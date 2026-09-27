package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/controller"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/testutils/mocks"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setBalanceServiceEnv() {
	os.Setenv("ENV", "TESTNET")
	os.Setenv("SPOT_PRODUCT_IDS", "0,2,4,6,8,10,12,14,16,18,20,22")
	os.Setenv("QUOTE_PRODUCT_ID", "4")
	os.Setenv("RPC_URL", "https://kartel-testnet.alt.technology")
	os.Setenv("AMM_SUBACCOUNT_ID", "0x000000000001d37eD507cA37Faa17079bFf5e46DcedE951577dB000000000001")
	os.Setenv("ORACLE_SERVER_URL", "https://oracle.hundred.exchange")
	os.Setenv("DSN", "host=127.0.0.1 user=postgres password=root dbname=postgres port=5432 sslmode=disable")
}

// Mock GetBrokerFeeFactor to return 50 (5 bps) for all broker IDs in tests
func mockGetBrokerFeeFactor() *gomonkey.Patches {
	// Monkey patch GetBrokerFeeFactor to always return 50 (5 bps)
	patches := gomonkey.ApplyFunc(cutils.GetBrokerFeeFactor, func(brokerId int) int64 {
		return 50
	})
	return patches
}

var _ = ginkgo.Describe("BalanceService", func() {
	var balanceService *services.BalanceService
	var redisClient *redis.Client
	var ctx = context.Background()
	var subaccountID string = "0x0000000000011111111111111111111111111111111111111111000000000001"
	var makerSubaccountID string = "0x0000000000012222222222222222222222222222222222222222000000000001"
	var ammSubaccountID string = "0x1100000000010000000000000000000000000000000000000001000000000001"
	var router *gin.Engine
	var mockRedis *miniredis.Miniredis
	var patches *gomonkey.Patches

	// Initialize Redis client and BalanceService before each test
	ginkgo.BeforeEach(func() {
		var err error
		mockRedis, err = miniredis.Run()
		gomega.Expect(err).To(gomega.BeNil())

		os.Setenv("SPOT_PRODUCT_IDS", "0,2,4,6,8,10,12,14,16,18,20,22")
		os.Setenv("QUOTE_PRODUCT_ID", "4")
		os.Setenv("REDIS_ADDR", mockRedis.Addr())
		os.Setenv("AMM_SUBACCOUNT_ID", ammSubaccountID)

		redisClient = redis.NewClient(&redis.Options{
			Addr: mockRedis.Addr(),
		})

		//Initialise redis pool and other redis global variables
		xredis.Initialize()

		// Apply the mock for GetBrokerFeeFactor
		patches = mockGetBrokerFeeFactor()
		defer patches.Reset()

		balanceService = services.NewBalanceService()
		balanceController := controller.NewBalanceController()

		router = gin.Default()
		balanceGroup := router.Group("/balance")
		{
			balanceGroup.POST("/update-token-balance", balanceController.UpdateTokenBalanceHandler)
		}
	})

	// Clear Redis state after each test
	ginkgo.AfterEach(func() {
		hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
		redisClient.Del(ctx, hashKey)
		makerhashKey := fmt.Sprintf("v0_balance_%s", makerSubaccountID)
		redisClient.Del(ctx, makerhashKey)
		mockRedis.Close()

		// Patches are automatically reset via defer
	})

	validateSubaccountID := func(subaccountID string) {
		matched, err := regexp.MatchString(`^0x[0-9a-fA-F]{64}$`, subaccountID)
		gomega.Expect(err).To(gomega.BeNil())
		gomega.Expect(matched).To(gomega.BeTrue())
	}

	performRequest := func(method, path string, body interface{}) *httptest.ResponseRecorder {
		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest(method, path, bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		return resp
	}

	// mockCumulativeFundingRate := func(symbol string, rate, timestamp string) {
	// 	cumulativeFundingRateKey := xredis.GetCumulativeFundingRateKey(symbol)
	// 	newCumulativeFundingRateDict := xredis.CumulativeFundingRateData{
	// 		CumulativeFundingRate:      rate,
	// 		CumulativeFundingTimestamp: timestamp,
	// 	}

	// 	newCumulativeFundingRateJson, err := json.Marshal(newCumulativeFundingRateDict)
	// 	if err != nil {
	// 		fmt.Printf("Error marshalling JSON: %v\n", err)
	// 		return
	// 	}
	// 	err = redisClient.Set(context.Background(), cumulativeFundingRateKey, newCumulativeFundingRateJson, 0).Err()
	// 	if err != nil {
	// 		fmt.Printf("Error storing funding rate: %v\n", err)
	// 	} else {
	// 		fmt.Printf("Cumulative Funding Rate value set for key %v: %v\n", cumulativeFundingRateKey, newCumulativeFundingRateDict)
	// 	}
	// }

	// Unit tests for UpdateTokenBalance
	ginkgo.Context("UpdateTokenBalance", func() {
		// Validate subaccountID before all tests
		ginkgo.BeforeEach(func() {
			validateSubaccountID(subaccountID)
		})

		// Test for successfully updating token balance with an even productId
		ginkgo.It("should update token balance successfully when productId is even", func() {
			productId := uint32(2)
			tokenBalance := "1000"

			err := balanceService.UpdateTokenBalance(subaccountID, productId, tokenBalance)
			gomega.Expect(err).To(gomega.BeNil())

			hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
			field := fmt.Sprintf("%d", productId)
			result, err := redisClient.HGet(ctx, hashKey, field).Result()
			gomega.Expect(err).To(gomega.BeNil())

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(balance.Available).To(gomega.Equal(tokenBalance))
		})

		// Test for returning an error when productId is odd
		ginkgo.It("should return an error when productId is odd", func() {
			productId := uint32(3)
			tokenBalance := "1000"

			err := balanceService.UpdateTokenBalance(subaccountID, productId, tokenBalance)
			gomega.Expect(err).ToNot(gomega.BeNil())
			gomega.Expect(err.Error()).To(gomega.Equal("productId must be even for spot balances"))
		})

		// Test for increasing token balance when tokenBalance is positive
		ginkgo.It("should increase token balance when tokenBalance is positive", func() {
			productId := uint32(2)
			initialBalance := "500"
			incrementBalance := "500"

			// Set initial balance
			err := balanceService.UpdateTokenBalance(subaccountID, productId, initialBalance)
			gomega.Expect(err).To(gomega.BeNil())

			// Increase balance
			err = balanceService.UpdateTokenBalance(subaccountID, productId, incrementBalance)
			gomega.Expect(err).To(gomega.BeNil())

			hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
			field := fmt.Sprintf("%d", productId)
			result, err := redisClient.HGet(ctx, hashKey, field).Result()
			gomega.Expect(err).To(gomega.BeNil())

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			gomega.Expect(err).To(gomega.BeNil())

			expectedBalance := new(big.Int)
			expectedBalance.SetString(initialBalance, 10)
			incrementBigInt, _ := new(big.Int).SetString(incrementBalance, 10)
			expectedBalance.Add(expectedBalance, incrementBigInt)

			gomega.Expect(balance.Available).To(gomega.Equal(expectedBalance.String()))
		})

		// Test for decreasing token balance when tokenBalance is negative
		ginkgo.It("should decrease token balance when tokenBalance is negative", func() {
			productId := uint32(2)
			initialBalance := "1000"
			decrementBalance := "-500"

			// Set initial balance
			err := balanceService.UpdateTokenBalance(subaccountID, productId, initialBalance)
			gomega.Expect(err).To(gomega.BeNil())

			// Decrease balance
			err = balanceService.UpdateTokenBalance(subaccountID, productId, decrementBalance)
			gomega.Expect(err).To(gomega.BeNil())

			hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
			field := fmt.Sprintf("%d", productId)
			result, err := redisClient.HGet(ctx, hashKey, field).Result()
			gomega.Expect(err).To(gomega.BeNil())

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			gomega.Expect(err).To(gomega.BeNil())

			expectedBalance := new(big.Int)
			expectedBalance.SetString(initialBalance, 10)
			decrementBigInt, _ := new(big.Int).SetString(decrementBalance, 10)
			expectedBalance.Add(expectedBalance, decrementBigInt)

			gomega.Expect(balance.Available).To(gomega.Equal(expectedBalance.String()))
		})
	})

	// Integration tests for UpdateTokenBalanceHandler
	ginkgo.Context("UpdateTokenBalanceHandler", func() {
		// Test for successfully updating token balance
		ginkgo.It("should update token balance successfully when productId is even", func() {
			subaccountID := subaccountID
			request := types.TokenBalanceRequest{
				SubaccountID: subaccountID,
				ProductId:    2,
				TokenBalance: "1000",
			}

			resp := performRequest("POST", "/balance/update-token-balance", request)
			gomega.Expect(resp.Code).To(gomega.Equal(http.StatusOK))

			hashKey := "v0_balance_" + strings.ToLower(subaccountID)
			field := "2"
			result, err := redisClient.HGet(ctx, hashKey, field).Result()
			gomega.Expect(err).To(gomega.BeNil())

			var balance types.Balance
			err = json.Unmarshal([]byte(result), &balance)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(balance.Available).To(gomega.Equal(request.TokenBalance))
		})

		// Test for returning an error when productId is odd
		ginkgo.It("should return an error when productId is odd", func() {
			subaccountID := subaccountID
			request := types.TokenBalanceRequest{
				SubaccountID: subaccountID,
				ProductId:    3,
				TokenBalance: "1000",
			}

			resp := performRequest("POST", "/balance/update-token-balance", request)
			gomega.Expect(resp.Code).To(gomega.Equal(http.StatusInternalServerError))

			var responseBody map[string]string
			err := json.Unmarshal(resp.Body.Bytes(), &responseBody)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(responseBody["error"]).To(gomega.Equal("productId must be even for spot balances"))
		})
	})

	// Unit tests for UpdatePerpBalance
	// ginkgo.Context("UpdatePerpBalance", func() {
	// 	// Test for returning an error when productId is even
	// 	ginkgo.It("should return an error when productId is even", func() {
	// 		productId := uint32(2)
	// 		req := types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          productId,
	// 			TakerAmount:        "1000",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "1000",
	// 		}

	// 		err := balanceService.UpdatePerpBalance(req)
	// 		gomega.Expect(err).ToNot(gomega.BeNil())
	// 		gomega.Expect(err.Error()).To(gomega.Equal("productId must be odd for perp balances"))
	// 	})

	// 	// Test for successfully updating perp balance with an odd productId
	// 	ginkgo.It("should update perp balance successfully when productId is odd", func() {
	// 		productId := uint32(1)
	// 		req := types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          productId,
	// 			TakerAmount:        "10",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "1000",
	// 		}
	// 		mockCumulativeFundingRate("ETH", "-180312499998846", "0")

	// 		err := balanceService.UpdatePerpBalance(req)
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		hashKey := fmt.Sprintf("v0_balance_%s", subaccountID)
	// 		field := fmt.Sprintf("%d", productId)
	// 		result, err := redisClient.HGet(ctx, hashKey, field).Result()
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		var balance types.PerpBalance
	// 		err = json.Unmarshal([]byte(result), &balance)
	// 		gomega.Expect(err).To(gomega.BeNil())
	// 		gomega.Expect(balance.Amount).To(gomega.Equal(req.TakerAmount))
	// 		gomega.Expect(balance.VQuoteBalance).To(gomega.Equal(req.TakerVQuoteBalance))
	// 	})

	// 	// Unit tests for updatePerpBalanceInRedis
	// 	ginkgo.It("should update perp balance in Redis correctly", func() {
	// 		subAccountId := subaccountID
	// 		productId := uint32(1)
	// 		updatedAmount := "2000"
	// 		updatedVQuoteBalance := "4000"
	// 		lastFundingRate := "1000"

	// 		err := balanceService.UpdatePerpBalanceInRedis(subAccountId, productId, updatedAmount, updatedVQuoteBalance, lastFundingRate)
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		hashKey := fmt.Sprintf("v0_balance_%s", subAccountId)
	// 		field := fmt.Sprintf("%d", productId)
	// 		result, err := redisClient.HGet(ctx, hashKey, field).Result()
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		var updatedBalance types.PerpBalance
	// 		err = json.Unmarshal([]byte(result), &updatedBalance)
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		gomega.Expect(updatedBalance.Amount).To(gomega.Equal(updatedAmount))
	// 		gomega.Expect(updatedBalance.VQuoteBalance).To(gomega.Equal(updatedVQuoteBalance))
	// 		gomega.Expect(updatedBalance.LastFundingRate).To(gomega.Equal(lastFundingRate))
	// 	})

	// 	ginkgo.It("should update balance for match order correctly", func() {
	// 		productId := uint32(1)
	// 		currentFundingRate := "-181312499998846"
	// 		subAccountId := subaccountID
	// 		newAmount := "40000000000000000000"
	// 		newVQuoteBalance := "20000000000000000000"
	// 		perpBalance := types.PerpBalance{
	// 			Amount:          "20000000000000000000",
	// 			VQuoteBalance:   "10000000000000000000",
	// 			LastFundingRate: "-180312499998846",
	// 		}
	// 		isTaker := true

	// 		balanceService.UpdateBalanceForMatchOrder(productId, currentFundingRate, subAccountId, newAmount, newVQuoteBalance, perpBalance, isTaker)

	// 		// Check the updated Perp balance in Redis
	// 		hashKey := fmt.Sprintf("v0_balance_%s", subAccountId)
	// 		field := fmt.Sprintf("%d", productId)
	// 		result, err := redisClient.HGet(ctx, hashKey, field).Result()
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		var updatedPerpBalance types.PerpBalance
	// 		err = json.Unmarshal([]byte(result), &updatedPerpBalance)
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		expectedAmount := "60000000000000000000"        // amount + newAmount
	// 		expectedVQuoteBalance := "30000000000000000000" // case when both sign of amount and new amount same

	// 		gomega.Expect(updatedPerpBalance.Amount).To(gomega.Equal(expectedAmount))
	// 		gomega.Expect(updatedPerpBalance.VQuoteBalance).To(gomega.Equal(expectedVQuoteBalance))

	// 		// Check the updated token balance for the quote ID in Redis
	// 		quoteProductId := os.Getenv("QUOTE_PRODUCT_ID")
	// 		quoteResult, err := redisClient.HGet(ctx, hashKey, quoteProductId).Result()
	// 		gomega.Expect(err).To(gomega.BeNil())

	// 		expectedTokenBalanceUpdate := "-10010000000000000"
	// 		var balance types.Balance
	// 		err = json.Unmarshal([]byte(quoteResult), &balance)
	// 		gomega.Expect(err).To(gomega.BeNil())
	// 		gomega.Expect(balance.Available).To(gomega.Equal(expectedTokenBalanceUpdate))
	// 	})

	// 	// Unit tests for calcFundingFees
	// 	ginkgo.It("should calculate funding fees correctly", func() {
	// 		perpBalance := types.PerpBalance{
	// 			Amount:          "20000000000000000000",
	// 			VQuoteBalance:   "10000000000000000000",
	// 			LastFundingRate: "-180312499998846",
	// 		}
	// 		currentFundingRate := big.NewInt(-181312499998846)

	// 		fundingFees := utils.CalcFundingFeesx18(perpBalance, currentFundingRate)
	// 		expectedFundingFees := big.NewInt(10000000000000)

	// 		gomega.Expect(fundingFees).To(gomega.Equal(expectedFundingFees))
	// 	})
	// Integration tests for UpdatePerpBalanceHandler
	// ginkgo.DescribeTable("UpdatePerpBalanceHandler",
	// 	func(req types.PerpBalanceRequest, expectedStatus int, expectedBodySubstring string) {
	// 		resp := performRequest("POST", "/balance/update-perp-balance", req)
	// 		gomega.Expect(resp.Code).To(gomega.Equal(expectedStatus))
	// 		if expectedBodySubstring != "" {
	// 			gomega.Expect(resp.Body.String()).To(gomega.ContainSubstring(expectedBodySubstring))
	// 		}
	// 	},
	// 	ginkgo.Entry("should return an error when productId is even",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          2,
	// 			TakerAmount:        "1000",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "1000",
	// 		}, http.StatusBadRequest, "Failed to pass parameter validation"),
	// 	ginkgo.Entry("should return bad request when amount and quote are of same sign",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          3,
	// 			TakerAmount:        "1000",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "1000",
	// 		}, http.StatusBadRequest, "Failed to pass parameter validation"),
	// 	ginkgo.Entry("should return bad request when taker amount or quote is zero",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          3,
	// 			TakerAmount:        "0",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "1000",
	// 		}, http.StatusBadRequest, "Failed to pass parameter validation"),
	// 	ginkgo.Entry("should return bad request when maker amount or quote is zero",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          3,
	// 			TakerAmount:        "1000",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "0",
	// 			MakerVQuoteBalance: "1000",
	// 		}, http.StatusBadRequest, "Failed to pass parameter validation"),
	// 	ginkgo.Entry("should return bad request when both amounts and quotes are invalid",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          3,
	// 			TakerAmount:        "0",
	// 			TakerVQuoteBalance: "0",
	// 			MakerAmount:        "0",
	// 			MakerVQuoteBalance: "0",
	// 		}, http.StatusBadRequest, "Failed to pass parameter validation"),
	// 	ginkgo.Entry("should update perp balance successfully when productId is odd",
	// 		types.PerpBalanceRequest{
	// 			TakerSubaccountID:  subaccountID,
	// 			MakerSubaccountID:  makerSubaccountID,
	// 			ProductId:          3,
	// 			TakerAmount:        "-1000",
	// 			TakerVQuoteBalance: "1000",
	// 			MakerAmount:        "1000",
	// 			MakerVQuoteBalance: "-1000",
	// 		}, http.StatusOK, ""),
	// )

})

func TestGetAvailableMarginv2(t *testing.T) {
	t.Skip("Skipping test because it requires a bit of mocking...")
	setBalanceServiceEnv()
	testutils.WithSetupMockRedis(t, func() {
		xclient.InitOracleClient()
		db.Init()

		balanceService := services.NewBalanceService()

		t.Run("should return available margin for a subaccount", func(t *testing.T) {
			balances := &subaccountTypes.SubaccountBalances{
				SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002",
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
					contractUtils.ETH_USDC: {
						Balancex18: cutils.Mulx18(big.NewInt(100)),
						Lockedx18:  cutils.GetBig0(),
					},
				},
				PerpBalances: map[uint32]subaccountTypes.PerpBalance{
					contractUtils.ETH_MARKET: {
						Amountx18:             cutils.Mulx18(big.NewInt(-1)),
						VQuoteBalancex18:      cutils.Mulx18(big.NewInt(100)),
						LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
					},
				},
			}

			data := &xredis.CumulativeFundingRateData{
				CumulativeFundingRate:      "1000000000000000000",
				CumulativeFundingTimestamp: "0",
			}

			_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
				err := subaccount.ReplaceBalanceInRedis(pipe, balances)
				if err != nil {
					return err
				}

				for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
					err = pipe.Set(context.Background(), xredis.GetCumulativeFundingRateKey(contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[productId]), *data, 0).Err()
					if err != nil {
						return err
					}
				}

				err = pipe.Set(context.Background(), xredis.GetCumulativeFundingRateKey(contractUtils.PRODUCT_ID_SYMBOL_TO_MAP[contractUtils.ETH_MARKET]), *data, 0).Err()
				if err != nil {
					return err
				}

				return nil
			})

			assert.Nilf(t, err, "Error setting up mock redis: %v", err)

			value, err := balanceService.GetAvailableMarginV2(balances.SubaccountId)
			if err == nil {
				fmt.Printf("Available Margin for subaccount %s: %s\n", balances.SubaccountId, value)
			} else {
				t.Errorf("Error getting available margin for subaccount %s: %v\n", balances.SubaccountId, err)
			}
		})
	})
}

func TestUpdateLocalBalanceForOrderMatch(t *testing.T) {
	setBalanceServiceEnv()

	// Apply the mock for GetBrokerFeeFactor at the beginning of the test
	patches := mockGetBrokerFeeFactor()
	defer patches.Reset()

	testutils.WithSetupMockRedis(t, func() {
		contractUtils.Init()
		subaccountHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"
		type TestCase struct {
			Name                       string
			SubaccountBalances         subaccountTypes.SubaccountBalances
			BalancePerpRequest         types.BalancePerpPayload
			ProductId                  uint32
			OraclePricesMap            map[string]ctypes.OraclePrice
			PerpetualMarketsMap        map[uint]subaccountTypes.PerpetualMarket
			CumulativeFundingRateMap   map[string]*big.Int
			ExpectedSubaccountBalances subaccountTypes.SubaccountBalances
			ExpectedRealizedPnl        *big.Int
		}

		testCases := []TestCase{
			{
				Name: "Just adding a new position as a taker",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				BalancePerpRequest: types.BalancePerpPayload{
					SubaccountId:     subaccountHex,
					Amountx18:        cutils.Mulx18(big.NewInt(100)),
					VQuoteBalancex18: cutils.Mulx18(big.NewInt(-100)), // $100 position -> Trading fees = $0.05
				},
				ProductId: contractUtils.ETH_MARKET,
				OraclePricesMap: map[string]ctypes.OraclePrice{
					"ETH": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "ETH",
					},
					"USDC": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "USDC",
					},
				},
				PerpetualMarketsMap: map[uint]subaccountTypes.PerpetualMarket{
					contractUtils.ETH_MARKET: {
						ProductId:                    contractUtils.ETH_MARKET,
						BaseAsset:                    "ETH",
						MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
						InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
					},
				},
				CumulativeFundingRateMap: map[string]*big.Int{
					"ETH": cutils.Mulx18(big.NewInt(0)),
				},
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						// Funds will be removed from QUOTE_PRODUCT_ID
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-5), 16), // $0.05
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
				},
				ExpectedRealizedPnl: cutils.Mulx18(big.NewInt(0)),
			},
			{
				Name: "Closing newly opened position as a taker",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						// Funds will be removed from QUOTE_PRODUCT_ID
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-5), 16),
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
				},
				BalancePerpRequest: types.BalancePerpPayload{
					SubaccountId:     subaccountHex,
					Amountx18:        cutils.Mulx18(big.NewInt(-100)),
					VQuoteBalancex18: cutils.Mulx18(big.NewInt(+99)), // $99 position -> Trading fees = $0.0495
				},
				ProductId: contractUtils.ETH_MARKET,
				OraclePricesMap: map[string]ctypes.OraclePrice{
					"ETH": {
						Pricex18: cutils.MulxCust(big.NewInt(99), 17), // $0.99
						Symbol:   "ETH",
					},
					"USDC": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "USDC",
					},
				},
				PerpetualMarketsMap: map[uint]subaccountTypes.PerpetualMarket{
					contractUtils.ETH_MARKET: {
						ProductId:                    contractUtils.ETH_MARKET,
						BaseAsset:                    "ETH",
						MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
						InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
					},
				},
				CumulativeFundingRateMap: map[string]*big.Int{
					"ETH": cutils.Mulx18(big.NewInt(0)),
				},
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-10995), 14), // $1 loss + $0.05 + $0.0495 = $1.0995
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				ExpectedRealizedPnl: cutils.Mulx18(big.NewInt(-1)),
			},
			{
				Name: "Changing the sign of position as a taker",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						// Funds will be removed from QUOTE_PRODUCT_ID
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-5), 16),
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
				},
				BalancePerpRequest: types.BalancePerpPayload{
					SubaccountId:     subaccountHex,
					Amountx18:        cutils.Mulx18(big.NewInt(-200)),
					VQuoteBalancex18: cutils.Mulx18(big.NewInt(+198)), // $198 position -> Trading fees = $0.099
				},
				ProductId: contractUtils.ETH_MARKET,
				OraclePricesMap: map[string]ctypes.OraclePrice{
					"ETH": {
						Pricex18: cutils.MulxCust(big.NewInt(99), 16), // $0.99
						Symbol:   "ETH",
					},
					"USDC": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "USDC",
					},
				},
				PerpetualMarketsMap: map[uint]subaccountTypes.PerpetualMarket{
					contractUtils.ETH_MARKET: {
						ProductId:                    contractUtils.ETH_MARKET,
						BaseAsset:                    "ETH",
						MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
						InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
					},
				},
				CumulativeFundingRateMap: map[string]*big.Int{
					"ETH": cutils.Mulx18(big.NewInt(0)),
				},
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-1149), 15), // $1 loss + $0.05 + $0.099 = $1.149
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(-100)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(+99)),
							LastCumFundingRatex18: cutils.Mulx18(big.NewInt(0)),
						},
					},
				},
				ExpectedRealizedPnl: cutils.Mulx18(big.NewInt(-1)),
			},
			{
				Name: "Closing position partially as a taker",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						// Funds will be removed from QUOTE_PRODUCT_ID
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.GetBig0(),
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
				},
				BalancePerpRequest: types.BalancePerpPayload{
					SubaccountId:     subaccountHex,
					Amountx18:        cutils.Mulx18(big.NewInt(-50)),
					VQuoteBalancex18: cutils.MulxCust(big.NewInt(495), 17), // $49.5 position -> Trading fees = $0.02475
				},
				ProductId: contractUtils.ETH_MARKET,
				OraclePricesMap: map[string]ctypes.OraclePrice{
					"ETH": {
						Pricex18: cutils.MulxCust(big.NewInt(99), 16), // $0.99
						Symbol:   "ETH",
					},
					"USDC": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "USDC",
					},
				},
				PerpetualMarketsMap: map[uint]subaccountTypes.PerpetualMarket{
					contractUtils.ETH_MARKET: {
						ProductId:                    contractUtils.ETH_MARKET,
						BaseAsset:                    "ETH",
						MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
						InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
					},
				},
				CumulativeFundingRateMap: map[string]*big.Int{
					"ETH": cutils.MulxCust(big.NewInt(1), 15), // 0.1% of $100 -> $0.1
				},
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-62475), 13), // $0.5 loss + $0.02475 + $0.1 = $0.62475
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{
						contractUtils.ETH_MARKET: {
							ProductId:             contractUtils.ETH_MARKET,
							Amountx18:             cutils.Mulx18(big.NewInt(50)),
							VQuoteBalancex18:      cutils.Mulx18(big.NewInt(-50)),
							LastCumFundingRatex18: cutils.MulxCust(big.NewInt(1), 15),
						},
					},
				},
				ExpectedRealizedPnl: big.NewInt(-600000000000000000),
			},
			{
				Name: "Just adding a new position as a taker with differnt ProductId",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				BalancePerpRequest: types.BalancePerpPayload{
					SubaccountId:     subaccountHex,
					Amountx18:        cutils.Mulx18(big.NewInt(100)),
					VQuoteBalancex18: cutils.Mulx18(big.NewInt(-100)), // $100 position -> Trading fees = $0.05
				},
				ProductId: contractUtils.ETH_MARKET,
				OraclePricesMap: map[string]ctypes.OraclePrice{
					"ETH": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "ETH",
					},
					"USDC": {
						Pricex18: cutils.Mulx18(big.NewInt(1)),
						Symbol:   "USDC",
					},
				},
				PerpetualMarketsMap: map[uint]subaccountTypes.PerpetualMarket{
					contractUtils.ETH_MARKET: {
						ProductId:                    contractUtils.BTC_MARKET,
						BaseAsset:                    "ETH",
						MaintenanceMarginFractionx18: cutils.MulxCust(big.NewInt(2), 16), // 2%
						InitialMarginFractionx18:     cutils.MulxCust(big.NewInt(5), 16), // 5%
					},
				},
				CumulativeFundingRateMap: map[string]*big.Int{
					"ETH": cutils.Mulx18(big.NewInt(0)),
				},
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(100)),
							Lockedx18:  cutils.GetBig0(),
						},
						// Funds will be removed from QUOTE_PRODUCT_ID
						contractUtils.QUOTE_TOKEN_PRODUCT_ID: {
							ProductId:  contractUtils.QUOTE_TOKEN_PRODUCT_ID,
							Balancex18: cutils.MulxCust(big.NewInt(-5), 16), // $0.05
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
				},
				ExpectedRealizedPnl: cutils.Mulx18(big.NewInt(0)),
			},
		}

		testutils.SetupBalanceEnv()
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				balanceService := services.NewBalanceService()
				updatedBalance, isHealthyBalanceUpdate, realized_pnl, _ := balanceService.UpdateLocalBalanceForOrderMatch(tc.SubaccountBalances, tc.BalancePerpRequest, uint32(tc.ProductId), true, tc.OraclePricesMap, tc.PerpetualMarketsMap, tc.CumulativeFundingRateMap)

				assert.True(t, isHealthyBalanceUpdate, "Expected healthy balance update")
				assert.True(t, updatedBalance.Equals(&tc.ExpectedSubaccountBalances), "Expected updated balance to match")
				assert.True(t, realized_pnl.Cmp(tc.ExpectedRealizedPnl) == 0, "Expected realized_pnl to match the expected value")
			})
		}
	})
}

func TestUpdateTokenBalance(t *testing.T) {
	setBalanceServiceEnv()
	testutils.WithSetupMockRedis(t, func() {
		contractUtils.Init()
		subaccountHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"
		type TestCase struct {
			Name                       string
			SubaccountBalances         subaccountTypes.SubaccountBalances
			BalanceTokenRequest        subaccountTypes.SpotBalance
			TokenRequestCount          int
			ExpectedSubaccountBalances subaccountTypes.SubaccountBalances
		}

		testCases := []TestCase{
			{
				Name: "Adding token balance once",
				SubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.GetBig0(),
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
				BalanceTokenRequest: subaccountTypes.SpotBalance{
					ProductId:  contractUtils.ARB_USDC,
					Balancex18: cutils.Mulx18(big.NewInt(100)),
				},
				TokenRequestCount: 10,
				ExpectedSubaccountBalances: subaccountTypes.SubaccountBalances{
					SubaccountId: subaccountHex,
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.Mulx18(big.NewInt(1000)),
							Lockedx18:  cutils.GetBig0(),
						},
					},
					PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
				},
			},
		}

		sbi := subaccount.NewSubaccountBalanceImpl()

		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				balanceService := services.NewBalanceService()
				_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					subaccount.ReplaceBalanceInRedis(pipe, &tc.SubaccountBalances)
					return nil
				})
				assert.Nil(t, err, "Error setting up mock redis")

				wg := sync.WaitGroup{}

				for i := 0; i < tc.TokenRequestCount; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						updatedBalance := balanceService.UpdateTokenBalance(tc.SubaccountBalances.SubaccountId, tc.BalanceTokenRequest.ProductId, tc.BalanceTokenRequest.Balancex18.String())
						assert.NoError(t, updatedBalance, "Expected updated balance to be non-nil")
					}()
				}

				wg.Wait()
				updatedBalance := sbi.MustGetSubaccountBalancesFromIds([]string{tc.SubaccountBalances.SubaccountId})[0]

				assert.Equal(t, tc.ExpectedSubaccountBalances.Equals(&updatedBalance), true, "Expected updated balance to match")
			})
		}
	})

}

func TestSettlePnl(t *testing.T) {
	testutils.SetMainnetEnv()
	testutils.SetupBalanceEnv()

	patches := gomonkey.NewPatches()
	defer patches.Reset()
	patches.ApplyFunc((*appstate.AppStateImp).GetAllOraclePrices, func(a appstate.AppState, _ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
		return mocks.GetMockOraclePrices(), nil
	})
	type TestCase struct {
		Name               string
		subaccounts        []string
		SubaccountBalances []subaccountTypes.SubaccountBalances
		ExpectedBalance    []subaccountTypes.SubaccountBalances
	}

	testCases := []TestCase{
		{
			Name: "Test 2 subaccount with -ve quote",
			subaccounts: []string{
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000001",
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d81000000000001",
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d82000000000001",
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d83000000000001",
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d84000000000001",
				"0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d85000000000001",
			},
			SubaccountBalances: []subaccountTypes.SubaccountBalances{
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("-50"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d81000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("-25"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d82000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d83000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("25"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d84000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("-150"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d85000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("-60"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
			},
			ExpectedBalance: []subaccountTypes.SubaccountBalances{
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("50"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d81000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("75"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d82000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d83000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("25"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("100"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d84000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("-50"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
				{
					SubaccountId: "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d85000000000001",
					SpotBalances: map[uint32]subaccountTypes.SpotBalance{
						contractUtils.ARB_USDC: {
							ProductId:  contractUtils.ARB_USDC,
							Balancex18: cutils.FloatStrToX18("0"),
							Lockedx18:  cutils.GetBig0(),
						},
						contractUtils.OSTRICH_USDC_UPDATED: {
							ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
							Balancex18: cutils.FloatStrToX18("40"),
							Lockedx18:  cutils.GetBig0(),
						},
					},
				},
			},
		},
	}

	testutils.WithSetupMockRedis(t, func() {
		balanceService := services.NewBalanceService()
		sbi := subaccount.NewSubaccountBalanceImpl()
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					for _, subaccountBalance := range tc.SubaccountBalances {
						subaccount.ReplaceBalanceInRedis(pipe, &subaccountBalance)
					}
					return nil
				})
				assert.Nil(t, err, "Error setting up mock redis")
				var newAccounts []string
				newAccounts = append(newAccounts, tc.subaccounts...)
				_, failedAccounts, _ := balanceService.SettlePnLForSubaccounts(newAccounts)
				xlog.Infof("Failed accounts: %v", failedAccounts)
				updatedBalances := sbi.MustGetSubaccountBalancesFromIds(tc.subaccounts)
				for index, finalBalance := range updatedBalances {
					assert.Equal(t, tc.ExpectedBalance[index].Equals(&finalBalance), true, "Expected updated balance to match")
				}

			})
		}
	})
}

func TestGetTotalSpotValuex18(t *testing.T) {
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc((*appstate.AppStateImp).GetAllOraclePrices, func(a appstate.AppState, _ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
		return mocks.GetMockOraclePrices(), nil
	})
	subaccountHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"

	type TestCase struct {
		Name                      string
		SubaccountBalances        subaccountTypes.SubaccountBalances
		ExpectedTotalSpotValuex18 *big.Int
	}

	testCases := []TestCase{
		{
			Name: "Case 1: Zero balance",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
			},
			ExpectedTotalSpotValuex18: big.NewInt(0),
		},
		{
			Name: "Case 2: Non-zero balance but only quote product",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.GetBig0(),
					},
					// This should not be included in the total spot value
					contractUtils.ST_LOGX: {
						ProductId:  contractUtils.ST_LOGX,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.GetBig0(),
					},
					// This should not be included in the total spot value
					contractUtils.LOGX: {
						ProductId:  contractUtils.LOGX,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.GetBig0(),
					},
				},
			},
			ExpectedTotalSpotValuex18: cutils.FloatStrToX18("100"),
		},
		{
			Name: "Case 3: Non-zero balance with multiple products",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.GetBig0(),
					},
					contractUtils.OSTRICH_USDC_UPDATED: {
						ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
						Balancex18: cutils.FloatStrToX18("200"),
						Lockedx18:  cutils.GetBig0(),
					},
				},
			},
			ExpectedTotalSpotValuex18: cutils.FloatStrToX18("300"),
		},
	}

	testutils.WithSetupMockRedis(t, func() {
		balanceService := services.NewBalanceService()
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					return subaccount.ReplaceBalanceInRedis(pipe, &tc.SubaccountBalances)
				})
				assert.Nil(t, err, "Error setting up mock redis")

				totalSpotValuex18, err := balanceService.GetTotalSpotValuex18(tc.SubaccountBalances.SubaccountId)
				assert.Nil(t, err, "Expected no error")
				assert.Equalf(t, tc.ExpectedTotalSpotValuex18.Cmp(totalSpotValuex18), 0, "Expected total spot value to match: %+v but got %+v", tc.ExpectedTotalSpotValuex18, totalSpotValuex18)
			})
		}
	})
}

func TestGetValueInQuoteToken(t *testing.T) {
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()

	patches := gomonkey.NewPatches()

	patches = patches.ApplyFunc((*appstate.AppStateImp).GetAllOraclePrices, func(a appstate.AppState, _ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
		return mocks.GetMockOraclePrices(), nil
	})

	defer patches.Reset()

	type TestCase struct {
		Name                    string
		ProductId               uint32
		Amountx18               *big.Int
		ExpectedValueInQuotex18 *big.Int
	}

	testCases := []TestCase{
		{
			Name:                    "Case 1: Zero Amount",
			ProductId:               contractUtils.ARB_USDC,
			Amountx18:               cutils.FloatStrToX18("0"),
			ExpectedValueInQuotex18: cutils.FloatStrToX18("0"),
		},
		{
			Name:                    "Case 2: Non-zero Quote Amount",
			ProductId:               contractUtils.ARB_USDC,
			Amountx18:               cutils.FloatStrToX18("100"),
			ExpectedValueInQuotex18: cutils.FloatStrToX18("100"),
		},
		{
			Name:                    "Case 3: Non-zero OSTRICH_USDC_UPDATED Amount",
			ProductId:               contractUtils.OSTRICH_USDC_UPDATED,
			Amountx18:               cutils.FloatStrToX18("100"),
			ExpectedValueInQuotex18: cutils.FloatStrToX18("100"),
		},
	}

	testutils.WithSetupMockRedis(t, func() {
		balanceService := services.NewBalanceService()
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				valueInQuote, err := balanceService.GetValueInQuoteTokenx18(tc.ProductId, tc.Amountx18)
				assert.Nil(t, err, "Expected no error")
				assert.Equalf(t, tc.ExpectedValueInQuotex18.Cmp(valueInQuote), 0, "Expected value in quote to match: %+v but got %+v", tc.ExpectedValueInQuotex18, valueInQuote)
			})
		}
	})
}

func TestGetSpotBalance(t *testing.T) {
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	subaccountHex := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"

	patches.ApplyFunc((*appstate.AppStateImp).GetAllOraclePrices, func(a appstate.AppState, _ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
		return mocks.GetMockOraclePrices(), nil
	})

	type TestCase struct {
		Name               string
		SubaccountBalances subaccountTypes.SubaccountBalances
		FreeBalancex18     string
		UsedBalancex18     string
	}

	testCases := []TestCase{
		{
			Name: "Case 1: Zero balance",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
			},
			FreeBalancex18: "0",
			UsedBalancex18: "0",
		},
		{
			Name: "Case 2: Non-zero balance but but nothing is locked",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.GetBig0(),
					},
				},
			},
			FreeBalancex18: cutils.FloatStrToX18("100").String(),
			UsedBalancex18: "0",
		},
		{
			Name: "Case 3: Non-zero balance with some locked",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.FloatStrToX18("99"),
						Lockedx18:  cutils.FloatStrToX18("50"),
					},
				},
			},
			FreeBalancex18: cutils.FloatStrToX18("99").String(),
			UsedBalancex18: cutils.FloatStrToX18("50").String(),
		},
		{
			Name: "Case 4: Non-zero balance with multiple tokens",
			SubaccountBalances: subaccountTypes.SubaccountBalances{
				SubaccountId: subaccountHex,
				SpotBalances: map[uint32]subaccountTypes.SpotBalance{
					contractUtils.ARB_USDC: {
						ProductId:  contractUtils.ARB_USDC,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.FloatStrToX18("100"),
					},
					contractUtils.OSTRICH_USDC_UPDATED: {
						ProductId:  contractUtils.OSTRICH_USDC_UPDATED,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.FloatStrToX18("0"),
					},
					contractUtils.ST_LOGX: {
						ProductId:  contractUtils.ST_LOGX,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.FloatStrToX18("0"),
					},
					contractUtils.LOGX: {
						ProductId:  contractUtils.LOGX,
						Balancex18: cutils.FloatStrToX18("100"),
						Lockedx18:  cutils.FloatStrToX18("0"),
					},
				},
			},
			FreeBalancex18: cutils.FloatStrToX18("199.975").String(),
			UsedBalancex18: cutils.FloatStrToX18("100").String(),
		},
	}

	testutils.WithSetupMockRedis(t, func() {
		balanceService := services.NewBalanceService()
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				_, err := xredis.GetRedisClient().TxPipelined(context.Background(), func(pipe redis.Pipeliner) error {
					return subaccount.ReplaceBalanceInRedis(pipe, &tc.SubaccountBalances)
				})
				assert.Nil(t, err, "Error setting up mock redis")

				_, freeBalance, usedBalance, err := balanceService.GetSpotBalance(tc.SubaccountBalances.SubaccountId)
				assert.Nil(t, err, "Expected no error")
				assert.Equalf(t, tc.FreeBalancex18, freeBalance, "Expected free balance to match: %+v but got %+v", tc.FreeBalancex18, freeBalance)
				assert.Equalf(t, tc.UsedBalancex18, usedBalance, "Expected used balance to match: %+v but got %+v", tc.UsedBalancex18, usedBalance)
			})
		}
	})

}
