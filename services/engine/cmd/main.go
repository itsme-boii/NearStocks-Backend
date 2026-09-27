package main

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/engine"
	"github/eugenix-io/logx-inf-backend/services/engine/api"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

// NOTE: If we are updating anything here ideally you should also testutils/engine.testutils.go file
func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		xlog.Errorf("Error loading .env file: %s", err)
	}

	xlog.Infof("Starting Engine server")
	contractUtils.Init()
	// Init db
	db.Init()

	// Init clients
	xclient.InitOracleClient()
	xclient.InitApiServerClient()
	xclient.InitBalanceClient()
	xclient.InitDiscordClient()

	// Init engine
	engine.Init()

	xlog.Infof("Engine starting with a delay of 10 seconds...")
	time.Sleep(10 * time.Second)

	xlog.Infof("Engine acquiring Redis lock...")
	// Acquire Redis lock to ensure only one instance of the server is running
	_, err = xredis.WithRedisLock(xredis.GetMatchingEngineLockKey(), func() (*xredis.NOOP, error) {
		xlog.Infof("Engine acquired Redis lock")
		// Init cron
		if os.Getenv("STOP_TRIGGER_ORDER_CRON") != "1" {
			xlog.Infof("Starting trigger order cron")
			cerrors.WithPanicRecover(engine.StartTriggerOrderCron)
		}

		// Start API server
		port := ":8081"
		if os.Getenv("PORT") != "" {
			port = ":" + os.Getenv("PORT")
		}

		server := api.NewApiServer(port)
		go func() {
			if err := server.Run(); err != nil {
				xlog.Fatalf("Failed to run server: %v", err)
			}
		}()

		xlog.Infof("Engine server started on port %s", port)
		xlog.Infof("Engine server is running...")

		// Create a channel to handle shutdown signals
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Block until we receive a shutdown signal
		sig := <-sigChan
		xlog.Infof("Received signal %v, shutting down...", sig)
		return nil, nil
	}, xredis.LockOptions{
		LockExpiry: cutils.Ptr(time.Hour * 24 * 365),
		MaxRetries: cutils.Ptr(100),
		RetryDelay: cutils.Ptr(5 * time.Second),
	})

	if err != nil {
		msg := fmt.Sprintf("Something went wrong. Stopping engine server: %v", err)
		xlog.Errorf(msg)
		xclient.GetGlobalDiscordClient().SendWebhookMessage(msg)
	}
}
