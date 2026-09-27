package api

import (
	"github/eugenix-io/logx-inf-backend/services/cron-server/governance"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine) {
	router.GET("/health", HealthCheck)

	balanceGroup := router.Group("/hello")
	{
		balanceGroup.GET("/", Hello)
	}

	governance.RegisterGovernanceRoutes(router)
}

func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// Define the handler function for the /funding endpoint
func Hello(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!")
}
