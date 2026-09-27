package main

import (
	"context"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/liquidation/api"
	"github/eugenix-io/logx-inf-backend/services/liquidation/initiator"
	"github/eugenix-io/logx-inf-backend/services/liquidation/stats"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		xlog.Warnf("Error loading .env file: %s", err)
	}
	contractUtils.Init()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Channel to receive signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	var wg sync.WaitGroup

	stats.Init()
	stats.InitFundsMonitorStats()
	
	// Set up the Gin router
	router := gin.Default()
	api.RegisterRoutes(router)

	// Start the server for health checks
	port := ":8085"
	if os.Getenv("PORT") != "" {
		port = ":" + os.Getenv("PORT")
	}
	go func() {
		if err := router.Run(port); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	type NOP struct{}

	// Acquire lock for 1 year | This is to ensure that only one instance of the liquidation service is running
	lockTime := time.Hour * 24 * 365
	retryDelay := time.Minute * 1
	maxRetries := 10
	lockOptions := xredis.LockOptions{
		LockExpiry: &lockTime,
		MaxRetries: &maxRetries,
		RetryDelay: &retryDelay,
	}

	xlog.Infof("Starting liquidation service and acquiring lock")
	_, err = xredis.WithRedisLock(xredis.GetLiquidationEngineLockKey(), func() (*NOP, error) {
		defer func() {
			if r := recover(); r != nil {
				xlog.Warnf("Recovered from panic: %+v", r)
			}
		}()

		var taskRunner initiator.TaskRunner
		liquidationStopped := os.Getenv("STOP_SERVICE") == "1"
		if liquidationStopped {
			xlog.Warnf("Liquidation service is stopped")
		} else {
			// Init connection to database
			db.Init()

			// Init Oracle client
			xclient.InitOracleClient()
			xclient.InitApiServerClient()
			xclient.InitDiscordClient()

			taskRunner = initiator.NewTaskRunner()
			if err := taskRunner.FetchCurrentStateFromRedis(); err != nil {
				liquidationStopped = true
				xlog.Errorf("Cannot run liquidation service: %v", err)
			} else {
				stats.GlobalLiquidationStats.UpdateRunningStatus(true)
				xlog.Infof("Running all jobs for liquidation")
				taskRunner.RunAllJobs(ctx, &wg)
			}
		}

		<-quit

		stats.GlobalLiquidationStats.UpdateRunningStatus(false)
		doneJobs := make(chan struct{})
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !liquidationStopped {
				// Prioritizing writing the current state to redis before stopping the service
				// NOTE: If this doesn't work then we will sync state every hour
				taskRunner.WriteCurrentStateToRedis()
			}
		}()

		go func() {
			cancel()
			wg.Wait()
			close(doneJobs)
		}()

		// Wait for both doneWritingState and doneJobs to finish
		select {
		case <-doneJobs:
			xlog.Infof("Closing all pending jobs and Done writing state to redis")
		case <-time.After(time.Second * 8): // Cloud run waits for 10 seconds to stop the container, so, we are waiting for 8 seconds and then we'll force exit
			xlog.Warnf("Forcing exit")
		}

		return &NOP{}, nil
	}, lockOptions)

	if err != nil {
		<-quit
	}

	xlog.Infof("Exiting liquidation service")
}
