package main

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/services/cron-server/api"
	"github/eugenix-io/logx-inf-backend/services/cron-server/dashboard"
	"github/eugenix-io/logx-inf-backend/services/cron-server/earning"
	"github/eugenix-io/logx-inf-backend/services/cron-server/funding"
	"github/eugenix-io/logx-inf-backend/services/cron-server/nearbatch"
	"github/eugenix-io/logx-inf-backend/services/cron-server/options"
	"github/eugenix-io/logx-inf-backend/services/cron-server/pnl"
	"github/eugenix-io/logx-inf-backend/services/cron-server/pruning"
	referralCron "github/eugenix-io/logx-inf-backend/services/cron-server/referral"
	"github/eugenix-io/logx-inf-backend/services/cron-server/relayer"
	"github/eugenix-io/logx-inf-backend/services/cron-server/sync"
	"github/eugenix-io/logx-inf-backend/services/cron-server/unrealisedpnl"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github/eugenix-io/logx-inf-backend/services/cron-server/governance"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
	//Initialise the database
	db.Init()
	contract.Init()

	// Init External Clients
	xclient.InitBalanceClient()
	xclient.InitDiscordClient()
	xclient.InitApiServerClient()
	xclient.InitOracleClient()

	router := gin.Default()

	// Register routes
	api.RegisterRoutes(router)

	go pnl.StartDailyPnlCron()

	// Start the funding CRON job
	stopFundingEngine := os.Getenv("STOP_FUNDING_ENGINE")
	if stopFundingEngine != "1" {
		funding.InitialiseFundingRates()
		go funding.StartFundingCron()
	} else {
		xlog.Infof("Cron Server - not starting funding engine because of env variable")
	}

	stopExternalFundingEngine := os.Getenv("STOP_EXTERNAL_FUNDING_ENGINE")
	if stopExternalFundingEngine != "1" {
		funding.InitialiseExternalFundingRates()
		go withPanicRecover(funding.StartExternalFundingRateCron)
	} else {
		xlog.Infof("Cron Server - not starting external funding engine because of env variable")
	}

	stopRelayerBalanceCheck := os.Getenv("STOP_RELAYER_BALANCE_CHECK")
	if stopRelayerBalanceCheck != "1" {
		go relayer.RelayerBalanceCheckCron()
	} else {
		xlog.Infof("Cron Server - not starting relayer balance check because of env variable")
	}
	stopEarningsEngine := os.Getenv("STOP_EARNINGS_ENGINE")
	if stopEarningsEngine != "1" {
		go earning.StartEarningCron()
	} else {
		xlog.Infof("Cron Server - not starting earnings engine because of env variable")
	}

	// start referral cron
	stopReferralCron := os.Getenv("STOP_REFERRAL_CRON")
	if stopReferralCron != "1" {
		go referralCron.StartReferralSyncCron()
	} else {
		xlog.Infof("Cron Server - not starting referral cron because of env variable")
	}

	// NEAR settlement (Development.md §8.3): batch sender, price tick and prune job
	if contractUtils.NearSettlement() && os.Getenv("STOP_BATCHING_SERVICE") != "1" {
		if err := nearbatch.Start(context.Background()); err != nil {
			xlog.Errorf("Cron Server - NEAR jobs not started: %v", err)
		} else {
			xlog.Infof("Cron Server - NEAR batch sender, price tick and prune jobs started")
		}
	}

	// Start batching CRON job (appchain; stays disabled)
	// stopBatchingService := os.Getenv("STOP_BATCHING_SERVICE")
	// if stopBatchingService != "1" {
	// 	go batching.StartBatchingCron()
	// } else {
	// 	xlog.Infof("Cron Server - not starting batching service because of env variable")
	// }

	// Start lottery CRON job
	// stopLotteryService := os.Getenv("STOP_LOTTERY_SERVICE")
	// if stopLotteryService != "1" {
	// 	go lottery.StartLotteryFlowCron()
	// } else {
	// 	xlog.Infof("Cron Server - not starting lottery service because of env variable")
	// }

	stopAmmHealthCheck := os.Getenv("STOP_AMM_HEALTH_CHECK")
	if stopAmmHealthCheck != "1" {
		go sync.CheckAmmHealth()
	} else {
		xlog.Infof("Cron Server - not starting AMM health check because of env variable")
	}
	// go sync.StartSyncCron()

	go dashboard.StartDashboardCron()

	stopUnrealisedPnlCron := os.Getenv("STOP_UNREALISED_PNL_SERVICE")
	if stopUnrealisedPnlCron != "1" {
		unrealisedpnl.InitializeUnrealisedPnlCron()
		go unrealisedpnl.StartUnrealisedPnlCron()
	} else {
		xlog.Infof("Cron Server - not starting unrealised pnl cron because of env variable")
	}

	stopOrderPruneCron := os.Getenv("STOP_ORDER_PRUNE_SERVICE")
	if stopOrderPruneCron != "1" {
		go withPanicRecover(func() { pruning.StartOrderPruneCron() })
	} else {
		xlog.Infof("Cron Server - not starting order prune cron because of env variable")
	}

	stopOptionsJobs := os.Getenv("STOP_OPTIONS_JOBS")
	if stopOptionsJobs != "1" {
		go withPanicRecover(func() { options.StartOptionsProcessingCron() })
	} else {
		xlog.Infof("Cron Server - not starting options jobs cron because of env variable")
	}

	stopGovernanceCron := os.Getenv("STOP_GOVERNANCE_CRON")
	if stopGovernanceCron != "1" {
		go governance.StartGovernanceCron()
	} else {
		xlog.Infof("Cron Server - not starting governance cron because of env variable")
	}

	port := ":8080"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}
	// Start the server
	router.Run(port)

	// Wait for an interrupt signal to gracefully shut down the service
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("Shutting down the service gracefully...")
}
