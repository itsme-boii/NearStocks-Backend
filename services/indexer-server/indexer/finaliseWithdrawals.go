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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/robfig/cron/v3"
	"golang.org/x/crypto/sha3"
)

type WithdrawalsFinalizer struct {
	rpcClients         map[string]*ethclient.Client // url -> rpcClient
	eventSignatureHash common.Hash
	lastReadBlockMap   map[uint32]uint64 // productId -> blockNumber
	mapLock            sync.RWMutex
}

func NewWithdrawalsFinalizer() *WithdrawalsFinalizer {
	return &WithdrawalsFinalizer{}
}
func (wf *WithdrawalsFinalizer) setLastReadBlockFromRedis() {
	// Get the max block from redis
	values, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetDestinationChainTrackerLastBlock()).Result()
	if err != nil {
		xlog.Errorf("Failed to get last block from redis: %v", err)
		return
	}
	result := make(map[uint32]uint64)
	for key, value := range values {
		productId, err := strconv.ParseUint(key, 10, 32)
		if err != nil {
			xlog.Errorf("Failed to parse product id: %v", err)
			continue
		}
		blockNumber, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			xlog.Errorf("Failed to parse block number: %v", err)
			continue
		}
		result[uint32(productId)] = blockNumber
	}

	wf.lastReadBlockMap = result
}

// TODO: Optimization - Instead of product wise run the job chain wise
func (wf *WithdrawalsFinalizer) Init() {
	// Initialize the RPC clients
	rpcClients := make(map[string]*ethclient.Client)
	for _, sourceChainInfo := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		if _, ok := rpcClients[sourceChainInfo.RPC_URL]; ok {
			continue
		}

		client, err := ethclient.Dial(sourceChainInfo.RPC_URL)
		if err != nil {
			xlog.Errorf("Failed to connect to the Ethereum client: %v", err)
			continue
		}
		rpcClients[sourceChainInfo.RPC_URL] = client
	}
	wf.rpcClients = rpcClients
	wf.eventSignatureHash = wf.GetEventSignature()
	wf.setLastReadBlockFromRedis() // set last read block from redis when starting service
}

type FinaliseStats struct {
	Hits   int
	Misses []string
}

func (wf *WithdrawalsFinalizer) Run() {
	withdrawFinaliserInterval := os.Getenv("WITHDRAW_SOURCE_INTERVAL")
	if withdrawFinaliserInterval == "" {
		withdrawFinaliserInterval = "*/20 * * * * *"
	} else {
		xlog.Infof("Using custom interval for withdrawal source job: %s", withdrawFinaliserInterval)
	}

	c := cron.New(cron.WithSeconds())
	for productId := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		xlog.Debugf("Adding WITHDRAW SOURCE job for product %d", productId)
		_, err := c.AddFunc(withdrawFinaliserInterval, func() {
			wf.RunJob(productId)
			go cerrors.WithPanicRecover(func() { wf.RunBackFillJob(productId) })
		})
		if err != nil {
			xlog.Errorf("Failed to add job for product %d: %v", productId, err)
		}
	}

	c.Start()
}

func (wf *WithdrawalsFinalizer) RunJob(productId uint32) {
	defer cutils.LogTime(time.Now(), fmt.Sprintf("WITHDRAW_SOURCE: %v", productId))

	client := wf.rpcClients[contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId].RPC_URL]
	contractAddress := common.HexToAddress(contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId].MailboxAddress)

	// read a startBlock and pass to the function underneath
	startBlock := wf.lastReadBlockMap[productId] + 1 // Can be done better: if this is zero fetch latest and start from (latest - some big value)
	pendingMessageCounts := (&db.DepositWithdrawDB{}).GetPendingUnfinalizedWithdrawalsCount(productId)
	results := wf.AttemptFinalisedWithdrawals(pendingMessageCounts > 0, client, contractAddress, productId, startBlock, false)

	msg := fmt.Sprintf("Backfilled finalized %d withdrawals for product %d | pre-execution pending message counts: %v", results.Hits, productId, pendingMessageCounts)
	xlog.Infof(msg)
	if pendingMessageCounts > 10 {
		xclient.GlobalDiscordClient.SendWebhookMessage(msg)
	}
}

func (wf *WithdrawalsFinalizer) GetBackFillConfig(productId uint32) (startBlock uint64, endBlock uint64, isPaused bool) {
	value, err := xredis.GetRedisClient().HGet(context.Background(), xredis.GetDestinationChainBackfillKey(productId), xredis.GetDestinationChainBackfillStartField()).Result()
	if err != nil {
		return 0, 0, true
	}

	startBlock, err = strconv.ParseUint(value, 10, 64)
	if err != nil {
		xlog.Errorf("Failed to parse start block for product %d: %v", productId, err)
		return 0, 0, true
	}

	value, err = xredis.GetRedisClient().HGet(context.Background(), xredis.GetDestinationChainBackfillKey(productId), xredis.GetDestinationChainBackfillEndField()).Result()
	if err != nil {
		return 0, 0, true
	}

	endBlock, err = strconv.ParseUint(value, 10, 64)
	if err != nil {
		xlog.Errorf("Failed to parse end block for product %d: %v", productId, err)
		return 0, 0, true
	}

	if startBlock > endBlock {
		return 0, 0, true
	}

	return startBlock, endBlock, false
}

func (wf *WithdrawalsFinalizer) RunBackFillJob(productId uint32) {
	defer cutils.LogTime(time.Now(), fmt.Sprintf("WITHDRAW_SOURCE_BACKFILL: %v", productId))

	client := wf.rpcClients[contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId].RPC_URL]
	contractAddress := common.HexToAddress(contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId].MailboxAddress)

	// Get the last read block from redis
	startBlock, _, isPaused := wf.GetBackFillConfig(productId)
	if isPaused {
		xlog.Infof("Backfill is paused for product %d, skipping backfill job", productId)
		return
	}

	pendingMessageCounts := (&db.DepositWithdrawDB{}).GetPendingUnfinalizedWithdrawalsCount(productId)
	results := wf.AttemptFinalisedWithdrawals(pendingMessageCounts > 0, client, contractAddress, productId, startBlock, true)

	msg := fmt.Sprintf("Backfilled finalized %d withdrawals for product %d | pre-execution pending message counts: %v", results.Hits, productId, pendingMessageCounts)
	xlog.Infof(msg)
	if pendingMessageCounts > 10 {
		xclient.GlobalDiscordClient.SendWebhookMessage(msg)
	}
}

func (*WithdrawalsFinalizer) GetEventSignature() common.Hash {
	// Generate the event signature hash for "ProcessId(bytes32)"
	eventSignature := []byte("ProcessId(bytes32)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(eventSignature)
	return common.BytesToHash(hash.Sum(nil))
}

func (wf *WithdrawalsFinalizer) AttemptFinalisedWithdrawals(messageExists bool, client *ethclient.Client, contractAddress common.Address, productId uint32, startBlock uint64, isBackFillJob bool) FinaliseStats {
	// Get the latest block
	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		xlog.Errorf("Failed to get the latest block number for Finalise Withdraw indexer.. something wrong with rpc of product: %v | err: %v", productId, err)
		// Making TotalFetched to -1 to undertand that this was a major failure
		return FinaliseStats{Hits: -1, Misses: nil}
	}

	if startBlock > currentBlock {
		xlog.Warnf("Finalise destination Indexer has reached the max block number for product id: %v,", productId) // this can happen when message IDS are not realyed -> check relayer if its working
		return FinaliseStats{Hits: -1, Misses: nil}
	}

	endBlockIncrement := uint64(499)
	endBlock := min(startBlock+endBlockIncrement, currentBlock)

	xlog.Infof("Attempting to finalize withdrawal for product %d and blocks %d - %d", productId, startBlock, endBlock)

	result := FinaliseStats{Hits: 0, Misses: nil}
	if messageExists {
		query := ethereum.FilterQuery{
			Addresses: []common.Address{contractAddress},
			Topics:    [][]common.Hash{{wf.eventSignatureHash}, {}},
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
		}

		// Query logs
		logs, err := client.FilterLogs(context.Background(), query)
		if err != nil {
			xlog.Errorf("Failed to filter WITHDRAW SOURCE logs for contract address: %v | productId: %v | err: %v", contractAddress, productId, err)
			return FinaliseStats{Hits: 0, Misses: nil}
		}

		// Finalize withdrawals in the database if logs are found
		if len(logs) > 0 {
			// xlog.Debugf("Found %d logs for product %d in block range %d - %d | %+v", len(logs), productId, startBlock, endBlock, logs)
			for _, vLog := range logs {
				messageId := vLog.Topics[1].Hex()
				(&db.DepositWithdrawDB{}).MarkWithdrawalFinalisedByMessageId(messageId, vLog.TxHash.Hex(), uint64(vLog.Index), vLog.BlockNumber)
			}
		}

		result = FinaliseStats{Hits: len(logs), Misses: nil}
	}

	if !isBackFillJob {
		wf.writeLastReadBlockToRedis(productId, endBlock) // update last read block to redis - persistent storage not in memory in case or service restart etc

		wf.mapLock.Lock()
		defer wf.mapLock.Unlock()
		wf.lastReadBlockMap[productId] = endBlock // update last read block with endBlock - local in memory update
	} else {
		wf.writeBackfillStartBlockToRedis(productId, endBlock+1)
	}

	return result
}

func (wf *WithdrawalsFinalizer) writeLastReadBlockToRedis(productId uint32, lastBlockNumber uint64) {
	_, err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetDestinationChainTrackerLastBlock(), xredis.GetSourceChainTrackerLastBlockField(productId), strconv.FormatUint(lastBlockNumber, 10)).Result()
	if err != nil {
		xlog.Errorf("Failed to write last read block to redis while finalising: %v", err)
	}
}

func (wf *WithdrawalsFinalizer) writeBackfillStartBlockToRedis(productId uint32, startBlockNumber uint64) {
	_, err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetDestinationChainBackfillKey(productId), xredis.GetDestinationChainBackfillStartField(), strconv.FormatUint(startBlockNumber, 10)).Result()
	if err != nil {
		xlog.Errorf("Failed to write backfill start block to redis for product %d: %v", productId, err)
		return
	}

	xlog.Infof("Successfully wrote backfill start block %d for product %d to redis", startBlockNumber, productId)
}
