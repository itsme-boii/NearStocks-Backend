package subaccount

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/funding"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"

	"log"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
)

type SubaccountBalance interface {
	GetAllSubaccounts(startGID uint) ([]subaccountTypes.SubaccountBalances, uint, error)
	MustGetSubaccountBalancesFromIds(subaccountIds []string) []subaccountTypes.SubaccountBalances
}

type SubaccountBalanceImpl struct {
	redisClient *redis.Client
}

// Validate that SubaccountBalanceImpl implements SubaccountBalance
var _ SubaccountBalance = &SubaccountBalanceImpl{}

func NewSubaccountBalanceImpl() *SubaccountBalanceImpl {
	return &SubaccountBalanceImpl{
		redisClient: xredis.GetRedisClient(),
	}
}

// Get all subaccount ids from db in batches
// MAX limit is 100_000
// NOTE: Assumes primary key GID is continuous without any gaps
func (s *SubaccountBalanceImpl) getAllSubaccountIdsBatched(startGID uint) (subaccountIds []string, lastGID uint) {
	defer cutils.LogTime(time.Now(), metric.LIQUIDATION_GET_ALL_SUBACCOUNT_IDS)
	limit := uint(10_000)

	xlog.Infof("Fetching subaccounts from GID: %v\n", startGID)

	for i := 0; i < 10; i++ {
		var newSubaccountIds *[]string
		newSubaccountIds, lastGID = (&db.SubaccountDB{}).GetAllIdsFromStartGID(startGID, limit)
		subaccountIds = append(subaccountIds, *newSubaccountIds...)
		if lastGID < startGID+limit-1 {
			break
		}
		startGID = lastGID + 1
	}

	xlog.Infof("Fetched %v subaccounts | lastGID: %v\n", len(subaccountIds), lastGID)
	return subaccountIds, lastGID
}

// Get all subaccounts from db and return them with empty spot and perp balances
func (s *SubaccountBalanceImpl) GetAllSubaccounts(startGID uint) ([]subaccountTypes.SubaccountBalances, uint, error) {
	// Get all subaccounts from db
	subaccountIds, lastFetchedGID := s.getAllSubaccountIdsBatched(startGID)
	// Create subaccounts with empty spot and perp balances
	subaccounts := cutils.MapSlice(subaccountIds, func(subaccountId string) subaccountTypes.SubaccountBalances {
		return subaccountTypes.SubaccountBalances{
			SubaccountId: subaccountId,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
		}
	})
	return subaccounts, lastFetchedGID, nil
}

// Fetch raw spot and perp balances from redis for requested subaccounts using pipeline fetch.
// Unmarshal the perp and spot balances
func (s *SubaccountBalanceImpl) MustGetSubaccountBalancesFromIds(subaccountIds []string) []subaccountTypes.SubaccountBalances {
	defer cutils.LogTime(time.Now(), metric.GET_SUBACCOUNTS_BALANCES_FAST)

	if len(subaccountIds) == 0 {
		return []subaccountTypes.SubaccountBalances{}
	}

	ctx := context.Background()
	allCmds := make([]redis.Cmder, 0)
	limit := 1_000

	for i := 0; i < len(subaccountIds); i += limit {
		end := min(i+limit, len(subaccountIds))
		subaccountIdsBatch := subaccountIds[i:end]
		cmds, err := s.redisClient.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for _, subaccountId := range subaccountIdsBatch {
				subaccountIdHex, err := cutils.SubaccountIdToHex(subaccountId)
				// Temporary hack: If error is not nil then it means subaccount id is already in hex format
				if err != nil {
					subaccountIdHex = subaccountId
				}
				hashKey := xredis.GetBalanceKey(subaccountIdHex)
				pipe.HGetAll(context.Background(), hashKey).Result()
			}
			return nil
		})

		// This is a critical error
		if err != nil {
			xlog.Errorf("err: %v | Couldn't fetch subaccount balances for subaccount ids[0]: %+v | len: %v | ", err, subaccountIdsBatch[0], len(subaccountIdsBatch))
			xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("Critical: Couldn't fetch subaccount balances for subaccount ids[0]: %+v | len: %v | err: %v", subaccountIdsBatch[0], len(subaccountIdsBatch), err))
		}

		allCmds = append(allCmds, cmds...)
	}

	return s.parseSubaccountBalancesFromRedisCmder(allCmds, subaccountIds)
}

func (s *SubaccountBalanceImpl) parseSubaccountBalancesFromRedisCmder(cmds []redis.Cmder, subaccountIds []string) []subaccountTypes.SubaccountBalances {
	var subaccounts []subaccountTypes.SubaccountBalances
	for idx, cmd := range cmds {
		subaccountId := subaccountIds[idx]
		subaccountIdHex, _ := cutils.SubaccountIdToHex(subaccountId)

		balances, err := cmd.(*redis.MapStringStringCmd).Result()
		var balanceFetchOrParseError error
		subaccountBalances := &subaccountTypes.SubaccountBalances{}

		if err != nil {
			if err == redis.Nil {
				xlog.Warnf("No token balances found for subaccountHex: %v\n", subaccountIdHex)
			} else {
				xlog.Warnf("Unable to get token balances: %v\n", err)
			}
			balanceFetchOrParseError = err
		} else {
			balanceFetchOrParseError = subaccountBalances.UnmarshalForRedis(balances, subaccountId)
		}

		if balanceFetchOrParseError != nil {
			xlog.Warnf("Error fetching or parsing balances for subaccountHex: %v | err: %v ... Taking empty balances", subaccountIdHex, balanceFetchOrParseError)
			subaccounts = append(subaccounts, subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountId,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
			})
		} else {
			subaccounts = append(subaccounts, *subaccountBalances)
		}
	}
	return subaccounts
}

func GetTotalEquityx36(unsettledSubaccount subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, cumulativeFundingRateMap map[string]*big.Int) *big.Int {
	// Total net equity
	totalNetEquityx36 := cutils.GetBig0()
	totalSpotBalancex36 := GetTotalSpotBalancex36(unsettledSubaccount, oraclePricesMap)
	// Calculate unrealised pnl and unrealised funding fee
	unrealisedPnlx36, unrealisedFundingFeex36 := GetUnrealisedSubaccountValue(unsettledSubaccount, oraclePricesMap, cumulativeFundingRateMap)
	totalNetEquityx36 = new(big.Int).Add(totalNetEquityx36, totalSpotBalancex36)
	totalNetEquityx36 = new(big.Int).Add(totalNetEquityx36, unrealisedPnlx36)
	// Funding fee should be deducted from total net equity
	totalNetEquityx36 = new(big.Int).Sub(totalNetEquityx36, unrealisedFundingFeex36)
	return totalNetEquityx36
}

// NOTE: For now we are taking oracle price for calculating current value of token. Ideally this should be mark price similar to hyperliquid but we are skipping it for now
// This function should return the unrealised pnl as well as unrealised funding fee
// Upnl = Current value + VQuoteBalancex18
// Unrealised funding fee = (Current cummulative - last cummulative) * (Long Amount + Short Amount)
func GetUnrealisedSubaccountValue(unsettledSubaccount subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, cumulativeFundingRateMap map[string]*big.Int) (unrealisedPnlx36 *big.Int, unrealisedFundingFeex36 *big.Int) {
	// Get the spot value
	unrealisedPnlx36 = cutils.GetBig0()
	unrealisedFundingFeex36 = cutils.GetBig0()

	for _, perpetual := range unsettledSubaccount.PerpBalances {
		oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(uint32(perpetual.ProductId))
		if !exists {
			xlog.Errorf("Oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", perpetual.ProductId)
			continue
		}

		if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
			// Current value of perp
			currentNotionalValuex36 := new(big.Int).Mul(perpetual.Amountx18, oraclePrice.Pricex18)
			vQuoteBalancex36 := cutils.Mulx18(perpetual.VQuoteBalancex18)
			upnlx36 := new(big.Int).Add(currentNotionalValuex36, vQuoteBalancex36)
			unrealisedPnlx36 = new(big.Int).Add(unrealisedPnlx36, upnlx36)
			uffex36 := funding.CalcFundingFeesx36(perpetual, cumulativeFundingRateMap[oracleSymbol])
			unrealisedFundingFeex36 = new(big.Int).Add(unrealisedFundingFeex36, uffex36)
		} else {
			xlog.Debugf("User's Perp balance: %+v", unsettledSubaccount)
			log.Panicf("Oracle price not found for symbol: %s. This means there is a bug in oracle", oracleSymbol)
		}
	}
	return unrealisedPnlx36, unrealisedFundingFeex36
}

func GetRequiredMarginValues(unsettledSubaccount subaccountTypes.SubaccountBalances, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, oraclePricesMap map[string]ctypes.OraclePrice) (maintenanceMarginx36 *big.Int, initialMarginx36 *big.Int) {
	// Get the spot value
	maintenanceMarginx36 = cutils.GetBig0()
	initialMarginx36 = cutils.GetBig0()

	for _, perpetual := range unsettledSubaccount.PerpBalances {
		oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(uint32(perpetual.ProductId))
		if !exists {
			log.Panicf("Oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", perpetual.ProductId)
		}
		perpetualMarket, ok := perpetualMarketsMap[uint(perpetual.ProductId)]
		if !ok {
			log.Panicf("Perpetual market not found for product id: %d", perpetual.ProductId)
		}
		if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
			// Take absolute value
			netNotionalAbsValuex36 := new(big.Int).Abs(new(big.Int).Mul(perpetual.Amountx18, oraclePrice.Pricex18))
			// Initial margin
			initialMarginx36 = new(big.Int).Add(initialMarginx36, cutils.Divx18(new(big.Int).Mul(netNotionalAbsValuex36, perpetualMarket.InitialMarginFractionx18)))
			// Maintenance margin
			maintenanceMarginx36 = new(big.Int).Add(maintenanceMarginx36, cutils.Divx18(new(big.Int).Mul(netNotionalAbsValuex36, perpetualMarket.MaintenanceMarginFractionx18)))
		}
	}

	return maintenanceMarginx36, initialMarginx36
}

// FIXME: Every spot will have a contribution factor. For now, we are assuming it to be 1. For tokens which doesn't have oracle price we assume they are not contributing to the net collateral
func GetTotalSpotBalancex36(subaccount subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice) *big.Int {
	totalSpotBalancex36 := cutils.GetBig0()
	for _, spotBalance := range subaccount.SpotBalances {
		if contractUtils.PRODUCT_MARKET_WEIGHTS[spotBalance.ProductId] == 0 {
			continue
		}

		oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(uint32(spotBalance.ProductId))
		if !exists {
			continue
		}

		if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
			totalSpotBalancex36 = new(big.Int).Add(totalSpotBalancex36, new(big.Int).Mul(spotBalance.Balancex18, oraclePrice.Pricex18))
		}
	}
	return totalSpotBalancex36
}

// Returns dollar value of locked collateral
func GetLockedValuex36(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice) *big.Int {
	lockedValuex36 := cutils.GetBig0()
	for _, spotBalance := range subaccountBalance.SpotBalances {
		if contractUtils.PRODUCT_MARKET_WEIGHTS[spotBalance.ProductId] == 0 {
			continue
		}

		oracleSymbol, exists := marketutils.GetBaseSymbolForProduct(uint32(spotBalance.ProductId))
		if !exists {
			continue
		}

		if oraclePrice, ok := oraclePricesMap[oracleSymbol]; ok {
			// We are taking max of 0 and locked balance
			lockedValuex36 = new(big.Int).Add(lockedValuex36, new(big.Int).Mul(cutils.MaxBigInt(cutils.GetBig0(), spotBalance.Lockedx18), oraclePrice.Pricex18))
		}
	}
	return lockedValuex36
}

// Returns Total Net Equity and Available Margin
func GetCollateralDetails(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) (totalNetEquityx36 *big.Int, availableMarginx36 *big.Int) {
	// Get total equity and margin
	totalNetEquityx36 = GetTotalEquityx36(subaccountBalance, oraclePricesMap, cumulativeFundingRateMap)
	_, initialMarginx36 := GetRequiredMarginValues(subaccountBalance, perpetualMarketsMap, oraclePricesMap)
	lockedBalancex36 := GetLockedValuex36(subaccountBalance, oraclePricesMap)

	// Calculate available margin
	availableMarginx36 = new(big.Int).Sub(totalNetEquityx36, initialMarginx36)
	availableMarginx36 = new(big.Int).Sub(availableMarginx36, lockedBalancex36)

	return totalNetEquityx36, availableMarginx36
}

// Returns withdrawable balance map <productId <-> balancex18 string> with x18 precision
func GetWithdrawableBalance(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int, assumePositionAndOrdersClosed ...bool) (map[uint32]string, error) {
	// Get total equity and margin
	totalNetEquityx36, availableMarginx36 := GetCollateralDetails(subaccountBalance, oraclePricesMap, perpetualMarketsMap, cumulativeFundingRateMap)

	if len(assumePositionAndOrdersClosed) > 0 && assumePositionAndOrdersClosed[0] {
		availableMarginx36 = totalNetEquityx36
	}

	// Distribute the available margin across all tokens
	withdrawableTokenBal := make(map[uint32]string)
	for _, productId := range cutils.ReverseSlice(contractUtils.ALL_COLLATERAL_SPOTS) {
		if availableMarginx36.Sign() <= 0 {
			break
		}

		symbol, _ := marketutils.GetBaseSymbolForProduct(productId)

		price, exists := oraclePricesMap[symbol]
		if !exists {
			xlog.Errorf("BS - Price not found for token: %s", symbol)
			return nil, fmt.Errorf("price not found for token")
		}
		currentAmountx18 := subaccountBalance.MustGetSpotBalance(productId).Balancex18

		if currentAmountx18.Sign() <= 0 {
			continue
		}

		dollarValuex36 := new(big.Int).Mul(currentAmountx18, price.Pricex18)

		if availableMarginx36.Cmp(dollarValuex36) >= 0 {
			withdrawableTokenBal[productId] = currentAmountx18.String()
			availableMarginx36.Sub(availableMarginx36, dollarValuex36)
		} else {
			freeAmountx18 := new(big.Int).Div(availableMarginx36, price.Pricex18)
			withdrawableTokenBal[productId] = freeAmountx18.String()
			dollarValuex36 = new(big.Int).Mul(freeAmountx18, price.Pricex18)
			availableMarginx36 = new(big.Int).Sub(availableMarginx36, dollarValuex36)
		}
	}

	return withdrawableTokenBal, nil
}

func ReplaceBalanceInRedis(pipe redis.Pipeliner, subaccountBalances *subaccountTypes.SubaccountBalances) error {
	key := xredis.GetBalanceKey(subaccountBalances.SubaccountId)
	// First delete the key: This is important otherwise the old data will remain
	err := pipe.Del(context.Background(), key).Err()
	if err != nil {
		xlog.Errorf("Error deleting subaccount balances in redis: %v", err)
		return err
	}

	// If there are no spot and perps no need to set anything
	// Marshal data for redis
	if len(subaccountBalances.SpotBalances) > 0 || len(subaccountBalances.PerpBalances) > 0 {
		data, err := subaccountBalances.MarshalForRedis()
		if err != nil {
			xlog.Errorf("Error marshalling subaccount balances for redis: %v", err)
			return err
		}

		err = pipe.HSet(context.Background(), key, data).Err()
		if err != nil {
			xlog.Errorf("Error setting subaccount balances in redis: %v", err)
			return err
		}
	}

	return nil
}

// SafetyMarginx18 is the balance-server's trade health measure (formerly getSafetyMarginV2):
// Divx18(equity - locked - 0.96 * initial margin). A trade is accepted when the result is >= 0
// or not lower than before the trade.
func SafetyMarginx18(subaccountBalance subaccountTypes.SubaccountBalances, oraclePricesMap map[string]ctypes.OraclePrice, perpetualMarketsMap map[uint]subaccountTypes.PerpetualMarket, cumulativeFundingRateMap map[string]*big.Int) *big.Int {
	totalNetEquityx36 := GetTotalEquityx36(subaccountBalance, oraclePricesMap, cumulativeFundingRateMap)
	lockedBalancex36 := GetLockedValuex36(subaccountBalance, oraclePricesMap)
	totalNetEquityx36 = new(big.Int).Sub(totalNetEquityx36, lockedBalancex36)
	_, initialMarginx36 := GetRequiredMarginValues(subaccountBalance, perpetualMarketsMap, oraclePricesMap)

	// 0.96 * Initial Margin
	initialMarginx36_0_96 := cutils.DivxCust(new(big.Int).Mul(initialMarginx36, big.NewInt(96)), 2)
	return cutils.Divx18(new(big.Int).Sub(totalNetEquityx36, initialMarginx36_0_96))
}
