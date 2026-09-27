package main

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/services/indexer-server/api"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/services/indexer-server/indexer"
	"github/eugenix-io/logx-inf-backend/services/indexer-server/nearindexer"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file: %s", err)
	}
	contractUtils.Init()

	// Initialize the database
	db.Init()
	contract.Init()

	// Initialize external clients
	xclient.InitDiscordClient()
	xclient.InitApiServerClient()

	// NEAR settlement (Development.md §8.4): mirror near-stocks.near deposits, withdrawal outcomes
	// and fee sweeps into the balance-server
	if contractUtils.NearSettlement() && os.Getenv("STOP_NEAR_INDEXER") != "1" {
		xclient.InitBalanceClient()
		go cerrors.WithPanicRecover(func() {
			nearindexer.Start(context.Background())
		})
	}

	if os.Getenv("STOP_WITHDRAWAL_FINALIZER") != "1" {
		// Initialize the withdrawals finalizer
		go cerrors.WithPanicRecover(func() {
			finalizer := indexer.NewWithdrawalsFinalizer()
			finalizer.Init()
			// Start the withdrawals finalizer in a separate goroutine
			finalizer.Run()
		})
	}

	if os.Getenv("STOP_DEPOSIT_SOURCE_TRACKER") != "1" {
		// Initialize the deposits tracker
		go cerrors.WithPanicRecover(func() {
			depositsTracker := indexer.NewDepositSourceTracker()
			depositsTracker.Init()
			// Start the deposits tracker in a separate goroutine
			depositsTracker.Run()
		})
	}

	// Set up the Gin router
	router := gin.Default()
	api.RegisterRoutes(router)

	// Start the server
	port := ":8080"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}
	go func() {
		if err := router.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	// Wait for an interrupt signal to gracefully shut down the service
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down the service gracefully...")
}
