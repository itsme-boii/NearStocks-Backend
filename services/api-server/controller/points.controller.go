package controller

import (
	"net/http"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/gin-gonic/gin"
)

type PointsController struct{}

// RegisterPointsController registers the points controller routes
func RegisterPointsController(rg *gin.RouterGroup) {
	pc := &PointsController{}

	// GET /api/v1/points/{address} - Get points for a specific address
	rg.GET("/points/:address", pc.GetPointsByAddress)
}

// GetPointsByAddress retrieves all points for a specific address
func (pc *PointsController) GetPointsByAddress(ctx *gin.Context) {
	address := ctx.Param("address")
	if address == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "address parameter is required")
		return
	}

	// Validate address format (basic validation)
	if len(address) < 10 {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "invalid address format")
		return
	}

	pointsDB := &db.PointsDB{}
	points, err := pointsDB.GetPointsByAddress(address)
	if err != nil {
		xlog.Errorf("Points Controller - Failed to get points for address %s: %v", address, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to retrieve points")
		return
	}

	// Calculate total points from the points array
	totalPoints := ctypes.NewBigIntFromString("0")
	for _, point := range points {
		totalPoints = totalPoints.Add(point.Points)
	}

	response := gin.H{
		"address":     address,
		"totalPoints": totalPoints,
		"points":      points,
	}

	ctx.JSON(http.StatusOK, response)
}
