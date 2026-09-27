package tests

import (
	"context"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/testutils"
	"math/big"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// This also checks MustGetSubaccountBalancesFromIds
func TestReplaceBalanceInRedis(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		subaccountBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.ARB_USDC: {
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  cutils.GetBig0(),
				},
				contractUtils.ETH_USDC: {
					ProductId:  contractUtils.ETH_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
					Lockedx18:  cutils.GetBig0(),
				},
			},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{
				contractUtils.ETH_MARKET: {
					ProductId:             contractUtils.ETH_MARKET,
					Amountx18:             cutils.Mulx18(big.NewInt(-1)),
					VQuoteBalancex18:      cutils.Mulx18(big.NewInt(100)),
					LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
				},
			},
		}

		_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalances)
		})
		assert.NoErrorf(t, err, "Error in replacing balance in redis")

		subaccountBal := subaccount.NewSubaccountBalanceImpl()
		// Fetch the balance from redis and compare
		subaccountBalancesFromRedis := subaccountBal.MustGetSubaccountBalancesFromIds([]string{subaccountBalances.SubaccountId})[0]
		assert.True(t, subaccountBalances.Equals(&subaccountBalancesFromRedis), "Balances are not equal")
	})
}

func TestReplaceWithEmptyBalanceInRedis(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		subaccountBalances := subaccountTypes.SubaccountBalances{
			SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
			SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
			PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
		}

		_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
			return subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalances)
		})
		assert.NoErrorf(t, err, "Error in replacing balance in redis")

		subaccountBal := subaccount.NewSubaccountBalanceImpl()
		// Fetch the balance from redis and compare
		subaccountBalancesFromRedis := subaccountBal.MustGetSubaccountBalancesFromIds([]string{subaccountBalances.SubaccountId})[0]
		assert.True(t, subaccountBalances.Equals(&subaccountBalancesFromRedis), "Balances are not equal")
	})
}

func TestGetTotalEquityx36(t *testing.T) {
	os.Setenv("ENV", "TESTNET")
	contractUtils.Init()
	type TestCase struct {
		name                     string
		subaccountBalances       subaccountTypes.SubaccountBalances
		oraclePricesMap          map[string]ctypes.OraclePrice
		cumulativeFundingRateMap map[string]*big.Int
		expectedTotalEquityx36   *big.Int
	}

	testCases := []TestCase{
		{
			name: "No spot. No upnl. Only unrealized funding",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.ETH_MARKET: {
						ProductId:             contractUtils.ETH_MARKET,
						Amountx18:             cutils.Mulx18(big.NewInt(-1)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(100)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(100)),
					Symbol:   "ETH",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{
				"ETH": cutils.Mulx18(big.NewInt(1)),
			},
			expectedTotalEquityx36: cutils.MulxCust(big.NewInt(100), 36), // User is holding short and funding is positive -> profit = 100 * 1
		},
		{
			name: "No spot. No upnl. Only unrealized funding. Negative funding",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.ETH_MARKET: {
						ProductId:             contractUtils.ETH_MARKET,
						Amountx18:             cutils.Mulx18(big.NewInt(-1)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(100)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(100)),
					Symbol:   "ETH",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{
				"ETH": cutils.Mulx18(big.NewInt(-1)),
			},
			expectedTotalEquityx36: cutils.MulxCust(big.NewInt(-100), 36), // User is holding short and funding is negative -> loss = -100 * 1
		},
		{
			name: "No spot. No urealized funding. Negative unrealized pnl",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.ETH_MARKET: {
						ProductId:             contractUtils.ETH_MARKET,
						Amountx18:             cutils.Mulx18(big.NewInt(-1)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(100)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(101)),
					Symbol:   "ETH",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{
				"ETH": cutils.Mulx18(big.NewInt(0)),
			},
			expectedTotalEquityx36: cutils.MulxCust(big.NewInt(-1), 36), // User is holding short and price increased -> profit = 100 - 101
		},
		{
			name: "No spot. No urealized funding. Only positive unrealized pnl",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.ETH_MARKET: {
						ProductId:             contractUtils.ETH_MARKET,
						Amountx18:             cutils.Mulx18(big.NewInt(10)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(101)),
					Symbol:   "ETH",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{
				"ETH": cutils.Mulx18(big.NewInt(0)),
			},
			expectedTotalEquityx36: cutils.MulxCust(big.NewInt(10), 36), // User is holding long and price increased -> profit = (1010- 1000)
		},
		{
			name: "No upnl. No urealized funding. Only one spot balance",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"USDC": {
					Pricex18: cutils.MulxCust(big.NewInt(10001), 14),
					Symbol:   "USDC",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{},
			expectedTotalEquityx36:   cutils.MulxCust(big.NewInt(10001), 34), // User is holding 100 USDC
		},
		{
			name: "No upnl. No urealized funding. Only one spot balance. Negative balance",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.Mulx18(big.NewInt(-100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"USDC": {
					Pricex18: cutils.MulxCust(big.NewInt(10001), 14),
					Symbol:   "USDC",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{},
			expectedTotalEquityx36:   cutils.MulxCust(big.NewInt(-10001), 34), // User is holding -100 USDC
		},
		{
			name: "Spot, upnl and unrealized funding",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
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
						Amountx18:             cutils.Mulx18(big.NewInt(10)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-1000)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(101)),
					Symbol:   "ETH",
				},
				"USDC": {
					Pricex18: cutils.MulxCust(big.NewInt(10001), 14),
					Symbol:   "USDC",
				},
			},
			cumulativeFundingRateMap: map[string]*big.Int{
				"ETH": cutils.Mulx18(big.NewInt(1)),
			},
			expectedTotalEquityx36: cutils.MulxCust(big.NewInt(-88999), 34), // User is holding 100 USDC and 10 ETH -> $10 pnl - $1000 funding + $100.01 spot balance
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalEquityx36 := subaccount.GetTotalEquityx36(tc.subaccountBalances, tc.oraclePricesMap, tc.cumulativeFundingRateMap)
			assert.Equalf(t, tc.expectedTotalEquityx36.Cmp(totalEquityx36), 0, "Expected total equity: %v, got: %v", tc.expectedTotalEquityx36, totalEquityx36)
		})
	}

}

func TestGetTotalSpotBalancex36(t *testing.T) {
	os.Setenv("ENV", "TESTNET")
	contractUtils.Init()
	type TestCase struct {
		name                   string
		subaccountBalances     subaccountTypes.SubaccountBalances
		oraclePricesMap        map[string]ctypes.OraclePrice
		expectedSpotBalancex36 *big.Int
	}

	testCases := []TestCase{
		{
			name: "Single spot balance with ARB_USDC and matching oracle price",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"ETH": {
					Pricex18: cutils.Mulx18(big.NewInt(101)),
					Symbol:   "ETH",
				},
				"USDC": {
					Pricex18: cutils.Mulx18(big.NewInt(100)),
					Symbol:   "USDC",
				},
			},
			expectedSpotBalancex36: cutils.MulxCust(big.NewInt(10000), 36), // 100 * 100 = 10000
		},
		{
			name: "For LOGX productID , expected balance is zero",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.LOGX: {
						ProductId:  contractUtils.LOGX,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"LOGX": {
					Pricex18: cutils.Mulx18(big.NewInt(50)),
					Symbol:   "LOGX",
				},
			},
			expectedSpotBalancex36: cutils.GetBig0(), // Expected zero balance
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			totalSpotBalancex36 := subaccount.GetTotalSpotBalancex36(tc.subaccountBalances, tc.oraclePricesMap)
			assert.Equalf(t, tc.expectedSpotBalancex36.Cmp(totalSpotBalancex36), 0, "Expected total spot balance: %v, got: %v", tc.expectedSpotBalancex36, totalSpotBalancex36)
		})
	}
}

func TestGetLockedValuex36(t *testing.T) {
	os.Setenv("ENV", "TESTNET")
	contractUtils.Init()
	type TestCase struct {
		name                   string
		subaccountBalances     subaccountTypes.SubaccountBalances
		oraclePricesMap        map[string]ctypes.OraclePrice
		expectedLockedValuex36 *big.Int
	}

	testCases := []TestCase{
		{
			name: "Single locked balance with ARB_USDC and matching oracle price",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.Mulx18(big.NewInt(50)),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"USDC": {
					Pricex18: cutils.Mulx18(big.NewInt(100)),
					Symbol:   "USDC",
				},
			},
			expectedLockedValuex36: cutils.MulxCust(big.NewInt(5000), 36), // 50 * 100 = 5000
		},
		{
			name: "For LOGX product ID, expected locked value is zero",
			subaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.LOGX: {
						ProductId:  contractUtils.LOGX,
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(), // No locked balance for LOGX
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"LOGX": {
					Pricex18: cutils.Mulx18(big.NewInt(50)),
					Symbol:   "LOGX",
				},
			},
			expectedLockedValuex36: cutils.GetBig0(), // Expected zero locked value
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lockedValuex36 := subaccount.GetLockedValuex36(tc.subaccountBalances, tc.oraclePricesMap)
			assert.Equalf(t, tc.expectedLockedValuex36.Cmp(lockedValuex36), 0, "Expected locked value: %v, got: %v", tc.expectedLockedValuex36, lockedValuex36)
		})
	}
}

func TestGetAllSubaccounts(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()

	mockSubaccountDb := &db.SubaccountDB{}

	// Add subaccount_tables entries
	mockSubaccountDb.MockCreate("0xabc", 1, 1)
	mockSubaccountDb.MockCreate("0xdef", 1, 10_000)
	mockSubaccountDb.MockCreate("0xghi", 1, 1_000_000)

	subaccountBal := subaccount.NewSubaccountBalanceImpl()

	subaccountIds, lastFetchedId, err := subaccountBal.GetAllSubaccounts(1)
	assert.NoError(t, err, "Error in fetching all subaccounts")
	assert.Equal(t, 3, len(subaccountIds), "Expected 3 subaccounts, got %d", len(subaccountIds))
	assert.Equal(t, uint(1_000_000), lastFetchedId)
}
