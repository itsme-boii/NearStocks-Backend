package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AddressController struct {
}

func RegisterAddressController(
	r *gin.RouterGroup,
) {
	addressController := AddressController{}
	rg := r.Group("/address")
	// Private Endpoints

	// Public Endpoints
	rg.GET("/:ethAddress/subaccounts", addressController.GetSubaccountList)
}

type AddressGetSubaccountListUri struct {
	EthAddress string `uri:"ethAddress" binding:"required"`
}
type AddressGetSubaccountListHeader struct {
	BrokerId uint `header:"Broker-Id" binding:"required"`
}
type AddressGetSubaccountListRequest struct {
	AddressGetSubaccountListUri
	AddressGetSubaccountListHeader
}

// 1. Fetch all subaccount uids
// 2. Count the numbers and generate next subaccount uuid
func (*AddressController) GetSubaccountList(ctx *gin.Context) {
	var requestUri AddressGetSubaccountListUri
	var requestHeader AddressGetSubaccountListHeader
	if err := ctx.BindUri(&requestUri); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if err := ctx.BindHeader(&requestHeader); err != nil {
		ctx.AbortWithError(http.StatusBadRequest, err)
		return
	}
	requestBody := AddressGetSubaccountListRequest{
		AddressGetSubaccountListUri:    requestUri,
		AddressGetSubaccountListHeader: requestHeader,
	}

	subacountIds := (&db.SubaccountDB{}).GetIDsByBrokerAddress(requestBody.BrokerId, requestBody.EthAddress)
	nextSubacountId := cutils.CreateSubaccountId(requestBody.BrokerId, requestBody.EthAddress, len(*subacountIds)+1)
	cutils.ApiResponse(ctx, gin.H{"subaccounts": subacountIds, "nextSubaccountId": nextSubacountId}, http.StatusOK, "")
}
