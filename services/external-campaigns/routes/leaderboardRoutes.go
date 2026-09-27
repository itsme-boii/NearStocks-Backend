package routes

import (
	"net/http"
	"services/external-campaigns/controllers"
	"services/external-campaigns/services"

	"github.com/gin-gonic/gin"
)

func SetupLeaderboardRoutes(router *gin.Engine) {
	rewardService := &services.RewardServices{}
	leaderboardController := controllers.NewLeaderboardController(rewardService)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	leaderboardGroup := router.Group("/leaderboard")
	{
		leaderboardGroup.GET("/pnl", leaderboardController.GetLeaderboard)
		leaderboardGroup.GET("/volume", leaderboardController.GetVolumeLeaderboard)
	}
	router.GET("/user-stats", leaderboardController.GetUserStats)
	router.GET("/referral-stats", leaderboardController.GetReferralStats)
}
