package main

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/freya/services"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	// Load env vars
	err := godotenv.Load(".env")

	if err != nil {
		xlog.Debugf("Error loading .env file: %s", err)
	}

	if os.Getenv("STOP_FREYA") == "1" {
		xlog.Warnf("Freya is stopped")
		return
	}

	//Initalise the service
	xlog.Infof("Initialising ...")
	contractUtils.Init()
	db.Init()

	//Run the Hedging Bot
	hedgerService := &services.HedgerService{}
	// Initialize the service (starts the notification listener)
	hedgerService.InitializeHedgerService()

	// Start processing hedge positions in a separate goroutine
	go hedgerService.HedgePositions()

	select {} // Block main thread to keep the goroutines running
}
