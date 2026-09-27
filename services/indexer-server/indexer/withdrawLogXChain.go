package indexer

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
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

type WithdrawalsLogXTracker struct {
	logxRpcClient      *ethclient.Client // url -> rpcClient
	eventSignatureHash common.Hash
}

func NewWithdrawalsLogXReceived() *WithdrawalsLogXTracker {
	return &WithdrawalsLogXTracker{}
}

func (wf *WithdrawalsLogXTracker) Init() {
	// Initialize the RPC clients
	var err error
	wf.logxRpcClient, err = ethclient.Dial(contractUtils.RPC_URL)
	if err != nil {
		xlog.Errorf("Failed to connect to the Ethereum client: %v", err)
	}
	wf.eventSignatureHash = wf.GetEventSignature()
}

type LogXWithdrawReceievedStats struct {
	Hits   int
	Misses []string
	Error  string
}

func (wf *WithdrawalsLogXTracker) Run() {
	withdrawLogxChainInterval := os.Getenv("WITHDRAW_LOGX_INTERVAL")
	if withdrawLogxChainInterval == "" {
		withdrawLogxChainInterval = "*/10 * * * * *"
	} else {
		xlog.Infof("Using custom interval for logx chain withdrawal: %s", withdrawLogxChainInterval)
	}

	c := cron.New(cron.WithSeconds())
	for productId := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		_, err := c.AddFunc(withdrawLogxChainInterval, func() {
			wf.RunJob(productId)
		})
		if err != nil {
			xlog.Errorf("Failed to add job for product %d: %v", productId, err)
		}
	}

	c.Start()
}

func (wf *WithdrawalsLogXTracker) RunJob(productId uint32) {
	defer cutils.LogTime(time.Now(), fmt.Sprintf("WITHDRAW_LOGX_CHAIN: %v", productId))

	limit := 0
	if os.Getenv("WITHDRAW_LOGX_LIMIT") != "" {
		limit, _ = strconv.Atoi(os.Getenv("WITHDRAW_LOGX_LIMIT"))
	}
	if limit == 0 {
		limit = 200
	}

	messageIdsToReceive := (&db.DepositWithdrawDB{}).GetFirstXUnreceivedWithdrawalsMessageIdsForProductId(productId, limit)
	if len(messageIdsToReceive) == 0 {
		xlog.Infof("No logx chain withdrawals to receive for product %d", productId)
		return
	}

	contractAddress := common.HexToAddress(contractUtils.CLEARING_HOUSE)
	results := wf.AttemptReceivedWithdrawals(messageIdsToReceive, wf.logxRpcClient, contractAddress, productId)

	xlog.Infof("Received %+v withdrawals on logx chain for product %d", results, productId)
	if len(results.Misses) > 9 || results.Error != "" {
		xclient.GlobalDiscordClient.SendWebhookMessage(fmt.Sprintf("Received %+v withdrawals for product %d | err : %v", results, productId, results.Error))
	}
}

func (*WithdrawalsLogXTracker) GetEventSignature() common.Hash {
	// Generate the event signature hash for "ProcessId(bytes32)"
	eventSignature := []byte("WithdrawToken(bytes32,bytes32,bytes,int128,int128,uint32)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(eventSignature)
	return common.BytesToHash(hash.Sum(nil))
}

func (wf *WithdrawalsLogXTracker) AttemptReceivedWithdrawals(messageIds []string, client *ethclient.Client, contractAddress common.Address, productId uint32) LogXWithdrawReceievedStats {
	// Get the latest block
	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		xlog.Errorf("Failed to get the latest block number for Finalise Withdraw indexer.. something wrong with rpc of product: %v | err: %v", productId, err)
		// Making TotalFetched to -1 to undertand that this was a major failure
		return LogXWithdrawReceievedStats{Hits: -1, Misses: messageIds, Error: fmt.Sprintf("Failed to get latest block number: %v", err)}
	}

	// Process and receive unreceived withdrawals
	messageIdsHased := make([]common.Hash, len(messageIds))
	for i, messageId := range messageIds {
		messageIdsHased[i] = common.HexToHash(messageId)
	}

	// Create the filter query
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{wf.eventSignatureHash}, nil, nil, messageIdsHased},
		FromBlock: big.NewInt(int64(currentBlock - 10000)),
		ToBlock:   big.NewInt(int64(currentBlock)),
	}

	// Query logs
	logs, err := client.FilterLogs(context.Background(), query)
	if err != nil {
		msg := fmt.Sprintf("Failed to filter WITHDRAW LOGX CHAIN logs for messages (first 10): %v | err: %v", messageIds[:min(10, len(messageIds))], err)
		xlog.Errorf(msg)
		return LogXWithdrawReceievedStats{Hits: 0, Misses: messageIds, Error: msg}
	}

	messageIdsMissed := map[string]bool{}
	for _, v := range messageIds {
		messageIdsMissed[v] = true
	}

	// Receive withdrawals in the database if logs are found
	if len(logs) > 0 {
		for _, vLog := range logs {
			newMessageId := vLog.Topics[1].Hex()
			oldMessageId := vLog.Topics[3].Hex()
			(&db.DepositWithdrawDB{}).MarkWithdrawReceivedByMessageId(oldMessageId, newMessageId, vLog.TxHash.Hex(), uint64(vLog.TxIndex), vLog.BlockNumber)
			delete(messageIdsMissed, oldMessageId)
		}
	}

	return LogXWithdrawReceievedStats{Hits: len(logs), Misses: cutils.MapKeys(messageIdsMissed)}
}
