package testutils

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/api"
	"github/eugenix-io/logx-inf-backend/services/balance-server/controller"
	"github/eugenix-io/logx-inf-backend/testutils/mocks"
	"github/eugenix-io/logx-inf-backend/xclient"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
)

func InitBalanceServer(t *testing.T) (func(), *gomonkey.Patches) {
	SetupBalanceMainnetEnv()

	SetupDBEnv(t)
	db.Init()

	contract.Init()

	// Init discord client
	xclient.InitDiscordClient()
	// Init oracle client
	xclient.InitOracleClient()

	patches := gomonkey.NewPatches()

	// Oracle patch
	patches.ApplyFunc((*appstate.AppStateImp).GetAllOraclePrices, func(a appstate.AppState, _ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
		return mocks.GetMockOraclePrices(), nil
	})

	// Funding patch
	patches.ApplyFuncReturn(xredis.GetCumulativeFundingRateForSymbol, cutils.GetBig0(), nil)

	// Initialize Gin router
	router := gin.Default()
	api.RegisterRoutes(router, controller.NewBalanceController())

	// Start API server
	port := os.Getenv("PORT")

	// Define the server and set its properties
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	closeServer := func() {
		xlog.Infof("Closing Balance Server")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Fatalf("Server close failed: %+v", err)
		} else {
			xlog.Infof("Balance Server closed")
		}
	}

	return closeServer, patches
}
