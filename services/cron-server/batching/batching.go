package batching

import (
	"strings"
)

// var batchDb = &db.BatchDB{}

// // Discord alert and update the redis key to pause the sequencer
// func pauseSequencerForSubaccount(subaccountHex string, errToSend error) {
// 	subaccountID := strings.ToLower(subaccountHex)
// 	err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(subaccountID), 1).Err()
// 	if err != nil {
// 		xlog.Errorf("Batching Cron - Error pausing sequencer for subaccount: %s - %v", subaccountHex, err)
// 	} else {
// 		xlog.Infof("Batching Cron - Sequencer is paused for subaccount: %s", subaccountHex)
// 		alertPauseOnDiscord(errors.Join(fmt.Errorf("sequencer is paused for subaccount: %s", subaccountHex), errToSend))
// 	}
// }

// func getAllPausedSubaccounts() []string {
// 	ctx := context.Background()
// 	subaccounts, err := xredis.GetRedisClient().HKeys(ctx, xredis.GetSubacountPauseSequencerKey()).Result()

// 	if err == redis.Nil {
// 		return []string{}
// 	} else if err != nil {
// 		xlog.Errorf("Batching Cron - Error getting all paused subaccounts: %v", err)
// 		// Knowingly bypassing
// 		return []string{}
// 	}
// 	return subaccounts
// }

// func SubmitTransactions(subaccountsToExclude []string, cutoffTime time.Time) (bool, error) {
// 	defer cutils.LogTime(time.Now(), metric.SEQUENCER_BATCHING)

// 	xlog.Infof("Starting batching cron job with cutoffTime: %v", cutoffTime)

// 	batchSizeStr := os.Getenv("BATCH_SIZE")
// 	batchSize, err := strconv.Atoi(batchSizeStr)
// 	if err != nil || batchSize <= 0 {
// 		// Default to 50 if the environment variable is not set or is invalid
// 		xlog.Errorf("Batching Cron - Batch size variable is not set, defaulting to 50 transactions per batch")
// 		batchSize = 50
// 	}

// 	batchPerMin, err := strconv.Atoi(os.Getenv("NUM_BATCHES_PER_MINUTE"))
// 	if err != nil || batchPerMin <= 0 {
// 		// Default to 20 if the environment variable is not set or is invalid
// 		xlog.Warnf("Batching Cron - Number of batches per minute variable is not set, defaulting to 20 batches per minute")
// 		batchPerMin = 20
// 	}

// 	limit := batchPerMin * batchSize

// 	transactions, signatures1, signatures2, ids, subaccountIds1, subaccountIds2, _, balances, nonces, _, functionNames, err := batchDb.GetAllTransactionsWithIDs(limit, subaccountsToExclude, cutoffTime)
// 	if err != nil {
// 		return false, err
// 	}

// 	totalTransactions := len(transactions)
// 	xlog.Infof("Batching Cron - Total transactions to be sent: %d", totalTransactions)

// 	if totalTransactions == 0 {
// 		xlog.Infof("Batching Cron - No transactions to be submitted in batch db")
// 		return true, nil
// 	}

// 	for i := 0; i < totalTransactions; i += batchSize {
// 		end := i + batchSize
// 		if end > totalTransactions {
// 			end = totalTransactions
// 		}

// 		batchTransactions := transactions[i:end]
// 		batchSignatures1 := signatures1[i:end]
// 		batchSignatures2 := signatures2[i:end]
// 		batchIds := ids[i:end]

// 		batchNumber := (i / batchSize) + 1
// 		xlog.Infof("Batching Cron - Submitting batch number %d with %d transactions", batchNumber, len(batchTransactions))

// 		// Retry submitting the batch transactions 3 times
// 		for attempt := 0; attempt < 3; attempt++ {
// 			err := attemptBatchSubmission(batchTransactions, batchSignatures1, batchSignatures2, batchIds)
// 			if err == nil {
// 				xlog.Infof("Batching Cron - Batch transactions submitted successfully: %d", batchNumber, len(batchTransactions))
// 				break
// 			} else if errors.As(err, &DBError{}) {
// 				xlog.Errorf("Batching Cron - Error submitting batch transactions: %v | batch ids: %v", err, batchIds)
// 				return false, err
// 			}
// 			if attempt < 2 {
// 				xlog.Warnf("Batching Cron - Error submitting batch transactions: %v - Attempt %d... Waiting for 5 seconds before next attempt", err, attempt+1)
// 				// AUDIT: Monitor the run time of the entire cron job and then improve this
// 				time.Sleep(2 * time.Second)
// 			} else {
// 				err = attemptBinaryExecution(batchTransactions, batchSignatures1, batchSignatures2, batchIds)
// 				return false, err
// 			}
// 		}

// 		time.Sleep(2 * time.Second)
// 	}

// 	// Pass the combined array to sync services  in a goroutine
// 	go func() {
// 		err := sync.ProcessSubAccounts(balances, nonces)
// 		if err != nil {
// 			xlog.Infof("Error processing subaccounts: %v\n", err)
// 		}
// 		xlog.Infof("Verified Balance sync between web2 and web3")
// 	}()

// 	// Start a new goroutine for settle Pnl
// 	//TODO: add tests to check only match orders are being sent.
// 	stopSettlePnlJob := os.Getenv("STOP_SETTLE_PNL_JOB")
// 	if stopSettlePnlJob != "1" {
// 		filterAccounts := filterForSettlePnl(subaccountIds1, subaccountIds2, functionNames)
// 		xlog.Infof("Settle Pnl - settling pnl for subaccounts: %v", filterAccounts)

// 		withPanicRecover(func() {
// 			go settlepnl.SettlePnlJob(filterAccounts)
// 		})

// 	} else {
// 		xlog.Infof("Settle Pnl Job - not starting because of env variable")
// 	}

// 	return true, nil
// }

type DBError struct {
	msg string
}

func (e DBError) Error() string {
	return e.msg
}

// func filterForSettlePnl(subaccounts1 []string, subaccounts2 []string, functionNames []string) []string {
// 	filteredAccounts := []string{}

// 	ctx := context.Background()
// 	redisKey := xredis.GetSettlePnlSubAccounts()
// 	subAccountsArr, err := xredis.GetRedisClient().HKeys(ctx, redisKey).Result()

// 	if err == redis.Nil {
// 		return filteredAccounts
// 	} else if err != nil {
// 		xlog.Errorf("SettlePnl - Error while filtering accounts for settle pnl. Err: %v", err)
// 		return filteredAccounts
// 	}

// 	subaccountMap := make(map[string]bool)
// 	exists := make(map[string]bool)

// 	for _, subAccount := range subAccountsArr {
// 		subaccountMap[subAccount] = true
// 	}

// 	for idx, functionName := range functionNames {
// 		if functionName == transaction.MATCH_ORDERS_FN {
// 			if subaccountMap[subaccounts1[idx]] && !exists[subaccounts1[idx]] {
// 				filteredAccounts = append(filteredAccounts, subaccounts1[idx])
// 				exists[subaccounts1[idx]] = true
// 			}
// 			if subaccountMap[subaccounts2[idx]] && !exists[subaccounts2[idx]] {
// 				filteredAccounts = append(filteredAccounts, subaccounts2[idx])
// 				exists[subaccounts2[idx]] = true
// 			}
// 		}
// 	}
// 	return filteredAccounts
// }

// TODO: Separately identify errors caused by contract call and database write
// func attemptBatchSubmission(batchTransactions [][]byte, batchSignatures1 [][]byte, batchSignatures2 [][]byte, batchIds []uint) error {
// 	submitted, idx, err := contract.GlobalContracts.EndpointContract.SubmitBatchTransactionsChecked(batchTransactions, batchSignatures1, batchSignatures2)
// 	if err == nil && submitted {
// 		if err := batchDb.MarkCompletedForIds(batchIds, idx); err != nil {
// 			return DBError{msg: fmt.Sprintf("Error deleting transactions for ids: %+v", batchIds)}
// 		}
// 		return nil
// 	}
// 	return errors.Join(fmt.Errorf("batch ids of failed txns: %+v |", batchIds), err)
// }

// func attemptIncreaseNonce(subaccountIdHex string, transactionCounter uint) error {
// 	err := contract.GlobalContracts.EndpointContract.SetNonce(subaccountIdHex, 1, transactionCounter)
// 	if err != nil {
// 		xlog.Errorf("Failed to increase nonce for subaccount: %s | err: %v", subaccountIdHex, err)
// 		return fmt.Errorf("failed to increase nonce: %v", err)
// 	}
// 	return nil
// }

// func attemptBinaryExecution(batchTransactions [][]byte, batchSignature1 [][]byte, batchSignature2 [][]byte, batchIds []uint) error {
// 	left := 0
// 	right := len(batchIds) - 1
// 	var err error
// 	for left <= right {
// 		// Get mid point which is rounding towards left
// 		mid := (left + right) / 2
// 		// Try to execute from left to mid
// 		// If it fails, then try to execute for the first half
// 		// If it passes, then try to execute for the remaining half
// 		err = attemptBatchSubmission(batchTransactions[left:mid+1], batchSignature1[left:mid+1], batchSignature2[left:mid+1], batchIds[left:mid+1])
// 		if errors.As(err, &DBError{}) {
// 			// terminate the execution because if there is any database error
// 			return err
// 		} else if err != nil {
// 			right = mid - 1
// 		} else {
// 			left = mid + 1
// 		}
// 	}
// 	return err
// }

type DiscordWebhookSync struct {
	Content string `json:"content"`
}

var DiscordMemberIds = strings.Join([]string{"<@757966098292277248>", "<@725368429703463002>", "<@362470326761685002>"}, " ")

// func alertPauseOnDiscord(errExec error) {
// 	message := fmt.Sprintf("%v %+v", DiscordMemberIds, errExec)

// 	requestBody := DiscordWebhookSync{Content: message}
// 	jsonPayload, err := json.Marshal(requestBody)
// 	if err != nil {
// 		xlog.Errorf("DW - error marshaling request body: %v", err)
// 		return
// 	}
// 	resp, err := http.Post(os.Getenv("DISCORD_NETWORK_SYNC_WEBHOOK"), "application/json", bytes.NewBuffer(jsonPayload))
// 	if err != nil {
// 		xlog.Errorf("Error posting to discord: %v", err)
// 		return
// 	}
// 	defer resp.Body.Close()
// }

// Once sequencer is paused, it will not submit any transactions
// NOTE: To restart a paused sequencer, manually remove the key "PAUSE_SEQUENCER" from redis
// func pauseSequencer(errExec error) {
// 	err := xredis.GetRedisClient().Set(context.Background(), xredis.GetPauseSequencerKey(), 1, 0).Err()
// 	if err != nil {
// 		xlog.Errorf("Batching Cron - Error pausing sequencer: %v", err)
// 	} else {
// 		alertPauseOnDiscord(errors.Join(fmt.Errorf("sequencer is paused due to some error in submission: "), errExec))
// 	}
// }

// func StartBatchingCron() {
// 	c := cron.New()
// 	delete_ocbm_txn := os.Getenv("DELETE_OCBM_TXN") == "1"
// 	delete_it_txn := os.Getenv("DELETE_IT_TXN") == "1"
// 	delete_failed_unstake_txn := os.Getenv("DELETE_FAILED_UNSTAKE_TXN") == "1"
// 	batchCronInterval := os.Getenv("BATCH_CRON_INTERVAL")
// 	delete_claim_nonce_txn := os.Getenv("DELETE_CLAIM_NONCE_TXN") == "1"
// 	if batchCronInterval == "" {
// 		batchCronInterval = "* * * * *"
// 	}

// 	_, err := c.AddFunc(batchCronInterval, func() {
// 		// Lock is added here to prevent cron functions to run concurrently
// 		xredis.WithRedisLock(xredis.GetBatchingLockKey(), func() (_ *xredis.NOOP, __ error) {
// 			// Keeping this so, we can manually pause the sequencer
// 			if cutils.IsBatchingCronPaused() {
// 				xlog.Infof("Batching Cron - Batching cron is paused. Not submitting transactions.")
// 				return
// 			}
// 			if !cutils.IsSequencerRunning() {
// 				xlog.Infof("Batching Cron - Sequencer is paused. Not submitting transactions.")
// 				return
// 			}

// 			subaccountsToExclude := getAllPausedSubaccounts()
// 			// NOTE: Beware of time zone issues
// 			cutoffTime := time.Now().Add(-time.Second * 10)
// 			success, err := SubmitTransactions(subaccountsToExclude, cutoffTime)

// 			if err != nil {
// 				// Get failure txn:
// 				transactions, _, _, ids, subaccountIds1, subaccountIds2, _, _, _, txnCounters, func_names, err2 := batchDb.GetAllTransactionsWithIDs(1, subaccountsToExclude, cutoffTime)
// 				if !(err2 == nil && len(transactions) > 0) {
// 					xlog.Warnf("Batching service - This is weird as all the transactions are already submitted but there was an error while submitting txn... Doing nothing")
// 					return
// 				}

// 				nonAMMSubaccountHex := subaccountIds1[0]
// 				if nonAMMSubaccountHex == contractUtils.AMM_SUBACCOUNT_ID {
// 					nonAMMSubaccountHex = subaccountIds2[0]
// 				}
// 				failureId := ids[0]
// 				txnCounter := txnCounters[0]

// 				var inifiteLoopCase bool = false

// 				if delete_claim_nonce_txn && func_names[0] == "ClaimLogX" {
// 					if strings.Contains(err.Error(), "sufficient") {
// 						xlog.Errorf("This is expected to fail because of insufficient balance: %v", err)
// 					} else if failureId < 162603 || failureId >= 163434 {
// 						xlog.Errorf("This is unexpected and should not happen: %v", err)
// 					} else if strings.Contains(err.Error(), "WN") {
// 						xlog.Errorf("This is expected to fail because this transaction already went through..... deleting: %v", err)
// 						xlog.Infof("Batching Cron - Deleting faulty claim nonce transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 						deletedAt := time.Date(2020, 8, 29, 0, 0, 0, 0, time.UTC)
// 						if err = batchDb.SetDeletedAtById(failureId, deletedAt); err != nil {
// 							xlog.Errorf("Batching Cron - Error deleting faulty claim nonce transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter)
// 						}
// 					}
// 				} else if delete_ocbm_txn && strings.Contains(err.Error(), "OCBM") {
// 					xlog.Infof("Batching Cron - Deleting faulty OCBM transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 					deletedAt := time.Date(2000, 8, 29, 0, 0, 0, 0, time.UTC)
// 					if err = batchDb.SetDeletedAtById(failureId, deletedAt); err != nil {
// 						xlog.Errorf("Batching Cron - Error deleting faulty OCBM transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 					}
// 				} else if delete_it_txn && (strings.Contains(err.Error(), "execution reverted: IT") || strings.Contains(err.Error(), "execution reverted: IM")) {
// 					if strings.Contains(err.Error(), "execution reverted: IT") {
// 						xlog.Infof("Batching Cron - Deleting faulty IT transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 					} else {
// 						xlog.Infof("Batching Cron - Deleting faulty IM transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 					}
// 					deletedAt := time.Date(2000, 8, 30, 0, 0, 0, 0, time.UTC)
// 					if err = batchDb.SetDeletedAtById(failureId, deletedAt); err != nil {
// 						xlog.Errorf("Batching Cron - Error deleting faulty IT transaction from db: %v", err)
// 					}
// 				} else if delete_failed_unstake_txn && strings.Contains(err.Error(), "execution reverted: LogxStaker: invalid _stakeId for _account") {
// 					xlog.Infof("Batching Cron - Deleting faulty failed unstake transaction id - %v | txn counter - %v | from db: %v", failureId, txnCounter, err)
// 					deletedAt := time.Date(2000, 8, 31, 0, 0, 0, 0, time.UTC)
// 					// Fetch the current txn and delete it + increase nonce
// 					if err = batchDb.SetDeletedAtById(failureId, deletedAt); err != nil {
// 						xlog.Errorf("Batching Cron - Error deleting faulty unstake transaction from db: %v", err)
// 					} else {
// 						xlog.Infof("Deleted faulty unstake transaction from db - ID: %d....... Now increasing nonce...", ids[0])
// 						if err = attemptIncreaseNonce(nonAMMSubaccountHex, txnCounters[0]); err != nil {
// 							xlog.Errorf("Batching Cron - Error increasing nonce for faulty Unstake transaction: %v", err)
// 						} else {
// 							xlog.Infof("Increased nonce for faulty Unstake transaction", ids[0])
// 						}
// 					}
// 				} else if strings.Contains(err.Error(), "connect: connection refused") {
// 					xlog.Errorf("Batching Cron - Error submitting transactions due to connection error: %v", err)
// 					inifiteLoopCase = true
// 				} else if strings.Contains(err.Error(), "502 Bad Gateway") {
// 					xlog.Errorf("Batching Cron - Error submitting transactions due to 502 Bad Gateway: %v", err)
// 					inifiteLoopCase = true
// 				}

// 				// It is possible that error is ocbm error and we are not able to delete the transaction
// 				if err != nil {
// 					if nonAMMSubaccountHex == "" || nonAMMSubaccountHex == contractUtils.AMM_SUBACCOUNT_ID || inifiteLoopCase {
// 						alertPauseOnDiscord(errors.Join(fmt.Errorf("going into infinite loop: %s", nonAMMSubaccountHex), err))
// 					} else {
// 						pauseSequencerForSubaccount(nonAMMSubaccountHex, err)
// 					}
// 				}
// 				return
// 			}

// 			if success {
// 				xlog.Infof("Batching Cron - Batch transactions submitted successfully.")
// 			} else {
// 				xlog.Errorf("Batching Cron - Batch transactions submission was not successful.")
// 			}
// 			return
// 		})
// 	})
// 	if err != nil {
// 		xlog.Errorf("Batching Cron - Error adding cron function: %v", err)
// 	}

// 	xlog.Infof("Batching Cron - Starting the cron")
// 	c.Start()

// 	// Block the function from exiting
// 	select {}
// }
