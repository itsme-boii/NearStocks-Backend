package routes

import (
	"services/external-campaigns/controllers"
	"services/external-campaigns/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(router *gin.Engine) {
	campaignsController := router.Group("/campaigns")
	{
		campaignsController.GET("/totalSupply", controllers.TotalSupplyHandler)
		campaignsController.GET("/circulatingSupply", controllers.CirculatingSupplyHandler)
		campaignsController.GET("/circulatingSupplyGecko", controllers.CirculatingSupplyHandlerGecko)
		campaignsController.GET("/referralCount", middlewares.ValidateAddressQueryParam(), controllers.ReferralCountHandler)
		campaignsController.GET("/userVolume", middlewares.ValidateAddressQueryParam(), controllers.UserVolumeHandler)
		campaignsController.GET("/userSpotbalance", middlewares.ValidateAddressQueryParam(), controllers.GetSpotBalances)
		campaignsController.GET("/espressoCampaign", middlewares.ValidateAddressQueryParam(), controllers.EspressoCampaignHandler)
		campaignsController.GET("/userStakedInfo", middlewares.ValidateAddressQueryParam(), controllers.GetStakingInfo)
		campaignsController.GET("/opQuest", middlewares.ValidateAddressQueryParam(), controllers.OPQuestHandler)
		campaignsController.GET("/rariQuest", middlewares.ValidateAddressQueryParam(), controllers.RariQuestHandler)
		campaignsController.GET("/mintQuest", middlewares.ValidateAddressQueryParam(), controllers.MintQuestHandler)
		campaignsController.GET("/optionsGalxe", middlewares.ValidateAddressQueryParam(), controllers.OptionsQuestHandler)
		campaignsController.GET("/tradedOptionsToday", middlewares.ValidateAddressQueryParam(), controllers.OptionsCountDailyHandler)
		campaignsController.GET("/tradeCount", middlewares.ValidateAddressQueryParam(), controllers.TradeCountHandler)
		campaignsController.GET("/micro3Campaign", middlewares.ValidateAddressQueryParam(), controllers.Micro3CampaignHandler)
		campaignsController.GET("/galxe/volumeQuest", middlewares.ValidateAddressQueryParam(), controllers.GalxeQuestVolumeHandler)
		campaignsController.GET("/galxe/rocketxTxnCheck", middlewares.ValidateAddressQueryParam(), controllers.RocketXTransactionHandler)
		campaignsController.POST("/zealyQuest", middlewares.ValidateZealyClaimDto(), controllers.ZealyQuestHandler)
		campaignsController.POST("/zealyQuestVolumeCheck", middlewares.ValidateZealyClaimDto(), controllers.OstrichZealyVolumeChecker)
		campaignsController.POST("/volumeQuestZealy", middlewares.ValidateZealyClaimDto(), controllers.ZealyQuestVolumeChecker)
		campaignsController.POST("/interact/everTraded", middlewares.ValidateInteractContactDto(), controllers.InteractEverTradedHandler)
		campaignsController.POST("/posttodiscord", middlewares.ValidateDiscordMessageRequestBody(), controllers.SendMessageToDiscordHandler)
	}
}
