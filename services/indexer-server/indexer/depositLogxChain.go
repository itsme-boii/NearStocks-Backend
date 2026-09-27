package indexer

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cerrors"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/sha3"
)

// Current implementation -
// 1. What we are doing here is looking on source chain for any deposits that have been made to the mailbox contract
// 2. We take the transaction hash of the deposit and mark the deposit as received in the database
// 3. Unique identifier for the deposit is the message id

type DepositLogXTracker struct {
	logxRpcClient      *ethclient.Client // url -> rpcClient
	eventSignatureHash common.Hash
	mailBoxAddresses   map[uint32]common.Address // productId -> mailbox address
}

func NewDepositLogXTracker() *DepositLogXTracker {
	return &DepositLogXTracker{}
}

func (wf *DepositLogXTracker) Init() {
	// Initialize the RPC clients
	client, err := ethclient.Dial(contractUtils.RPC_URL)
	if err != nil {
		xlog.Errorf("Failed to connect to the Ethereum client: %v", err)
	}
	wf.logxRpcClient = client
	wf.eventSignatureHash = wf.GetEventSignature()

	// Initialize mailbox addresses for each product
	wf.mailBoxAddresses = make(map[uint32]common.Address)
	for productId := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		mailboxAddr := contractUtils.PRODUCT_ID_TO_MAILBOX_ADDRESS[productId]
		wf.mailBoxAddresses[productId] = common.HexToAddress(mailboxAddr)
		xlog.Infof("Initialized mailbox for product %d: %s", productId, mailboxAddr)
	}
}

type LogxReceivedStats struct {
	LatestBlock  uint64
	TotalFetched int
}

func (wf *DepositLogXTracker) Run() {
	DepositLogXTrackerInterval := os.Getenv("DEPOSIT_LOGX_INTERVAL")
	if DepositLogXTrackerInterval == "" {
		DepositLogXTrackerInterval = "*/30 * * * * *"
	} else {
		xlog.Infof("Using custom interval for deposit LogX tracker: %s", DepositLogXTrackerInterval)
	}

	c := cron.New(cron.WithSeconds())
	for productId := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		_, err := c.AddFunc(DepositLogXTrackerInterval, func() {
			wf.RunJob(productId)
		})
		if err != nil {
			xlog.Errorf("Failed to add job for product %d: %v", productId, err)
		}
	}

	envEnabled := os.Getenv("IS_DEPOSIT_LOGX_BACKFILL_ENABLED")
	if envEnabled == "1" {
		_, err1 := c.AddFunc(DepositLogXTrackerInterval, func() {
			cerrors.WithPanicRecover(func() { wf.RunBackFillJob() })
		})
		if err1 != nil {
			xlog.Errorf("Failed to add backfill job: %v", err1)
		}
	}

	c.Start()
}

func (wf *DepositLogXTracker) RunJob(productId uint32) {
	defer cutils.LogTime(time.Now(), fmt.Sprintf("DEPOSIT_LOGX_TRACKER: %v", productId))

	limit := 0
	if os.Getenv("DEPOSIT_LOGX_LIMIT") != "" {
		limit, _ = strconv.Atoi(os.Getenv("DEPOSIT_LOGX_LIMIT"))
	}
	if limit == 0 {
		limit = 200
	}

	messageIdsToReceived := (&db.DepositWithdrawDB{}).GetFirstXSourceOnlyReceivedDepositsMessageIdsForProductId(productId, limit)
	if len(messageIdsToReceived) == 0 {
		xlog.Infof("No deposits to mark received for product %d", productId)
		return
	}

	mailboxAddress, exists := wf.mailBoxAddresses[productId]
	if !exists {
		xlog.Errorf("Mailbox address not found for product %d", productId)
		return
	}

	results := wf.AttemptMarkDepositReceived(messageIdsToReceived, wf.logxRpcClient, mailboxAddress, productId)

	xlog.Infof("Received deposits LogX chain logs for messages (first 10 out of %v): Hits: %v, Misses: %+v for product %d", len(results.Misses), results.Hits, results.Misses[:min(10, len(results.Misses))], productId)
	if len(results.Misses) > 9 {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Deposit Logx: %+v for product %d", results, productId))
	}
}

func (*DepositLogXTracker) GetEventSignature() common.Hash {
	// Generate the event signature hash for "ProcessId(bytes32)"
	eventSignature := []byte("ProcessId(bytes32)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(eventSignature)
	return common.BytesToHash(hash.Sum(nil))
}

func (wf *DepositLogXTracker) AttemptMarkDepositReceived(messageIds []string, client *ethclient.Client, contractAddress common.Address, productId uint32) FinaliseStats {
	// Get the latest block
	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		xlog.Errorf("Failed to get the latest block number for Finalise Withdraw indexer.. something wrong with rpc of product: %v | err: %v", productId, err)
		// Making TotalFetched to -1 to undertand that this was a major failure
		return FinaliseStats{Hits: -1, Misses: messageIds}
	}

	// Process and finalize unfinalized withdrawals
	messageIdsHased := make([]common.Hash, len(messageIds))
	for i, messageId := range messageIds {
		messageIdsHased[i] = common.HexToHash(messageId)
	}

	// Create the filter query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{wf.eventSignatureHash}, messageIdsHased},
		FromBlock: big.NewInt(int64(currentBlock - 10000)),
		ToBlock:   big.NewInt(int64(currentBlock)),
	}

	// Query logs
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		xlog.Errorf("Failed to filter logs for attempMarkDeposit: %v | message ids len: %v | product id: ", err, len(messageIds), productId)
		return FinaliseStats{Hits: 0, Misses: messageIds}
	}

	messageIdsMissed := map[string]bool{}
	for _, v := range messageIds {
		messageIdsMissed[v] = true
	}

	// Finalize withdrawals in the database if logs are found
	if len(logs) > 0 {
		for _, vLog := range logs {
			messageId := vLog.Topics[1].Hex()
			(&db.DepositWithdrawDB{}).MarkDepositReceivedByMessageId(messageId)
			delete(messageIdsMissed, messageId)
		}

		err := xclient.GlobalApiServerClient.FinaliseDeposits()
		if err != nil {
			xlog.Errorf("Failed to mark logx chain deposits as finalised. You can manually hit api and get the same work done: %v", err)
		}
	}

	return FinaliseStats{Hits: len(logs), Misses: cutils.MapKeys(messageIdsMissed)}
}

func (wf *DepositLogXTracker) GetBackFillConfig() (startBlock uint64, endBlock uint64, isPaused bool) {
	value, err := xredis.GetRedisClient().Get(context.Background(), xredis.GetDepositLogXChainBackfillStartBlockKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get start block: %v", err)
		return 0, 0, true
	}

	startBlock, err = strconv.ParseUint(value, 10, 64)
	if err != nil {
		xlog.Errorf("Failed to parse start block: %v", err)
		return 0, 0, true
	}

	value, err = xredis.GetRedisClient().Get(context.Background(), xredis.GetDepositLogXChainBackfillEndBlockKey()).Result()
	if err != nil {
		xlog.Errorf("Failed to get end block: %v", err)
		return 0, 0, true
	}

	endBlock, err = strconv.ParseUint(value, 10, 64)
	if err != nil {
		xlog.Errorf("Failed to parse end block: %v", err)
		return 0, 0, true
	}

	if startBlock > endBlock {
		xlog.Infof("Backfill completed (start: %d, end: %d)", startBlock, endBlock)
		return 0, 0, true
	}

	return startBlock, endBlock, false
}

func (wf *DepositLogXTracker) RunBackFillJob() {
	defer cutils.LogTime(time.Now(), "DEPOSIT_LOGX_BACKFILL")

	startBlock, endBlock, isPaused := wf.GetBackFillConfig()
	if isPaused {
		xlog.Infof("Backfill is paused, skipping backfill job")
		return
	}

	// For backfill, we need to check all mailboxes since we don't know which products (Ostrich or LogX) the events belong to
	results := wf.AttemptMarkDepositReceivedBackfill(wf.logxRpcClient, startBlock, endBlock)

	if len(results.Misses) > 9 {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Backfilled deposit Logx: %+v", results))
	}
}

func (wf *DepositLogXTracker) AttemptMarkDepositReceivedBackfill(client *ethclient.Client, startBlock uint64, endBlock uint64) FinaliseStats {
	// Get the latest block
	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		xlog.Errorf("Failed to get the latest block number for Deposit LogX Backfill indexer.. something wrong with rpc: %v", err)
		return FinaliseStats{Hits: -1, Misses: []string{}}
	}

	if startBlock > currentBlock {
		xlog.Warnf("Deposit LogX Backfill Indexer has reached the max block number %d , start block %d", currentBlock, startBlock)
		return FinaliseStats{Hits: -1, Misses: []string{}}
	}

	endBlockIncrement := uint64(5000)
	if os.Getenv("DEPOSIT_LOGX_BACKFILL_INCREMENT") != "" {
		endBlockIncrement, _ = strconv.ParseUint(os.Getenv("DEPOSIT_LOGX_BACKFILL_INCREMENT"), 10, 64)
	}

	// Respect the configured end block, don't exceed current block
	processingEndBlock := min(startBlock+endBlockIncrement, currentBlock, endBlock)

	xlog.Infof("Attempting to backfill deposit received for blocks %d - %d", startBlock, processingEndBlock)

	uniqueMailboxes := make(map[common.Address]bool)
	var mailboxAddresses []common.Address
	for _, mailboxAddr := range wf.mailBoxAddresses {
		if !uniqueMailboxes[mailboxAddr] {
			uniqueMailboxes[mailboxAddr] = true
			mailboxAddresses = append(mailboxAddresses, mailboxAddr)
		}
	}

	xlog.Infof("Backfilling across %d mailboxes: %v", len(mailboxAddresses), mailboxAddresses)

	query := ethereum.FilterQuery{
		Addresses: mailboxAddresses,
		Topics:    [][]common.Hash{{wf.eventSignatureHash}, {}},
		FromBlock: big.NewInt(int64(startBlock)),
		ToBlock:   big.NewInt(int64(processingEndBlock)),
	}

	// Query logs
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		xlog.Errorf("Failed to filter logs for deposit LogX backfill: %v", err)
		return FinaliseStats{Hits: 0, Misses: []string{}}
	}

	if len(logs) > 0 {
		for _, vLog := range logs {
			messageId := vLog.Topics[1].Hex()
			(&db.DepositWithdrawDB{}).MarkDepositReceivedByMessageId(messageId)
		}

		err := xclient.GlobalApiServerClient.FinaliseDeposits()
		if err != nil {
			xlog.Errorf("Failed to mark logx chain deposits as finalised during backfill. You can manually hit api and get the same work done: %v", err)
		}
	}

	wf.writeBackfillStartBlockToRedis(processingEndBlock + 1)

	return FinaliseStats{Hits: len(logs), Misses: []string{}}
}

func (wf *DepositLogXTracker) writeBackfillStartBlockToRedis(startBlockNumber uint64) {
	_, err := xredis.GetRedisClient().Set(context.Background(), xredis.GetDepositLogXChainBackfillStartBlockKey(), startBlockNumber, 0).Result()
	if err != nil {
		xlog.Errorf("Failed to write backfill start block to redis: %v", err)
		return
	}

	xlog.Infof("Successfully wrote backfill start block %d to redis", startBlockNumber)
}
