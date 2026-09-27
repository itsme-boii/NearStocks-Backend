package db

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"time"
)

type PnlDB struct{}

// CreatePnl creates a new entry in the pnl table
func (*PnlDB) CreatePnl(pnlData PnlTable) (*PnlTable, error) {

	// Entry does not exist, create a new one
	if err := db.Create(&pnlData).Error; err != nil {
		return nil, err
	}
	return &pnlData, nil
}

func (*PnlDB) GetDailyPnL(brokerId uint) (map[string][]interface{}, map[uint]map[string][]interface{}, error) {
	aggregatedPnL := map[string][]interface{}{
		"date":           {},
		"realized_pnl":   {},
		"unrealized_pnl": {},
	}
	productPnL := make(map[uint]map[string][]interface{})
	now := time.Now().UTC().Truncate(24 * time.Hour)
	dates := make([]time.Time, 30)

	for i := 0; i < 30; i++ {
		dates[i] = now.AddDate(0, 0, -i)
	}

	for i, startOfDay := range dates {
		endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)

		// Aggregated values
		var aggregatedData struct {
			Date          time.Time
			RealizedPnl   ctypes.BigInt
			UnrealizedPnl string
		}

		err := db.Raw(`
		SELECT 
			SUM(CAST(realized_pnl AS NUMERIC)) as realized_pnl, 
			SUM(CAST(unrealized_pnl AS NUMERIC)) as unrealized_pnl
		FROM pnl_tables
		WHERE created_at BETWEEN ? AND ? AND broker_id = ?;
		`, startOfDay, endOfDay, brokerId).Scan(&aggregatedData).Error

		if err != nil {
			xlog.Errorf("Error querying aggregated PnL for date range %v - %v: %v", startOfDay, endOfDay, err)
			return nil, nil, err
		}

		aggregatedPnL["date"] = append(aggregatedPnL["date"], startOfDay)
		if !aggregatedData.RealizedPnl.IsNil() {
			aggregatedPnL["realized_pnl"] = append(aggregatedPnL["realized_pnl"], aggregatedData.RealizedPnl.String())
			aggregatedPnL["unrealized_pnl"] = append(aggregatedPnL["unrealized_pnl"], aggregatedData.UnrealizedPnl)
		} else {
			aggregatedPnL["realized_pnl"] = append(aggregatedPnL["realized_pnl"], "0")
			aggregatedPnL["unrealized_pnl"] = append(aggregatedPnL["unrealized_pnl"], "0")
		}

		// Product-specific values
		var productData []struct {
			ProductID     uint
			RealizedPnl   ctypes.BigInt
			UnrealizedPnl string
		}
		err = db.Raw(`
		SELECT 
			product_id, 
			SUM(CAST(realized_pnl AS NUMERIC)) as realized_pnl, 
			SUM(CAST(unrealized_pnl AS NUMERIC)) as unrealized_pnl
		FROM pnl_tables
		WHERE created_at BETWEEN ? AND ? AND broker_id = ?
		GROUP BY product_id
		ORDER BY product_id;
		`, startOfDay, endOfDay, brokerId).Scan(&productData).Error

		if err != nil {
			xlog.Errorf("Error querying product-specific PnL for date range %v - %v: %v", startOfDay, endOfDay, err)
			return nil, nil, err
		}

		for _, p := range productData {
			if _, ok := productPnL[p.ProductID]; !ok {
				productPnL[p.ProductID] = map[string][]interface{}{
					"realized_pnl":   make([]interface{}, 30),
					"unrealized_pnl": make([]interface{}, 30),
				}
				// Initialize with "0" for all 30 entries
				for j := 0; j < 30; j++ {
					productPnL[p.ProductID]["realized_pnl"][j] = "0"
					productPnL[p.ProductID]["unrealized_pnl"][j] = "0"
				}
			}
			productPnL[p.ProductID]["realized_pnl"][i] = p.RealizedPnl.String()
			productPnL[p.ProductID]["unrealized_pnl"][i] = p.UnrealizedPnl
		}
	}

	return aggregatedPnL, productPnL, nil
}
