package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/health", HealthCheck)

	balanceGroup := router.Group("/amm")
	{
		balanceGroup.GET("/", AmmHandler)
	}
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Define the handler function for the /funding endpoint
func AmmHandler(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!")
}
