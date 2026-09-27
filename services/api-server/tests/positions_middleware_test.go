package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/middleware"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/alicebob/miniredis/v2"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
)

func setup() {
	testutils.SetupContractEnv()
	contractUtils.Init()
	contract.Init()
}

func TestRequireValidPositionLimitsMiddleware(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "RequireValidPositionLimits Suite")
}

var _ = ginkgo.Describe("RequireValidPositionLimits Middleware", func() {
	var mockRedis *miniredis.Miniredis
	var mockSubaccount *db.SubaccountTable
	var patches *gomonkey.Patches
	var ValidatePatches *gomonkey.Patches
	var OracelPatches *gomonkey.Patches
	var FetchTotalPositionPatches *gomonkey.Patches
	ammSubaccountID := "0x1100000000010000000000000000000000000000000000000001000000000001"
	testSubaccountID := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"

	ginkgo.BeforeEach(func() {
		var err error

		// Start Miniredis server
		mockRedis, err = miniredis.Run()
		gomega.Expect(err).To(gomega.BeNil())
		setup()
		os.Setenv("AMM_SUBACCOUNT_ID", ammSubaccountID)
		os.Setenv("REDIS_ADDR", mockRedis.Addr())
		xredis.Initialize()

		// Mock SubaccountTable object
		mockSubaccount = &db.SubaccountTable{
			GID:        1,
			BrokerId:   12345,
			EthAddress: "0x0000000000000000000000000000000000000000",
			ID:         testSubaccountID,
			UserName:   nil,
		}

		// Apply mock for getCurrentSubaccount
		patches = gomonkey.ApplyFunc(middleware.GetCurrentSubaccount, func(ctx *gin.Context) (*db.SubaccountTable, error) {
			return mockSubaccount, nil
		})
		// Mock CheckSubaccountPositions function
		ValidatePatches = gomonkey.ApplyFunc(middleware.CheckSubaccountPositions, func(marketId uint32, subaccountID string, isBuy bool, newVQuote *big.Int) (bool, *big.Int, error) {
			// Simulate a successful validation by returning true (position is allowed) and a mock currentVQuote
			mockCurrentVQuote := big.NewInt(1000) // Example value for currentVQuote
			return true, mockCurrentVQuote, nil
		})

		// Mock the fetchOraclePrice function
		OracelPatches = patches.ApplyFunc(middleware.FetchOraclePrice, func(marketId uint32) (*big.Int, error) {
			// Return a mocked price for the given market ID
			if marketId == 1 {
				return big.NewInt(10000000000000000), nil // 1 ETH in 18 decimals
			}
			return nil, fmt.Errorf("market ID not found")
		})
		FetchTotalPositionPatches = gomonkey.ApplyFunc(middleware.FetchTotalPosition, func(marketId uint32, priceStr *big.Int) (*big.Int, *big.Int, error) {
			// Mocked values
			mockLongPosition := big.NewInt(1000)
			mockShortPosition := big.NewInt(500)

			// Simulate no error
			return mockLongPosition, mockShortPosition, nil
		})
	})

	ginkgo.AfterEach(func() {
		mockRedis.FlushAll() // Clear all keys in mock Redis
		mockRedis.Close()    // Stop the Miniredis server
		patches.Reset()      // Reset patches
		ValidatePatches.Reset()
		OracelPatches.Reset()
		FetchTotalPositionPatches.Reset()
	})

	ginkgo.It("should pass for valid position limits for ORDER_TYPE_MARKET", func() {
		// Prepare request body for ORDER_TYPE_MARKET
		marketID := uint(1)
		isBuy := true
		expiryTs := uint64(1690000000)
		isReduce := false

		createOrderReq := ctypes.CreateOrderBody{
			MarketId:  &marketID,
			IsBuy:     &isBuy,
			OrderType: ctypes.ORDER_TYPE_MARKET,
			AmountStr: "100",
			PriceStr:  "1",
			ExpiryTs:  &expiryTs,
			Party:     ctypes.PARTY_SOLVER,
			IsReduce:  &isReduce,
		}

		body, err := json.Marshal(createOrderReq)
		gomega.Expect(err).To(gomega.BeNil())

		// Create a new httptest ResponseRecorder and gin.Context
		w := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(w)

		// Manually create a request and assign it to the context
		req := httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		ctx.Request = req

		// Set the mock subaccount in the context
		ctx.Set("current-subaccount", mockSubaccount)

		// Call middleware directly
		middleware.RequireValidPositionLimits(ctx)

		// Should return internal server error since the default limit which will be set is 0
		gomega.Expect(w.Code).To(gomega.Equal(http.StatusInternalServerError))
	})
})
