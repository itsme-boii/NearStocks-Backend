package lottery

// type LotteryWinnerNotification struct {
// 	Content string `json:"content"`
// }

// var lotteryTable = &db.LotteryFlowDB{}

// func StartLotteryFlowCron() {
// 	xlog.Infof("Lottery Flow Cron job started")
// 	c := cron.New(cron.WithSeconds())
// 	// at 12 utc everyday
// 	_, err := c.AddFunc("0 0 12 * * *", func() {
// 		xlog.Infof("Lottery Flow Cron job running: %v", time.Now())

// 		// Set up the date range from 12:00 UTC of the previous day to 12:00 UTC of the current day
// 		endTime := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour) // 12:00 UTC today
// 		startTime := endTime.Add(-24 * time.Hour)

// 		// Retrieve a random lottery entry for the previous day
// 		lotteryEntry, err := lotteryTable.GetRandomEntryByTimeRange(startTime, endTime)
// 		if err != nil {
// 			xlog.Errorf("Error retrieving random lottery entry: %v", err)
// 			return
// 		}
// 		if lotteryEntry == nil {
// 			xlog.Infof("No lottery entry found for the previous day")
// 			return
// 		}

// 		// Send a notification to Discord
// 		err = sendDiscordNotification(struct {
// 			SubaccountId string
// 			LotteryCode  string
// 		}{
// 			SubaccountId: lotteryEntry.SubaccountId,
// 			LotteryCode:  lotteryEntry.LotteryCode,
// 		})
// 		if err != nil {
// 			xlog.Errorf("Error sending Discord notification: %v", err)
// 		}
// 		if err != nil {
// 			xlog.Errorf("Error sending Discord notification: %v", err)
// 		}

// 		// // Call the function to handle the winner
// 		err = handleLotteryWinner(lotteryEntry.SubaccountId)
// 		if err != nil {
// 			xlog.Errorf("Error handling lottery winner for subaccount %s: %v", lotteryEntry.SubaccountId, err)
// 			return
// 		}

// 		xlog.Infof("Lottery Flow Cron job completed for previous day")
// 	})
// 	if err != nil {
// 		xlog.Errorf("Error adding cron function: %v", err)
// 	}

// 	c.Start()
// 	xlog.Infof("Lottery Flow Cron scheduler started")

// 	select {}
// }

// func sendDiscordNotification(lotteryEntry struct {
// 	SubaccountId string
// 	LotteryCode  string
// }) error {

// 	// Extract the Ethereum address from the subaccount ID
// 	ethAddress, err := cutils.ExtractEthereumAddress(lotteryEntry.SubaccountId)
// 	if err != nil || ethAddress == "" {
// 		ethAddress = lotteryEntry.SubaccountId // Fallback to subaccount ID if Ethereum address extraction fails
// 	}

// 	// Create an array with the single Ethereum address
// 	ethAddresses := []string{ethAddress}

// 	// Initialize displayName with a fallback to ethAddress
// 	displayName := ethAddress

// 	// Call the function to get the username for the Ethereum address
// 	usernameMapping, err := (&db.SubaccountDB{}).GetUserNamesByEthAddresses(ethAddresses)
// 	if err == nil {
// 		// Check if a username exists for the Ethereum address, otherwise use the Ethereum address
// 		if username, found := usernameMapping[ethAddress]; found {
// 			displayName = username
// 		}
// 	}

// 	// Retrieve and parse the reward amount from the environment variable
// 	rewardAmountStr := os.Getenv("LOGX_REWARD_AMOUNT")
// 	rewardAmountWei, ok := new(big.Int).SetString(rewardAmountStr, 10)
// 	if !ok {
// 		xlog.Errorf("Invalid reward amount format in environment variable LOGX_REWARD_AMOUNT")
// 	}

// 	// Convert the reward amount
// 	rewardAmountLOGX := new(big.Int).Div(rewardAmountWei, cutils.GetBigx18())

// 	rewardAmountDisplay := fmt.Sprintf("%.0f", new(big.Float).SetInt(rewardAmountLOGX))

// 	endTime := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour)
// 	startTime := endTime.Add(-24 * time.Hour)

// 	// Format the start and end times with date and time (excluding the year)
// 	startTimeFormatted := startTime.Format("02 Jan 15:04 UTC")
// 	endTimeFormatted := endTime.Format("02 Jan 15:04 UTC")

// 	// Prepare the message content for the Discord webhook with the new format, using displayName and rewardAmountDisplay
// 	message := fmt.Sprintf(
// 		"🚀 **Today’s Daily Lottery Winner!** 🚀\n\n"+
// 			"The results are in! Today’s lucky winner is … 🥁 drumroll please …\n\n"+
// 			"✨ **UserName:** %s \n"+
// 			"🎫 **Ticket #%s**\n\n"+
// 			"🎉 Congratulations! You've won **%s $LOGX**! \n\n"+
// 			"📅 **Winning Period:** %s to %s\n\n\n"+
// 			"Didn’t win today? Grab your ticket now for another shot!\n\n"+
// 			"To check more details, refer to the pinned message.\n\n"+
// 			"Good luck, everyone, and see you for tomorrow’s big reveal! ✨✨",
// 		displayName,
// 		lotteryEntry.LotteryCode,
// 		rewardAmountDisplay,
// 		startTimeFormatted,
// 		endTimeFormatted,
// 	)

// 	imageUrl := os.Getenv("LOTTERY_IMAGE_URL")
// 	if imageUrl == "" {
// 		imageUrl = "https://default-image-url.com/default-image.png" // Fallback URL if environment variable is not set
// 	}
// 	// Prepare the JSON payload with content and embed for the image
// 	payload := map[string]interface{}{
// 		"content": message,
// 		"embeds": []map[string]interface{}{
// 			{
// 				"image": map[string]interface{}{
// 					"url": imageUrl,
// 				},
// 			},
// 		},
// 	}

// 	// Marshal the payload to JSON
// 	jsonPayload, err := json.Marshal(payload)
// 	if err != nil {
// 		return fmt.Errorf("error marshaling Discord request body: %v", err)
// 	}

// 	// Get the Discord webhook URL from the environment variable
// 	discordWebhookURL := os.Getenv("LOTTERY_DISCORD_NETWORK_SYNC_WEBHOOK")
// 	if discordWebhookURL == "" {
// 		return fmt.Errorf("missing Discord webhook URL configuration")
// 	}

// 	// Send the POST request to Discord
// 	resp, err := http.Post(discordWebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
// 	if err != nil {
// 		return fmt.Errorf("error sending Discord webhook request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	// Check for a successful response
// 	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
// 		return fmt.Errorf("discord webhook responded with status: %d", resp.StatusCode)
// 	}

// 	xlog.Infof("Discord notification sent successfully for lottery winner")
// 	return nil
// }
