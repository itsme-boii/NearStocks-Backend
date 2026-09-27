package tests

// D-7 fee account and R-5 liquidator checks in the balance-server (behavior-spec §4, §6).

import (
	"context"
	"errors"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

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
	"github.com/stretchr/testify/require"
)

const btc = uint32(3) // mainnet product 3 = BTC

// fixedAppState: constant BTC and USDC prices, BTC margins 10% / 5%, zero funding.
type fixedAppState struct{ ammCap *big.Int }

func (f fixedAppState) GetAppState(_ ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	return f.prices(), f.markets(), map[string]*big.Int{"BTC": big.NewInt(0)}, nil
}
func (f fixedAppState) prices() map[string]ctypes.OraclePrice {
	return map[string]ctypes.OraclePrice{"BTC": {Pricex18: cutils.Mulx18(big.NewInt(60_000)), Symbol: "BTC"}, "USDC": {Pricex18: cutils.Mulx18(big.NewInt(1)), Symbol: "USDC"}}
}
func (f fixedAppState) markets() map[uint]subaccountTypes.PerpetualMarket {
	return map[uint]subaccountTypes.PerpetualMarket{uint(btc): {ProductId: uint(btc), BaseAsset: "BTC", InitialMarginFractionx18: cutils.FloatStrToX18("0.1"), MaintenanceMarginFractionx18: cutils.FloatStrToX18("0.05"), AmmMaxPositionx18: f.ammCap}}
}
func (f fixedAppState) GetAllPerpMarkets(_ ...time.Duration) map[uint]subaccountTypes.PerpetualMarket {
	return f.markets()
}
func (f fixedAppState) GetAllFundingRates(_ ...time.Duration) map[string]*big.Int {
	return map[string]*big.Int{"BTC": big.NewInt(0)}
}
func (f fixedAppState) GetAllOraclePrices(_ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
	return f.prices(), nil
}

func quoteAccount(id string, usdc int64) *subaccountTypes.SubaccountBalances {
	return &subaccountTypes.SubaccountBalances{
		SubaccountId: id,
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{4: {ProductId: 4, Balancex18: cutils.Mulx18(big.NewInt(usdc)), Lockedx18: big.NewInt(0)}},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
	}
}

func seedAccounts(t *testing.T, accounts ...*subaccountTypes.SubaccountBalances) {
	ctx := context.Background()
	_, err := xredis.GetRedisClient().TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, a := range accounts {
			if err := subaccount.ReplaceBalanceInRedis(pipe, a); err != nil {
				return err
			}
		}
		return pipe.Set(ctx, xredis.GetCumulativeFundingRateKey("BTC"), xredis.CumulativeFundingRateData{CumulativeFundingRate: "0", CumulativeFundingTimestamp: "0"}, 0).Err()
	})
	require.NoError(t, err)
}

func balanceOf(id string) *subaccountTypes.SubaccountBalances {
	b := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{id})[0]
	return &b
}

// userSub builds a broker-2 subaccount id with a distinct address per i.
func userSub(i int) string {
	return sub32(2, "0x"+strconv.FormatInt(int64(0x10000000+i), 16)+"00000000000000000000000000000000", 0)
}

func TestFeeAccrualFoldIsLossless(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()
	testutils.WithSetupMockRedis(t, func() {
		rc := xredis.GetRedisClient()
		sb := subaccount.NewSubaccountBalanceImpl()
		var wg sync.WaitGroup
		want := big.NewInt(0)
		for i := 1; i <= 40; i++ {
			fee := new(big.Int).Mul(big.NewInt(int64(i)), big.NewInt(1_000_000_000_000_000_007))
			want.Add(want, fee)
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := rc.TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					subaccount.AccrueFee(pipe, fee)
					subaccount.AccrueFee(pipe, big.NewInt(0)) // ignored
					return nil
				})
				assert.NoError(t, err)
				_, err = subaccount.FoldFeeAccruals(rc, sb)
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
		_, err := subaccount.FoldFeeAccruals(rc, sb)
		require.NoError(t, err)
		n, _ := rc.LLen(context.Background(), subaccount.FeeAccrualKey()).Result()
		assert.Equal(t, int64(0), n, "queue fully folded")
		assert.Equal(t, want.String(), balanceOf(contractUtils.TRADING_FEES_SUBACCOUNT_ID).MustGetSpotBalance(4).Balancex18.String())
	})
}

// Many takers hit the AMM at once. Every match must land (no lost updates on the shared AMM and
// fee accounts), and the fee account must hold exactly the takers' fees.
func TestConcurrentMatchesAgainstAmmLoseNothing(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()
	testutils.WithSetupMockRedis(t, func() {
		const n = 24
		bs := services.NewBalanceServiceWithAppState(fixedAppState{})
		accounts := []*subaccountTypes.SubaccountBalances{quoteAccount(contractUtils.AMM_SUBACCOUNT_ID, 0)}
		for i := 0; i < n; i++ {
			accounts = append(accounts, quoteAccount(userSub(i), 10_000))
		}
		seedAccounts(t, accounts...)

		amount := cutils.FloatStrToX18("0.01")
		price := cutils.Mulx18(big.NewInt(60_000))
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				counter := uint(i + 1)
				q := cutils.Divx18(new(big.Int).Mul(amount, price))
				e, _, _, _, _ := bs.AtomicUpdateBalanceForOrderMatch(types.UpdateSubaccountForMatchRequest{
					TxnCounter: &counter, MarketId: uint(btc),
					Maker: types.BalancePerpPayload{SubaccountId: contractUtils.AMM_SUBACCOUNT_ID, Amountx18: new(big.Int).Neg(amount), VQuoteBalancex18: new(big.Int).Set(q), UnlockQuotex18: big.NewInt(0), SkipUnlock: true},
					Taker: types.BalancePerpPayload{SubaccountId: userSub(i), Amountx18: new(big.Int).Set(amount), VQuoteBalancex18: new(big.Int).Neg(q), UnlockQuotex18: big.NewInt(0), SkipUnlock: true},
				})
				assert.Nil(t, e)
			}(i)
		}
		wg.Wait()
		_, err := subaccount.FoldFeeAccruals(xredis.GetRedisClient(), subaccount.NewSubaccountBalanceImpl())
		require.NoError(t, err)

		amm := balanceOf(contractUtils.AMM_SUBACCOUNT_ID)
		assert.Equal(t, new(big.Int).Mul(amount, big.NewInt(-n)).String(), amm.MustGetPerpBalance(btc).Amountx18.String(), "every match reached the AMM")
		// broker 2 pays 0.06% of 600 = 0.36 per taker; the AMM pays nothing
		wantFees := new(big.Int).Mul(cutils.FloatStrToX18("0.36"), big.NewInt(n))
		assert.Equal(t, wantFees.String(), balanceOf(contractUtils.TRADING_FEES_SUBACCOUNT_ID).MustGetSpotBalance(4).Balancex18.String())
	})
}

func TestAmmCapRejectsGrowthInMatch(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()
	testutils.WithSetupMockRedis(t, func() {
		bs := services.NewBalanceServiceWithAppState(fixedAppState{ammCap: cutils.FloatStrToX18("0.015")})
		seedAccounts(t, quoteAccount(contractUtils.AMM_SUBACCOUNT_ID, 0), quoteAccount(userSub(1), 10_000))
		amount := cutils.FloatStrToX18("0.01")
		q := cutils.Divx18(new(big.Int).Mul(amount, cutils.Mulx18(big.NewInt(60_000))))
		match := func(counter uint) *services.ErrorUpdatingBalance {
			e, _, _, _, _ := bs.AtomicUpdateBalanceForOrderMatch(types.UpdateSubaccountForMatchRequest{
				TxnCounter: &counter, MarketId: uint(btc),
				Maker: types.BalancePerpPayload{SubaccountId: contractUtils.AMM_SUBACCOUNT_ID, Amountx18: new(big.Int).Neg(amount), VQuoteBalancex18: new(big.Int).Set(q), UnlockQuotex18: big.NewInt(0), SkipUnlock: true},
				Taker: types.BalancePerpPayload{SubaccountId: userSub(1), Amountx18: new(big.Int).Set(amount), VQuoteBalancex18: new(big.Int).Neg(q), UnlockQuotex18: big.NewInt(0), SkipUnlock: true},
			})
			return e
		}
		assert.Nil(t, match(1)) // AMM short 0.01
		e := match(2)           // would be 0.02 > 0.015
		require.NotNil(t, e)
		assert.True(t, e.MakerError)
		assert.Equal(t, cutils.FloatStrToX18("-0.01").String(), balanceOf(contractUtils.AMM_SUBACCOUNT_ID).MustGetPerpBalance(btc).Amountx18.String())
	})
}

func TestFinaliseLiquidationLiquidatorChecksAndFeeAccount(t *testing.T) {
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()
	testutils.WithSetupMockRedis(t, func() {
		victim := userSub(1)
		// long 0.1 BTC entered at 65k, 700 USDC: below maintenance at 60k
		newVictim := func() *subaccountTypes.SubaccountBalances {
			v := quoteAccount(victim, 700)
			v.PerpBalances[btc] = subaccountTypes.PerpBalance{ProductId: btc, Amountx18: cutils.FloatStrToX18("0.1"), VQuoteBalancex18: cutils.Mulx18(big.NewInt(-6_500)), LastCumFundingRatex18: big.NewInt(0)}
			return v
		}
		req := func(liquidator string) *types.FinaliseLiquidationRequest {
			return &types.FinaliseLiquidationRequest{
				TxnCounter: new(uint), LiquidatorSubaccountId: liquidator, LiquidateeSubaccountId: victim, ProductId: btc,
				AmountX18: cutils.FloatStrToX18("-0.1"), MatchPriceX18: cutils.Mulx18(big.NewInt(60_000)),
				PerpOraclePricesX18: map[uint32]*big.Int{btc: cutils.Mulx18(big.NewInt(60_000))}, SpotOraclePricesX18: map[uint32]*big.Int{4: cutils.Mulx18(big.NewInt(1))},
			}
		}
		ls := &services.LiquidationServiceImpl{RedisClient: xredis.GetRedisClient(), SubaccountBal: subaccount.NewSubaccountBalanceImpl(), AppState: fixedAppState{ammCap: cutils.FloatStrToX18("0.05")}}
		oiKey := xredis.GetTotalLongPositionKey(strconv.Itoa(int(btc)))

		// R-5: a 100 USDC user cannot take on 0.1 BTC (IM 600); nothing is written
		poor := userSub(2)
		seedAccounts(t, newVictim(), quoteAccount(poor, 100), quoteAccount(contractUtils.INSURANCE_SUBACCOUNT_ID, 1_000))
		_ = xredis.GetRedisClient().Set(context.Background(), oiKey, "100000000000000000", 0).Err()
		_, err := ls.FinaliseLiquidation(req(poor))
		assert.True(t, errors.Is(err, services.ErrLiquidatorUnhealthy), "%v", err)
		assert.Equal(t, "100000000000000000", balanceOf(victim).MustGetPerpBalance(btc).Amountx18.String(), "victim untouched")
		oi, _ := xredis.GetRedisClient().Get(context.Background(), oiKey).Result()
		assert.Equal(t, "100000000000000000", oi, "open interest untouched")

		// the AMM is bound by its 0.05 cap
		seedAccounts(t, quoteAccount(contractUtils.AMM_SUBACCOUNT_ID, 0))
		_, err = ls.FinaliseLiquidation(req(contractUtils.AMM_SUBACCOUNT_ID))
		assert.True(t, errors.Is(err, services.ErrAmmPositionCap), "%v", err)

		// a funded user liquidates; D-2 fee to insurance, D-7 trading fee to the fee account
		rich := userSub(3)
		seedAccounts(t, quoteAccount(rich, 100_000))
		_, err = ls.FinaliseLiquidation(req(rich))
		require.NoError(t, err)
		assert.Equal(t, "0", balanceOf(victim).MustGetPerpBalance(btc).Amountx18.String())
		assert.Equal(t, cutils.Mulx18(big.NewInt(1_090)).String(), balanceOf(contractUtils.INSURANCE_SUBACCOUNT_ID).MustGetSpotBalance(4).Balancex18.String())
		assert.Equal(t, cutils.FloatStrToX18("3.6").String(), balanceOf(contractUtils.TRADING_FEES_SUBACCOUNT_ID).MustGetSpotBalance(4).Balancex18.String())
	})
}

// redisFundingAppState: 10% / 5% margins on every perp, no stored prices (liquidation requests carry
// them), and cumulative funding read from Redis like FinaliseLiquidation does.
type redisFundingAppState struct{ fixedAppState }

func (redisFundingAppState) GetAppState(_ ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	markets := map[uint]subaccountTypes.PerpetualMarket{}
	cum := map[string]*big.Int{}
	for _, pid := range contractUtils.ALL_PERPS_ON_CONTRACT {
		sym := contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[pid]
		markets[uint(pid)] = subaccountTypes.PerpetualMarket{ProductId: uint(pid), BaseAsset: sym, InitialMarginFractionx18: cutils.FloatStrToX18("0.1"), MaintenanceMarginFractionx18: cutils.FloatStrToX18("0.05")}
		c, err := xredis.GetCumulativeFundingRateForSymbol(xredis.GetRedisClient(), sym)
		if err != nil || c == nil {
			c = big.NewInt(0)
		}
		cum[sym] = c
	}
	return map[string]ctypes.OraclePrice{}, markets, cum, nil
}
