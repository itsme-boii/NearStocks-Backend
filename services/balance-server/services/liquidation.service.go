package services

import (
	"context"
	"errors"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type LiquidationService interface {
	SettlePnlForLiquidator(liquidationReq *types.FinaliseLiquidationRequest, liquidateeSubaccountBalance *subaccountTypes.SubaccountBalances, cumulativeFundingRatex18 *big.Int) (*subaccountTypes.SubaccountBalances, *big.Int, *big.Int)
	SettlePnlForLiquidatee(liquidationReq *types.FinaliseLiquidationRequest, liquidateeSubaccountBalance *subaccountTypes.SubaccountBalances, cumulativeFundingRatex18 *big.Int) (*subaccountTypes.SubaccountBalances, *big.Int, *big.Int)
	// SettleLiqPnlUsingPerps(liquidationReq *types.FinaliseLiquidationRequest, totalPnlx36 *big.Int, subaccountBalances *types.SubaccountBalances) *big.Int
	FinaliseLiquidation(liquidationReq *types.FinaliseLiquidationRequest) (*types.FinaliseLiquidationResponse, error)
	MustGetBalanceFromRedis(subaccountId string) *subaccountTypes.SubaccountBalances
	MustGetSubaccountBalancesFromIds(subaccountIds []string) []subaccountTypes.SubaccountBalances
	SettleUsingInsurance(settleUsingInsuranceRequest *types.SettleWithInsuranceRequest) error
	GetRedisClient() *redis.Client
}

func NewLiquidationService() LiquidationService {
	return &LiquidationServiceImpl{
		RedisClient:   xredis.GetRedisClient(),
		SubaccountBal: subaccount.NewSubaccountBalanceImpl(),
		AppState:      appstate.NewAppState(),
	}
}

type LiquidationServiceImpl struct {
	RedisClient   *redis.Client
	SubaccountBal subaccount.SubaccountBalance
	// Oracle prices, perp markets and cumulative funding for the liquidator checks (R-5).
	AppState appstate.AppState
}

// ErrLiquidatorUnhealthy: the liquidator could not carry the position it would take on. The NEAR
// contract refuses the same liquidation (behavior-spec R-5), so the backend must not apply it.
var ErrLiquidatorUnhealthy = errors.New("liquidator would be unhealthy")

// ErrAmmPositionCap: the AMM would exceed the market's AmmMaxPositionx18.
var ErrAmmPositionCap = errors.New("AMM position cap")

// Validate that LiquidationServiceImp implements LiquidationService
var _ LiquidationService = &LiquidationServiceImpl{}

func (l *LiquidationServiceImpl) GetRedisClient() *redis.Client {
	return l.RedisClient
}

// Removes the spot balances from the subaccount balances and returns the remaining pnl
// Returns the quote balance after settling the pnl
// NOTE: Requires pnl in x36
// Handles positive pnl as well
func SettleLiqPnlUsingSpots(spotPricesx18 map[uint32]*big.Int, totalPnlx36 *big.Int, subaccountBalances *subaccountTypes.SubaccountBalances) *big.Int {
	// Settle liquidation pnl using spots
	for _, productId := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		if totalPnlx36.Sign() >= 0 {
			break
		}

		// Get the spot balance
		spotBalance := subaccountBalances.MustGetSpotBalance(productId)
		// Get the spot price
		spotPricex18 := spotPricesx18[productId]
		// It is possible that spot price is not available or spotbalance is not present for the product id
		if contractUtils.PRODUCT_MARKET_WEIGHTS[productId] == 0 || spotBalance.Balancex18.Sign() <= 0 || spotPricex18 == nil || spotPricex18.Sign() == 0 {
			continue
		}

		spotAssetDollarValuex36 := spotBalance.GetAssetDollarValuex18(spotPricex18)
		// If Spot asset max exceeds total pnl then take the total pnl equivalent of spot asset
		if spotAssetDollarValuex36.CmpAbs(totalPnlx36) > 0 {
			spotBalanceToDeduct := new(big.Int).Div(new(big.Int).Neg(totalPnlx36), spotPricex18)
			// Warning: spotBalanceToDeduct * Pricex18 might not be equal to totalPnlx36 due to rounding errors
			// Try this if error is significant: totalPnlx36 = new(big.Int).Sub(totalPnlx36, new(big.Int).Mul(spotBalanceToDeduct, spotPricex18))
			totalPnlx36 = cutils.GetBig0()
			// Reduce balance
			spotBalance.UpdateBalance(new(big.Int).Neg(spotBalanceToDeduct))
			// Update subaccount balances
			subaccountBalances.SpotBalances[productId] = spotBalance
		} else {
			totalPnlx36 = new(big.Int).Add(totalPnlx36, spotAssetDollarValuex36)
			delete(subaccountBalances.SpotBalances, productId)
		}
	}

	// Update remaining pnl in quote balance
	quoteSpotBalance := subaccountBalances.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID)
	quoteSpotBalance.UpdateBalance(cutils.Divx18(totalPnlx36))
	subaccountBalances.SpotBalances[contractUtils.QUOTE_TOKEN_PRODUCT_ID] = quoteSpotBalance
	return quoteSpotBalance.Balancex18
}

func (l *LiquidationServiceImpl) MustGetBalanceFromRedis(subaccountId string) *subaccountTypes.SubaccountBalances {
	return &l.SubaccountBal.MustGetSubaccountBalancesFromIds([]string{subaccountId})[0]
}

func (l *LiquidationServiceImpl) MustGetSubaccountBalancesFromIds(subaccountIds []string) []subaccountTypes.SubaccountBalances {
	return l.SubaccountBal.MustGetSubaccountBalancesFromIds(subaccountIds)
}

// All redis operations will be atomic
// 1. Collect Liquidation Fee
// 2. Collect Trading Fee
// 3. Update perp balance
// 4. Settle Liquidation PnL
// 4.a Settle using spots
// 4.b else Settle using perps
// 4.c else Use Insurance Funds
func (l *LiquidationServiceImpl) FinaliseLiquidation(liquidationReq *types.FinaliseLiquidationRequest) (*types.FinaliseLiquidationResponse, error) {
	// Lock liquidatee subaccount balance
	mutexLiquidatee, err := xredis.AcquireLockWithRetry(xredis.GetBalanceRedisLockKey(liquidationReq.LiquidateeSubaccountId))
	if err != nil {
		xlog.Errorf("Error acquiring lock for liquidatee subaccount: %v", err)
		return &types.FinaliseLiquidationResponse{}, err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutexLiquidatee); err != nil {
			xlog.Errorf("Failed to release lock for liquidatee subaccount:", err)
		}
	}()
	mutexLiquidator, err := xredis.AcquireLockWithRetry(xredis.GetBalanceRedisLockKey(liquidationReq.LiquidatorSubaccountId))
	if err != nil {
		xlog.Errorf("Error acquiring lock for liquidator subaccount: %v", err)
		return &types.FinaliseLiquidationResponse{}, err
	}
	defer func() {
		// Release lock
		if err := xredis.ReleaseLock(mutexLiquidator); err != nil {
			xlog.Errorf("Failed to release lock for liquidator subaccount:", err)
		}
	}()

	// The liquidation fee is credited to the insurance fund, so its balance is updated in the same
	// pipeline. Lock order (users first, insurance last) matches SettleUsingInsurance, so no cycle.
	insuranceIsParty := strings.EqualFold(liquidationReq.LiquidateeSubaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID) ||
		strings.EqualFold(liquidationReq.LiquidatorSubaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID)
	if !insuranceIsParty {
		mutexInsurance, err := xredis.AcquireLockWithRetry(xredis.GetBalanceRedisLockKey(contractUtils.INSURANCE_SUBACCOUNT_ID))
		if err != nil {
			xlog.Errorf("Error acquiring lock for insurance subaccount: %v", err)
			return &types.FinaliseLiquidationResponse{}, err
		}
		defer func() {
			if err := xredis.ReleaseLock(mutexInsurance); err != nil {
				xlog.Errorf("Failed to release lock for insurance subaccount: %v", err)
			}
		}()
	}

	balances := l.MustGetSubaccountBalancesFromIds([]string{liquidationReq.LiquidateeSubaccountId, liquidationReq.LiquidatorSubaccountId})
	liquidateeSubaccountBalance := &balances[0]
	liquidatorSubaccountBalance := &balances[1]

	if err != nil {
		xlog.Errorf("Error getting liquidator subaccount balance from redis: %v", err)
		return &types.FinaliseLiquidationResponse{}, err
	}

	symbol, exists := marketutils.GetFundingSymbolForProduct(liquidationReq.ProductId)
	if !exists {
		return &types.FinaliseLiquidationResponse{}, fmt.Errorf("funding symbol not found for product id: %d", liquidationReq.ProductId)
	}
	cumulativeFundingRatex18, err := xredis.GetCumulativeFundingRateForSymbol(l.RedisClient, symbol)
	if err != nil {
		xlog.Errorf("Error getting cumulative funding rate for symbol: %v", err)
		return &types.FinaliseLiquidationResponse{}, err
	}

	liquidatorBefore := cloneBalances(liquidatorSubaccountBalance)
	oldLiquidateeAmount := new(big.Int).Set(liquidateeSubaccountBalance.MustGetPerpBalance(liquidationReq.ProductId).Amountx18)
	oldLiquidatorAmount := new(big.Int).Set(liquidatorSubaccountBalance.MustGetPerpBalance(liquidationReq.ProductId).Amountx18)
	liquidateeSubaccountBalance, realisedPnlLiquidateex18, liquidateeFundingFeesx18 := l.SettlePnlForLiquidatee(liquidationReq, liquidateeSubaccountBalance, cumulativeFundingRatex18)
	liquidatorSubaccountBalance, realisedPnlLiquidatorx18, liquidatorFundingFeesx18 := l.SettlePnlForLiquidator(liquidationReq, liquidatorSubaccountBalance, cumulativeFundingRatex18)
	if err := l.checkLiquidator(liquidationReq, liquidatorBefore, liquidatorSubaccountBalance); err != nil {
		// nothing has been written yet; the liquidation is simply not applied
		return &types.FinaliseLiquidationResponse{}, err
	}
	// open interest, for non-AMM sides (as before, but only once the liquidation will apply)
	pu := perputils.NewPerpUtils()
	for _, side := range []struct {
		id         string
		old, delta *big.Int
	}{
		{liquidationReq.LiquidateeSubaccountId, oldLiquidateeAmount, liquidationReq.AmountX18},
		{liquidationReq.LiquidatorSubaccountId, oldLiquidatorAmount, new(big.Int).Neg(liquidationReq.AmountX18)},
	} {
		if side.id == contractUtils.AMM_SUBACCOUNT_ID {
			continue
		}
		if err := pu.AddLongShortOIAtReddis(liquidationReq.ProductId, side.old, side.delta); err != nil {
			xlog.Infof("Error updating OI position for productID %d: %v\n", liquidationReq.ProductId, err)
		}
	}

	liquidationFeex18 := LiquidationFeex18(liquidationReq)
	var insuranceBalance *subaccountTypes.SubaccountBalances
	switch {
	case strings.EqualFold(liquidationReq.LiquidatorSubaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID):
		insuranceBalance = liquidatorSubaccountBalance // already locked and about to be written
	case strings.EqualFold(liquidationReq.LiquidateeSubaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID):
		insuranceBalance = liquidateeSubaccountBalance
	default:
		insuranceBalance = l.MustGetBalanceFromRedis(contractUtils.INSURANCE_SUBACCOUNT_ID)
	}
	insuranceBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, liquidationFeex18)

	_, err = l.RedisClient.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
		if !insuranceIsParty {
			if err := subaccount.ReplaceBalanceInRedis(pipe, insuranceBalance); err != nil {
				xlog.Errorf("Error replacing insurance subaccount balance in redis: %v", err)
				return err
			}
		}

		err = subaccount.ReplaceBalanceInRedis(pipe, liquidateeSubaccountBalance)
		if err != nil {
			xlog.Errorf("Error replacing liquidatee subaccount balance in redis: %v", err)
			return err
		}

		err = subaccount.ReplaceBalanceInRedis(pipe, liquidatorSubaccountBalance)
		if err != nil {
			xlog.Errorf("Error replacing liquidator subaccount balance in redis: %v", err)
			return err
		}

		// D-7: the liquidatee's trading fee goes to the fee account, queued with this write
		subaccount.AccrueFee(pipe, LiquidateeTradingFeex18(liquidationReq))

		if err := transaction.WriteBalanceUpdateForTxn(pipe, *liquidationReq.TxnCounter, transaction.TxnBalanceUpdate{
			Subaccount1: *liquidateeSubaccountBalance,
			Subaccount2: *liquidatorSubaccountBalance,
		}); err != nil {
			xlog.Errorf("Error writing balance update for txn: %v", err)
			// Skip the error since this is not a critical operation
		}

		return nil
	})
	if err == nil {
		if _, errFold := subaccount.FoldFeeAccruals(l.RedisClient, l.SubaccountBal); errFold != nil {
			xlog.Errorf("fee accrual fold failed (fees stay queued): %v", errFold)
		}
	}
	return &types.FinaliseLiquidationResponse{
		RealisedLiquidatorPnlx18: realisedPnlLiquidatorx18,
		RealisedLiquidateePnlx18: realisedPnlLiquidateex18,
		LiquidatorFundingFeesx18: liquidatorFundingFeesx18,
		LiquidateeFundingFeesx18: liquidateeFundingFeesx18,
	}, err
}

// LiquidationFeex18 is the fee charged to the liquidatee: liquidationFraction * |amount * matchPrice|.
// FinaliseLiquidation credits it to the insurance fund (behavior-spec D-2).
func LiquidationFeex18(liquidationReq *types.FinaliseLiquidationRequest) *big.Int {
	notionalx18 := cutils.Divx18(new(big.Int).Mul(liquidationReq.AmountX18, liquidationReq.MatchPriceX18))
	return cutils.Divx18(new(big.Int).Mul(marketutils.GetLiquidationFractionx18(liquidationReq.ProductId), new(big.Int).Abs(notionalx18)))
}

// LiquidateeTradingFeex18 is the taker-style trading fee charged to the liquidatee:
// Divx18(factor * 1e13 * |Divx18(amount * matchPrice)|). SettlePnlForLiquidatee deducts it.
func LiquidateeTradingFeex18(liquidationReq *types.FinaliseLiquidationRequest) *big.Int {
	notionalx18 := new(big.Int).Abs(cutils.Divx18(new(big.Int).Mul(liquidationReq.AmountX18, liquidationReq.MatchPriceX18)))
	factor := cutils.GetBrokerFeeFactor(int(cutils.ExtractBrokerIdFromSubaccountHex(liquidationReq.LiquidateeSubaccountId)))
	return cutils.Divx18(new(big.Int).Mul(cutils.MulxCust(big.NewInt(factor), 13), notionalx18))
}

func cloneBalances(a *subaccountTypes.SubaccountBalances) subaccountTypes.SubaccountBalances {
	c := subaccountTypes.SubaccountBalances{SubaccountId: a.SubaccountId, SpotBalances: map[uint32]subaccountTypes.SpotBalance{}, PerpBalances: map[uint32]subaccountTypes.PerpBalance{}}
	for k, v := range a.SpotBalances {
		c.SpotBalances[k] = subaccountTypes.SpotBalance{ProductId: v.ProductId, Balancex18: new(big.Int).Set(v.Balancex18), Lockedx18: new(big.Int).Set(v.Lockedx18)}
	}
	for k, v := range a.PerpBalances {
		c.PerpBalances[k] = subaccountTypes.PerpBalance{ProductId: v.ProductId, Amountx18: new(big.Int).Set(v.Amountx18), VQuoteBalancex18: new(big.Int).Set(v.VQuoteBalancex18), LastCumFundingRatex18: new(big.Int).Set(v.LastCumFundingRatex18)}
	}
	return c
}

// checkLiquidator applies the NEAR contract's liquidator rules (behavior-spec R-5): the AMM stays
// within its position cap; any other liquidator must pass the match health rule
// (safety >= 0, or no worse than before) at the liquidation's prices.
func (l *LiquidationServiceImpl) checkLiquidator(req *types.FinaliseLiquidationRequest, before subaccountTypes.SubaccountBalances, after *subaccountTypes.SubaccountBalances) error {
	if l.AppState == nil {
		return fmt.Errorf("liquidation service has no app state")
	}
	prices, markets, cum, err := l.AppState.GetAppState(appstate.GetAppConfig(500*time.Millisecond, 5*time.Second, 24*time.Hour))
	if err != nil {
		return fmt.Errorf("app state for liquidator check: %w", err)
	}
	if strings.EqualFold(req.LiquidatorSubaccountId, contractUtils.AMM_SUBACCOUNT_ID) {
		if !markets[uint(req.ProductId)].AmmPositionAllowed(before.MustGetPerpBalance(req.ProductId).Amountx18, after.MustGetPerpBalance(req.ProductId).Amountx18) {
			return ErrAmmPositionCap
		}
		return nil
	}
	// the prices supplied with the liquidation win, as on-chain
	merged := make(map[string]ctypes.OraclePrice, len(prices))
	for k, v := range prices {
		merged[k] = v
	}
	for _, m := range []map[uint32]*big.Int{req.PerpOraclePricesX18, req.SpotOraclePricesX18} {
		for pid, p := range m {
			if sym, ok := marketutils.GetBaseSymbolForProduct(pid); ok && p != nil {
				merged[sym] = ctypes.OraclePrice{Pricex18: p, Symbol: sym}
			}
		}
	}
	// the liquidator's position was updated at the Redis cumulative funding used for the trade
	sb := subaccount.SafetyMarginx18(before, merged, markets, cum)
	sa := subaccount.SafetyMarginx18(*after, merged, markets, cum)
	if sa.Sign() >= 0 || sa.Cmp(sb) >= 0 {
		return nil
	}
	return ErrLiquidatorUnhealthy
}

// Returns realised pnl x18 (excluding fees) for liquidatee
func (l *LiquidationServiceImpl) SettlePnlForLiquidatee(liquidationReq *types.FinaliseLiquidationRequest, liquidateeSubaccountBalance *subaccountTypes.SubaccountBalances, cumulativeFundingRatex18 *big.Int) (*subaccountTypes.SubaccountBalances, *big.Int, *big.Int) {
	// Update perp balance
	// Warning: This can cause rounding errors
	notionalx18 := cutils.Divx18(new(big.Int).Mul(liquidationReq.AmountX18, liquidationReq.MatchPriceX18))
	liquidateePerpBalance := liquidateeSubaccountBalance.MustGetPerpBalance(liquidationReq.ProductId)
	// open interest is updated by FinaliseLiquidation once the liquidation is known to apply

	realisedPnlLiquidateex18, fundingFeesLiquidateex18 := liquidateePerpBalance.UpdateBalance(liquidationReq.AmountX18, new(big.Int).Neg(notionalx18), cumulativeFundingRatex18)
	// Update liquidatee subaccount balance
	liquidateeSubaccountBalance.PerpBalances[liquidationReq.ProductId] = liquidateePerpBalance
	// 1.5% of notional value will be collected as liquidation fee
	liquidationFeex18 := LiquidationFeex18(liquidationReq)
	// Taker fee of 0.05-0.06% will be collected as trading fee (credited to the D-7 fee account)
	tradingFeex18 := LiquidateeTradingFeex18(liquidationReq)

	// Pnl = realisedPnlLiquidateex18 - liquidationFeex18 - tradingFeex18
	pnlx36 := cutils.Mulx18(new(big.Int).Sub(realisedPnlLiquidateex18, new(big.Int).Add(liquidationFeex18, tradingFeex18)))

	// Include quote balance in pnlx36 and set quote balance to 0
	balanceToSettlex18 := liquidateeSubaccountBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18
	// Set quote balance to 0 and attempt to settle using other spots
	liquidateeSubaccountBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(balanceToSettlex18))
	pnlx36 = new(big.Int).Add(pnlx36, cutils.Mulx18(balanceToSettlex18))
	SettleLiqPnlUsingSpots(liquidationReq.SpotOraclePricesX18, pnlx36, liquidateeSubaccountBalance)

	return liquidateeSubaccountBalance, realisedPnlLiquidateex18, fundingFeesLiquidateex18
}

func (l *LiquidationServiceImpl) SettlePnlForLiquidator(liquidationReq *types.FinaliseLiquidationRequest, liquidatorSubaccountBalance *subaccountTypes.SubaccountBalances, cumulativeFundingRatex18 *big.Int) (*subaccountTypes.SubaccountBalances, *big.Int, *big.Int) {
	// Update perp balance
	// Warning: This can cause rounding errors
	amountx18 := new(big.Int).Neg(liquidationReq.AmountX18)
	notionalx18 := cutils.Divx18(new(big.Int).Mul(amountx18, liquidationReq.MatchPriceX18))
	perpBalance := liquidatorSubaccountBalance.MustGetPerpBalance(liquidationReq.ProductId)
	realisedPnlLiquidatorx18, fundingFeesLiquidatorx18 := perpBalance.UpdateBalance(amountx18, new(big.Int).Neg(notionalx18), cumulativeFundingRatex18)
	// Update liquidator subaccount perp balance
	liquidatorSubaccountBalance.PerpBalances[liquidationReq.ProductId] = perpBalance
	// No trading fee for liquidator
	tradingFeex18 := cutils.GetBig0()

	// Pnl = realisedPnlLiquidateex18 - liquidationFeex18 - tradingFeex18
	pnlx18 := new(big.Int).Sub(realisedPnlLiquidatorx18, tradingFeex18)

	// Update Quote Spot balance of liquidatee
	liquidatorSpotBalance := liquidatorSubaccountBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID)
	liquidatorSpotBalance.UpdateBalance(pnlx18)
	liquidatorSubaccountBalance.SpotBalances[contractUtils.QUOTE_TOKEN_PRODUCT_ID] = liquidatorSpotBalance

	return liquidatorSubaccountBalance, realisedPnlLiquidatorx18, fundingFeesLiquidatorx18
}

// Fetch balance
// Perform settleUsingSpots
// Then make the quote balance zero if negative
// Update the balance in redis
func (l *LiquidationServiceImpl) SettleUsingInsurance(settleUsingInsuranceRequest *types.SettleWithInsuranceRequest) error {
	// Lock subaccount's balance
	remainingQuoteX18 := cutils.GetBig0()
	_, err := xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(settleUsingInsuranceRequest.SubaccountId), func() (*xredis.NOOP, error) {
		return xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(contractUtils.INSURANCE_SUBACCOUNT_ID), func() (*xredis.NOOP, error) {
			balances := l.MustGetSubaccountBalancesFromIds([]string{settleUsingInsuranceRequest.SubaccountId, contractUtils.INSURANCE_SUBACCOUNT_ID})
			subaccountBalance := balances[0]
			insuranceBalance := balances[1]

			balanceToSettlex18 := subaccountBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18
			// Set quote balance to 0 and attempt to settle using other spots
			subaccountBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(balanceToSettlex18))
			remainingQuoteX18 = SettleLiqPnlUsingSpots(settleUsingInsuranceRequest.SpotOraclePricesX18, cutils.Mulx18(balanceToSettlex18), &subaccountBalance)

			if remainingQuoteX18.Sign() < 0 {
				insuranceBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, remainingQuoteX18)
				insuranceQuoteAfterSettlement := insuranceBalance.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18
				if insuranceQuoteAfterSettlement.Sign() < 0 {
					if xclient.GlobalDiscordClient != nil {
						xclient.GlobalDiscordClient.SendWebhookMessage("Insurance is out of funds. Require additional amount - " + new(big.Int).Neg(insuranceQuoteAfterSettlement).String() + " for " + settleUsingInsuranceRequest.SubaccountId)
					}
					return nil, fmt.Errorf("insurance is out of funds. Require additional amount - %v for %v", new(big.Int).Neg(insuranceQuoteAfterSettlement), settleUsingInsuranceRequest.SubaccountId)
				}

				// Make the quote balance for subaccount zero
				subaccountBalance.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(remainingQuoteX18))
			}

			// Update balances in redis
			_, err := l.RedisClient.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				errI := subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
				if errI != nil {
					xlog.Errorf("Error replacing subaccount balance in redis: %v", errI)
					return errI
				}
				errI = subaccount.ReplaceBalanceInRedis(pipe, &insuranceBalance)
				if errI != nil {
					xlog.Errorf("Error replacing insurance balance in redis: %v", errI)
				}
				return errI
			})

			return nil, err
		})
	})

	if err != nil {
		xlog.Errorf("Error settling using insurance: %v", err)
		return err
	}

	if remainingQuoteX18.Sign() < 0 {
		xlog.Infof("Insurance funds paid out: %v for subaccountIdHex: %v", new(big.Int).Neg(remainingQuoteX18), settleUsingInsuranceRequest.SubaccountId)
		if xclient.GlobalDiscordClient != nil {
			xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Insurance funds paid out: %v for subaccountHex: %v", new(big.Int).Neg(remainingQuoteX18).String(), settleUsingInsuranceRequest.SubaccountId))
		}
	} else {
		xlog.Infof("Insurance funds not required for subaccountIdHex: %v", settleUsingInsuranceRequest.SubaccountId)
	}

	return err
}
