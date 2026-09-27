package main

import (
	"fmt"
	"log"
	"os"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/services/balance-server/api"
	"github/eugenix-io/logx-inf-backend/services/balance-server/controller"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// If you are making changes to the main function, you should also update testutils/balance.testutils.go file
func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file: %s", err)
	}

	// Get AMM subaccount ID
	ammSubaccountID := os.Getenv("AMM_SUBACCOUNT_ID")
	if ammSubaccountID == "" {
		log.Fatal("AMM_SUBACCOUNT_ID environment variable not set")
	}
	contractUtils.Init()

	db.Init()
	contract.Init()

	// Create balance controller
	balanceController := controller.NewBalanceController()
	// Init discord client
	xclient.InitDiscordClient()
	// Init oracle client
	xclient.InitOracleClient()

	// Initialize Gin router
	router := gin.Default()

	// Register routes
	api.RegisterRoutes(router, balanceController)

	// Start API server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	err = router.Run(fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}
}
