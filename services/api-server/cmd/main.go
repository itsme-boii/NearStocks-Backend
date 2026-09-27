package main

import (
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/api-server/api"
	"github/eugenix-io/logx-inf-backend/services/api-server/client"
	"github/eugenix-io/logx-inf-backend/xclient"

	"log"
	"os"

	"github/eugenix-io/logx-inf-backend/services/api-server/services"

	"github.com/joho/godotenv"
)

func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("Error loading .env file: %s", err)
	}

	xlog.Infof("Initialising api server...")
	contractUtils.Init()
	// Init smart contract connection
	contract.Init()
	// Init db
	db.Init()
	perputils.InitializeOpenInterest()

	// Init engine
	client.InitEngineClient()
	xclient.InitBalanceClient()
	xclient.InitOracleClient()
	xclient.InitLiquidationClient()
	xclient.InitDiscordClient()

	// NEAR 1Click deposit poller (Phase 2). Crediting is idempotent; enable on one or more instances.
	if os.Getenv("NEAR_INTENTS_POLLER") == "1" {
		if _, err := services.NearConfigFromEnv(); err != nil {
			log.Fatalf("NEAR_INTENTS_POLLER=1 but NEAR config is invalid: %v", err)
		}
		services.StartOneClickPoller()
	}

	// Init server
	server := api.NewApiServer(":8080")
	err = server.Run()

	if err != nil {
		log.Fatal(err)
	}
}
