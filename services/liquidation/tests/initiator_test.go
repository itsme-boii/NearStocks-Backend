package tests

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/services/liquidation/initiator"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTaskRunnerRunLiquidationJob(t *testing.T) {
	t.Skip("Skip TestTaskRunnerRunLiquidationJob")
	// Create a new TaskRunnerImpl
	taskRunner := &initiator.TaskRunnerImpl{}

	// Call the RunLiquidationJob method
	err := taskRunner.RunLiquidationJob1()

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
}

func TestCheckSubaccountCollateralization2(t *testing.T) {
	type TestCase struct {
		name                             string
		subaccount                       subaccountTypes.SubaccountBalances
		oraclePricesMap                  map[string]ctypes.OraclePrice
		perpetualMarketMap               map[uint]subaccountTypes.PerpetualMarket
		cumulativeFundingRateMap         map[string]*big.Int
		expectedIsBelowInitialMargin     bool
		expectedIsBelowMaintenanceMargin bool
		expectedRequiresInsurance        bool
	}

	os.Setenv("AMM_SUBACCOUNT_ID", "1_0xd37eD507cA37Faa17079bFf5e46DcedE951577dB_1")
	os.Setenv("DEBUG_MODE", "1")
	os.Setenv("ENV", "TESTNET")
	contractUtils.Init()

	testCases := []TestCase{
		{
			subaccount: subaccountTypes.SubaccountBalances{
				SubaccountId: "0x1",
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					// DOGE
					9: {
						ProductId:             9,
						Amountx18:             cutils.StrToBigInt("3388810000000000000000"),
						VQuoteBalancex18:      cutils.StrToBigInt("-342156323730000000000"),
						LastCumFundingRatex18: big.NewInt(-193749999998760),
					},
					// HARRIS
					13: {
						ProductId:             13,
						Amountx18:             cutils.StrToBigInt("90384600000000000000"),
						VQuoteBalancex18:      cutils.StrToBigInt("-47046991992000000000"),
						LastCumFundingRatex18: cutils.StrToBigInt("-193749999998760"),
					},
					// LINK
					17: {
						ProductId:             17,
						Amountx18:             cutils.StrToBigInt("10577900000000000000"),
						VQuoteBalancex18:      cutils.StrToBigInt("-108108359359000000000"),
						LastCumFundingRatex18: cutils.StrToBigInt("-193749999998760"),
					},
				},
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					4: {
						ProductId:  4,
						Balancex18: big.NewInt(-149193502524300000),
						Lockedx18:  big.NewInt(0),
					},
				},
			},
			oraclePricesMap: map[string]ctypes.OraclePrice{
				"DOGE": {
					Pricex18: big.NewInt(98000000000000000),
					Symbol:   "DOGE",
				},
				"HARRIS": {
					Pricex18: big.NewInt(473000000000000000),
					Symbol:   "HARRIS",
				},
				// LINK:{Pricex18:+10245000000000000000 Symbol:LINK}
				"LINK": {
					Pricex18: cutils.StrToBigInt("10245000000000000000"),
					Symbol:   "LINK",
				},
				"USDC": {
					Pricex18: big.NewInt(1000000000000000000),
					Symbol:   "USDC",
				},
			},
			perpetualMarketMap: map[uint]subaccountTypes.PerpetualMarket{
				// 9:{ProductId:9 BaseAsset:DOGE MaintenanceMarginFractionx18:+20000000000000000 InitialMarginFractionx18:+50000000000000000}
				9: {
					ProductId:                    9,
					BaseAsset:                    "DOGE",
					MaintenanceMarginFractionx18: big.NewInt(20000000000000000),
					InitialMarginFractionx18:     big.NewInt(50000000000000000),
				},
				// 13:{ProductId:13 BaseAsset:HARRIS MaintenanceMarginFractionx18:+20000000000000000 InitialMarginFractionx18:+50000000000000000}
				13: {
					ProductId:                    13,
					BaseAsset:                    "HARRIS",
					MaintenanceMarginFractionx18: big.NewInt(20000000000000000),
					InitialMarginFractionx18:     big.NewInt(50000000000000000),
				},
				// 17:{ProductId:17 BaseAsset:LINK MaintenanceMarginFractionx18:+20000000000000000 InitialMarginFractionx18:+50000000000000000}
				17: {
					ProductId:                    17,
					BaseAsset:                    "LINK",
					MaintenanceMarginFractionx18: big.NewInt(20000000000000000),
					InitialMarginFractionx18:     big.NewInt(50000000000000000),
				},
			},
			// cumulativeFundingRateMap: map[ARB:+2430904832110185 BTC:+934573997245517 DOGE:-67457254004767678 ETH:+3626505212701608 HARRIS:+8355195615175087 LINK:-45722476797546894 NEAR:-36587767642889575 PEPE:-74092245370361305 SOL:+3383252068672868 TRUMP:+2343389809601675 XRP:-74193519280886328]
			cumulativeFundingRateMap: map[string]*big.Int{
				"DOGE":   big.NewInt(-67457254004767678),
				"HARRIS": big.NewInt(8355195615175087),
				"LINK":   big.NewInt(-45722476797546894),
			},
			expectedIsBelowInitialMargin:     true,
			expectedIsBelowMaintenanceMargin: false,
			expectedRequiresInsurance:        false,
		},
	}

	taskRunner := initiator.NewTaskRunner()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin := taskRunner.CheckSubaccountCollateralization(tc.subaccount, tc.oraclePricesMap, tc.perpetualMarketMap, tc.cumulativeFundingRateMap)

			assert.Equal(t, tc.expectedIsBelowInitialMargin, isBelowInitialMargin, "Expected isBelowInitialMargin to be %v, got: %v", tc.expectedIsBelowInitialMargin, isBelowInitialMargin)
			assert.Equal(t, tc.expectedIsBelowMaintenanceMargin, isBelowMaintenanceMargin, "Expected isBelowMaintenanceMargin to be %v, got: %v", tc.expectedIsBelowMaintenanceMargin, isBelowMaintenanceMargin)
			assert.Equal(t, tc.expectedRequiresInsurance, requiresInsurance, "Expected requiresInsurance to be %v, got: %v", tc.expectedRequiresInsurance, requiresInsurance)
			assert.Equal(t, tc.expectedIsBelowMaintenanceMargin, isBelowLiquidationMargin, "Expected isBelowLiquidationMargin to be %v, got: %v", tc.expectedIsBelowMaintenanceMargin, isBelowLiquidationMargin)
		})
	}

}

func TestCheckSubaccountCollateralization(t *testing.T) {
	// Set env vars
	os.Setenv("AMM_SUBACCOUNT_ID", "1_0xd37eD507cA37Faa17079bFf5e46DcedE951577dB_1")

	taskRunner := initiator.NewTaskRunner()
	x18_1 := cutils.Mulx18(new(big.Int).SetInt64(1))
	// x18_minus2 := cutils.Mulx18(new(big.Int).SetInt64(-2))
	x18_10 := cutils.Mulx18(new(big.Int).SetInt64(10))
	x18_minus_110 := cutils.Mulx18(new(big.Int).SetInt64(-110))

	unsettledSubaccount := subaccountTypes.SubaccountBalances{
		SubaccountId: "0x1",
		// Perp position value: 110 USDC
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{
			contractUtils.ETH_MARKET: {
				ProductId:             contractUtils.ETH_MARKET, // ETH-USD Perp
				Amountx18:             x18_1,
				VQuoteBalancex18:      x18_minus_110, // Assuming no funding here
				LastCumFundingRatex18: cutils.GetBig0(),
			},
		},
		// Spot position value: 10 USDC
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{
			contractUtils.ETH_USDC: {
				ProductId:  contractUtils.ETH_USDC,
				Balancex18: x18_10,
				Lockedx18:  cutils.GetBig0(),
			},
		},
	}

	x18_100 := cutils.Mulx18(new(big.Int).SetInt64(100))

	oraclePricesMap := map[string]ctypes.OraclePrice{
		"USDC": {
			Pricex18: x18_1,
			Symbol:   "USDC",
		},
		"ETH": {
			Pricex18: x18_100,
			Symbol:   "ETH",
		},
	}

	perpetualMarketMap := map[uint]subaccountTypes.PerpetualMarket{
		1: {
			ProductId:                    1,
			BaseAsset:                    "ETH",
			MaintenanceMarginFractionx18: new(big.Int).Div(cutils.Mulx18(new(big.Int).SetInt64(10000)), cutils.GetBigx6()),  // 10^4 / 10^6 = 10^-2 in (x18)
			InitialMarginFractionx18:     new(big.Int).Div(cutils.Mulx18(new(big.Int).SetInt64(100000)), cutils.GetBigx6()), // 10^5 / 10^6 = 10^-1 in (x18)
		},
	}

	cumulativeFundingRateMap := map[string]*big.Int{
		"ETH": cutils.Mulx18(new(big.Int).SetInt64(0)),
	}

	// Test: 1
	isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin := taskRunner.CheckSubaccountCollateralization(unsettledSubaccount, oraclePricesMap, perpetualMarketMap, cumulativeFundingRateMap)
	// Calculation for the above values:
	// Total equity = Total spot balance + total unrealised pnl on perps
	// Total spot balance = 10 USDC
	// Total initial perp value + funding  = 110 USDC
	// Total current perp value = 100 * 1 = 100 USDC
	// Total unrealised pnl = 100 - 110 = -10 USDC
	// Total equity = 10 - 10 = 0 USDC
	// Total initial margin = 100 * 10^-1 = 10 USDC
	// Total maintenance margin = 100 * 10^-2 = 1 USDC
	if !isBelowInitialMargin {
		t.Errorf("Test: 1 - Expected isBelowInitialMargin to be true, got: %v", isBelowInitialMargin)
	}
	if !isBelowMaintenanceMargin {
		t.Errorf("Test: 1 - Expected isBelowMaintenanceMargin to be true, got: %v", isBelowMaintenanceMargin)
	}
	if requiresInsurance {
		t.Errorf("Test: 1 - Expected requiresInsurance to be false, got: %v", requiresInsurance)
	}
	if !isBelowLiquidationMargin {
		t.Errorf("Test: 1 - Expected isBelowLiquidationMargin to be true, got: %v", isBelowLiquidationMargin)
	}

	// Test: 2
	// Total equity = 11 - 10 = 1 USDC
	// Total initial margin = 100 * 10^-1 = 10 USDC
	// Total maintenance margin = 100 * 10^-2 = 1 USDC
	x18_11 := cutils.Mulx18(new(big.Int).SetInt64(11))
	quoteBalance := unsettledSubaccount.MustGetSpotBalance(contractUtils.ETH_USDC)
	quoteBalance.Balancex18 = x18_11
	unsettledSubaccount.SpotBalances[contractUtils.ETH_USDC] = quoteBalance
	isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin = taskRunner.CheckSubaccountCollateralization(unsettledSubaccount, oraclePricesMap, perpetualMarketMap, cumulativeFundingRateMap)
	if !isBelowInitialMargin {
		t.Errorf("Test: 2 - Expected isBelowInitialMargin to be true, got: %v", isBelowInitialMargin)
	}
	if isBelowMaintenanceMargin {
		t.Errorf("Test: 2 - Expected isBelowMaintenanceMargin to be true, got: %v", isBelowMaintenanceMargin)
	}
	if requiresInsurance {
		t.Errorf("Test: 2 - Expected requiresInsurance to be false, got: %v", requiresInsurance)
	}
	if isBelowLiquidationMargin {
		t.Errorf("Test: 2 - Expected isBelowLiquidationMargin to be false, got: %v", isBelowLiquidationMargin)
	}

	// Test: 3
	// Total equity = 21 - 10 = 11 USDC
	// Total initial margin = 100 * 10^-1 = 10 USDC
	// Total maintenance margin = 100 * 10^-2 = 1 USDC
	x18_21 := cutils.Mulx18(new(big.Int).SetInt64(21))
	quoteBalance = unsettledSubaccount.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID)
	quoteBalance.Balancex18 = x18_21
	unsettledSubaccount.SpotBalances[contractUtils.ETH_USDC] = quoteBalance
	isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin = taskRunner.CheckSubaccountCollateralization(unsettledSubaccount, oraclePricesMap, perpetualMarketMap, cumulativeFundingRateMap)
	if isBelowInitialMargin {
		t.Errorf("Test: 3 - Expected isBelowInitialMargin to be false, got: %v", isBelowInitialMargin)
	}
	if isBelowMaintenanceMargin {
		t.Errorf("Test: 3 - Expected isBelowMaintenanceMargin to be false, got: %v", isBelowMaintenanceMargin)
	}
	if requiresInsurance {
		t.Errorf("Test: 3 - Expected requiresInsurance to be false, got: %v", requiresInsurance)
	}
	if isBelowLiquidationMargin {
		t.Errorf("Test: 3 - Expected isBelowLiquidationMargin to be false, got: %v", isBelowLiquidationMargin)
	}

	// Test: 4
	// Total equity = -1 USDC
	// Total initial margin = 0
	// Total maintenance margin = 0
	x18_9 := cutils.Mulx18(new(big.Int).SetInt64(9))
	quoteBalance = unsettledSubaccount.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID)
	quoteBalance.Balancex18 = new(big.Int).Neg(x18_9)
	unsettledSubaccount.SpotBalances[contractUtils.QUOTE_TOKEN_PRODUCT_ID] = quoteBalance

	unsettledSubaccount.PerpBalances = map[uint32]subaccountTypes.PerpBalance{}
	isBelowInitialMargin, isBelowMaintenanceMargin, requiresInsurance, isBelowLiquidationMargin = taskRunner.CheckSubaccountCollateralization(unsettledSubaccount, oraclePricesMap, perpetualMarketMap, cumulativeFundingRateMap)
	if isBelowInitialMargin {
		t.Errorf("Test: 4 - Expected isBelowInitialMargin to be false, got: %v", isBelowInitialMargin)
	}
	if isBelowMaintenanceMargin {
		t.Errorf("Test: 4 - Expected isBelowMaintenanceMargin to be false, got: %v", isBelowMaintenanceMargin)
	}
	if !requiresInsurance {
		t.Errorf("Test: 4 - Expected requiresInsurance to be true, got: %v", requiresInsurance)
	}
	if isBelowLiquidationMargin {
		t.Errorf("Test: 4 - Expected isBelowLiquidationMargin to be false, got: %v", isBelowLiquidationMargin)
	}
}
