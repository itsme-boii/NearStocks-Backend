package sync

import (
	"strings"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/robfig/cron/v3"
)

var (
	BatchTable   = &db.BatchDB{}
	MinimumDelta = cutils.GetBigxCust(1)
	// redisClient *redis.Client = xredis.GetRedisClient()
)

type DiscordWebhookSync struct {
	Content string `json:"content"`
}

const MAX_SUBACCOUNTS_PER_BATCH = 20000

// var redisClient *redis.Client = xredis.GetRedisClient()

// Note - not adding distributed redis lock since the key-value pair is referenced only in this function
// func StartSyncCron() {
// 	xlog.Infof("Sync Cron job started")
// 	c := cron.New(cron.WithSeconds())
// 	// cron job runs every 1 minutes
// 	_, err := c.AddFunc("0 */1 * * * *", func() {
// 		xlog.Infof("Sync Cron job running: %v", time.Now())
// 		ctx := context.Background()
// 		startIndexStr, err := redisClient.Get(ctx, "sync_start_index").Result()
// 		if err != nil && err != redis.Nil {
// 			xlog.Errorf("Error fetching start index from Redis: %v", err)
// 			return
// 		}

// 		startIndex := uint64(0)
// 		if startIndexStr != "" {
// 			startIndex, err = strconv.ParseUint(startIndexStr, 10, 64)
// 			if err != nil {
// 				xlog.Errorf("Error parsing start index from Redis: %v", err)
// 				return
// 			}
// 		}
// 		xlog.Infof("Sync server startIndex", startIndex)

// 		subaccountIDs, lastIndex, err := BatchTable.GetUniqueSubAccountIDsAfterID(uint(startIndex), MAX_SUBACCOUNTS_PER_BATCH)
// 		if err != nil {
// 			xlog.Errorf("Error fetching subaccount IDs: %v", err)
// 			return
// 		}

// 		if err := processSubAccounts(subaccountIDs); err != nil {
// 			xlog.Errorf("Error processing subaccounts: %v", err)
// 			return
// 		}

// 		err = redisClient.Set(ctx, "sync_start_index", strconv.FormatUint(lastIndex, 10), 0).Err()
// 		if err != nil {
// 			xlog.Errorf("Error storing new start index in Redis: %v", err)
// 		}
// 		xlog.Infof("Sync server endIndex", lastIndex)
// 	})
// 	if err != nil {
// 		xlog.Errorf("Error adding cron function: %v", err)
// 	}

// 	// Start the cron scheduler
// 	c.Start()
// 	xlog.Infof("Cron scheduler started")

// 	// Keep the cron scheduler running in the background
// 	select {}
// }

// func ProcessSubAccounts(balances map[string]string, nonces map[string]string) error {
// 	if len(balances) == 0 {
// 		return nil
// 	}
// 	if err := fetchAndCompareSpotBalances(balances); err != nil {
// 		xlog.Errorf("Error processing spot balances: %v", err)
// 		return err
// 	}
// 	if err := fetchAndComparePerpPositions(balances); err != nil {
// 		xlog.Errorf("Error processing perpetual positions: %v", err)
// 		return err
// 	}
// 	if len(nonces) > 0 {
// 		if err := fetchAndCompareNonces(nonces); err != nil {
// 			xlog.Errorf("Error processing nonces: %v", err)
// 			return err
// 		}
// 	}

// 	// Send the sync health check notification to Discord
// 	message := "********************* Sync health check done on Balances and Nonces *********************"
// 	requestBody := DiscordWebhookSync{Content: message}
// 	jsonPayload, err := json.Marshal(requestBody)
// 	if err != nil {
// 		xlog.Errorf("DW - error marshaling request body: %v", err)
// 		return err
// 	}

// 	resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 	if err != nil {
// 		xlog.Errorf("DW - error sending notification to Discord: %v", err)
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	return nil
// }

// func fetchAndCompareNonces(nonces map[string]string) error {
// 	// Extract subaccount IDs from the map
// 	subaccountIDs := make([]string, 0, len(nonces))
// 	for subaccountID := range nonces {
// 		subaccountIDs = append(subaccountIDs, subaccountID)
// 	}

// 	// Fetch nonces for all subaccounts in batch from the contract
// 	contractNonces, err := contract.GlobalContracts.EndpointContract.GetNoncesForSubaccounts(subaccountIDs)
// 	if err != nil {
// 		return fmt.Errorf("error fetching nonces from contract: %v", err)
// 	}

// 	// Iterate over the fetched contract nonces and compare with client nonces
// 	for i, subaccountID := range subaccountIDs {
// 		clientNonce := nonces[subaccountID]
// 		contractNonce := fmt.Sprintf("%d", contractNonces[i])

// 		// Compare client (Redis) nonce and contract nonce
// 		if clientNonce != contractNonce {
// 			message := fmt.Sprintf("Difference in nonce for subaccount %s: Client Nonce: %s, Contract Nonce: %s", subaccountID, clientNonce, contractNonce)
// 			// xlog.Infof(message)
// 			requestBody := DiscordWebhookSync{Content: message}
// 			jsonPayload, err := json.Marshal(requestBody)
// 			if err != nil {
// 				return fmt.Errorf("DW - error marshaling request body: %v", err)
// 			}
// 			resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 			if err != nil {
// 				return err
// 			}
// 			defer resp.Body.Close()
// 		}
// 	}

// 	return nil
// }

// func fetchAndCompareSpotBalances(balances map[string]string) error {
// 	// Extract subaccount IDs from the balances map
// 	subaccountIDs := make([]string, 0, len(balances))
// 	for subaccountID := range balances {
// 		if subaccountID == "cumulative_funding_rates" {
// 			ok, err := CheckFundingRateSync(balances[subaccountID])
// 			if !ok || err != nil {
// 				xlog.Errorf("Error in funding rate sync check: %v", err)
// 			}
// 			continue
// 		}
// 		subaccountIDs = append(subaccountIDs, subaccountID)
// 	}

// 	// Fetch contract balances for all subaccounts
// 	contractSpotBalances, err := contract.GlobalContracts.SpotContract.GetAllBalancesOfSubaccounts(subaccountIDs)
// 	if err != nil {
// 		return fmt.Errorf("error fetching spot balances from contract: %v", err)
// 	}

// 	// Iterate over each subaccount
// 	for subaccountID, contractBalances := range contractSpotBalances {
// 		// Temporarily skip the AMM subaccount
// 		if subaccountID == contractUtils.AMM_SUBACCOUNT_ID {
// 			continue
// 		}

// 		clientBalanceJSON, exists := balances[subaccountID]
// 		if !exists {
// 			xlog.Infof("Subaccount %s exists in contract balance but not in client balance", subaccountID)
// 			continue
// 		}
// 		if clientBalanceJSON == "" {
// 			continue
// 		}
// 		// Create a map for contract balances using productID as the key
// 		contractBalanceMap := make(map[uint32]*big.Int)
// 		for index, contractBalance := range contractBalances {
// 			productID := getSpotProductID(index)
// 			contractBalanceMap[productID] = contractBalance
// 		}

// 		// Unmarshal the outer JSON string into a map[string]string
// 		var rawBalances map[string]string
// 		if err := json.Unmarshal([]byte(clientBalanceJSON), &rawBalances); err != nil {
// 			xlog.Errorf("Unable to unmarshal client balance data for subaccount %s: %v", subaccountID, err)
// 			continue
// 		}

// 		// Unmarshal each inner JSON string into the respective struct
// 		clientBalances := make(map[string]struct {
// 			Available string `json:"available"`
// 			Locked    string `json:"locked"`
// 		})

// 		for productIDStr, rawBalance := range rawBalances {
// 			var tokenBalance struct {
// 				Available string `json:"available"`
// 				Locked    string `json:"locked"`
// 			}
// 			if err := json.Unmarshal([]byte(rawBalance), &tokenBalance); err != nil {
// 				xlog.Errorf("Unable to unmarshal token balance data for subaccount %s, productID %s: %v", subaccountID, productIDStr, err)
// 				continue
// 			}
// 			clientBalances[productIDStr] = tokenBalance
// 		}

// 		// Compare client and contract balances for even productIDs
// 		for productIDStr, tokenBalance := range clientBalances {
// 			productID, err := strconv.Atoi(productIDStr)
// 			if err != nil {
// 				xlog.Errorf("Invalid product ID %s for subaccount %s: %v", productIDStr, subaccountID, err)
// 				continue
// 			}

// 			// Only compare if the productID is even
// 			if productID%2 == 0 {
// 				contractAmount, exists := contractBalanceMap[uint32(productID)]
// 				if !exists {
// 					xlog.Infof("ProductID %d exists in client balance but not in contract balance for subaccount %s", productID, subaccountID)
// 					continue
// 				}

// 				clientAmount := new(big.Int)
// 				clientAmount.SetString(tokenBalance.Available, 10)
// 				diff := new(big.Int).Sub(contractAmount, clientAmount)
// 				//TODO: remove diff amount check after deposit flow change
// 				if clientAmount.Cmp(contractAmount) != 0 && new(big.Int).Abs(diff).Cmp(MinimumDelta) == 1 {
// 					message := fmt.Sprintf("Difference in spot balance for subaccount %s, productID %d: Client Balance: %v, Contract Balance: %v", subaccountID, productID, clientAmount, contractAmount)
// 					// xlog.Infof(message)
// 					requestBody := DiscordWebhookSync{Content: message}
// 					jsonPayload, err := json.Marshal(requestBody)
// 					if err != nil {
// 						return fmt.Errorf("DW - error marshaling request body: %v", err)
// 					}
// 					resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 					if err != nil {
// 						return err
// 					}
// 					defer resp.Body.Close()
// 				}
// 			}
// 		}
// 	}
// 	return nil
// }

// func fetchAndComparePerpPositions(balances map[string]string) error {
// 	// Define the Balance struct with strings for initial unmarshalling
// 	type BalanceRaw struct {
// 		Amount                   string `json:"amount"`
// 		VQuoteBalance            string `json:"vQuoteBalance"`
// 		LastCumulativeFundingX18 string `json:"lastFundingRate"`
// 	}

// 	// Define the Balance struct with *big.Int for final use
// 	type Balance struct {
// 		Amount                   *big.Int
// 		VQuoteBalance            *big.Int
// 		LastCumulativeFundingX18 *big.Int
// 	}

// 	// Extract subaccount IDs from the balances map
// 	subaccountIDs := make([]string, 0, len(balances))
// 	for subaccountID := range balances {
// 		if subaccountID == "cumulative_funding_rates" {
// 			continue
// 		}
// 		subaccountIDs = append(subaccountIDs, subaccountID)
// 	}

// 	// Fetch contract balances for all subaccounts
// 	contractPositions, err := contract.GlobalContracts.PerpContract.GetAllBalancesOfSubaccounts(subaccountIDs)
// 	if err != nil {
// 		return fmt.Errorf("error fetching perpetual positions from contract: %v", err)
// 	}

// 	// Iterate over each subaccount
// 	for subaccountID, contractBalances := range contractPositions {
// 		// Temporarily skip the AMM subaccount to avoid spamming in Discord
// 		if subaccountID == contractUtils.AMM_SUBACCOUNT_ID {
// 			continue
// 		}
// 		clientBalanceJSON, exists := balances[subaccountID]
// 		if !exists {
// 			xlog.Infof("Subaccount %s exists in contract positions but not in client balance", subaccountID)
// 			continue
// 		}
// 		if clientBalanceJSON == "" {
// 			continue
// 		}
// 		// Unmarshal the outer JSON string into a map[string]string
// 		var rawBalances map[string]string
// 		if err := json.Unmarshal([]byte(clientBalanceJSON), &rawBalances); err != nil {
// 			xlog.Errorf("Unable to unmarshal client balance data for subaccount %s: %v", subaccountID, err)
// 			continue
// 		}

// 		// Unmarshal each inner JSON string into the BalanceRaw struct only if the productID is odd, then convert to Balance
// 		clientBalances := make(map[string]Balance)
// 		for productIDStr, rawBalance := range rawBalances {
// 			productID, err := strconv.Atoi(productIDStr)
// 			if err != nil {
// 				xlog.Errorf("Invalid product ID %s for subaccount %s: %v", productIDStr, subaccountID, err)
// 				continue
// 			}

// 			// Only unmarshal if the productID is odd
// 			if productID%2 != 0 {
// 				var rawBalanceData BalanceRaw
// 				if err := json.Unmarshal([]byte(rawBalance), &rawBalanceData); err != nil {
// 					xlog.Errorf("Unable to unmarshal perp balance data for subaccount %s, productID %s: %v", subaccountID, productIDStr, err)
// 					continue
// 				}

// 				// Convert string fields to *big.Int
// 				amount := new(big.Int)
// 				if _, ok := amount.SetString(rawBalanceData.Amount, 10); !ok {
// 					xlog.Errorf("Invalid amount for subaccount %s, productID %s: %s", subaccountID, productIDStr, rawBalanceData.Amount)
// 					continue
// 				}

// 				vQuoteBalance := new(big.Int)
// 				if _, ok := vQuoteBalance.SetString(rawBalanceData.VQuoteBalance, 10); !ok {
// 					xlog.Errorf("Invalid vQuoteBalance for subaccount %s, productID %s: %s", subaccountID, productIDStr, rawBalanceData.VQuoteBalance)
// 					continue
// 				}

// 				lastCumulativeFundingX18 := new(big.Int)
// 				if _, ok := lastCumulativeFundingX18.SetString(rawBalanceData.LastCumulativeFundingX18, 10); !ok {
// 					xlog.Errorf("Invalid lastFundingRate for subaccount %s, productID %s: %s", subaccountID, productIDStr, rawBalanceData.LastCumulativeFundingX18)
// 					continue
// 				}

// 				clientBalances[productIDStr] = Balance{
// 					Amount:                   amount,
// 					VQuoteBalance:            vQuoteBalance,
// 					LastCumulativeFundingX18: lastCumulativeFundingX18,
// 				}
// 			}
// 		}

// 		// Create a map for contract balances using productID as the key
// 		contractBalanceMap := make(map[uint32]Balance)
// 		for index, contractBalance := range contractBalances {
// 			productID := getPerpProductID(index)
// 			contractBalanceMap[productID] = Balance{
// 				Amount:                   contractBalance.Amount,
// 				VQuoteBalance:            contractBalance.VQuoteBalance,
// 				LastCumulativeFundingX18: contractBalance.LastCumulativeFundingX18,
// 			}
// 		}
// 		// Compare client and contract positions
// 		for productIDStr, perpBalance := range clientBalances {
// 			productID, err := strconv.Atoi(productIDStr)
// 			if err != nil {
// 				xlog.Errorf("Invalid product ID %s for subaccount %s: %v", productIDStr, subaccountID, err)
// 				continue
// 			}

// 			contractAmount, exists := contractBalanceMap[uint32(productID)]
// 			if !exists {
// 				xlog.Infof("ProductID %d exists in client balance but not in contract positions for subaccount %s", productID, subaccountID)
// 				continue
// 			}
// 			diffFundingX18 := new(big.Int).Sub(contractAmount.LastCumulativeFundingX18, perpBalance.LastCumulativeFundingX18)
// 			diffQuoteBalance := new(big.Int).Sub(contractAmount.VQuoteBalance, perpBalance.VQuoteBalance)
// 			diffAmount := new(big.Int).Sub(contractAmount.Amount, perpBalance.Amount)

// 			if new(big.Int).Abs(diffFundingX18).Cmp(MinimumDelta) == 1 ||
// 				new(big.Int).Abs(diffQuoteBalance).Cmp(MinimumDelta) == 1 ||
// 				new(big.Int).Abs(diffAmount).Cmp(MinimumDelta) == 1 {
// 				message := fmt.Sprintf("Difference in perpetual positions for subaccount %s, productID %d: Client Position: %+v, Contract Position: %+v", subaccountID, productID, perpBalance, contractAmount)
// 				// xlog.Infof(message)
// 				requestBody := DiscordWebhookSync{Content: message}
// 				jsonPayload, err := json.Marshal(requestBody)
// 				if err != nil {
// 					return fmt.Errorf("DW - error marshaling request body: %v", err)
// 				}
// 				resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 				if err != nil {
// 					return err
// 				}
// 				defer resp.Body.Close()
// 			}
// 		}
// 	}
// 	return nil
// }

// func getSpotProductID(index int) uint32 {
// 	switch index {
// 	case 0:
// 		return 4
// 	case 1:
// 		return 0
// 	case 2:
// 		return 2
// 	default:
// 		return uint32(2 * index)
// 	}
// }

// func getPerpProductID(index int) uint32 {
// 	return uint32(2*index + 1)
// }

var DiscordMemberIds = strings.Join([]string{"<@743007249420779522>", "<@757966098292277248>"}, " ")

// func CheckFundingRateSync(cumulativeFundingRateStr string) (bool, error) {
// 	// Fetch cumulative funding rates from the contract
// 	fundingRatesFromContract, _, err := contract.GlobalContracts.PerpContract.GetCumulativeFundingRates()
// 	if err != nil {
// 		return false, fmt.Errorf("error fetching cumulative funding rates from contract: %v", err)
// 	}

// 	// Split the input string into productID:rate pairs
// 	pairs := strings.Split(cumulativeFundingRateStr, ";")

// 	// Iterate over the pairs and compare each rate with the contract value
// 	for _, pair := range pairs {
// 		if pair == "" {
// 			continue // Skip empty strings
// 		}

// 		// Split the pair into productID and rate
// 		parts := strings.Split(pair, ":")
// 		if len(parts) != 2 {
// 			return false, fmt.Errorf("invalid format for cumulative funding rate pair: %s", pair)
// 		}

// 		// Convert productID to an integer
// 		productID, err := strconv.Atoi(parts[0])
// 		if err != nil {
// 			return false, fmt.Errorf("error converting productID to integer: %v", err)
// 		}

// 		// Convert the rate from the string to a *big.Int
// 		rateFromDB := new(big.Int)
// 		rateFromDB, ok := rateFromDB.SetString(parts[1], 10)
// 		if !ok {
// 			return false, fmt.Errorf("error converting rate to *big.Int: %s", parts[1])
// 		}

// 		// Compare the rate with the contract rate
// 		contractRate := fundingRatesFromContract[productID/2]
// 		if contractRate == nil || rateFromDB == nil {
// 			return false, fmt.Errorf("nil value encountered for productID: %d", productID)
// 		}

// 		if contractRate.Cmp(rateFromDB) != 0 {
// 			xlog.Infof("Difference in funding rate for product %v: Web3 Rate: %s, Web2 Rate: %s", productID, contractRate.String(), rateFromDB.String())
// 			message := fmt.Sprintf("%v. Difference in funding rate for product %v: Web3 Rate: %s, Web2 Rate: %s",
// 				DiscordMemberIds,
// 				productID,
// 				contractRate.String(),
// 				rateFromDB.String())

// 			requestBody := DiscordWebhookSync{Content: message}
// 			jsonPayload, err := json.Marshal(requestBody)
// 			if err != nil {
// 				return false, fmt.Errorf("DW - error marshaling request body: %v", err)
// 			}

// 			// Send the notification to Discord
// 			resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 			if err != nil {
// 				return false, err
// 			}
// 			defer resp.Body.Close()
// 		}
// 	}

// 	message := "********************* Sync health check done on funding rate *********************"
// 	requestBody := DiscordWebhookSync{Content: message}
// 	jsonPayload, err := json.Marshal(requestBody)
// 	if err != nil {
// 		return false, fmt.Errorf("DW - error marshaling request body: %v", err)
// 	}

// 	// Send the notification to Discord
// 	resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 	if err != nil {
// 		return false, err
// 	}
// 	defer resp.Body.Close()

// 	return true, nil
// }

func CheckAmmHealth() {
	// Check AMM health
	c := cron.New(cron.WithSeconds())

	// Runs every minute
	_, err := c.AddFunc("0 */1 * * * *", func() {
		health, err := xclient.GlobalBalanceClient.GetHealth(contractUtils.AMM_SUBACCOUNT_ID)
		if err != nil {
			xlog.Errorf("Error fetching AMM health: %v", err)
			xclient.GlobalDiscordClient.SendWebhookMessage("Error fetching AMM health: " + err.Error())
			return
		}

		if health.BelowMaintenanceMargin {
			message := "AMM is below maintenance margin. Please add more funds"
			xlog.Infof(message)
			xclient.GlobalDiscordClient.SendWebhookMessage(message)
		} else if health.BelowInitialMargin {
			message := "AMM is below initial margin. Please add more funds"
			xlog.Infof(message)
			xclient.GlobalDiscordClient.SendWebhookMessage(message)
		} else {
			xlog.Infof("AMM health is good")
			// xclient.GlobalDiscordClient.SendWebhookMessage("################## AMM's health check done ########################")
		}
	})

	if err != nil {
		xlog.Errorf("Error adding cron function: %v", err)
	}

	// Start the cron scheduler
	xlog.Infof("Check AMM health cron job started")
	c.Start()
}
