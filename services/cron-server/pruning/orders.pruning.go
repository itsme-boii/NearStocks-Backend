package pruning

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/metric"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

func pruneOrders() {
	defer cutils.LogTime(time.Now(), metric.PRUNE_ORDERS)
	// Your pruning logic here
	xlog.Infof("Pruning unmatched amm orders")
	err := (&db.OrderDB{}).DeleteUnmatchedAMMOrders()
	if err != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Error pruning orders: err: %+v", err.Error()))
	}
	xlog.Infof("Pruning unmatched amm orders done")
}

func StartOrderPruneCron() {
	c := cron.New(cron.WithSeconds())
	orderPruningCronInterval := os.Getenv("ORDER_PRUNE_INTERVAL")
	if orderPruningCronInterval == "" {
		orderPruningCronInterval = "0 0 * * * *"
	}
	_, err := c.AddFunc(orderPruningCronInterval, pruneOrders)
	if err != nil {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Error adding cron job for pruning orders: err: %+v", err.Error()))
		return
	}
	c.Start()

	// Keep the program running
	select {}
}
