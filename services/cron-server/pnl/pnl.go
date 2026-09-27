package pnl

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"math/big"
	"os"
	"time"

	"github.com/robfig/cron/v3"
)

var PnlTable = &db.PnlDB{}

const DEFAULT_BROKER_ID = uint(1)

func StartDailyPnlCron() {

	fmt.Println("Daily PnL Cron job started")
	c := cron.New(cron.WithSeconds())
	// cron job runs at every 00:02 utc everyday
	_, err := c.AddFunc("0 2 0 * * *", func() {
		fmt.Println("Daily PnL Cron job running:", time.Now())
		subaccountID := os.Getenv("AMM_SUBACCOUNT_ID")
		if subaccountID == "" {
			fmt.Println("AMM_SUBACCOUNT_ID environment variable not set")
			return
		}
		calculateAndStoreDailyPnL(subaccountID)
	})
	if err != nil {
		fmt.Printf("Error adding cron function: %v\n", err)
	}

	// Start the cron scheduler
	c.Start()
	fmt.Println("Cron scheduler started")

	// Keep the cron scheduler running in the background
	select {}
}

func calculateAndStoreDailyPnL(subaccountID string) {
	fmt.Printf("Calculating PnL for subaccount: %s\n", subaccountID)

	// Fetch the daily realized PnL
	pnLPerToken, _, err := getDailyRealizedPnL(subaccountID)
	if err != nil {
		fmt.Printf("Error fetching daily realized PnL: %v\n", err)
		pnLPerToken = make(map[uint]ctypes.BigInt) // Initialize as empty map
	}

	// Fetch the daily unrealized PnL
	unrealizedPnLPerToken, err := cutils.GetUnrealizedPnL(subaccountID)
	if err != nil {
		fmt.Printf("Error fetching unrealized PnL: %v\n", err)
		unrealizedPnLPerToken = make(map[uint]string) // Initialize as empty map
	}

	// Create a set of all ProductIDs to consider all products
	productIDs := make(map[uint]struct{})
	for productID := range pnLPerToken {
		productIDs[productID] = struct{}{}
	}
	for productID := range unrealizedPnLPerToken {
		productIDs[productID] = struct{}{}
	}

	// Process each productID
	for productID := range productIDs {
		realizedPnl, ok := pnLPerToken[productID]
		if !ok {
			realizedPnl = ctypes.NewBigInt(big.NewInt(0)) // Default to zero if no realized PnL
		}

		unrealizedPnl, ok := unrealizedPnLPerToken[productID]
		if !ok {
			unrealizedPnl = "0" // Default to zero if no unrealized PnL
		}

		pnlData := db.PnlTable{
			BrokerId:      DEFAULT_BROKER_ID,
			ProductID:     productID,
			RealizedPnl:   realizedPnl,
			UnrealizedPnl: unrealizedPnl,
		}
		if _, err := PnlTable.CreatePnl(pnlData); err != nil {
			fmt.Printf("Error creating PnL entry for ProductID %d: %v\n", productID, err)
		}
	}

	fmt.Println("Successfully created PnL entries")
}

func getDailyRealizedPnL(subaccountID string) (map[uint]ctypes.BigInt, ctypes.BigInt, error) {
	fillDB := &db.FillDB{}
	// Set dateToQuery to the start of yesterday
	dateToQuery := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	pnLPerToken, totalPnL, err := fillDB.GetDailyRealizedPnL(subaccountID, dateToQuery)
	if err != nil {
		return nil, ctypes.BigInt{}, err
	}
	return pnLPerToken, totalPnL, nil
}
