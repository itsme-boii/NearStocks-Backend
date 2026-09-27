package main

import (
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/amm/config"
	"github/eugenix-io/logx-inf-backend/services/amm/controller"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type NOP struct{}

func main() {
	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		xlog.Debugf("Error loading .env file: %s", err)
	}

	if os.Getenv("STOP_AMM") == "1" {
		xlog.Warnf("AMM is stopped")
		return
	}
	xlog.Infof("Initialising AMM service...")
	contractUtils.Init()
	config.Init()
	db.Init()

	xclient.InitDiscordClient()
	xclient.InitOracleClient()

	router := gin.Default()

	xlog.Infof("Initializing Global Variables...")
	controller.InitializeGlobalVariables()

	xlog.Infof("Starting AMM service and acquiring lock with a delay of 10 seconds...")
	time.Sleep(10 * time.Second)

	_, err = xredis.WithRedisLock(xredis.GetAMMLockKey(), func() (*NOP, error) {

		xlog.Infof("Starting Tickers ...")
		go controller.StartPriceCron()

		stopAmmQuotes := os.Getenv("STOP_AMM_QUOTES")
		if stopAmmQuotes != "1" {
			// Wait a bit to let other routines initialize.
			time.Sleep(5 * time.Second)
			xlog.Infof("Sending Quotes ...")
			go cerrors.WithPanicRecover(controller.RfqClient)
		} else {
			xlog.Errorf("AMM Quotes - Not sending AMM quotes because of env variable")
			xclient.GlobalDiscordClient.SendWebhookMessage("AMM Quotes - Not sending AMM quotes because of env variable")
		}

		port := ":8080"
		if os.Getenv("PORT") != "" {
			port = ":" + os.Getenv("PORT")
		}

		// Set up channel to listen for termination signals.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			if err := router.Run(port); err != nil {
				xlog.Errorf("Failed to run server: %v", err)
			}
		}()

		// Block here until a termination signal is received.
		<-quit
		xlog.Infof("Shutting down AMM service...")

		return &NOP{}, nil
	}, xredis.LockOptions{
		LockExpiry: cutils.Ptr(time.Hour * 24 * 365),
		MaxRetries: cutils.Ptr(100),
		RetryDelay: cutils.Ptr(5 * time.Second),
	})

	// If acquiring the lock failed, log the error.
	if err != nil {
		msg := "AMM service is already running or failed to acquire lock"
		xlog.Errorf(err.Error())
		xclient.GlobalDiscordClient.SendWebhookMessage(msg)
	}

	xlog.Infof("Exiting AMM service")
}
