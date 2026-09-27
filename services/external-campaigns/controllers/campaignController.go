package controllers

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"io"
	"math/big"
	"net/http"
	"os"
	"services/external-campaigns/dtos"
	"services/external-campaigns/services"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/gin-gonic/gin"
)

func SendMessageToDiscordHandler(ctx *gin.Context) {
	validatedMessage, exists := ctx.Get("validatedMessage")
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message": "Validated message not found",
		})
		return
	}
	message := validatedMessage.(string)
	err := xclient.GlobalDiscordClient.SendWebhookMessage(message)
	if err != nil {
		xlog.Errorf("Error sending message to Discord: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message": "Failed to send message to Discord",
		})
		return
	}

	// Success response
	ctx.JSON(http.StatusOK, gin.H{
		"message": "Message sent to Discord successfully",
	})
}
func ReferralCountHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	referralCount, err := services.ReferralCountForAddress(address)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	returnObj := gin.H{
		"data": gin.H{
			"userAddress":   address,
			"referralCount": referralCount,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}

func TotalSupplyHandler(ctx *gin.Context) {
	totalSupply := 1000000000
	ctx.String(http.StatusOK, "%d", totalSupply)
}

func CirculatingSupplyHandler(ctx *gin.Context) {
	circulatingSupply := 547330000
	ctx.String(http.StatusOK, "%d", circulatingSupply)
}

func CirculatingSupplyHandlerGecko(ctx *gin.Context) {
	circulatingSupply := 547330000
	response := gin.H{
		"result": strconv.Itoa(circulatingSupply),
	}
	ctx.JSON(http.StatusOK, response)
}

// DEX2 - NOTE: BROKER ID IS HARDCODED TO 1
func GetStakingInfo(ctx *gin.Context) {
	address := ctx.Query("address")

	checksumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checksumAddress + "_1"

	subaccountHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount ID to hex: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	spotBalance, _, _, err := xclient.GlobalBalanceClient.GetSpotBalance(subaccountHex)
	if err != nil {
		xlog.Errorf("Error fetching spot balance for %s: %v", subaccountHex, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staking info"})
		return
	}

	stakedAmount := "0"
	for _, balance := range spotBalance {
		if balance.ProductId == 2 {
			stakedAmount = balance.TokenBalance
			break
		}
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"userAddress":  checksumAddress,
			"stakedAmount": stakedAmount,
		},
		"message": "Success",
	})
}

func GetSpotBalances(ctx *gin.Context) {
	address := ctx.Query("address")

	checksumAddress := common.HexToAddress(address).Hex()
	subaccountId := "1_" + checksumAddress + "_1"

	subaccountHex, err := cutils.SubaccountIdToHex(subaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount ID to hex: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subaccount ID"})
		return
	}

	spotBalance, _, _, err := xclient.GlobalBalanceClient.GetSpotBalance(subaccountHex)
	if err != nil {
		xlog.Errorf("Error fetching spot balance for %s: %v", subaccountHex, err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to spot balances info"})
		return
	}

	productBalances := make(map[uint32]string)
	for _, balance := range spotBalance {
		productBalances[balance.ProductId] = balance.TokenBalance
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"userAddress":     checksumAddress,
			"productBalances": productBalances,
		},
		"message": "Success",
	})
}

func EspressoCampaignHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing address query parameter"})
		return
	}

	startTime := time.Date(2025, 5, 18, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user volume"})
		return
	}

	threshold := big.NewFloat(100)
	if userVolume.Cmp(threshold) >= 0 {
		ctx.JSON(http.StatusOK, gin.H{"status": "success"})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"status": "failed"})
	}
}

func UserVolumeHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when Kroma campaign starts
	startTime := time.Date(2024, 10, 15, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	returnObj := gin.H{
		"data": gin.H{
			"userAddress": address,
			"userVolume":  userVolume,
			"volumeAfter": startTime,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}

func OPQuestHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when the campaign starts
	startTime := time.Date(2024, 10, 22, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	depositCount, err := services.CheckIfDepositPresentFromChain(address, uint64(contractUtils.OPTIMISM))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	returnObj := gin.H{
		"data": gin.H{
			"userAddress": address,
			"userVolume":  userVolume,
			"volumeAfter": startTime,
			"opDeposit":   depositCount,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}
func RariQuestHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when the campaign starts
	startTime := time.Date(2024, 12, 2, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	depositCount, err := services.CheckIfDepositPresentFromChain(address, uint64(contractUtils.RARI))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	returnObj := gin.H{
		"data": gin.H{
			"userAddress": address,
			"userVolume":  userVolume,
			"volumeAfter": startTime,
			"rariDeposit": depositCount,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}
func MintQuestHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when the campaign starts
	startTime := time.Date(2024, 12, 8, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	depositCount, err := services.CheckIfDepositPresentFromChain(address, uint64(contractUtils.MINT))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	returnObj := gin.H{
		"data": gin.H{
			"userAddress": address,
			"userVolume":  userVolume,
			"volumeAfter": startTime,
			"mintDeposit": depositCount,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}
func OptionsQuestHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when the campaign starts
	startTime := time.Date(2025, 2, 26, 0, 0, 0, 0, time.UTC)
	userVolume, err := services.GetOptionsVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	returnObj := gin.H{
		"data": gin.H{
			"userAddress":       address,
			"userOptionsVolume": userVolume,
			"volumeAfter":       startTime,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}

func OptionsCountDailyHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	// Setting start to the time when the campaign starts
	count, err := services.GetOptionsCountToday(address)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	returnObj := gin.H{
		"data": gin.H{
			"userAddress":       address,
			"optionsTradeCount": count,
			"date":              time.Now().Format("2006-01-02"), // Formatting date as YYYY-MM-DD
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}

func TradeCountHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	tradeCount, err := services.TradeCountForAddress(address)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Create the return object
	returnObj := gin.H{
		"data": gin.H{
			"userAddress": address,
			"tradeCount":  tradeCount,
		},
		"message": "Success",
	}

	ctx.JSON(http.StatusOK, returnObj)
}

func ZealyQuestHandler(ctx *gin.Context) {
	reqBody, exists := ctx.Get("reqBody")
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Request body not found"})
		return
	}
	questClaim := reqBody.(dtos.ZealyQuestClaimDto)
	tradeCount, err := services.TradeCountForAddress(questClaim.Accounts.Wallet)

	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message":   "Internal server error",
			"requestId": questClaim.RequestId,
		})
		return
	}
	if tradeCount == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message":   "User has not traded with us yet: " + questClaim.Accounts.Wallet,
			"requestId": questClaim.RequestId,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User completed the action",
	})
}

func OstrichZealyVolumeChecker(ctx *gin.Context) {
	reqBody, exists := ctx.Get("reqBody")
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Request body not found"})
		return
	}
	questClaim := reqBody.(dtos.ZealyQuestClaimDto)
	startTime := time.Date(2025, 9, 12, 0, 0, 0, 0, time.UTC)
	tradeVolume, err := services.UserVolumeCheck(questClaim.Accounts.Wallet, startTime, 2)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message":   "Internal server error",
			"requestId": questClaim.RequestId,
		})
		return
	}
	threshold := big.NewFloat(100)
	if tradeVolume.Cmp(threshold) == -1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message":   "User has not completed $100 worth trade: " + questClaim.Accounts.Wallet,
			"requestId": questClaim.RequestId,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User completed the action",
	})
}

func InteractEverTradedHandler(c *gin.Context) {
	reqBody, _ := c.Get("reqBody")
	userContact := reqBody.(dtos.InteractDto)
	tradeCount, err := services.TradeCountForAddress(userContact.Address)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"error": gin.H{
				"code":    1,
				"message": err.Error(),
			},
			"data": gin.H{
				"result": false,
			},
		})
		return
	}

	result := tradeCount > 0
	c.JSON(http.StatusOK, gin.H{
		"error": nil,
		"data": gin.H{
			"result": result,
		},
	})
}

func calculateCollateralInvestedAndOrderType(quoteDelta *big.Int) (*big.Int, string) {

	// Calculate 5% of the absolute value of QuoteDelta
	absQuoteDelta := new(big.Int).Abs(quoteDelta)
	fivePercent := new(big.Int).Div(new(big.Int).Mul(absQuoteDelta, big.NewInt(5)), big.NewInt(100))

	// Determine order type based on the sign of QuoteDelta
	orderType := "SELL"
	if quoteDelta.Sign() < 0 {
		orderType = "BUY"
	}

	return fivePercent, orderType
}

func ZealyQuestVolumeChecker(ctx *gin.Context) {
	reqBody, exists := ctx.Get("reqBody")
	if !exists {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "Request body not found"})
		return
	}
	questClaim := reqBody.(dtos.ZealyQuestClaimDto)

	startTime := time.Date(2024, 11, 04, 0, 0, 0, 0, time.UTC)

	tradeVolume, err := services.UserVolumeCheck(questClaim.Accounts.Wallet, startTime, 1)
	if err != nil {
		xlog.Errorf("Rewards Controller - Error reading from db subaccountID: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"message":   "Internal server error",
			"requestId": questClaim.RequestId,
		})
		return
	}

	threshold := big.NewFloat(300)
	if tradeVolume.Cmp(threshold) == -1 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"message":   "User has not completed $300 worth trade: " + questClaim.Accounts.Wallet,
			"requestId": questClaim.RequestId,
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "User completed the action",
	})
}

func Micro3CampaignHandler(ctx *gin.Context) {
	address := ctx.Query("address")
	if address == "" {
		ctx.JSON(http.StatusOK, gin.H{
			"error": gin.H{
				"code":    1,
				"message": "Missing address query parameter",
			},
			"data": gin.H{
				"result": false,
			},
		})
		return
	}

	startTimeStr := os.Getenv("MICRO3_CAMPAIGN_START_DATE")
	var startTime time.Time
	if startTimeStr == "" {
		startTime = time.Date(2025, 6, 22, 0, 0, 0, 0, time.UTC)
	} else {
		var err error
		startTime, err = time.Parse("2006-01-02 15:04:05", startTimeStr)
		if err != nil {
			startTime = time.Date(2025, 6, 22, 0, 0, 0, 0, time.UTC)
			xlog.Errorf("Failed to parse MICRO3_CAMPAIGN_START_DATE: %v, using default", err)
		}
	}

	userVolume, err := services.GetUserVolumeAfterCertainDate(address, startTime)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"error": gin.H{
				"code":    2,
				"message": "Failed to fetch user volume: " + err.Error(),
			},
			"data": gin.H{
				"result": false,
			},
		})
		return
	}

	thresholdStr := os.Getenv("MICRO3_CAMPAIGN_VOLUME_THRESHOLD")
	var threshold *big.Float
	if thresholdStr == "" {
		threshold = big.NewFloat(50)
	} else {
		thresholdFloat, err := strconv.ParseFloat(thresholdStr, 64)
		if err != nil {
			threshold = big.NewFloat(50)
			xlog.Errorf("Failed to parse MICRO3_CAMPAIGN_VOLUME_THRESHOLD: %v, using default", err)
		} else {
			threshold = big.NewFloat(thresholdFloat)
		}
	}

	result := userVolume.Cmp(threshold) >= 0

	if result {
		ctx.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"result": true,
			},
		})
	} else {
		ctx.JSON(http.StatusOK, gin.H{
			"error": gin.H{
				"code":    3,
				"message": "User has not completed the required volume",
			},
			"data": gin.H{
				"result": false,
			},
		})
	}
}

func GalxeQuestVolumeHandler(ctx *gin.Context) {
	address := ctx.Query("address")

	returnObj := gin.H{
		"data": gin.H{
			"totalVolume": -1,
		},
		"message": "",
	}

	if address == "" {
		returnObj["message"] = "Address query param is required"
		ctx.JSON(http.StatusBadRequest, returnObj)
		return
	}

	// Validate and normalize the address
	checksumAddress := common.HexToAddress(address).Hex()

	startTime := time.Date(2025, 9, 18, 16, 30, 0, 0, time.UTC)

	userVolume, err := services.GetUserVolumeAfterCertainDate(checksumAddress, startTime)
	if err != nil {
		xlog.Errorf("Error fetching volume for Galxe quest: %v", err)
		returnObj["message"] = "Server Error"
		ctx.JSON(http.StatusInternalServerError, returnObj)
		return
	}

	returnObj["data"] = gin.H{
		"totalVolume": userVolume,
	}
	returnObj["message"] = "Success"

	ctx.JSON(http.StatusOK, returnObj)
}

func RocketXTransactionHandler(ctx *gin.Context) {
	address := ctx.Query("address")

	returnObj := gin.H{
		"data": gin.H{
			"hasTransactions": false,
		},
		"message": "",
	}

	if address == "" {
		returnObj["message"] = "Address query param is required"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}

	// Validate and normalize the address
	checksumAddress := common.HexToAddress(address).Hex()

	// Build RocketX API request
	apiURL := fmt.Sprintf("https://api.rocketx.exchange/rocketx/v1/transactionHistory/%s", checksumAddress)

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		returnObj["message"] = "Failed to create request"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}

	// Add API key header
	apiKey := os.Getenv("ROCKETX_API_KEY")
	if apiKey == "" {
		returnObj["message"] = "API configuration error"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}
	req.Header.Set("x-api", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		returnObj["message"] = "Failed to fetch data"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		returnObj["message"] = "External service error"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		returnObj["message"] = "Failed to process response"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}

	var transactions []interface{}
	if err := json.Unmarshal(body, &transactions); err != nil {
		returnObj["message"] = "Failed to parse data"
		ctx.JSON(http.StatusOK, returnObj)
		return
	}

	returnObj["data"] = gin.H{
		"hasTransactions": len(transactions) > 0,
	}
	returnObj["message"] = "Success"

	ctx.JSON(http.StatusOK, returnObj)
}
