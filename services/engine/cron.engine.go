package engine

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"strconv"

	"github.com/robfig/cron/v3"
)

func StartTriggerOrderCron() {
	c := cron.New(cron.WithSeconds())
	appState := appstate.NewAppState()

	triggerCron := os.Getenv("TRIGGER_CRON")
	if triggerCron == "" {
		triggerCron = "*/10 * * * * *"
	}

	orderLimit := int64(5)
	_orderLimit := os.Getenv("TRIGGER_ORDER_LIMIT")
	if _orderLimit != "" {
		var err error
		orderLimit, err = strconv.ParseInt(_orderLimit, 10, 64)
		if err != nil {
			xlog.Warnf("Error while parsing TRIGGER_ORDER_LIMIT. Using default value")
		}
	} else {
		xlog.Warnf("Couldn't parse TRIGGER_ORDER_LIMIT. Using default value")
	}

	c.AddFunc(triggerCron, func() {
		cerrors.WithPanicRecover(
			func() {
				xlog.Infof("Executing trigger order cron")
				oraclePrices, err := appState.GetAllOraclePrices()
				if err != nil {
					xlog.Errorf("Error while fetching oracle prices. Failed with error: %v", err)
					return
				}

				triggerOrderbooks := GlobalEngine.GetTriggerOrderbooks()
				for marketId, tob := range triggerOrderbooks {
					symbol, ok := marketutils.GetBaseSymbolForProduct(uint32(marketId))
					if !ok {
						xlog.Errorf("Oracle symbol not found for product id: %d. This means there is a bug in contractUtils.PRODUCT_ID_SYMBOL_TO_MAP", marketId)
						continue
					}

					oraclePrice, exists := oraclePrices[symbol]
					if !exists {
						xlog.Errorf("Oracle price not found for product: %v", marketId)
						xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Trigger order - Oracle price not found for product: %v", marketId))
						continue
					}

					go cerrors.WithPanicRecover(func() { tob.ExecuteTriggerOrders(oraclePrice.Pricex18, orderLimit) })
				}
			},
		)
	})

	c.Start()
	xlog.Infof("Trigger order cron started")
	// NOTE: Check if select {} is required here
}
