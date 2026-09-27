package controller

import (
	"encoding/json"
	"net/http"

	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/gin-gonic/gin"
)

type AirdropController struct{}

func RegisterAirdropController(rg *gin.RouterGroup) {
	ac := &AirdropController{}
	rg.GET("/airdrop-allocation/:address", ac.GetAirdropAllocation)
}

func (ac *AirdropController) GetAirdropAllocation(ctx *gin.Context) {
	address := ctx.Param("address")
	if address == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "address parameter is required")
		return
	}

	// Validate address format
	if len(address) != 42 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid address format")
		return
	}

	airdropDB := &db.AirdropAllocationDB{}
	allocation, err := airdropDB.GetByAddress(address)
	if err != nil {
		xlog.Errorf("Airdrop Controller - Failed to get allocation for address %s: %v", address, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to retrieve allocation")
		return
	}

	// Define empty data
	emptyData := map[string]int{
		"POINTS_PROGRAM":        0,
		"SAVE_THE_OSTRICH":      0,
		"BUG_BOUNTY":            0,
		"COMMUNITY_CONTRIBUTOR": 0,
		"AFFILIATE_PROGRAM":     0,
		"STRONG_PARTNERS":       0,
		"ARBITRUM_COMMUNITY":    0,
		"TRILLION_HOLDERS":      0,
	}

	// If no allocation found, return allocation as 0 and empty data
	if allocation == nil {
		response := gin.H{
			"allocation": 0,
			"data":       emptyData,
		}
		ctx.JSON(http.StatusOK, response)
		return
	}

	normalizedAllocation := cutils.Divx18(allocation.Allocation.Val)

	response := gin.H{
		"allocation": normalizedAllocation,
	}

	// If data exists, parse and include it in the response
	if allocation.Data != "" {
		var parsedData map[string]int
		err := json.Unmarshal([]byte(allocation.Data), &parsedData)
		if err != nil {
			xlog.Errorf("Airdrop Controller - Failed to parse airdrop data for address %s: %v", address, err)
			response["data"] = emptyData
		} else {
			for key, value := range emptyData {
				if _, exists := parsedData[key]; !exists {
					parsedData[key] = value
				}
			}
			response["data"] = parsedData
		}
	} else {
		// If no data exists, return empty data object
		response["data"] = emptyData
	}

	ctx.JSON(http.StatusOK, response)
}
