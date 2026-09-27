package controller

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
)

type CantonAddressController struct {
}

func RegisterCantonAddressController(r *gin.RouterGroup) {
	cantonAddressController := CantonAddressController{}
	rg := r.Group("/canton-address")

	// Public Endpoints
	rg.POST("", cantonAddressController.CreateOrGetMapping)
}

type CreateCantonAddressRequest struct {
	CantonAddress string `json:"canton_address" binding:"required"`
}

type CantonAddressResponse struct {
	Address        string `json:"address"`
	CantonAddress  string `json:"canton_address"`
	SigningAddress string `json:"signing_address,omitempty"` // Signing key address for authentication
}

// CreateOrGetMapping accepts a Canton address and returns the EVM address (creates new wallet if not exists)
// Also creates subaccount for Canton users
func (*CantonAddressController) CreateOrGetMapping(ctx *gin.Context) {
	var request CreateCantonAddressRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid request body")
		return
	}

	xlog.Infof("Canton Address Controller - Processing request for Canton address: %s", request.CantonAddress)

	// Default broker ID (can be made configurable if needed)
	brokerId := uint(2)

	// Use the Canton service to get or create party details
	result, err := services.GetOrCreateCantonPartyDetails(request.CantonAddress, brokerId)
	if err != nil {
		xlog.Errorf("Canton Address Controller - Error getting/creating Canton party details: %v", err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to get or create Canton party details")
		return
	}

	// Convert stored address to checksummed format for response
	addressFromDB := common.HexToAddress(result.EvmAddress).Hex()

	response := CantonAddressResponse{
		Address:       addressFromDB,
		CantonAddress: request.CantonAddress,
	}

	xlog.Infof("Canton Address Controller - Successfully processed Canton address: %s, EVM Address: %s", request.CantonAddress, addressFromDB)
	cutils.ApiResponse(ctx, response, http.StatusOK, "")
}

