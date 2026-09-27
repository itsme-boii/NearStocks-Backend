package flagusercron

import (
	"context"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"

	"github.com/robfig/cron/v3"
)

func flagUsersJob() {
	xlog.Infof("FlagUser cron job running")
	ctx := context.Background()
	redisClient := xredis.GetRedisClient()

	// Fetch POSITIONS_MAP from Redis
	positionsMapKey := xredis.GetPositionsMapKey()
	val, err := redisClient.Get(ctx, positionsMapKey).Result()
	if err != nil {
		xlog.Errorf("Error fetching POSITIONS_MAP: %v", err)
		return
	}
	var positionMap map[string]*ctypes.PositionSummary
	if err := json.Unmarshal([]byte(val), &positionMap); err != nil {
		xlog.Errorf("Error unmarshalling POSITIONS_MAP: %v", err)
		return
	}

	// Get all unique subaccount IDs from the position map
	subaccountIdSet := make(map[string]struct{})
	for _, pos := range positionMap {
		if pos != nil && pos.SubaccountID != "" {
			subaccountIdSet[pos.SubaccountID] = struct{}{}
		}
	}
	subaccountIds := make([]string, 0, len(subaccountIdSet))
	for subId := range subaccountIdSet {
		subaccountIds = append(subaccountIds, subId)
	}

	// Fetch all subaccount balances
	subaccountImpl := subaccount.NewSubaccountBalanceImpl()
	subaccountBalances := subaccountImpl.MustGetSubaccountBalancesFromIds(subaccountIds)

	appState := appstate.NewAppState()
	oraclePrices, err := appState.GetAllOraclePrices()
	if err != nil {
		xlog.Errorf("Error fetching oracle prices: %v", err)
		return
	}
	cumulativeFundingRateMap := appState.GetAllFundingRates()

	// Map subaccount -> sum of unrealised PnL
	subaccountUnrealised := make(map[string]*big.Int)
	thresholdValue, err := redisClient.Get(ctx, xredis.GetFlagUserThresholdKey()).Result()
	if err != nil || thresholdValue == "" {
		thresholdValue = "2000"
	}
	threshold := new(big.Int).Mul(cutils.FloatStrToX18(thresholdValue), big.NewInt(1e18))

	for _, subBal := range subaccountBalances {
		upnlx36, _ := subaccount.GetUnrealisedSubaccountValue(subBal, oraclePrices, cumulativeFundingRateMap)
		subaccountUnrealised[subBal.SubaccountId] = upnlx36
	}

	// Fetch 24h realised PnL for all subaccounts
	fillDB := &db.FillDB{}
	realisedMap, err := fillDB.Get24hRealisedPnlPerSubaccount()
	if err != nil {
		xlog.Errorf("Error fetching 24h realised PnL: %v", err)
		return
	}

	flaggedKey := xredis.GetFlaggedSubaccountsKey()

	// Create union of subaccounts with either unrealized or realized PnL
	allSubaccounts := make(map[string]struct{})
	for subaccount := range subaccountUnrealised {
		allSubaccounts[subaccount] = struct{}{}
	}
	for subaccount := range realisedMap {
		allSubaccounts[subaccount] = struct{}{}
	}

	for subaccount := range allSubaccounts {
		unrealisedx36 := big.NewInt(0)
		if v, ok := subaccountUnrealised[subaccount]; ok {
			unrealisedx36 = v
		}
		realisedx36 := big.NewInt(0)
		if v, ok := realisedMap[subaccount]; ok {
			// realisedMap is in x18, convert to x36
			realisedx36 = new(big.Int).Mul(v, big.NewInt(1e18))
		}
		totalx36 := new(big.Int).Add(unrealisedx36, realisedx36)

		// Check if user exceeds threshold
		exceedsThreshold := unrealisedx36.Cmp(threshold) > 0 || totalx36.Cmp(threshold) > 0

		if exceedsThreshold {
			// Check current flag status
			currentStatus, err := cutils.GetSubaccountFlagStatus(ctx, subaccount)
			if err != nil {
				xlog.Errorf("Error getting flag status for subaccount %s: %v", subaccount, err)
				continue
			}

			// If user is unflagged (status = "2"), don't flag them but send Discord alert
			if currentStatus == "2" {
				// Send Discord alert for unflagged user with high PnL
				msg := fmt.Sprintf("Unflagged user with high PnL: %s, Unrealised x36: %s, Realised+Unrealised x36: %s",
					subaccount, unrealisedx36.String(), totalx36.String())
				xclient.GlobalDiscordClient.SendWebhookMessage(msg)
				continue
			}

			// If user is not flagged (status = "0" or doesn't exist), flag them
			if currentStatus == "0" || currentStatus == "" {
				// Store in Redis as "1" (flagged)
				err := redisClient.HSet(ctx, flaggedKey, subaccount, "1").Err()
				if err != nil {
					xlog.Errorf("Error flagging subaccount %s: %v", subaccount, err)
					continue
				}

				// Send Discord alert for newly flagged user
				msg := fmt.Sprintf("Flagged subaccount: %s, Unrealised x36: %s, Realised+Unrealised x36: %s",
					subaccount, unrealisedx36.String(), totalx36.String())
				xclient.GlobalDiscordClient.SendWebhookMessage(msg)
			}
			// If user is already flagged (status = "1"), do nothing
		}
	}
}

func StartFlagUserCron() {

	c := cron.New(cron.WithSeconds())

	// cron job to run every 5 minutes
	c.AddFunc("0 */5 * * * *", func() {
		cerrors.WithPanicRecover(flagUsersJob)
	})

	c.Start()
	// Run the job immediately once when the service starts
	go cerrors.WithPanicRecover(flagUsersJob)

	// Keep the scheduler running
	select {}
}
