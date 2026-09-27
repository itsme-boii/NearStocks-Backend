package controller

import (
	"encoding/json"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/types"
	balanceTypes "github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
)

type InsuranceController struct{}

func RegisterInsuranceController(r *gin.RouterGroup) {
	insuranceController := InsuranceController{}
	rg := r.Group("/insurance")

	// Private Endpoints
	rg.POST("/settle", insuranceController.SettleWithInsurance)
}

func (ic *InsuranceController) SettleWithInsurance(ctx *gin.Context) {
	var requestPayload types.SettleWithInsuranceRequest
	if err := ctx.BindJSON(&requestPayload); err != nil {
		xlog.Errorf("Failed to bind request payload with err: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid request payload")
		return
	}

	subaccountIdHex, err := cutils.SubaccountIdToHex(requestPayload.SubaccountId)
	if err != nil {
		xlog.Errorf("Error converting subaccount id to hex: %v", err)
		cutils.ApiAbort(ctx, http.StatusBadRequest, "Invalid subaccount id")
		return
	}

	// Get subaccount to check if it exists
	subaccount := (&db.SubaccountDB{}).GetById(requestPayload.SubaccountId)
	if subaccount == nil {
		xlog.Errorf("Subaccount not found with id: %v", requestPayload.SubaccountId)
		cutils.ApiAbort(ctx, http.StatusNotFound, "Subaccount not found")
		return
	}

	transactionCounter, err := transaction.IncrementCounter(1)
	if err != nil {
		xlog.Errorf("Error incrementing transaction counter: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error incrementing transaction counter")
		return
	}

	// Get oracle prices
	tokenPriceMap, err := xclient.GlobalOracleClient.GetAllCollateralTokenPrices()
	if err != nil {
		xlog.Errorf("Error getting oracle prices: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error getting oracle prices")
		return
	}

	// Move this to common utils
	oraclePricesx18 := make(map[string]*big.Int, len(tokenPriceMap))
	for symbol, price := range tokenPriceMap {
		oraclePricesx18[symbol] = cutils.ConvertXCustToX18(price.PriceX, *price.Expo)
	}

	// Create spot prices map
	spotPricesx18Map := marketutils.GetSpotPricesX18Map(oraclePricesx18, true)

	request := balanceTypes.SettleWithInsuranceRequest{
		SubaccountId:        subaccountIdHex,
		SpotOraclePricesX18: spotPricesx18Map,
	}

	// Update in balance server
	err = xclient.GlobalBalanceClient.SettleUsingInsuranceFunds(request)
	if err != nil {
		xlog.Errorf("Error settling negative balance of subaccount with insurance funds: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error settling subaccount with insurance funds")
		return
	}

	insuranceSpotPricesMap := make(map[uint32]*big.Int)
	for _, productId := range contractUtils.ALL_SPOTS_FOR_INSURANCE {
		insuranceSpotPricesMap[uint32(productId)] = spotPricesx18Map[uint32(productId)]
	}

	// Create product id slice and corresponding price slice
	productIds := cutils.MapKeys(insuranceSpotPricesMap)
	spotPrices := cutils.MapValues(insuranceSpotPricesMap)

	contractRequest := contract.SocialiseSubaccountRequest{
		SubaccountId:    subaccount.ID,
		ProductIds:      productIds,
		OraclePricesX18: spotPrices,
	}

	requestJson, err := json.Marshal(contractRequest)
	if err != nil {
		xlog.Errorf("Error marshalling contract request: %v", err)
	} else {
		xlog.Infof("Socialise subaccount request: %+v", string(requestJson))
	}

	// Update in contract
	err = contract.GlobalContracts.EndpointContract.SocialiseSubaccount(contractRequest, transactionCounter)
	if err != nil {
		xlog.Errorf("Error settling negative balance of subaccount with insurance funds: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "Error in batching txn to contract")
		return
	}

	cutils.ApiSuccess(ctx, nil, "Subaccount settled with insurance funds")
}
