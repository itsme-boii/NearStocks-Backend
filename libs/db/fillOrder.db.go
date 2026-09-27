package db

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"
)

type FillOrderDB struct{}

func (*FillOrderDB) FillExists(createFillOrder FillOrderTable) bool {
	var count int64
	// Check if the fills order already exists in the database
	db.Raw(`SELECT COUNT(*) FROM fill_order_tables WHERE txn_hash = ? and index = ?`, createFillOrder.TxnHash, createFillOrder.Index).Scan(&count)
	return count > 0
}

// Returns nil if entry already exists
// CreateFillOrder creates a new entry in the settle_pnl table, but skips if the same entry already exists
func (fodb *FillOrderDB) CreateFillOrder(createFillOrder FillOrderTable) (*FillOrderTable, error) {
	if fodb.FillExists(createFillOrder) {
		return nil, nil
	}

	// Entry does not exist, create a new one
	if err := db.Create(&createFillOrder).Error; err != nil {
		return nil, err
	}

	return &createFillOrder, nil
}

// GetMaxBlockNumber fetches the maximum block number from the settle_pnl table
func (*FillOrderDB) GetMaxBlockNumber() uint64 {
	var maxBlockNumber uint64
	err := db.Model(&FillOrderTable{}).Select("COALESCE(MAX(block_number), 0)").Row().Scan(&maxBlockNumber)
	if err != nil {
		log.Printf("Error fetching max block number from database: %v", err)
		return 0
	}
	return maxBlockNumber
}

// GetAllBySubaccountId fetches all entries from the fill_order table for a given subaccount ID.
func (*FillOrderDB) GetAllBySubaccountId(subaccountId string) *[]FillOrderTable {
	fills := []FillOrderTable{}
	return GetDBObjOrEmptyList(db.Where("sub_account_id = ?", subaccountId).Find(&fills), &fills)
}

// To get orders within a certain time range and for subaccount that has deposits from a certain source chain before or during that time period
func (*FillOrderDB) GetFillOrdersByTimeRangeAndDepositDate(sourceChainID uint64, startTimeUnix, endTimeUnix int64) *[]FillOrderTable {
	fills := []FillOrderTable{}
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	subaccounts := []string{}
	db.Table("subaccount_last_deposit_tables").Where("source_chain_id = ? AND last_deposit_date < to_timestamp(?) AND subaccount_id != ?", sourceChainID, endTimeUnix, ammSubaccountID).Pluck("subaccount_id", &subaccounts)

	if len(subaccounts) == 0 {
		return &fills
	}
	return GetDBObjOrEmptyList(db.Where("sub_account_id IN ? AND created_at BETWEEN to_timestamp(?) AND to_timestamp(?)", subaccounts, startTimeUnix, endTimeUnix).Find(&fills), &fills)
}

func (*FillOrderDB) GetDailyRealizedPnL(subaccountID string, specificDate time.Time) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	startOfDay := specificDate.UTC().Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

	var pnlData []struct {
		ProductID   uint
		RealisedPnl ctypes.BigInt
	}
	if err := db.Model(&FillOrderTable{}).
		Select("product_id, SUM(CAST(realised_pnl AS NUMERIC)) AS realised_pnl").
		Where("sub_account_id = ? AND created_at BETWEEN ? AND ?", subaccountID, startOfDay, endOfDay).
		Group("product_id").
		Scan(&pnlData).Error; err != nil {
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)
	for _, pnl := range pnlData {
		if currentPnl, exists := pnLPerToken[pnl.ProductID]; exists {
			pnLPerToken[pnl.ProductID] = currentPnl.Add(pnl.RealisedPnl)
		} else {
			pnLPerToken[pnl.ProductID] = pnl.RealisedPnl
		}
		totalRealizedPnL = totalRealizedPnL.Add(pnl.RealisedPnl)
	}

	return pnLPerToken, totalRealizedPnL, nil
}

func (*FillOrderDB) GetDailyFees(brokerId uint, startDate ...time.Time) (map[string][]interface{}, map[uint]map[string][]interface{}, error) {
	var startOfPeriod time.Time
	var dayCount int
	now := time.Now().UTC().Truncate(24 * time.Hour)

	if len(startDate) > 0 {
		startOfPeriod = startDate[0]
		dayCount = int(now.Sub(startOfPeriod).Hours()/24) + 1
	} else {
		startOfPeriod = now.AddDate(0, 0, -29)
		dayCount = 30
	}

	aggregatedFees := map[string][]interface{}{
		"date":        make([]interface{}, dayCount),
		"funding_fee": make([]interface{}, dayCount),
		"trading_fee": make([]interface{}, dayCount),
	}
	productFees := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	endOfPeriod := now.Add(24 * time.Hour).Add(-time.Nanosecond)

	var aggregatedData []struct {
		Date        time.Time
		FundingFees ctypes.BigInt
		TradingFees ctypes.BigInt
	}
	err := db.Raw(`
        SELECT 
            DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
            SUM(CAST(funding_fees AS NUMERIC)) AS funding_fees,
            SUM(CAST(fee_amount AS NUMERIC)) AS trading_fees
        FROM 
            fill_order_tables
        WHERE 
            created_at BETWEEN ? AND ? AND broker_id = ?
            AND sub_account_id != ?
        GROUP BY 
            date
        ORDER BY 
            date DESC
    `, startOfPeriod, endOfPeriod, brokerId, ammSubaccountID).Scan(&aggregatedData).Error

	if err != nil {
		return nil, nil, err
	}

	for i := 0; i < dayCount; i++ {
		date := now.AddDate(0, 0, -i)
		aggregatedFees["date"][i] = date
		aggregatedFees["funding_fee"][i] = "0"
		aggregatedFees["trading_fee"][i] = "0"
	}

	for _, dayData := range aggregatedData {
		dayIndex := int(now.Sub(dayData.Date).Hours() / 24)
		if dayIndex < dayCount {
			aggregatedFees["funding_fee"][dayIndex] = dayData.FundingFees.String()
			aggregatedFees["trading_fee"][dayIndex] = dayData.TradingFees.String()
		}
	}

	var productData []struct {
		Date        time.Time
		ProductID   uint
		FundingFees ctypes.BigInt
		TradingFees ctypes.BigInt
	}
	err = db.Raw(`
        SELECT 
            DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
            product_id,
            SUM(CAST(funding_fees AS NUMERIC)) AS funding_fees,
            SUM(CAST(fee_amount AS NUMERIC)) AS trading_fees
        FROM 
            fill_order_tables
        WHERE 
            created_at BETWEEN ? AND ? AND broker_id = ?
            AND sub_account_id != ?
        GROUP BY 
            date, product_id
        ORDER BY 
            date DESC
    `, startOfPeriod, endOfPeriod, brokerId, ammSubaccountID).Scan(&productData).Error

	if err != nil {
		return nil, nil, err
	}

	for _, data := range productData {
		if _, exists := productFees[data.ProductID]; !exists {
			productFees[data.ProductID] = map[string][]interface{}{
				"funding_fee": make([]interface{}, dayCount),
				"trading_fee": make([]interface{}, dayCount),
			}
			for i := 0; i < dayCount; i++ {
				productFees[data.ProductID]["funding_fee"][i] = "0"
				productFees[data.ProductID]["trading_fee"][i] = "0"
			}
		}
	}

	for _, dayData := range productData {
		dayIndex := int(now.Sub(dayData.Date).Hours() / 24)
		if dayIndex < dayCount {
			productFees[dayData.ProductID]["funding_fee"][dayIndex] = dayData.FundingFees.String()
			productFees[dayData.ProductID]["trading_fee"][dayIndex] = dayData.TradingFees.String()
		}
	}

	// Manual fee data for specific dates (only for broker ID 1)
	if brokerId == 1 {
		manualFees := map[time.Time]string{
			time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC): "1811231018382918867194",
			time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC): "1627834818382918867194",
			time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC): "1498936286403948304061",
			time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC): "1301287026894111241241",
			time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC): "1291782326894111241241",
			time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC): "1301287026894111241241",
			time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC): "1291782326894111241241",
			time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC): "1862936286403948304061",
			time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC): "1583561456278654321291",
			time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC): "1483561456278654321291",
			time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC): "1602231456278654321291",
			time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC): "1227834818382918867194",
			time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC): "2056147921190945642275",
			time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC): "1733561456278654321291",
			time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC): "1665561456278654321291",
			time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC): "10000",
		}

		for i, date := range aggregatedFees["date"] {
			if fee, exists := manualFees[date.(time.Time)]; exists {
				aggregatedFees["trading_fee"][i] = fee
			}
		}
	}

	return aggregatedFees, productFees, nil
}

func (*FillOrderDB) GetCumulativeTradingFees(endDate time.Time, brokerId uint) (string, error) {
	var cumulativeFees float64
	startDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS(CAST(fee_amount AS NUMERIC))) / 1e18, 0) AS cumulative_fees
		FROM 
			fill_order_tables
		WHERE 
			created_at AT TIME ZONE 'UTC' BETWEEN ? AND ? AND sub_account_id != ? AND broker_id = ?
	`, startDate, endDate, ammSubaccountID, brokerId).Scan(&cumulativeFees).Error

	if err != nil {
		xlog.Errorf("Error while fetching cumulative trading fees: %v", err)
		return "0", err
	}

	october31 := time.Date(2024, 10, 31, 0, 0, 0, 0, time.UTC)
	november9 := time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC)
	march02 := time.Date(2024, 03, 02, 0, 0, 0, 0, time.UTC)
	march03 := time.Date(2024, 03, 03, 0, 0, 0, 0, time.UTC)
	april25 := time.Date(2025, 04, 25, 0, 0, 0, 0, time.UTC)
	april26 := time.Date(2025, 04, 26, 0, 0, 0, 0, time.UTC)
	april27 := time.Date(2025, 04, 27, 0, 0, 0, 0, time.UTC)
	april20 := time.Date(2025, 04, 20, 0, 0, 0, 0, time.UTC)
	may04 := time.Date(2025, 05, 04, 0, 0, 0, 0, time.UTC)
	may09 := time.Date(2025, 05, 9, 0, 0, 0, 0, time.UTC)
	may10 := time.Date(2025, 05, 10, 0, 0, 0, 0, time.UTC)
	may11 := time.Date(2025, 05, 11, 0, 0, 0, 0, time.UTC)
	may17 := time.Date(2025, 05, 17, 0, 0, 0, 0, time.UTC)
	may18 := time.Date(2025, 05, 18, 0, 0, 0, 0, time.UTC)
	may24 := time.Date(2025, 05, 24, 0, 0, 0, 0, time.UTC)
	may25 := time.Date(2025, 05, 25, 0, 0, 0, 0, time.UTC)
	june01 := time.Date(2025, 06, 01, 0, 0, 0, 0, time.UTC)
	june02 := time.Date(2025, 06, 02, 0, 0, 0, 0, time.UTC)
	june04 := time.Date(2025, 06, 04, 0, 0, 0, 0, time.UTC)
	june05 := time.Date(2025, 06, 05, 0, 0, 0, 0, time.UTC)
	june06 := time.Date(2025, 06, 06, 0, 0, 0, 0, time.UTC)
	june07 := time.Date(2025, 06, 07, 0, 0, 0, 0, time.UTC)
	june10 := time.Date(2025, 06, 10, 0, 0, 0, 0, time.UTC)
	june11 := time.Date(2025, 06, 11, 0, 0, 0, 0, time.UTC)
	june12 := time.Date(2025, 06, 12, 0, 0, 0, 0, time.UTC)
	june13 := time.Date(2025, 06, 13, 0, 0, 0, 0, time.UTC)
	june14 := time.Date(2025, 06, 14, 0, 0, 0, 0, time.UTC)
	june15 := time.Date(2025, 06, 15, 0, 0, 0, 0, time.UTC)
	june17 := time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC)
	june18 := time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC)
	june19 := time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC)
	june20 := time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC)
	june21 := time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC)
	june22 := time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC)
	june23 := time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC)
	june24 := time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC)
	june26 := time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC)
	june27 := time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC)
	june28 := time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC)
	june29 := time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC)
	june30 := time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC)
	july01 := time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC)
	july03 := time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC)
	july04 := time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC)
	july05 := time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC)
	july06 := time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC)
	july07 := time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC)
	july10 := time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC)
	july11 := time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC)
	july12 := time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC)
	july27 := time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC)

	// Manual cumulative fee adjustments only for broker ID 1
	if brokerId == 1 {
		// these are for missed dates for internal subaccount fees
		if endDate.After(october31) {
			cumulativeFees += 4546.88
		}
		if endDate.After(november9) {
			cumulativeFees += 3141.51
		}

		if endDate.After(march02) {
			cumulativeFees += 11062.51
		}
		if endDate.After(march03) {
			cumulativeFees += 12556.38
		}
		if endDate.After(april25) {
			cumulativeFees += 2227.83
		}
		if endDate.After(april26) {
			cumulativeFees += 2862.94
		}
		if endDate.After(april27) {
			cumulativeFees += 2056.15
		}
		if endDate.After(may04) {
			cumulativeFees += 2583.56
		}
		if endDate.After(may09) {
			cumulativeFees += 3602.23
		}
		if endDate.After(may10) {
			cumulativeFees += 2301.29
		}
		if endDate.After(may11) {
			cumulativeFees += 2602.23
		}

		if endDate.After(april20) {
			cumulativeFees += 2492.67
		}

		if endDate.After(may17) {
			cumulativeFees += 2583.56
		}
		if endDate.After(may18) {
			cumulativeFees += 1765.56
		}
		if endDate.After(may24) {
			cumulativeFees += 2056.15
		}
		if endDate.After(may25) {
			cumulativeFees += 2227.83
		}
		if endDate.After(june01) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june02) {
			cumulativeFees += 2862.94
		}
		if endDate.After(june04) {
			cumulativeFees += 2498.32
		}
		if endDate.After(june05) {
			cumulativeFees += 1733.56
		}
		if endDate.After(june06) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june07) {
			cumulativeFees += 2583.56
		}
		if endDate.After(june10) {
			cumulativeFees += 2227.83
		}
		if endDate.After(june11) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june12) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june13) {
			cumulativeFees += 2056.15
		}
		if endDate.After(june14) {
			cumulativeFees += 1765.56
		}
		if endDate.After(june15) {
			cumulativeFees += 1827.83
		}
		if endDate.After(june17) {
			cumulativeFees += 1811.23
		}
		if endDate.After(june18) {
			cumulativeFees += 1627.83
		}
		if endDate.After(june19) {
			cumulativeFees += 1583.56
		}
		if endDate.After(june20) {
			cumulativeFees += 1498.94
		}
		if endDate.After(june21) {
			cumulativeFees += 1733.56
		}
		if endDate.After(june22) {
			cumulativeFees += 1227.83
		}
		if endDate.After(june23) {
			cumulativeFees += 1301.29
		}
		if endDate.After(june24) {
			cumulativeFees += 1291.78
		}
		if endDate.After(june26) {
			cumulativeFees += 1583.56
		}
		if endDate.After(june27) {
			cumulativeFees += 1227.83
		}
		if endDate.After(june28) {
			cumulativeFees += 1301.29
		}
		if endDate.After(june29) {
			cumulativeFees += 1291.78
		}
		if endDate.After(june30) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july01) {
			cumulativeFees += 1862.94
		}
		if endDate.After(july03) {
			cumulativeFees += 1583.56
		}
		if endDate.After(july04) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july05) {
			cumulativeFees += 1862.94
		}
		if endDate.After(july06) {
			cumulativeFees += 1602.23
		}
		if endDate.After(july07) {
			cumulativeFees += 1227.83
		}
		if endDate.After(july10) {
			cumulativeFees += 2056.15
		}
		if endDate.After(july11) {
			cumulativeFees += 1733.56
		}
		if endDate.After(july12) {
			cumulativeFees += 1665.56
		}
		if endDate.After(july27) {
			cumulativeFees += 0
		}
	}
	cumulativeFeesString := strconv.FormatFloat(cumulativeFees, 'f', 2, 64)

	return cumulativeFeesString, nil
}

func (*FillOrderDB) GetCumulativeVolume(endDate time.Time, brokerId uint) (string, error) {
	var cumulativeVolume float64
	startDate := time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC)
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS(CAST(quote_delta AS NUMERIC))) / 1e18, 0) AS cumulative_volume
		FROM 
			fill_order_tables
		WHERE 
			created_at AT TIME ZONE 'UTC' BETWEEN ? AND ? AND sub_account_id != ? AND broker_id = ?
	`, startDate, endDate, ammSubaccountID, brokerId).Scan(&cumulativeVolume).Error

	if err != nil {
		xlog.Errorf("Error while fetching cumulative volume: %v", err)
		return "0", err
	}

	october31 := time.Date(2024, 10, 31, 0, 0, 0, 0, time.UTC)
	november9 := time.Date(2024, 11, 9, 0, 0, 0, 0, time.UTC)
	feb19 := time.Date(2025, 02, 19, 0, 0, 0, 0, time.UTC)
	march02 := time.Date(2024, 03, 2, 0, 0, 0, 0, time.UTC)
	march03 := time.Date(2024, 03, 3, 0, 0, 0, 0, time.UTC)
	april25 := time.Date(2025, 04, 25, 0, 0, 0, 0, time.UTC)
	april26 := time.Date(2025, 04, 26, 0, 0, 0, 0, time.UTC)
	april27 := time.Date(2025, 04, 27, 0, 0, 0, 0, time.UTC)
	april20 := time.Date(2025, 04, 20, 0, 0, 0, 0, time.UTC)
	may04 := time.Date(2025, 05, 04, 0, 0, 0, 0, time.UTC)
	may09 := time.Date(2025, 05, 9, 0, 0, 0, 0, time.UTC)
	may10 := time.Date(2025, 05, 10, 0, 0, 0, 0, time.UTC)
	may11 := time.Date(2025, 05, 11, 0, 0, 0, 0, time.UTC)
	may17 := time.Date(2025, 05, 17, 0, 0, 0, 0, time.UTC)
	may18 := time.Date(2025, 05, 18, 0, 0, 0, 0, time.UTC)
	may24 := time.Date(2025, 05, 24, 0, 0, 0, 0, time.UTC)
	may25 := time.Date(2025, 05, 25, 0, 0, 0, 0, time.UTC)
	june01 := time.Date(2025, 06, 01, 0, 0, 0, 0, time.UTC)
	june02 := time.Date(2025, 06, 02, 0, 0, 0, 0, time.UTC)
	june04 := time.Date(2025, 06, 04, 0, 0, 0, 0, time.UTC)
	june05 := time.Date(2025, 06, 05, 0, 0, 0, 0, time.UTC)
	june06 := time.Date(2025, 06, 06, 0, 0, 0, 0, time.UTC)
	june07 := time.Date(2025, 06, 07, 0, 0, 0, 0, time.UTC)
	june10 := time.Date(2025, 06, 10, 0, 0, 0, 0, time.UTC)
	june11 := time.Date(2025, 06, 11, 0, 0, 0, 0, time.UTC)
	june12 := time.Date(2025, 06, 12, 0, 0, 0, 0, time.UTC)
	june13 := time.Date(2025, 06, 13, 0, 0, 0, 0, time.UTC)
	june14 := time.Date(2025, 06, 14, 0, 0, 0, 0, time.UTC)
	june15 := time.Date(2025, 06, 15, 0, 0, 0, 0, time.UTC)
	june17 := time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC)
	june18 := time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC)
	june19 := time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC)
	june20 := time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC)
	june21 := time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC)
	june22 := time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC)
	june23 := time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC)
	june24 := time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC)
	june26 := time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC)
	june27 := time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC)
	june28 := time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC)
	june29 := time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC)
	june30 := time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC)
	july01 := time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC)
	july03 := time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC)
	july04 := time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC)
	july05 := time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC)
	july06 := time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC)
	july07 := time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC)
	july10 := time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC)
	july11 := time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC)
	july12 := time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC)
	july27 := time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC)

	// Manual cumulative volume adjustments only for broker ID 1
	if brokerId == 1 {
		// these are for missed dates for internal subaccount volumes
		if endDate.After(october31) {
			cumulativeVolume += 9093763
		}
		if endDate.After(november9) {
			cumulativeVolume += 6283012
		}
		if endDate.After(feb19) {
			cumulativeVolume += 12167123
		}
		if endDate.After(march02) {
			cumulativeVolume += 20434700
		}
		if endDate.After(march03) {
			cumulativeVolume += 24105552
		}
		if endDate.After(april25) {
			cumulativeVolume += 10113154
		}
		if endDate.After(april26) {
			cumulativeVolume += 9112532
		}
		if endDate.After(april27) {
			cumulativeVolume += 11112310
		}
		if endDate.After(may04) {
			cumulativeVolume += 25835614
		}
		if endDate.After(may09) {
			cumulativeVolume += 18112310
		}
		if endDate.After(may10) {
			cumulativeVolume += 23012870
		}
		if endDate.After(may11) {
			cumulativeVolume += 20611472
		}
		if endDate.After(april20) {
			cumulativeVolume += 24926733
		}
		if endDate.After(may17) {
			cumulativeVolume += 15835614
		}
		if endDate.After(may18) {
			cumulativeVolume += 12926733
		}
		if endDate.After(june01) {
			cumulativeVolume += 15856145
		}
		if endDate.After(june02) {
			cumulativeVolume += 18629363
		}
		if endDate.After(june04) {
			cumulativeVolume += 24983232
		}
		if endDate.After(june05) {
			cumulativeVolume += 17335614
		}
		if endDate.After(june06) {
			cumulativeVolume += 15835614
		}
		if endDate.After(june07) {
			cumulativeVolume += 18629363
		}
		if endDate.After(june10) {
			cumulativeVolume += 18278348
		}
		if endDate.After(june11) {
			cumulativeVolume += 20561479
		}
		if endDate.After(june12) {
			cumulativeVolume += 17917823
		}
		if endDate.After(june13) {
			cumulativeVolume += 205615
		}
		if endDate.After(june14) {
			cumulativeVolume += 17655614
		}
		if endDate.After(june26) {
			cumulativeVolume += 15835614
		}
		if endDate.After(may24) {
			cumulativeVolume += 20561500
		}
		if endDate.After(may25) {
			cumulativeVolume += 22278300
		}
		if endDate.After(june15) {
			cumulativeVolume += 18278300
		}
		if endDate.After(june17) {
			cumulativeVolume += 18112300
		}
		if endDate.After(june18) {
			cumulativeVolume += 16278300
		}
		if endDate.After(june19) {
			cumulativeVolume += 15835614
		}
		if endDate.After(june20) {
			cumulativeVolume += 14989400
		}
		if endDate.After(june21) {
			cumulativeVolume += 17335614
		}
		if endDate.After(june22) {
			cumulativeVolume += 12278300
		}
		if endDate.After(june23) {
			cumulativeVolume += 13012900
		}
		if endDate.After(june24) {
			cumulativeVolume += 12917800
		}
		if endDate.After(june27) {
			cumulativeVolume += 12278300
		}
		if endDate.After(june28) {
			cumulativeVolume += 13012900
		}
		if endDate.After(june29) {
			cumulativeVolume += 12917800
		}
		if endDate.After(june30) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july01) {
			cumulativeVolume += 18629400
		}
		if endDate.After(july03) {
			cumulativeVolume += 15835614
		}
		if endDate.After(july04) {
			cumulativeVolume += 14835614
		}
		if endDate.After(july05) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july06) {
			cumulativeVolume += 16022314
		}
		if endDate.After(july07) {
			cumulativeVolume += 12278300
		}
		if endDate.After(july10) {
			cumulativeVolume += 20561500
		}
		if endDate.After(july11) {
			cumulativeVolume += 17335614
		}
		if endDate.After(july12) {
			cumulativeVolume += 16655614
		}
		if endDate.After(july27) {
			cumulativeVolume += 53245123
		}
	}
	cumulativeVolumeString := strconv.FormatFloat(cumulativeVolume, 'f', 2, 64)

	return cumulativeVolumeString, nil
}

func (*FillOrderDB) GetDailyOverallVolumes(brokerId uint, startDate ...time.Time) ([]DailyData, error) {
	var startOfPeriod time.Time
	if len(startDate) > 0 {
		startOfPeriod = startDate[0]
	} else {
		startOfPeriod = time.Now().AddDate(0, 0, -30).UTC()
	}
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	var results []DailyData

	err := db.Raw(`
		SELECT 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date, 
			FLOOR(SUM(ABS(CAST(quote_delta AS NUMERIC)))/1e18) AS count
		FROM 
			fill_order_tables
		WHERE 
			created_at >= ?  
			AND sub_account_id != ? AND broker_id = ?
		GROUP BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
		ORDER BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') ASC
		`, startOfPeriod, ammSubaccountID, brokerId).Scan(&results).Error

	if err != nil {
		xlog.Errorf("Error querying daily total volume: %v", err)
		return nil, err
	}
	// Override volumes for specific dates (only for broker ID 1, only from June 16th onwards)
	if brokerId == 1 {
		overrideVolumes := map[string]int64{
			"2025-06-17": 18112310,
			"2025-06-18": 16278348,
			"2025-06-19": 15835614,
			"2025-06-20": 14989362,
			"2025-06-21": 17335614,
			"2025-06-22": 12278348,
			"2025-06-23": 13012870,
			"2025-06-24": 12917823,
			"2025-06-26": 15835614,
			"2025-06-27": 12278348,
			"2025-06-28": 13012870,
			"2025-06-29": 12917823,
			"2025-06-30": 17335614,
			"2025-07-01": 18629363,
			"2025-07-03": 15835614,
			"2025-07-04": 14835614,
			"2025-07-05": 17335614,
			"2025-07-06": 16022314,
			"2025-07-07": 12278348,
			"2025-07-10": 20561479,
			"2025-07-11": 17335614,
			"2025-07-12": 16655614,
			"2025-07-27": 53245123,
		}

		for i, data := range results {
			dateStr := data.Date.Format("2006-01-02")
			if volume, exists := overrideVolumes[dateStr]; exists {
				results[i].Count = volume
			}
		}
	}
	return results, nil
}

// GetDailyVolumes retrieves the daily volume for each productID and the total lifetime volume
func (*FillOrderDB) GetDailyVolumes(brokerId uint, startDate ...time.Time) (map[uint]map[string][]interface{}, ctypes.BigInt, []time.Time, error) {
	productVolumes := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	now := time.Now().UTC().Truncate(24 * time.Hour)
	var dayCount int

	if len(startDate) > 0 {
		dayCount = int(now.Sub(startDate[0]).Hours()/24) + 1
	} else {
		dayCount = 30
	}

	dates := make([]time.Time, dayCount)
	for i := 0; i < dayCount; i++ {
		dates[i] = now.AddDate(0, 0, -i)
	}

	var lifetimeVolumeData struct {
		Volume ctypes.BigInt
	}
	err := db.Raw(`
        SELECT 
            SUM(ABS(CAST(quote_delta AS NUMERIC))) as volume
        FROM 
            fill_order_tables
        WHERE 
            sub_account_id NOT IN (?) AND broker_id = ?
    `, ammSubaccountID, brokerId).Scan(&lifetimeVolumeData).Error

	if err != nil {
		xlog.Errorf("Error querying lifetime volume: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}

	// Add manual volume adjustment only for broker ID 1
	totalLifetimeVolume := lifetimeVolumeData.Volume
	if brokerId == 1 {
		additionalVolume := new(big.Int)
		additionalVolume.SetString("15376775000000000000000000", 10)
		if totalLifetimeVolume.Val == nil {
			totalLifetimeVolume.Val = new(big.Int)
		}
		totalLifetimeVolume.Val.Add(totalLifetimeVolume.Val, additionalVolume)
	}

	var volumeData []struct {
		ProductID uint
		Volume    ctypes.BigInt
		Date      time.Time
	}
	endDate := dates[0].Add(24 * time.Hour)
	err = db.Raw(`
            SELECT 
                product_id, 
                SUM(ABS(CAST(quote_delta AS NUMERIC))) as volume,
                DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date
            FROM 
                fill_order_tables
            WHERE 
                created_at BETWEEN ? AND ? AND broker_id = ?
                AND sub_account_id NOT IN (?)
            GROUP BY 
                product_id, DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
            ORDER BY 
                date DESC
        `, dates[29], endDate, brokerId, ammSubaccountID).Scan(&volumeData).Error

	if err != nil {
		xlog.Errorf("Error querying product volumes: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}

	latestDate := dates[0]

	// Manual volume adjustments for specific dates (only for broker ID 1, only from June 16th onwards)
	var targetDates map[time.Time]*big.Float
	var targetDateVolumes map[time.Time]map[uint]*big.Int

	if brokerId == 1 {
		targetDates = map[time.Time]*big.Float{
			time.Date(2025, 06, 17, 0, 0, 0, 0, time.UTC): big.NewFloat(18112310000000000000000000),
			time.Date(2025, 06, 18, 0, 0, 0, 0, time.UTC): big.NewFloat(16278348000000000000000000),
			time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 06, 20, 0, 0, 0, 0, time.UTC): big.NewFloat(14989362000000000000000000),
			time.Date(2025, 06, 21, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 06, 22, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 06, 23, 0, 0, 0, 0, time.UTC): big.NewFloat(13012870000000000000000000),
			time.Date(2025, 06, 24, 0, 0, 0, 0, time.UTC): big.NewFloat(12917823000000000000000000),
			time.Date(2025, 06, 26, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 06, 27, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 06, 28, 0, 0, 0, 0, time.UTC): big.NewFloat(13012870000000000000000000),
			time.Date(2025, 06, 29, 0, 0, 0, 0, time.UTC): big.NewFloat(12917823000000000000000000),
			time.Date(2025, 06, 30, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 01, 0, 0, 0, 0, time.UTC): big.NewFloat(18629363000000000000000000),
			time.Date(2025, 07, 03, 0, 0, 0, 0, time.UTC): big.NewFloat(15835614000000000000000000),
			time.Date(2025, 07, 04, 0, 0, 0, 0, time.UTC): big.NewFloat(14835614000000000000000000),
			time.Date(2025, 07, 05, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 06, 0, 0, 0, 0, time.UTC): big.NewFloat(16022314000000000000000000),
			time.Date(2025, 07, 07, 0, 0, 0, 0, time.UTC): big.NewFloat(12278348000000000000000000),
			time.Date(2025, 07, 10, 0, 0, 0, 0, time.UTC): big.NewFloat(20561479000000000000000000),
			time.Date(2025, 07, 11, 0, 0, 0, 0, time.UTC): big.NewFloat(17335614000000000000000000),
			time.Date(2025, 07, 12, 0, 0, 0, 0, time.UTC): big.NewFloat(16655614000000000000000000),
			time.Date(2025, 07, 27, 0, 0, 0, 0, time.UTC): big.NewFloat(53245123000000000000000000),
		}

		targetDateVolumes = make(map[time.Time]map[uint]*big.Int)
		for targetDate := range targetDates {
			targetDateVolumes[targetDate] = make(map[uint]*big.Int)
		}
	}

	for _, data := range volumeData {
		dateIndex := int(latestDate.Sub(data.Date.UTC()).Hours() / 24)
		if dateIndex < 0 || dateIndex >= dayCount {
			continue
		}

		if _, ok := productVolumes[data.ProductID]; !ok {
			productVolumes[data.ProductID] = map[string][]interface{}{
				"volume": make([]interface{}, dayCount),
			}
			zeroBigInt := ctypes.NewBigInt(big.NewInt(0))
			for j := 0; j < dayCount; j++ {
				productVolumes[data.ProductID]["volume"][j] = (&zeroBigInt).String()
			}
		}
		productVolumes[data.ProductID]["volume"][dateIndex] = (&data.Volume).String()

		if volumes, exists := targetDateVolumes[data.Date.UTC()]; exists {
			volumes[data.ProductID] = data.Volume.Val
		}
	}
	// Apply manual volume adjustments only for broker ID 1
	if brokerId == 1 && targetDates != nil {
		// Product IDs for BTC, ETH, and SOL
		btcEthSolIDs := map[uint]bool{3: true, 1: true, 5: true}

		for targetDate, targetVolume := range targetDates {
			currentSum := big.NewInt(0)

			// Only consider BTC, ETH, and SOL
			for productID, volume := range targetDateVolumes[targetDate] {
				if btcEthSolIDs[productID] {
					currentSum.Add(currentSum, volume)
				}
			}

			// Adding manual volume adjustment for specific dates
			if currentSum.Cmp(big.NewInt(0)) > 0 {
				increaseFactor := new(big.Float).Quo(targetVolume, new(big.Float).SetInt(currentSum))
				for productID, volume := range targetDateVolumes[targetDate] {
					if btcEthSolIDs[productID] {
						adjustedVolume := new(big.Float).Mul(new(big.Float).SetInt(volume), increaseFactor)
						adjustedVolumeInt, _ := adjustedVolume.Int(nil)

						adjustedBigInt := ctypes.NewBigInt(adjustedVolumeInt)
						dateIndex := int(latestDate.Sub(targetDate).Hours() / 24)
						productVolumes[productID]["volume"][dateIndex] = (&adjustedBigInt).String()
					}
				}
			}
		}
	}

	return productVolumes, totalLifetimeVolume, dates, nil
}

func (*FillOrderDB) GetDailyVolumesInternal() (map[uint]map[string][]interface{}, ctypes.BigInt, []time.Time, error) {
	productVolumes := make(map[uint]map[string][]interface{})
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM subaccount and internal subaccounts into one slice
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Initialize dates array for the last 30 days from today
	dates := make([]time.Time, 30)
	for i := 0; i < 30; i++ {
		dates[i] = time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -i)
	}

	// Calculate the total lifetime volume excluding specified subaccount IDs
	var lifetimeVolumeData struct {
		Volume ctypes.BigInt
	}
	err := db.Raw(`
		SELECT 
			SUM(ABS(CAST(quote_delta AS NUMERIC))) as volume
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id NOT IN (?)
	`, excludeSubaccountIDs).Scan(&lifetimeVolumeData).Error

	if err != nil {
		xlog.Errorf("Error querying lifetime volume: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}
	totalLifetimeVolume := lifetimeVolumeData.Volume

	// Query product volumes grouped by day for the last 30 days
	var volumeData []struct {
		ProductID uint
		Volume    ctypes.BigInt
		Date      time.Time
	}
	endDate := dates[0].Add(24 * time.Hour)
	err = db.Raw(`
			SELECT 
				product_id, 
				SUM(ABS(CAST(quote_delta AS NUMERIC))) as volume,
				DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') as date
			FROM 
				fill_order_tables
			WHERE 
				created_at BETWEEN ? AND ? 
				AND sub_account_id NOT IN (?)
			GROUP BY 
				product_id, DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
			ORDER BY 
				date DESC
		`, dates[29], endDate, excludeSubaccountIDs).Scan(&volumeData).Error

	if err != nil {
		xlog.Errorf("Error querying product volumes: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), nil, err
	}

	// Set the reference date to the latest date in the data (i.e., dates[0] or today)
	latestDate := dates[0]

	// Process the query results and map data to productVolumes
	for _, data := range volumeData {
		// Calculate the date index based on latestDate
		dateIndex := int(latestDate.Sub(data.Date.UTC()).Hours() / 24)
		if dateIndex < 0 || dateIndex >= 30 {
			continue // Skip out-of-range data
		}

		// Initialize volume array if it doesn't exist
		if _, ok := productVolumes[data.ProductID]; !ok {
			productVolumes[data.ProductID] = map[string][]interface{}{
				"volume": make([]interface{}, 30),
			}
			for j := 0; j < 30; j++ {
				productVolumes[data.ProductID]["volume"][j] = ctypes.NewBigInt(big.NewInt(0))
			}
		}
		productVolumes[data.ProductID]["volume"][dateIndex] = data.Volume.String()
	}

	return productVolumes, totalLifetimeVolume, dates, nil
}

func (*FillOrderDB) GetTotalVolume(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	startOfPeriod := time.Date(2024, 8, 14, 15, 52, 43, 0, time.UTC) // Specified start time
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Struct to store the total volume
	var volumeData []struct {
		ProductID uint
		Volume    ctypes.BigInt
	}

	// Query to get the total volume since the specified timestamp, excluding a specific sub_account_id
	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(ABS(CAST(quote_delta AS NUMERIC))) AS volume
		FROM 
			fill_order_tables
		WHERE 
			created_at > ? 
			AND sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
				product_id
	`, startOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&volumeData).Error

	if err != nil {
		xlog.Errorf("Error querying total volume since the specified timestamp: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalVolume := ctypes.NewBigInt(big.NewInt(0))
	volumePerMarket := make(map[uint]ctypes.BigInt)
	for _, marketVolume := range volumeData {
		if currentFee, exists := volumePerMarket[marketVolume.ProductID]; exists {
			volumePerMarket[marketVolume.ProductID] = currentFee.Add(marketVolume.Volume)
		} else {
			volumePerMarket[marketVolume.ProductID] = marketVolume.Volume
		}
		totalVolume = totalVolume.Add(marketVolume.Volume)
	}

	return volumePerMarket, totalVolume, nil
}

func (*FillOrderDB) GetDailyActiveTraders(brokerId uint) ([]map[string]interface{}, error) {
	// Define the struct within the function scope
	type DailyActiveTrader struct {
		Count int       `json:"count"`
		Date  time.Time `json:"date"`
	}

	// Collect exclusion subaccount IDs from environment variables
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM and internal subaccounts
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Slice to hold query results
	var results []DailyActiveTrader

	// Execute the query to get daily active traders
	err := db.Raw(`
		SELECT 
			COUNT(DISTINCT user_address) AS count,
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC')
		ORDER BY 
			date DESC
		LIMIT 30
	`, excludeSubaccountIDs, brokerId).Scan(&results).Error

	if err != nil {
		return nil, fmt.Errorf("error querying daily active traders: %v", err)
	}

	// Convert results to []map[string]interface{} for flexibility
	dailyActiveTraders := make([]map[string]interface{}, len(results))
	for i, record := range results {
		dailyActiveTraders[i] = map[string]interface{}{
			"count": record.Count,
			"date":  record.Date,
		}
	}

	return dailyActiveTraders, nil
}

func (*FillOrderDB) GetPerpUserStatistics(brokerId uint) (map[string]interface{}, error) {
	// Collect exclusion subaccount IDs from environment variables
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM and internal subaccounts
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Variables to store results
	var last24hUsers int
	var totalUsers int

	// Query to get the number of unique users in the last 24 hours
	err := db.Raw(`
		SELECT COUNT(DISTINCT user_address)
		FROM fill_order_tables
		WHERE sub_account_id NOT IN (?) AND broker_id = ?
		AND created_at >= NOW() - INTERVAL '24 hours'
	`, excludeSubaccountIDs, brokerId).Scan(&last24hUsers).Error

	if err != nil {
		return nil, fmt.Errorf("error querying 24-hour users: %v", err)
	}

	// Query to get the total number of unique users
	err = db.Raw(`
		SELECT COUNT(DISTINCT user_address)
		FROM fill_order_tables
		WHERE sub_account_id NOT IN (?) AND broker_id = ?
	`, excludeSubaccountIDs, brokerId).Scan(&totalUsers).Error

	if err != nil {
		return nil, fmt.Errorf("error querying total users: %v", err)
	}

	// Return results as a map
	return map[string]interface{}{
		"24h_users":   last24hUsers,
		"total_users": totalUsers,
	}, nil
}

func (*FillOrderDB) DefillamaStats(brokerId uint, endTime time.Time) (ctypes.BigInt, ctypes.BigInt, error) {
	// Set the fixed start time (startOfPeriod) and calculate last 24 hours from the provided endTime
	startOfPeriod := time.Date(2024, 8, 14, 15, 52, 43, 0, time.UTC) // Fixed start time
	last24Hours := endTime.Add(-24 * time.Hour)                      // Last 24 hours from endTime
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID

	// Variables to store the total volume and last 24-hour volume
	var totalVolumeStr, last24HourVolumeStr string

	// Query to get both total volume since the specified start time (startOfPeriod) and the volume in the last 24 hours
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(ABS(CAST(quote_delta AS NUMERIC))) FILTER (WHERE created_at >= ? AND created_at <= ?), '0') AS last_24_hour_volume,
			COALESCE(SUM(ABS(CAST(quote_delta AS NUMERIC))) FILTER (WHERE created_at >= ? AND created_at <= ?), '0') AS total_volume
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id != ?  AND broker_id = ?
	`, last24Hours, endTime, startOfPeriod, endTime, ammSubaccountID, brokerId).Row().Scan(&last24HourVolumeStr, &totalVolumeStr)

	if err != nil {
		xlog.Errorf("Error querying total and last 24-hour volume: %v", err)
		return ctypes.NewBigInt(big.NewInt(0)), ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Convert the string results to *big.Int
	totalVolume := new(big.Int)
	totalVolume.SetString(totalVolumeStr, 10)

	last24HourVolume := new(big.Int)
	last24HourVolume.SetString(last24HourVolumeStr, 10)

	return ctypes.NewBigInt(totalVolume), ctypes.NewBigInt(last24HourVolume), nil
}

func (*FillOrderDB) Get24hVolumes(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	productVolumes := make(map[uint]ctypes.BigInt)
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour) // Start time 24 hours ago
	endOfPeriod := now                        // End time is now
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Query to get the 24-hour volume for each product and the total volume in one go
	var productData []struct {
		ProductID uint
		Volume    ctypes.BigInt
	}
	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(ABS(CAST(quote_delta AS NUMERIC))) as volume
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&productData).Error

	if err != nil {
		xlog.Errorf("Error querying product volumes for the last 24 hours: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Initialize the total volume
	totalVolume := ctypes.NewBigInt(big.NewInt(0))

	// Populate the productVolumes map with the queried data and calculate the total volume
	for _, p := range productData {
		productVolumes[p.ProductID] = p.Volume

		// Add the current product volume to the total volume
		totalVolume = totalVolume.Add(p.Volume)
	}

	return productVolumes, totalVolume, nil
}

func (*FillOrderDB) Get24hRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	type PnlData struct {
		ProductID   uint
		RealisedPnl ctypes.BigInt
	}

	var ammPnlData, internalAccountPnlData []PnlData
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err1 := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realised_pnl AS NUMERIC)) as realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND sub_account_id = ? AND broker_id = ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, brokerId).Scan(&ammPnlData).Error

	if err1 != nil {
		xlog.Errorf("Error while fetching 24h realised PnL %v", err1)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err1
	}

	err2 := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realised_pnl AS NUMERIC)) as realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND sub_account_id IN (?) AND broker_id = ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod, internalSubaccountIDs, brokerId).Scan(&internalAccountPnlData).Error

	if err2 != nil {
		xlog.Errorf("Error while fetching 24h realised PnL %v", err2)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err2
	}

	// Create a map for internalAccountPnlData for fast lookup
	internalPnLMap := make(map[uint]ctypes.BigInt)
	for _, pnl := range internalAccountPnlData {
		internalPnLMap[pnl.ProductID] = pnl.RealisedPnl
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)
	for _, pnl := range ammPnlData {
		productId := pnl.ProductID
		ammProductPnl := pnl.RealisedPnl

		// Check if the productId exists in internalAccountPnlData
		if internalProductPnl, exists := internalPnLMap[productId]; exists {
			// If it exists, add the AMM and internal account PnL
			finalPnl := ammProductPnl.Add(internalProductPnl)
			pnLPerToken[productId] = finalPnl
			totalRealizedPnL = totalRealizedPnL.Add(finalPnl)
		} else {
			// If it doesn't exist, just add the AMM PnL
			pnLPerToken[productId] = ammProductPnl
			totalRealizedPnL = totalRealizedPnL.Add(ammProductPnl)
		}
	}

	return pnLPerToken, totalRealizedPnL, nil
}

func (*FillOrderDB) Get24hTradingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	var feeData []struct {
		ProductID uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(fee_amount AS NUMERIC)) as fee_amount
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&feeData).Error

	if err != nil {
		xlog.Errorf("Error executing query:", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFeeAmount := ctypes.NewBigInt(big.NewInt(0))
	feePerToken := make(map[uint]ctypes.BigInt)
	for _, fee := range feeData {
		if currentFee, exists := feePerToken[fee.ProductID]; exists {
			feePerToken[fee.ProductID] = currentFee.Add(fee.FeeAmount)
		} else {
			feePerToken[fee.ProductID] = fee.FeeAmount
		}
		totalFeeAmount = totalFeeAmount.Add(fee.FeeAmount)
	}

	return feePerToken, totalFeeAmount, nil
}

func (*FillOrderDB) Get24hFundingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	var feeData []struct {
		ProductID uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(funding_fees AS NUMERIC)) as fee_amount
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ? 
			AND sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&feeData).Error

	if err != nil {
		xlog.Errorf("Error executing query:", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFeeAmount := ctypes.NewBigInt(big.NewInt(0))
	feePerToken := make(map[uint]ctypes.BigInt)
	for _, fee := range feeData {
		if currentFee, exists := feePerToken[fee.ProductID]; exists {
			feePerToken[fee.ProductID] = currentFee.Add(fee.FeeAmount)
		} else {
			feePerToken[fee.ProductID] = fee.FeeAmount
		}
		totalFeeAmount = totalFeeAmount.Add(fee.FeeAmount)
	}

	return feePerToken, totalFeeAmount, nil
}

func (*FillOrderDB) GetTotalRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	type PnlData struct {
		ProductID   uint
		RealisedPnl ctypes.BigInt
	}

	var ammPnlData, internalAccountPnlData []PnlData

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err1 := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realised_pnl AS NUMERIC)) as realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id = ? AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			product_id
	`, ammSubaccountID, brokerId).Scan(&ammPnlData).Error

	if err1 != nil {
		xlog.Errorf("Error while fetching total realised PnL for AMM subaccount: %v", err1)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err1
	}

	err2 := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realised_pnl AS NUMERIC)) as realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			product_id
	`, internalSubaccountIDs, brokerId).Scan(&internalAccountPnlData).Error

	if err2 != nil {
		xlog.Errorf("Error while fetching total realised PnL %v", err2)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err2
	}

	// Create a map for internalAccountPnlData for fast lookup
	internalPnLMap := make(map[uint]ctypes.BigInt)
	for _, pnl := range internalAccountPnlData {
		internalPnLMap[pnl.ProductID] = pnl.RealisedPnl
	}

	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)
	for _, pnl := range ammPnlData {
		productId := pnl.ProductID
		ammProductPnl := pnl.RealisedPnl

		// Check if the productId exists in internalAccountPnlData
		if internalProductPnl, exists := internalPnLMap[productId]; exists {
			// If it exists, add the AMM and internal account PnL
			finalPnl := ammProductPnl.Add(internalProductPnl)
			pnLPerToken[productId] = finalPnl
			totalRealizedPnL = totalRealizedPnL.Add(finalPnl)
		} else {
			// If it doesn't exist, just add the AMM PnL
			pnLPerToken[productId] = ammProductPnl
			totalRealizedPnL = totalRealizedPnL.Add(ammProductPnl)
		}
	}

	return pnLPerToken, totalRealizedPnL, nil
}

func (*FillOrderDB) GetTotalUserRealisedPnL(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	type PnlData struct {
		ProductID   uint
		RealisedPnl ctypes.BigInt
	}

	var pnLData []PnlData

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	// Combine the AMM subaccount and internal subaccounts into one slice
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	// Perform the query
	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realised_pnl AS NUMERIC)) as realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id NOT IN (?) AND broker_id = ?
		GROUP BY 
			product_id
	`, excludeSubaccountIDs, brokerId).Scan(&pnLData).Error

	if err != nil {
		xlog.Errorf("Error while fetching total realised PnL: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	// Process the result and calculate total realized PnL
	totalRealizedPnL := ctypes.NewBigInt(big.NewInt(0))
	pnLPerToken := make(map[uint]ctypes.BigInt)

	for _, pnl := range pnLData {
		pnLPerToken[pnl.ProductID] = pnl.RealisedPnl
		totalRealizedPnL = totalRealizedPnL.Add(pnl.RealisedPnl)
	}

	return pnLPerToken, totalRealizedPnL, nil
}

func (*FillOrderDB) GetTotalTradingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	var tradingFeeData []struct {
		ProductID uint
		FeeAmount ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(fee_amount AS NUMERIC)) as fee_amount
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			product_id
	`, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&tradingFeeData).Error

	if err != nil {
		xlog.Errorf("Error while fetching total realised PnL for AMM subaccount: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalTradingFee := ctypes.NewBigInt(big.NewInt(0))
	tradingFeePerToken := make(map[uint]ctypes.BigInt)
	for _, tradingFee := range tradingFeeData {
		if currentFee, exists := tradingFeePerToken[tradingFee.ProductID]; exists {
			tradingFeePerToken[tradingFee.ProductID] = currentFee.Add(tradingFee.FeeAmount)
		} else {
			tradingFeePerToken[tradingFee.ProductID] = tradingFee.FeeAmount
		}
		totalTradingFee = totalTradingFee.Add(tradingFee.FeeAmount)
	}

	return tradingFeePerToken, totalTradingFee, nil
}

func (*FillOrderDB) GetTotalFundingFee(brokerId uint) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	var fundingFeeData []struct {
		ProductID   uint
		FundingFees ctypes.BigInt
	}

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	internalSubaccountIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	err := db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(funding_fees AS NUMERIC)) as funding_fees
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id != ?
			AND sub_account_id NOT IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
		GROUP BY 
			product_id
	`, ammSubaccountID, internalSubaccountIDs, brokerId).Scan(&fundingFeeData).Error

	if err != nil {
		xlog.Errorf("Error while fetching total realised PnL for AMM subaccount: %v", err)
		return nil, ctypes.NewBigInt(big.NewInt(0)), err
	}

	totalFundingFee := ctypes.NewBigInt(big.NewInt(0))
	fundingFeePerToken := make(map[uint]ctypes.BigInt)
	for _, fundingFee := range fundingFeeData {
		if currentFee, exists := fundingFeePerToken[fundingFee.ProductID]; exists {
			fundingFeePerToken[fundingFee.ProductID] = currentFee.Add(fundingFee.FundingFees)
		} else {
			fundingFeePerToken[fundingFee.ProductID] = fundingFee.FundingFees
		}
		totalFundingFee = totalFundingFee.Add(fundingFee.FundingFees)
	}

	return fundingFeePerToken, totalFundingFee, nil
}

// DEX2: Not filtering by brokerId as it feel like we don't really need to
func (*FillOrderDB) Get24hPriceRange() (map[uint]struct {
	MaxPrice ctypes.BigInt
	MinPrice ctypes.BigInt
}, error) {
	priceRanges := make(map[uint]struct {
		MaxPrice ctypes.BigInt
		MinPrice ctypes.BigInt
	})
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour) // Start time 24 hours ago
	endOfPeriod := now                        // End time is now

	// Query to get the max and min price for each product over the past 24 hours
	var priceData []struct {
		ProductID uint
		MaxPrice  ctypes.BigInt
		MinPrice  ctypes.BigInt
	}
	err := db.Raw(`
		SELECT 
			product_id, 
			MAX(CAST(price_x18 AS NUMERIC)) as max_price, 
			MIN(CAST(price_x18 AS NUMERIC)) as min_price
		FROM 
			fill_order_tables
		WHERE 
			created_at BETWEEN ? AND ?
		GROUP BY 
			product_id
	`, startOfPeriod, endOfPeriod).Scan(&priceData).Error

	if err != nil {
		xlog.Errorf("Error querying price ranges for the last 24 hours: %v", err)
		return nil, err
	}

	// Populate the priceRanges map with the queried data
	for _, p := range priceData {
		priceRanges[p.ProductID] = struct {
			MaxPrice ctypes.BigInt
			MinPrice ctypes.BigInt
		}{
			MaxPrice: p.MaxPrice,
			MinPrice: p.MinPrice,
		}
	}

	return priceRanges, nil
}

func (*FillOrderDB) GetTotalFeesIncludingInternal(brokerId uint) (map[string]string, error) {
	// Get AMM subaccount ID and deposit skip subaccounts
	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID
	depositSkipSubaccountIDs := strings.Split(os.Getenv("DEPOSIT_SUBACCOUNT_SKIP"), ",")

	// Combine all subaccount IDs to exclude
	excludeSubaccountIDs := append([]string{ammSubaccountID}, depositSkipSubaccountIDs...)

	// Struct to hold our results
	var result struct {
		TotalTradingFee  ctypes.BigInt `json:"total_trading_fee"`
		TotalFundingFee  ctypes.BigInt `json:"total_funding_fee"`
		TotalRealisedPnl ctypes.BigInt `json:"total_realised_pnl"`
	}

	// Single query to get all three totals at once
	err := db.Raw(`
		SELECT 
			COALESCE(SUM(CAST(fee_amount AS NUMERIC)), 0) as total_trading_fee,
			COALESCE(SUM(CAST(funding_fees AS NUMERIC)), 0) as total_funding_fee,
			COALESCE(SUM(CAST(realised_pnl AS NUMERIC)), 0) as total_realised_pnl
		FROM 
			fill_order_tables
		WHERE 
			sub_account_id NOT IN (?) AND broker_id = ?
			AND created_at > '2024-08-14 15:52:43'
	`, excludeSubaccountIDs, brokerId).Scan(&result).Error

	if err != nil {
		xlog.Errorf("Error fetching total fees and PnL: %v", err)
		return nil, err
	}

	// Construct the response
	response := map[string]string{
		"total_trading_fee":  result.TotalTradingFee.String(),
		"total_funding_fee":  result.TotalFundingFee.String(),
		"total_realised_pnl": result.TotalRealisedPnl.String(),
	}

	return response, nil
}

// const (
// 	USER_DASHBOARD_DATA_KEY = "user_dashboard:data:"
// 	// TODO: When users increase then caching logic should be updated for individual users
// 	USER_DASHBOARD_BY_ADDRESS_KEY = "user_dashboard:address:"
// 	CACHE_EXPIRY_DURATION         = 15 * time.Minute
// )

// GetUserDashboardData retrieves paginated user dashboard data for users who have deposited product ID 72
func (*FillOrderDB) GetUserDashboardData(limit, offset int) ([]map[string]interface{}, int64, error) {
	redisClient := xredis.GetRedisClient()
	ctx := context.Background()

	cacheKey := fmt.Sprintf("%s%d_%d", USER_DASHBOARD_DATA_KEY, limit, offset)

	// Try cache first
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var results []map[string]interface{}
		if err := json.Unmarshal([]byte(cachedData), &results); err == nil {
			return results, int64(len(results)), nil
		}
	}
	brokerId := uint(2)

	results, err := (*FillOrderDB)(nil).fetchUserDashboardData(brokerId, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	if len(results) > 0 {
		if dataJSON, err := json.Marshal(results); err == nil {
			redisClient.Set(ctx, cacheKey, dataJSON, CACHE_EXPIRY_DURATION)
		} else {
			xlog.Errorf("Error marshaling data for cache: %v", err)
		}
	}

	return results, int64(len(results)), nil
}

func (*FillOrderDB) fetchUserDashboardData(brokerId uint, limit, offset int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	err := db.Raw(`
		WITH user_stats AS (
			SELECT 
				fo.sub_account_id,
				fo.user_address,
				COUNT(*) as total_trades,
				SUM(ABS(CAST(fo.quote_delta AS NUMERIC))) / 1e18 as total_volume_usd,
				SUM(CAST(fo.realised_pnl AS NUMERIC)) / 1e18 as total_net_pnl_usd,
				SUM(CASE WHEN CAST(fo.realised_pnl AS NUMERIC) > 0 THEN 1 ELSE 0 END) as winning_trades,
				AVG(ABS(CAST(fo.quote_delta AS NUMERIC))) / 1e18 as avg_trade_size,
				MODE() WITHIN GROUP (ORDER BY fo.product_id) as favorite_product_id,
				COUNT(DISTINCT DATE(fo.created_at)) as active_days,
				MAX(fo.created_at) as last_trade,
				MIN(fo.created_at) as first_trade
			FROM fill_order_tables fo
			WHERE fo.broker_id = ?
			GROUP BY fo.sub_account_id, fo.user_address
		)
		SELECT 
			us.sub_account_id,
			us.user_address,
			us.total_trades,
			us.total_volume_usd,
			us.total_net_pnl_usd,
			us.winning_trades,
			CASE 
				WHEN us.total_trades > 0 THEN (us.winning_trades::DECIMAL / us.total_trades::DECIMAL * 100)
				ELSE 0 
			END as win_rate,
			us.avg_trade_size,
			us.favorite_product_id,
			us.active_days,
			us.last_trade,
			us.first_trade,
			EXTRACT(EPOCH FROM (NOW() - us.first_trade)) / 86400 as account_age_days
		FROM user_stats us
		ORDER BY us.total_volume_usd DESC
		LIMIT ? OFFSET ?
	`, brokerId, limit, offset).Scan(&results).Error

	if err != nil {
		xlog.Errorf("Error fetching user dashboard data: %v", err)
		return nil, err
	}

	return results, nil
}

// fetchUserDashboardByAddress fetches dashboard data for a specific user address
func (*FillOrderDB) fetchUserDashboardByAddress(userAddress string) (map[string]interface{}, error) {
	brokerId := uint(2) // TODO: Make this configurable

	var result map[string]interface{}
	err := db.Raw(`
		WITH user_stats AS (
			SELECT 
				fo.sub_account_id,
				fo.user_address,
				COUNT(*) as total_trades,
				SUM(ABS(CAST(fo.quote_delta AS NUMERIC))) / 1e18 as total_volume_usd,
				SUM(CAST(fo.realised_pnl AS NUMERIC)) / 1e18 as total_net_pnl_usd,
				SUM(CASE WHEN CAST(fo.realised_pnl AS NUMERIC) > 0 THEN 1 ELSE 0 END) as winning_trades,
				AVG(ABS(CAST(fo.quote_delta AS NUMERIC))) / 1e18 as avg_trade_size,
				MODE() WITHIN GROUP (ORDER BY fo.product_id) as favorite_product_id,
				COUNT(DISTINCT DATE(fo.created_at)) as active_days,
				MAX(fo.created_at) as last_trade,
				MIN(fo.created_at) as first_trade
			FROM fill_order_tables fo
			WHERE fo.user_address = ? and fo.broker_id = ?
			GROUP BY fo.sub_account_id, fo.user_address
		)
		SELECT 
			us.user_address,
			us.total_trades,
			us.total_volume_usd,
			us.total_net_pnl_usd,
			us.winning_trades,
			CASE 
				WHEN us.total_trades > 0 THEN (us.winning_trades::DECIMAL / us.total_trades::DECIMAL * 100)
				ELSE 0 
			END as win_rate,
			us.avg_trade_size,
			us.favorite_product_id,
			us.active_days,
			us.last_trade,
			us.first_trade,
			EXTRACT(EPOCH FROM (NOW() - us.first_trade)) / 86400 as account_age_days
		FROM user_stats us
	`, userAddress, brokerId).Scan(&result).Error

	return result, err
}

// GetUserDashboardByAddress retrieves dashboard data for a specific user address
func (*FillOrderDB) GetUserDashboardByAddress(userAddress string) (map[string]interface{}, error) {
	redisClient := xredis.GetRedisClient()
	ctx := context.Background()

	// Create cache key for this specific address
	cacheKey := fmt.Sprintf("%s%s", USER_DASHBOARD_BY_ADDRESS_KEY, userAddress)

	// Try to get from cache first
	cachedData, err := redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(cachedData), &result); err == nil {
			return result, nil
		}
	}

	// Fetch user dashboard data
	result, err := (*FillOrderDB)(nil).fetchUserDashboardByAddress(userAddress)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if len(result) > 0 {
		if dataJSON, err := json.Marshal(result); err == nil {
			redisClient.Set(ctx, cacheKey, dataJSON, CACHE_EXPIRY_DURATION)
		} else {
			xlog.Errorf("Error marshaling data for cache: %v", err)
		}
	}

	return result, nil
}

// GetPreviousMonthStats gets all previous month statistics in one call
func (*FillOrderDB) GetPreviousMonthStats() (map[string]string, error) {
	now := time.Now().UTC()
	firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDayOfPreviousMonth := firstDayOfCurrentMonth.Add(-time.Second)
	firstDayOfPreviousMonth := time.Date(lastDayOfPreviousMonth.Year(), lastDayOfPreviousMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Get excluded subaccount IDs using GetTrimmedSplitEnv
	excludeSubaccountIDs := cutils.GetTrimmedSplitEnv("EXCLUDED_SUBACCOUNT_IDS", ",")
	fundingFeeExcludeIDs := cutils.ParseSubaccountIDsAsHexsFromEnv("INTERNAL_SUBACCOUNT_IDS", ",")

	type FeeStats struct {
		TradingFees     string `json:"trading_fees"`
		LiquidationFees string `json:"liquidation_fees"`
	}

	// Build trading/liquidation fees query with proper NOT IN clause
	tradingQuery := `
		SELECT
		  COALESCE(SUM(CASE WHEN type != 'LIQUIDATION' THEN CAST(feex18 AS NUMERIC) END)/1e18, 0) AS trading_fees,
		  COALESCE(SUM(CASE WHEN type = 'LIQUIDATION' THEN CAST(feex18 AS NUMERIC) END)/1e18, 0) AS liquidation_fees
		FROM fill_tables
		WHERE created_at >= ? AND created_at <= ?`

	// Add NOT IN clause only if there are excluded IDs
	if len(excludeSubaccountIDs) > 0 {
		tradingQuery += ` AND subaccount_id NOT IN (`
		for i := range excludeSubaccountIDs {
			if i > 0 {
				tradingQuery += ","
			}
			tradingQuery += "?"
		}
		tradingQuery += ")"
	}

	// Build parameters for trading query
	tradingParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
	for _, id := range excludeSubaccountIDs {
		tradingParams = append(tradingParams, id)
	}

	var feeStats FeeStats
	err := db.Raw(tradingQuery, tradingParams...).Scan(&feeStats).Error
	if err != nil {
		xlog.Errorf("Error fetching previous month trading/liquidation fees: %v", err)
		return nil, err
	}

	// Build funding fees query with proper NOT IN clause
	fundingQuery := `
		SELECT COALESCE(SUM(CAST(funding_fees AS NUMERIC))/1e18, 0) as funding_fees
		FROM fill_order_tables
		WHERE created_at >= ? AND created_at <= ?`

	// Add NOT IN clause only if there are excluded IDs
	if len(fundingFeeExcludeIDs) > 0 {
		fundingQuery += ` AND sub_account_id NOT IN (`
		for i := range fundingFeeExcludeIDs {
			if i > 0 {
				fundingQuery += ","
			}
			fundingQuery += "?"
		}
		fundingQuery += ")"
	}

	// Build parameters for funding query
	fundingParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
	for _, id := range fundingFeeExcludeIDs {
		fundingParams = append(fundingParams, id)
	}

	var fundingFees string
	err = db.Raw(fundingQuery, fundingParams...).Row().Scan(&fundingFees)
	if err != nil {
		xlog.Errorf("Error fetching previous month funding fees: %v", err)
		return nil, err
	}

	// AMM PnL query using environment variables
	ammExcludeIDs := cutils.GetTrimmedSplitEnv("AMM_EXCLUDED_SUBACCOUNT_IDS", ",")

	ammQuery := `
		SELECT COALESCE(SUM(CAST(realized_pnlx18 AS NUMERIC))/1e18, 0) as amm_pnl
		FROM fill_tables
		WHERE created_at >= ? AND created_at <= ?`

	// Add NOT IN clause only if there are excluded IDs
	if len(ammExcludeIDs) > 0 {
		ammQuery += ` AND subaccount_id NOT IN (`
		for i := range ammExcludeIDs {
			if i > 0 {
				ammQuery += ","
			}
			ammQuery += "?"
		}
		ammQuery += ")"
	}

	// Build parameters for AMM query
	ammParams := []interface{}{firstDayOfPreviousMonth, lastDayOfPreviousMonth}
	for _, id := range ammExcludeIDs {
		ammParams = append(ammParams, id)
	}

	var ammPnL string
	err = db.Raw(ammQuery, ammParams...).Row().Scan(&ammPnL)
	if err != nil {
		xlog.Errorf("Error fetching previous month AMM PnL: %v", err)
		return nil, err
	}

	return map[string]string{
		"trading_fees":     feeStats.TradingFees,
		"funding_fees":     cutils.X18ToFloatStr(cutils.FloatStrToX18(fundingFees)),
		"liquidation_fees": feeStats.LiquidationFees,
		"amm_pnl":          ammPnL,
		"month":            firstDayOfPreviousMonth.Format("2006-01"),
		"start_date":       firstDayOfPreviousMonth.Format("2006-01-02"),
		"end_date":         lastDayOfPreviousMonth.Format("2006-01-02"),
	}, nil
}

// Returns: map[subaccountID]*big.Int
func (*FillOrderDB) Get24hRealisedPnlPerSubaccount() (map[string]*big.Int, error) {
	now := time.Now().UTC()
	startOfPeriod := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	endOfPeriod := now.Format("2006-01-02 15:04:05")

	ammSubaccountID := contractUtils.AMM_SUBACCOUNT_ID_1
	internalSubaccountIDs := cutils.GetTrimmedSplitEnv("INTERNAL_SUBACCOUNT_IDS", ",")
	excludeSubaccountIDs := append([]string{ammSubaccountID}, internalSubaccountIDs...)

	type Result struct {
		SubaccountID string
		RealisedPnl  string
	}
	var results []Result

	err := db.Raw(`
		SELECT subaccount_id, SUM(CAST(realized_pnlx18 AS NUMERIC)) AS realised_pnl
		FROM fill_tables
		WHERE created_at BETWEEN ? AND ? AND subaccount_id NOT IN (?)
		GROUP BY subaccount_id
	`, startOfPeriod, endOfPeriod, excludeSubaccountIDs).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	out := make(map[string]*big.Int)
	for _, r := range results {
		val := new(big.Int)
		val.SetString(r.RealisedPnl, 10)
		out[r.SubaccountID] = val
	}
	return out, nil
}

// GetFundingFeeBySubaccountId returns the total funding fee for the given subaccountId
func (*FillOrderDB) GetFundingFeeBySubaccountId(subaccountId string, brokerId uint) (string, error) {
	var fundingFee string
	err := db.Raw(`
		SELECT COALESCE(SUM(CAST(funding_fees AS NUMERIC))/1e18, 0) as funding_fee
		FROM fill_order_tables
		WHERE sub_account_id = ? AND broker_id = ?
	`, subaccountId, brokerId).Row().Scan(&fundingFee)
	if err != nil {
		return "0", err
	}
	return fundingFee, nil
}

// GetDailyPnlGraph returns daily realized PnL for the given subaccountId
func (*FillOrderDB) GetDailyPnlGraph(subaccountId string, brokerId uint) ([]map[string]interface{}, error) {
	var results []struct {
		Date        time.Time `gorm:"column:date"`
		RealizedPnL string    `gorm:"column:realized_pnl"`
	}

	err := db.Raw(`
		SELECT 
			DATE_TRUNC('day', created_at AT TIME ZONE 'UTC') AS date,
		COALESCE(SUM(CAST(realised_pnl AS NUMERIC))/1e18, 0) AS realized_pnl
		FROM fill_order_tables
	    WHERE sub_account_id = ? AND broker_id = ?
		GROUP BY date
		ORDER BY date ASC
	`, subaccountId, brokerId).Scan(&results).Error

	if err != nil {
		return nil, err
	}

	var graphData []map[string]any

	for _, row := range results {
		pnlFloat, err := strconv.ParseFloat(row.RealizedPnL, 64)
		if err != nil {
			fmt.Printf("Error parsing RealizedPnL: %v\n", err)
			pnlFloat = 0 // fallback
		}
		graphData = append(graphData, map[string]interface{}{
			"date":         row.Date.Format("02-01-2006"),
			"realized_pnl": fmt.Sprintf("%.18f", pnlFloat),
		})
	}
	return graphData, nil
}

// GetTradeHistoryBySubaccountId fetches paginated trade history with only required fields
func (*FillOrderDB) GetTradeHistoryBySubaccountId(subaccountId string, offset, limit int) (*[]FillOrderTable, error) {
	fills := []FillOrderTable{}

	// Select only the fields we need for trade history
	err := db.Select("txn_hash, amount, product_id, realised_pnl, created_at").
		Where("sub_account_id = ?", subaccountId).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&fills).Error

	if err != nil {
		return nil, err
	}

	return &fills, nil
}

// GetTradeHistoryCountBySubaccountId returns total count of trade history records
func (*FillOrderDB) GetTradeHistoryCountBySubaccountId(subaccountId string) (int64, error) {
	var count int64
	err := db.Model(&FillOrderTable{}).
		Where("sub_account_id = ?", subaccountId).
		Count(&count).Error

	return count, err
}
