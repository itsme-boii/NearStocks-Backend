package main

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"os"
	"runtime/debug"
	"services/external-campaigns/bobrewards"
	"services/external-campaigns/flagusercron"
	"services/external-campaigns/lastdeposit"
	"services/external-campaigns/pricecron"
	"services/external-campaigns/routes"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
)

func withPanicRecover(function func()) {
	defer func() {
		if r := recover(); r != nil {
			xlog.Errorf("Recovered from panic: %v | stack trace: %v", r, string(debug.Stack()))
			xclient.GlobalDiscordClient.SendWebhookMessage("Recovered from panic" + fmt.Sprintf("Recovered from panic: %v | stack trace: %v", r, string(debug.Stack())))
		}
	}()
	function()
}

func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file: %s", err)
	}
	contractUtils.Init()
	// Initiate DB
	db.Init()
	// CORS middleware configuration to allow all origins and headers
	corsConfig := cors.New(cors.Options{
		AllowedOrigins: cutils.CORSAllowedOrigins(),
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		//ToDo - Add only allowed headers
		AllowedHeaders:   []string{"*"}, // Allow all headers
		AllowCredentials: false,         // clients authenticate with headers, never cookies
	})
	stopDepositScanning := os.Getenv("STOP_DEPOSIT_SCANNING")
	if stopDepositScanning != "1" {
		withPanicRecover(func() {
			err := lastdeposit.InitializeLastDepositCheckpoint()
			if err != nil {
				xlog.Fatalf("Failed to initialize deposit scanning checkpoint: %v", err)
			}
			go lastdeposit.StartLastDepositCron()
		})

	} else {
		xlog.Infof("External Campaign Service started - not starting deposit scanning cron because of env variable")
	}
	stopDistributingBobRewards := os.Getenv(("STOP_DISTRIBUTING_BOB_REWARDS"))
	if stopDistributingBobRewards != "1" {
		withPanicRecover(func() {
			go bobrewards.StartRewardDistribution()
		})
	} else {
		xlog.Infof("External Campaign Service started - not distributing bob rewards because of env variable")
	}

	// Start price fetching cron job
	stopPriceFetching := os.Getenv("STOP_STORK_PRICE_FETCHING")
	if stopPriceFetching != "1" {
		withPanicRecover(func() {
			go pricecron.StartPriceCron()
		})
	} else {
		xlog.Infof("External Campaign Service started - not starting price fetching cron because of env variable")
	}

	xclient.InitDiscordClient()
	xclient.InitApiServerClient()
	xclient.InitBalanceClient()
	xclient.InitOracleClient()

	router := gin.Default()

	// Apply the CORS middleware to the router
	// Apply the CORS middleware to the router using gin's middleware functionality
	router.Use(func(c *gin.Context) {
		corsConfig.HandlerFunc(c.Writer, c.Request)
		c.Next()
	})
	routes.SetupRoutes(router)
	routes.SetupLeaderboardRoutes(router)
	withPanicRecover(func() {
		routes.PublicRoutesSetup(router)
	})
	go cerrors.WithPanicRecover(flagusercron.StartFlagUserCron)
	router.Run(":8080")
}
