package settlepnl

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
)

func SettlePnlJob(subaccountIds []string) {

	xlog.Infof("Settle Pnl - Settling PnL for %d accounts", len(subaccountIds))

	if len(subaccountIds) == 0 {
		xlog.Infof("Settle Pnl - No subaccount Ids to settle PnL")
		return
	}

	// Process subaccounts in batches of 100
	for i := 0; i < len(subaccountIds); i += contractUtils.SETTLE_PNL_BATCH_SIZE {
		end := i + contractUtils.SETTLE_PNL_BATCH_SIZE
		if end > len(subaccountIds) {
			end = len(subaccountIds)
		}
		batchSubaccountIds := subaccountIds[i:end]

		var productIds []uint32
		var PricesX18 []*big.Int

		// Execute the PnL settlement contract call for the batch
		transactionCounter, err := transaction.IncrementCounter(1)
		if err != nil {
			xlog.Errorf("Settle Pnl - Failed to increment transaction counter for batch: %v", err)
			continue
		}

		// Fetch oraclePrices for the batch subaccounts
		oraclePrices, failedSubAccounts, err := xclient.GlobalBalanceClient.SettlePnLForSubaccounts(batchSubaccountIds)
		if err != nil {
			xlog.Errorf("Settle Pnl - Failed to fetch oracle prices for subaccounts: %v", err)
			continue
		}

		for _, productID := range contractUtils.ALL_COLLATERAL_SPOTS {
			PricesX18 = append(PricesX18, oraclePrices[productID])
			productIds = append(productIds, productID)
		}

		successfulSubaccounts := removeFailedSubaccounts(batchSubaccountIds, failedSubAccounts)
		if(len(successfulSubaccounts)==0){
			return
		}
		// Create the struct for batch settlement
		settleSubaccountPnlBody := contractUtils.SettleUserPnlRequest{
			SubaccountIds: successfulSubaccounts,
			ProductIds:    productIds,
			OraclePrices:  PricesX18, // Use *big.Int prices
		}

		discordErrorMsg := ""
		for failedsubaccount, errString := range failedSubAccounts {
			discordErrorMsg += fmt.Sprintf("%v -> %v\n", failedsubaccount, errString)
		}

		if discordErrorMsg != "" {
			xclient.GlobalDiscordClient.SendWebhookMessage(discordErrorMsg)
		}

		err = contract.GlobalContracts.EndpointContract.SettleUserPnl(settleSubaccountPnlBody, transactionCounter)
		if err != nil {
			xlog.Errorf("Settle Pnl - Error making batch PnL settlement contract call for subaccounts: %v", err)
			continue
		}
	}
}

func removeFailedSubaccounts(subaccounts []string, failedSubaccounts map[string]string) []string {
	var filteredSubaccounts []string
	for _, subaccount := range subaccounts {
		_, exists := failedSubaccounts[subaccount]
		if !exists {
			filteredSubaccounts = append(filteredSubaccounts, subaccount)
		}
	}
	return filteredSubaccounts
}
