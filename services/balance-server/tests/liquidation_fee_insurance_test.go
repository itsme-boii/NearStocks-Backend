package tests

import (
	"context"
	"math/big"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// behavior-spec H-3 / D-2: the liquidation fee charged to the liquidatee is credited to the insurance fund.
func TestLiquidationFeeCreditedToInsurance(t *testing.T) {
	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()

	liquidatee := "0x000000000001d37ed507ca37faa17079bff5e46dcede951577db000000000002"
	userLiquidator := "0x0000000000015631e1e23aeefae435106017bbcee2b91676e43b000000000002"

	quote := func(sb *subaccountTypes.SubaccountBalances) *big.Int {
		return sb.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18
	}
	newLiquidatee := func() *subaccountTypes.SubaccountBalances {
		return &subaccountTypes.SubaccountBalances{
			SubaccountId: liquidatee,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.QUOTE_TOKEN_PRODUCT_ID: {ProductId: contractUtils.QUOTE_TOKEN_PRODUCT_ID, Balancex18: cutils.Mulx18(big.NewInt(100)), Lockedx18: big.NewInt(0)},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.TRUMP_MARKET: {ProductId: contractUtils.TRUMP_MARKET, Amountx18: cutils.Mulx18(big.NewInt(1)), VQuoteBalancex18: cutils.Mulx18(big.NewInt(-300)), LastCumFundingRatex18: cutils.GetBig0()},
			},
		}
	}
	newLiquidator := func(id string, quoteUnits int64) *subaccountTypes.SubaccountBalances {
		return &subaccountTypes.SubaccountBalances{
			SubaccountId: id,
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.QUOTE_TOKEN_PRODUCT_ID: {ProductId: contractUtils.QUOTE_TOKEN_PRODUCT_ID, Balancex18: cutils.Mulx18(big.NewInt(quoteUnits)), Lockedx18: big.NewInt(0)},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
		}
	}
	request := func(liquidator string) *types.FinaliseLiquidationRequest {
		return &types.FinaliseLiquidationRequest{
			TxnCounter:             &ONE,
			LiquidateeOrderId:      1,
			LiquidateeSubaccountId: liquidatee,
			LiquidatorSubaccountId: liquidator,
			ProductId:              contractUtils.TRUMP_MARKET,
			AmountX18:              cutils.Mulx18(big.NewInt(-1)),
			MatchPriceX18:          cutils.Mulx18(big.NewInt(100)),
			PerpOraclePricesX18:    map[uint32]*big.Int{contractUtils.TRUMP_MARKET: cutils.Mulx18(big.NewInt(100))},
			SpotOraclePricesX18:    map[uint32]*big.Int{contractUtils.QUOTE_TOKEN_PRODUCT_ID: cutils.Mulx18(big.NewInt(1))},
		}
	}

	testutils.WithSetupMockRedis(t, func() {
		// TRUMP (product 11) at 100, margins 10% / 5%: the user liquidators below pass the R-5 check
		ls := &services.LiquidationServiceImpl{RedisClient: xredis.GetRedisClient(), SubaccountBal: subaccount.NewSubaccountBalanceImpl(), AppState: trumpAppState{}}
		seed := func(accounts ...*subaccountTypes.SubaccountBalances) {
			_, err := ls.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				for _, a := range accounts {
					if err := subaccount.ReplaceBalanceInRedis(pipe, a); err != nil {
						return err
					}
				}
				return pipe.Set(context.Background(), xredis.GetCumulativeFundingRateKey(contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.TRUMP_MARKET]),
					xredis.CumulativeFundingRateData{CumulativeFundingRate: "0", CumulativeFundingTimestamp: "0"}, 0).Err()
			})
			assert.NoError(t, err)
		}

		t.Run("user liquidator: insurance gains exactly the fee", func(t *testing.T) {
			req := request(userLiquidator)
			fee := services.LiquidationFeex18(req)
			assert.Equal(t, 1, fee.Sign(), "fee must be positive for this case")

			insuranceStart := cutils.Mulx18(big.NewInt(1000))
			seed(newLiquidatee(), newLiquidator(userLiquidator, 500), newLiquidator(contractUtils.INSURANCE_SUBACCOUNT_ID, 1000))

			_, err := ls.FinaliseLiquidation(req)
			assert.NoError(t, err)

			ins := ls.MustGetBalanceFromRedis(contractUtils.INSURANCE_SUBACCOUNT_ID)
			assert.Equal(t, new(big.Int).Add(insuranceStart, fee).String(), quote(ins).String())
			_, _ = ls.GetRedisClient().FlushAll(context.Background()).Result()
		})

		t.Run("insurance as liquidator: fee credited once, no self-deadlock", func(t *testing.T) {
			req := request(contractUtils.INSURANCE_SUBACCOUNT_ID)
			fee := services.LiquidationFeex18(req)

			// Reference: same liquidation with a user liquidator gives the liquidator's pnl effect.
			seed(newLiquidatee(), newLiquidator(userLiquidator, 1000), newLiquidator(contractUtils.INSURANCE_SUBACCOUNT_ID, 0))
			_, err := ls.FinaliseLiquidation(request(userLiquidator))
			assert.NoError(t, err)
			liquidatorQuoteAsUser := quote(ls.MustGetBalanceFromRedis(userLiquidator))
			_, _ = ls.GetRedisClient().FlushAll(context.Background()).Result()

			seed(newLiquidatee(), newLiquidator(contractUtils.INSURANCE_SUBACCOUNT_ID, 1000))
			_, err = ls.FinaliseLiquidation(req)
			assert.NoError(t, err)
			ins := ls.MustGetBalanceFromRedis(contractUtils.INSURANCE_SUBACCOUNT_ID)
			assert.Equal(t, new(big.Int).Add(liquidatorQuoteAsUser, fee).String(), quote(ins).String())
			_, _ = ls.GetRedisClient().FlushAll(context.Background()).Result()
		})
	})
}

type trumpAppState struct{ fixedAppState }

func (trumpAppState) GetAppState(_ ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	sym := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.TRUMP_MARKET]
	return map[string]ctypes.OraclePrice{sym: {Pricex18: cutils.Mulx18(big.NewInt(100)), Symbol: sym}, "USDC": {Pricex18: cutils.Mulx18(big.NewInt(1)), Symbol: "USDC"}},
		map[uint]subaccountTypes.PerpetualMarket{uint(contractUtils.TRUMP_MARKET): {ProductId: uint(contractUtils.TRUMP_MARKET), BaseAsset: sym, InitialMarginFractionx18: cutils.FloatStrToX18("0.1"), MaintenanceMarginFractionx18: cutils.FloatStrToX18("0.05")}},
		map[string]*big.Int{sym: big.NewInt(0)}, nil
}
