package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// SHARED TEST DATA SETUP
// ============================================================================
// These variables store test data that can be reused across all tests
// ============================================================================

// TestData holds shared test data and parameters
type TestData struct {
	SubaccountID string
	BrokerID     uint
	UserAddress  string
	MarketID     uint
	Date         time.Time
}

// creates and inserts test data into both fill_tables and fill_order_tables
func setupTestData(t *testing.T) *TestData {
	// Define test parameters
	td := &TestData{
		SubaccountID: "2_0x1234567890123456789012345678901234567890_1",
		BrokerID:     uint(1),
		UserAddress:  "0x1234567890123456789012345678901234567890",
		MarketID:     uint(1),
		Date:         time.Now().UTC().Truncate(24 * time.Hour),
	}
	// Clear any existing data first to ensure clean test state
	db.ExecSQL("DELETE FROM fill_tables")
	db.ExecSQL("DELETE FROM fill_order_tables")

	// ============================================================================
	// Create test data for fill_tables
	// ============================================================================
	fillTableData := []db.FillTable{
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date.AddDate(0, 0, -1), // Yesterday
			},
			BrokerId:       td.BrokerID,
			SubaccountId:   td.SubaccountID,
			OrderId:        97098,
			MarketId:       td.MarketID,
			Side:           ctypes.ORDER_SIDE_BUY,
			Liquidity:      ctypes.LIQUIDITY_TAKER,
			Type:           ctypes.FILL_MARKET,
			Amountx18:      ctypes.NewBigIntFromString("1000000000000000000"),    // 1.0 in x18
			Pricex18:       ctypes.NewBigIntFromString("2000000000000000000000"), // 2000.0 in x18
			Feex18:         ctypes.NewBigIntFromString("10000000000000000"),      // 0.01 in x18
			RealizedPnlx18: ctypes.NewBigIntFromString("50000000000000000"),      // 0.05 in x18
			FundingFeesx18: ctypes.NewBigIntFromString("20000000000000000"),      // 0.02 in x18 (positive = user pays)
			IsReduce:       false,
		},
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date, // Today
			},
			BrokerId:       td.BrokerID,
			SubaccountId:   td.SubaccountID,
			OrderId:        97099,
			MarketId:       td.MarketID,
			Side:           ctypes.ORDER_SIDE_SELL,
			Liquidity:      ctypes.LIQUIDITY_MAKER,
			Type:           ctypes.FILL_MARKET,
			Amountx18:      ctypes.NewBigIntFromString("500000000000000000"),     // 0.5 in x18
			Pricex18:       ctypes.NewBigIntFromString("2100000000000000000000"), // 2100.0 in x18
			Feex18:         ctypes.NewBigIntFromString("5000000000000000"),       // 0.005 in x18
			RealizedPnlx18: ctypes.NewBigIntFromString("25000000000000000"),      // 0.025 in x18
			FundingFeesx18: ctypes.NewBigIntFromString("-10000000000000000"),     // -0.01 in x18 (negative = user receives)
			IsReduce:       false,
		},
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date.Add(2 * time.Hour), // Today at 2 AM
			},
			BrokerId:       td.BrokerID,
			SubaccountId:   td.SubaccountID,
			OrderId:        97100,
			MarketId:       td.MarketID,
			Side:           ctypes.ORDER_SIDE_BUY,
			Liquidity:      ctypes.LIQUIDITY_TAKER,
			Type:           ctypes.FILL_MARKET,
			Amountx18:      ctypes.NewBigIntFromString("2000000000000000000"),    // 2.0 in x18
			Pricex18:       ctypes.NewBigIntFromString("2200000000000000000000"), // 2200.0 in x18
			Feex18:         ctypes.NewBigIntFromString("20000000000000000"),      // 0.02 in x18
			RealizedPnlx18: ctypes.NewBigIntFromString("100000000000000000"),     // 0.1 in x18
			FundingFeesx18: ctypes.NewBigIntFromString("30000000000000000"),      // 0.03 in x18
			IsReduce:       false,
		},
	}

	// ============================================================================
	// Create equivalent test data for fill_order_tables
	// Note: SubAccountID in fill_order_tables uses hex format, not string format
	// ============================================================================
	subaccountIDHex, err := cutils.SubaccountIdToHex(td.SubaccountID)
	if err != nil {
		t.Fatalf("Failed to convert subaccount ID to hex: %v", err)
	}

	fillOrderTableData := []db.FillOrderTable{
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date.AddDate(0, 0, -1),
			},
			BrokerID:     td.BrokerID,
			SubAccountID: subaccountIDHex, // Use hex format
			UserAddress:  td.UserAddress,
			ProductID:    td.MarketID,
			PriceX18:     ctypes.NewBigIntFromString("2000000000000000000000"),
			Amount:       ctypes.NewBigIntFromString("1000000000000000000"),
			IsTaker:      true, // LIQUIDITY_TAKER
			FeeAmount:    ctypes.NewBigIntFromString("10000000000000000"),
			BaseDelta:    ctypes.NewBigIntFromString("1000000000000000000"),
			QuoteDelta:   ctypes.NewBigIntFromString("2000000000000000000000"),
			RealisedPnl:  ctypes.NewBigIntFromString("50000000000000000"),
			FundingFees:  ctypes.NewBigIntFromString("20000000000000000"),
			BlockNumber:  0,
			Signature:    []byte{0xFB, 0x4F},
			TxnHash:      "0xth97098",
			Index:        0,
		},
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date,
			},
			BrokerID:     td.BrokerID,
			SubAccountID: subaccountIDHex, // Use hex format
			UserAddress:  td.UserAddress,
			ProductID:    td.MarketID,
			PriceX18:     ctypes.NewBigIntFromString("2100000000000000000000"),
			Amount:       ctypes.NewBigIntFromString("500000000000000000"),
			IsTaker:      false, // LIQUIDITY_MAKER
			FeeAmount:    ctypes.NewBigIntFromString("5000000000000000"),
			BaseDelta:    ctypes.NewBigIntFromString("500000000000000000"),
			QuoteDelta:   ctypes.NewBigIntFromString("1050000000000000000000"),
			RealisedPnl:  ctypes.NewBigIntFromString("25000000000000000"),
			FundingFees:  ctypes.NewBigIntFromString("-10000000000000000"),
			BlockNumber:  0,
			Signature:    []byte{0xFB, 0x50},
			TxnHash:      "0xth97099",
			Index:        0,
		},
		{
			BaseTable: db.BaseTable{
				CreatedAt: td.Date.Add(2 * time.Hour),
			},
			BrokerID:     td.BrokerID,
			SubAccountID: subaccountIDHex, // Use hex format
			UserAddress:  td.UserAddress,
			ProductID:    td.MarketID,
			PriceX18:     ctypes.NewBigIntFromString("2200000000000000000000"),
			Amount:       ctypes.NewBigIntFromString("2000000000000000000"),
			IsTaker:      true, // LIQUIDITY_TAKER
			FeeAmount:    ctypes.NewBigIntFromString("20000000000000000"),
			BaseDelta:    ctypes.NewBigIntFromString("2000000000000000000"),
			QuoteDelta:   ctypes.NewBigIntFromString("4400000000000000000000"),
			RealisedPnl:  ctypes.NewBigIntFromString("100000000000000000"),
			FundingFees:  ctypes.NewBigIntFromString("30000000000000000"),
			BlockNumber:  0,
			Signature:    []byte{0xFB, 0x51},
			TxnHash:      "0xth97100",
			Index:        0,
		},
	}

	// Insert data into fill_tables
	for _, fill := range fillTableData {
		result := (&db.FillDB{}).Create(&fill)
		if result == nil {
			t.Fatalf("Failed to create FillTable")
		}
	}

	// Insert data into fill_order_tables
	for i, fillOrder := range fillOrderTableData {
		fillOrder.TxnHash = fmt.Sprintf("0x%064x", i+1)
		fillOrder.Index = uint(i + 1)
		_, err := (&db.FillOrderDB{}).CreateFillOrder(fillOrder)
		if err != nil {
			t.Fatalf("Failed to create FillOrderTable: %v", err)
		}
	}

	t.Logf("Setup complete: Inserted %d FillTable records and %d FillOrderTable records",
		len(fillTableData), len(fillOrderTableData))

	return td
}
func TestMain(m *testing.M) {
	if os.Getenv("INTERNAL_SUBACCOUNT_IDS") == "" {
		os.Setenv("INTERNAL_SUBACCOUNT_IDS", "1_0x65ddd96914436E084F1Cc332c7eFb100ecb980B3_1,1_0x9C93f5C8aFe5e5b2642eC76E0EDFEc34DAdCFE2D_1,1_0x439d907B94437BAD645697a903f8610114563571_1,1_0x287dE2AC7a77Bd0DD13886e570e1A089e21Ed402_1,1_0x19CbC6Df7F74d6e75cEad93a25deCE671Ce985f8_1,1_0x582628702B25AF512A88782F60C1b6BcD63C8666_1,1_0x1c4039855b68E3A3Cf27d9695268f92B834D7739_1,1_0xe84e8f2dA14C4F6C1460D6973bED37776FCE8988_1,1_0x876aF73bbF695c8B7d079B82e3fF84e0BD49404B_1,1_0xa5D6823b8F57477Bd7D12662B50096F508f1c926_1,1_0xf1d39Cf8FD9086D943490063e441D9FA2afea6D9_1,2_0x8221a68d129f70877750029D9D45a58E395111Ec_1,2_0xA548dC4ce939eBC62e55F154293357fD50A32f0C_1,2_0x8521953D26F17A3aE0B2B99739757e37d68abd2a_1,1_0x48e7DB25cB247Bf02B25402108e4499A6636EAa9_1,1_0xEBe09B0C96B109039745B3C9BBD56E194d9b5783_1,1_0x5defe4f60965528030EE48bba305Ed017AB84998_1,1_0xbCc830633A7AF4aC054491333B62E611ca55B94D_1,1_0xc73D55eC17c806709def8FbB3bB3BCb299383d56_1,1_0xb3023BAa698DD410a8AfCFAE5F104005eD7d71Bf_1,1_0xa3280C1260D7be3Cd9Bf93e3fF7087D09Fd1821c_1")
	}

	os.Exit(m.Run())
}

// TestGetFundingFeeBySubaccountId compares FillDB.GetFundingFeeBySubaccountId with FillOrderDB.GetFundingFeeBySubaccountId
func TestGetFundingFeeBySubaccountId(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	subaccountIDHex, err := cutils.SubaccountIdToHex(testData.SubaccountID)
	require.NoError(t, err, "Should convert subaccount ID to hex")

	// Call both functions with hex format
	newResult, newErr := (&db.FillDB{}).GetFundingFeeBySubaccountId(subaccountIDHex, testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetFundingFeeBySubaccountId(subaccountIDHex, testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %s", newResult)
	t.Logf("FillOrderDB result: %s", oldResult)
	assert.Equal(t, newResult, oldResult, "FillDB and FillOrderDB should return same value")
}

// TestGetDailyRealizedPnL compares FillDB.GetDailyRealizedPnL with FillOrderDB.GetDailyRealizedPnL
func TestGetDailyRealizedPnL(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	subaccountIDHex, err := cutils.SubaccountIdToHex(testData.SubaccountID)
	require.NoError(t, err, "Should convert subaccount ID to hex")

	// Use today's date from test data
	specificDate := testData.Date

	// Call both functions with hex format
	newPnLPerToken, newTotal, newErr := (&db.FillDB{}).GetDailyRealizedPnL(subaccountIDHex, specificDate)
	oldPnLPerToken, oldTotal, oldErr := (&db.FillOrderDB{}).GetDailyRealizedPnL(subaccountIDHex, specificDate)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB PnL per token: %v, Total: %s", newPnLPerToken, newTotal.String())
	t.Logf("FillOrderDB PnL per token: %v, Total: %s", oldPnLPerToken, oldTotal.String())

	// Compare total PnL
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total PnL should match")

	// Compare per-token PnL
	assert.EqualValues(t, newPnLPerToken, oldPnLPerToken, "PnL per token should match")
}

// TestGetDailyPnlGraph compares FillDB.GetDailyPnlGraph with FillOrderDB.GetDailyPnlGraph
func TestGetDailyPnlGraph(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	subaccountIDHex, err := cutils.SubaccountIdToHex(testData.SubaccountID)
	require.NoError(t, err, "Should convert subaccount ID to hex")

	// Call both functions with hex format
	newResult, newErr := (&db.FillDB{}).GetDailyPnlGraph(subaccountIDHex, testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetDailyPnlGraph(subaccountIDHex, testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)
	assert.EqualValues(t, newResult, oldResult, "Daily PnL graph should match")
}

// TestGetDailyFees compares FillDB.GetDailyFees with FillOrderDB.GetDailyFees
func TestGetDailyFees(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newFeesByDate, newFeesByMarket, newErr := (&db.FillDB{}).GetDailyFees(testData.BrokerID)
	oldFeesByDate, oldFeesByMarket, oldErr := (&db.FillOrderDB{}).GetDailyFees(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB fees by date: %v", newFeesByDate)
	t.Logf("FillOrderDB fees by date: %v", oldFeesByDate)
	t.Logf("FillDB fees by market: %v", newFeesByMarket)
	t.Logf("FillOrderDB fees by market: %v", oldFeesByMarket)

	// Compare fees by date
	assert.EqualValues(t, newFeesByDate, oldFeesByDate, "Fees by date should match")

	// Compare fees by market
	assert.EqualValues(t, newFeesByMarket, oldFeesByMarket, "Fees by market should match")
}

// TestGetDailyVolumes compares FillDB.GetDailyVolumes with FillOrderDB.GetDailyVolumes
func TestGetDailyVolumes(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newVolumesByMarket, newTotal, newDates, newErr := (&db.FillDB{}).GetDailyVolumes(testData.BrokerID)
	oldVolumesByMarket, oldTotal, oldDates, oldErr := (&db.FillOrderDB{}).GetDailyVolumes(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total volume: %s", newTotal.String())
	t.Logf("FillOrderDB total volume: %s", oldTotal.String())
	t.Logf("FillDB volumes by market: %v", newVolumesByMarket)
	t.Logf("FillOrderDB volumes by market: %v", oldVolumesByMarket)

	// Compare total volume
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total volume should match")

	// Compare dates
	assert.Equal(t, len(newDates), len(oldDates), "Number of dates should match")

	// Compare volumes by market
	assert.Equal(t, len(newVolumesByMarket), len(oldVolumesByMarket), "Number of markets should match")
}

// TestGetDailyVolumesInternal compares FillDB.GetDailyVolumesInternal with FillOrderDB.GetDailyVolumesInternal
func TestGetDailyVolumesInternal(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	setupTestData(t)
	// Call both functions
	newVolumesByMarket, newTotal, newDates, newErr := (&db.FillDB{}).GetDailyVolumesInternal()
	oldVolumesByMarket, oldTotal, oldDates, oldErr := (&db.FillOrderDB{}).GetDailyVolumesInternal()

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total volume: %s", newTotal.String())
	t.Logf("FillOrderDB total volume: %s", oldTotal.String())

	// Compare total volume
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total volume should match")

	// Compare dates
	assert.Equal(t, len(newDates), len(oldDates), "Number of dates should match")

	// Compare volumes by market
	assert.Equal(t, len(newVolumesByMarket), len(oldVolumesByMarket), "Number of markets should match")
}

// TestGetDailyOverallVolumes compares FillDB.GetDailyOverallVolumes with FillOrderDB.GetDailyOverallVolumes
func TestGetDailyOverallVolumes(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetDailyOverallVolumes(testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetDailyOverallVolumes(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)
	assert.EqualValues(t, newResult, oldResult, "Daily overall volumes should match")
}

// TestGetCumulativeTradingFees compares FillDB.GetCumulativeTradingFees with FillOrderDB.GetCumulativeTradingFees
func TestGetCumulativeTradingFees(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	endDate := testData.Date.Add(24 * time.Hour)

	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetCumulativeTradingFees(endDate, testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetCumulativeTradingFees(endDate, testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %s", newResult)
	t.Logf("FillOrderDB result: %s", oldResult)
	assert.Equal(t, newResult, oldResult, "Cumulative trading fees should match")
}

// TestGetCumulativeVolume compares FillDB.GetCumulativeVolume with FillOrderDB.GetCumulativeVolume
func TestGetCumulativeVolume(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	endDate := testData.Date.Add(24 * time.Hour)

	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetCumulativeVolume(endDate, testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetCumulativeVolume(endDate, testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %s", newResult)
	t.Logf("FillOrderDB result: %s", oldResult)
	assert.Equal(t, newResult, oldResult, "Cumulative volume should match")
}

// TestGetTotalVolume compares FillDB.GetTotalVolume with FillOrderDB.GetTotalVolume
func TestGetTotalVolume(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newVolumesByMarket, newTotal, newErr := (&db.FillDB{}).GetTotalVolume(testData.BrokerID)
	oldVolumesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).GetTotalVolume(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total volume: %s", newTotal.String())
	t.Logf("FillOrderDB total volume: %s", oldTotal.String())

	// Compare total volume
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total volume should match")

	// Compare volumes by market
	assert.EqualValues(t, newVolumesByMarket, oldVolumesByMarket, "Volumes by market should match")
}

// TestGet24hRealisedPnL compares FillDB.Get24hRealisedPnL with FillOrderDB.Get24hRealisedPnL
func TestGet24hRealisedPnL(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newPnLByMarket, newTotal, newErr := (&db.FillDB{}).Get24hRealisedPnL(testData.BrokerID)
	oldPnLByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).Get24hRealisedPnL(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total PnL: %s", newTotal.String())
	t.Logf("FillOrderDB total PnL: %s", oldTotal.String())

	// Compare total PnL
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total PnL should match")

	// Compare PnL by market
	assert.Equal(t, len(newPnLByMarket), len(oldPnLByMarket), "Number of markets should match")
}

// TestGet24hTradingFee compares FillDB.Get24hTradingFee with FillOrderDB.Get24hTradingFee
func TestGet24hTradingFee(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newFeesByMarket, newTotal, newErr := (&db.FillDB{}).Get24hTradingFee(testData.BrokerID)
	oldFeesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).Get24hTradingFee(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total fee: %s", newTotal.String())
	t.Logf("FillOrderDB total fee: %s", oldTotal.String())

	// Compare total fee
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total trading fee should match")

	// Compare fees by market
	assert.Equal(t, len(newFeesByMarket), len(oldFeesByMarket), "Number of markets should match")
}

// TestGet24hFundingFee compares FillDB.Get24hFundingFee with FillOrderDB.Get24hFundingFee
func TestGet24hFundingFee(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newFeesByMarket, newTotal, newErr := (&db.FillDB{}).Get24hFundingFee(testData.BrokerID)
	oldFeesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).Get24hFundingFee(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total fee: %s", newTotal.String())
	t.Logf("FillOrderDB total fee: %s", oldTotal.String())

	// Compare total fee
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total funding fee should match")

	// Compare fees by market
	assert.Equal(t, len(newFeesByMarket), len(oldFeesByMarket), "Number of markets should match")
}

// TestGetTotalRealisedPnL compares FillDB.GetTotalRealisedPnL with FillOrderDB.GetTotalRealisedPnL
func TestGetTotalRealisedPnL(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newPnLByMarket, newTotal, newErr := (&db.FillDB{}).GetTotalRealisedPnL(testData.BrokerID)
	oldPnLByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).GetTotalRealisedPnL(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total PnL: %s", newTotal.String())
	t.Logf("FillOrderDB total PnL: %s", oldTotal.String())

	// Compare total PnL
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total PnL should match")

	// Compare PnL by market
	assert.Equal(t, len(newPnLByMarket), len(oldPnLByMarket), "Number of markets should match")
}

// TestGetTotalUserRealisedPnL compares FillDB.GetTotalUserRealisedPnL with FillOrderDB.GetTotalUserRealisedPnL
func TestGetTotalUserRealisedPnL(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newPnLByMarket, newTotal, newErr := (&db.FillDB{}).GetTotalUserRealisedPnL(testData.BrokerID)
	oldPnLByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).GetTotalUserRealisedPnL(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total PnL: %s", newTotal.String())
	t.Logf("FillOrderDB total PnL: %s", oldTotal.String())

	// Compare total PnL
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total user PnL should match")

	// Compare PnL by market
	assert.Equal(t, len(newPnLByMarket), len(oldPnLByMarket), "Number of markets should match")
}

// TestGetTotalTradingFee compares FillDB.GetTotalTradingFee with FillOrderDB.GetTotalTradingFee
func TestGetTotalTradingFee(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newFeesByMarket, newTotal, newErr := (&db.FillDB{}).GetTotalTradingFee(testData.BrokerID)
	oldFeesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).GetTotalTradingFee(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total fee: %s", newTotal.String())
	t.Logf("FillOrderDB total fee: %s", oldTotal.String())

	// Compare total fee
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total trading fee should match")

	// Compare fees by market
	assert.Equal(t, len(newFeesByMarket), len(oldFeesByMarket), "Number of markets should match")
}

// TestGetTotalFundingFee compares FillDB.GetTotalFundingFee with FillOrderDB.GetTotalFundingFee
func TestGetTotalFundingFee(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newFeesByMarket, newTotal, newErr := (&db.FillDB{}).GetTotalFundingFee(testData.BrokerID)
	oldFeesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).GetTotalFundingFee(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total fee: %s", newTotal.String())
	t.Logf("FillOrderDB total fee: %s", oldTotal.String())

	// Compare total fee
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total funding fee should match")

	// Compare fees by market
	assert.Equal(t, len(newFeesByMarket), len(oldFeesByMarket), "Number of markets should match")
}

// TestGetDailyActiveTraders compares FillDB.GetDailyActiveTraders with FillOrderDB.GetDailyActiveTraders
func TestGetDailyActiveTraders(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetDailyActiveTraders(testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetDailyActiveTraders(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)
	assert.Equal(t, len(newResult), len(oldResult), "Number of daily active traders should match")
}

// TestGet24hVolumes compares FillDB.Get24hVolumes with FillOrderDB.Get24hVolumes
func TestGet24hVolumes(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newVolumesByMarket, newTotal, newErr := (&db.FillDB{}).Get24hVolumes(testData.BrokerID)
	oldVolumesByMarket, oldTotal, oldErr := (&db.FillOrderDB{}).Get24hVolumes(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB total volume: %s", newTotal.String())
	t.Logf("FillOrderDB total volume: %s", oldTotal.String())

	// Compare total volume
	assert.Equal(t, newTotal.String(), oldTotal.String(), "Total 24h volume should match")

	// Compare volumes by market
	assert.Equal(t, len(newVolumesByMarket), len(oldVolumesByMarket), "Number of markets should match")
}

// TestGetPerpUserStatistics compares FillDB.GetPerpUserStatistics with FillOrderDB.GetPerpUserStatistics
func TestGetPerpUserStatistics(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetPerpUserStatistics(testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetPerpUserStatistics(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)

	// Compare key fields if they exist
	if newStats, ok := newResult["stats"].(map[string]interface{}); ok {
		if oldStats, ok := oldResult["stats"].(map[string]interface{}); ok {
			assert.Equal(t, len(newStats), len(oldStats), "Number of stats should match")
		}
	}
}

// TestGetTotalFeesIncludingInternal compares FillDB.GetTotalFeesIncludingInternal with FillOrderDB.GetTotalFeesIncludingInternal
func TestGetTotalFeesIncludingInternal(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetTotalFeesIncludingInternal(testData.BrokerID)
	oldResult, oldErr := (&db.FillOrderDB{}).GetTotalFeesIncludingInternal(testData.BrokerID)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)
	assert.EqualValues(t, newResult, oldResult, "Total fees including internal should match")
}

// TestGetPreviousMonthStats compares FillDB.GetPreviousMonthStats with FillOrderDB.GetPreviousMonthStats
func TestGetPreviousMonthStats(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetPreviousMonthStats()
	oldResult, oldErr := (&db.FillOrderDB{}).GetPreviousMonthStats()

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)
	assert.EqualValues(t, newResult, oldResult, "Previous month stats should match")
}

// TestGet24hRealisedPnlPerSubaccount compares FillDB.Get24hRealisedPnlPerSubaccount with FillOrderDB.Get24hRealisedPnlPerSubaccount
func TestGet24hRealisedPnlPerSubaccount(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).Get24hRealisedPnlPerSubaccount()
	oldResult, oldErr := (&db.FillOrderDB{}).Get24hRealisedPnlPerSubaccount()

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result count: %d", len(newResult))
	t.Logf("FillOrderDB result count: %d", len(oldResult))

	// Note: Subaccount IDs might be in different formats (hex vs string), so we compare counts
	// For exact comparison, we'd need to convert keys
	assert.Equal(t, len(newResult), len(oldResult), "Number of subaccounts should match")
}

// TestGetUserDashboardData compares FillDB.GetUserDashboardData with FillOrderDB.GetUserDashboardData
func TestGetUserDashboardData(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	setupTestData(t)
	limit := 10
	offset := 0

	// Call both functions
	newData, newCount, newErr := (&db.FillDB{}).GetUserDashboardData(limit, offset)
	oldData, oldCount, oldErr := (&db.FillOrderDB{}).GetUserDashboardData(limit, offset)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB count: %d, data length: %d", newCount, len(newData))
	t.Logf("FillOrderDB count: %d, data length: %d", oldCount, len(oldData))

	// Compare counts
	assert.Equal(t, newCount, oldCount, "Total count should match")

	// Compare data length (should be min of limit and count)
	expectedLength := int(limit)
	if newCount < int64(limit) {
		expectedLength = int(newCount)
	}
	assert.Equal(t, len(newData), len(oldData), "Data length should match")
	assert.Equal(t, expectedLength, len(newData), "Data length should match expected")
}

// TestGetUserDashboardByAddress compares FillDB.GetUserDashboardByAddress with FillOrderDB.GetUserDashboardByAddress
func TestGetUserDashboardByAddress(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	testData := setupTestData(t)
	// Call both functions
	newResult, newErr := (&db.FillDB{}).GetUserDashboardByAddress(testData.UserAddress)
	oldResult, oldErr := (&db.FillOrderDB{}).GetUserDashboardByAddress(testData.UserAddress)

	// Ensure both functions succeed
	require.NoError(t, newErr)
	require.NoError(t, oldErr)

	// Compare results
	t.Logf("FillDB result: %v", newResult)
	t.Logf("FillOrderDB result: %v", oldResult)

	// Compare key fields
	assert.Equal(t, len(newResult), len(oldResult), "Number of fields should match")

	// Compare specific fields if they exist
	if newTotalVolume, ok := newResult["total_volume_usd"].(string); ok {
		if oldTotalVolume, ok := oldResult["total_volume_usd"].(string); ok {
			assert.Equal(t, newTotalVolume, oldTotalVolume, "Total volume should match")
		}
	}
}
