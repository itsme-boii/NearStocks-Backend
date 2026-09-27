package tests

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"math/big"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// TestUpdateLocalBalanceForOrderMatchWithFundingFees verifies that funding fees are returned
// from UpdateLocalBalanceForOrderMatch method
func TestUpdateLocalBalanceForOrderMatchWithFundingFees(t *testing.T) {
	setBalanceServiceEnv()

	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()

	testutils.WithSetupMockRedis(t, func() {
		contractUtils.Init()
		subaccountHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"

		// Setup subaccount with existing position
		subaccountBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: subaccountHex,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.ARB_USDC: {
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  cutils.GetBig0(),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(100)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
					LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
				},
			},
		}

		balancePerpRequest := types.BalancePerpPayload{
			SubaccountId:     subaccountHex,
			Amountx18:        cutils.Mulx18(big.NewInt(-50)),       // Closing half position
			VQuoteBalancex18: cutils.MulxCust(big.NewInt(495), 17), // $49.5 position
		}

		oraclePricesMap := map[string]ctypes.OraclePrice{
			"ETH": {
				Pricex18: cutils.MulxCust(big.NewInt(99), 16), // $0.99
				Symbol:   "ETH",
			},
			"USDC": {
				Pricex18: cutils.Mulx18(big.NewInt(1)),
				Symbol:   "USDC",
			},
		}

		perpetualMarketsMap := map[uint]subaccountTypes.PerpetualMarket{
			contractUtils.ETH_MARKET: {
				ProductId:                    contractUtils.ETH_MARKET,
				BaseAsset:                    "ETH",
				MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
				InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
			},
		}

		// Set funding rate change: 0.1% (0.001) which will result in funding fees
		cumulativeFundingRateMap := map[string]*big.Int{
			"ETH": cutils.MulxCust(big.NewInt(1), 15), // 0.1% = 0.001
		}

		balanceService := services.NewBalanceService()
		_, isHealthyBalanceUpdate, _, fundingFees := balanceService.UpdateLocalBalanceForOrderMatch(
			subaccountBalances,
			balancePerpRequest,
			contractUtils.ETH_MARKET,
			true, // isTaker
			oraclePricesMap,
			perpetualMarketsMap,
			cumulativeFundingRateMap,
		)

		// Verify balance update was successful
		assert.True(t, isHealthyBalanceUpdate, "Expected healthy balance update")

		// Verify funding fees are calculated correctly
		// Formula: fundingFee = -((currentFundingRate - lastFundingRate) * VQuoteBalance)
		// = -((0.001 - 0) * (-100)) = -(-0.1) = 0.1
		// Positive funding fee means user pays (money goes out from user's account)
		assert.NotNil(t, fundingFees, "Funding fees should not be nil")
		expectedFundingFee := cutils.MulxCust(big.NewInt(1), 17) // 0.1
		assert.Equalf(t, fundingFees.Cmp(expectedFundingFee), 0,
			"Funding fees should be %s (0.1), got %s",
			expectedFundingFee.String(), fundingFees.String())

		xlog.Infof("✅ Balance Service - Funding fees verified: %s", fundingFees.String())
	})
}

// To test that funding fees are returned from FinaliseLiquidation method
func TestFinaliseLiquidationWithFundingFees(t *testing.T) {
	setBalanceServiceEnv()

	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()

	testutils.WithSetupMockRedis(t, func() {
		contractUtils.Init()

		liquidateeHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"
		liquidatorHex := "0x0000000000015631e1e23aeefae435106017bbcee2b91676e43b000000000002"

		// Setup liquidatee with existing position
		liquidateeBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: liquidateeHex,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.ARB_USDC: {
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  cutils.GetBig0(),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(1)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-100)),
					LastCumFundingRatex18: cutils.GetBig0(),
				},
			},
		}

		// Setup liquidator with spot balance
		liquidatorBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: liquidatorHex,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.ARB_USDC: {
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(500)),
					Lockedx18:  cutils.GetBig0(),
				},
			},
		}

		// Store balances in Redis
		_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			if err := subaccount.ReplaceBalanceInRedis(pipe, &liquidateeBalances); err != nil {
				return err
			}
			return subaccount.ReplaceBalanceInRedis(pipe, &liquidatorBalances)
		})
		assert.NoError(t, err)

		symbol, _ := marketutils.GetFundingSymbolForProduct(contractUtils.ETH_MARKET)

		// Set funding rate change of 0.1% (0.001)
		cumulativeFundingRatex18 := cutils.MulxCust(big.NewInt(1), 15) // 0.1% = 0.001
		data := &xredis.CumulativeFundingRateData{
			CumulativeFundingRate:      cumulativeFundingRatex18.String(),
			CumulativeFundingTimestamp: "0",
		}
		// Store cumulative funding rate in Redis
		_, err = xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return pipe.Set(context.Background(), xredis.GetCumulativeFundingRateKey(symbol), data, 0).Err()
		})
		assert.NoError(t, err)

		txnCounter := uint(1)
		liquidationReq := &types.FinaliseLiquidationRequest{
			TxnCounter:             &txnCounter,
			LiquidateeOrderId:      1,
			LiquidateeSubaccountId: liquidateeHex,
			LiquidatorSubaccountId: liquidatorHex,
			ProductId:              contractUtils.ETH_MARKET,
			AmountX18:              cutils.Mulx18(big.NewInt(-1)),  // Closing position
			MatchPriceX18:          cutils.Mulx18(big.NewInt(200)), // Selling at $200
			PerpOraclePricesX18: map[uint32]*big.Int{
				contractUtils.ETH_MARKET: cutils.Mulx18(big.NewInt(200)), // $200
			},
			SpotOraclePricesX18: map[uint32]*big.Int{
				contractUtils.ARB_USDC: cutils.Mulx18(big.NewInt(1)), // USDC = $1
			},
		}

		// ETH at $200, margins 10% / 5%, the same cumulative funding as Redis: the liquidator
		// (500 USDC taking 1 ETH) passes the R-5 check
		liquidationService := &services.LiquidationServiceImpl{RedisClient: xredis.GetRedisClient(), SubaccountBal: subaccount.NewSubaccountBalanceImpl(), AppState: ethAppState{symbol: symbol, cum: cumulativeFundingRatex18}}
		response, err := liquidationService.FinaliseLiquidation(liquidationReq)

		// Verify no error
		assert.NoError(t, err, "FinaliseLiquidation should not return error")

		assert.NotNil(t, response.LiquidateeFundingFeesx18, "LiquidateeFundingFeesx18 should not be nil")
		expectedLiquidateeFundingFee := cutils.MulxCust(big.NewInt(1), 17) // 0.1
		assert.Equalf(t, response.LiquidateeFundingFeesx18.Cmp(expectedLiquidateeFundingFee), 0,
			"LiquidateeFundingFeesx18 should be %s (0.1), got %s",
			expectedLiquidateeFundingFee.String(), response.LiquidateeFundingFeesx18.String())

		xlog.Infof("✅ Balance Service Liquidation - LiquidateeFundingFeesx18: %s, LiquidatorFundingFeesx18: %s",
			response.LiquidateeFundingFeesx18.String(), response.LiquidatorFundingFeesx18.String())
	})
}

type ethAppState struct {
	fixedAppState
	symbol string
	cum    *big.Int
}

func (e ethAppState) GetAppState(_ ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	return map[string]ctypes.OraclePrice{e.symbol: {Pricex18: cutils.Mulx18(big.NewInt(200)), Symbol: e.symbol}, "USDC": {Pricex18: cutils.Mulx18(big.NewInt(1)), Symbol: "USDC"}},
		map[uint]subaccountTypes.PerpetualMarket{uint(contractUtils.ETH_MARKET): {ProductId: uint(contractUtils.ETH_MARKET), BaseAsset: e.symbol, InitialMarginFractionx18: cutils.FloatStrToX18("0.1"), MaintenanceMarginFractionx18: cutils.FloatStrToX18("0.05")}},
		map[string]*big.Int{e.symbol: e.cum}, nil
}
