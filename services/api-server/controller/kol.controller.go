package controller

import (
	"context"
	"net/http"
	"strings"

	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type KolController struct {
	redisClient *redis.Client
}

func RegisterKolController(rg *gin.RouterGroup) {
	kc := &KolController{
		redisClient: xredis.GetRedisClient(),
	}
	rg.GET("/kol-allocation/:username", kc.GetKolAllocation)
}

func (kc *KolController) GetKolAllocation(ctx *gin.Context) {
	username := ctx.Param("username")
	if username == "" {
		cutils.ApiAbort(ctx, http.StatusBadRequest, "username parameter is required")
		return
	}

	// convert username to lowercase
	username = strings.ToLower(username)

	// Get allocation from Redis hash
	hashKey := xredis.GetKolAllocationHashKey()
	allocation, err := kc.redisClient.HGet(context.Background(), hashKey, username).Result()

	if err != nil {
		if err == redis.Nil {
			// User not found, return 0 allocation
			response := gin.H{
				"username":   username,
				"allocation": "0",
			}
			ctx.JSON(http.StatusOK, response)
			return
		}
		xlog.Errorf("Kol Controller - Failed to get allocation for username %s: %v", username, err)
		cutils.ApiAbort(ctx, http.StatusInternalServerError, "failed to retrieve allocation")
		return
	}

	response := gin.H{
		"username":   username,
		"allocation": allocation,
	}

	ctx.JSON(http.StatusOK, response)
}
