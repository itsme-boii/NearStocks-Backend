package indexer

import (
	"context"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/contract/gen"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/robfig/cron/v3"
	"github.com/status-im/keycard-go/hexutils"

	"golang.org/x/crypto/sha3"
)

// Current implementation -
// 1. What we are doing here is looking on source chain for any deposits that have been made to the mailbox contract
// 2. We take the transaction hash of the deposit and mark the deposit as received in the database
// 3. Unique identifier for the deposit is the message id

type DepositSourceTracker struct {
	rpcClients          map[string]*ethclient.Client // url -> rpcClient
	filterers           map[uint32]*gen.HypErcFilterer
	lastFetchedBlockMap map[uint32]uint64 // productId -> blockNumber
	mapLock             sync.RWMutex
}

func NewDepositSourceTracker() *DepositSourceTracker {
	return &DepositSourceTracker{}
}

func (wf *DepositSourceTracker) setLastBlockFromRedis() {
	// Get the max block from redis
	values, err := xredis.GetRedisClient().HGetAll(context.Background(), xredis.GetSourceChainTrackerLastBlock()).Result()
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
	wf.lastFetchedBlockMap = result
}

func (wf *DepositSourceTracker) Init() {
	// Initialize the RPC clients
	xlog.Debugf("Initialising DepositSourceTracker")

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

	filterers := make(map[uint32]*gen.HypErcFilterer)
	for productId, sourceChainInfo := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		if _, ok := wf.rpcClients[sourceChainInfo.RPC_URL]; ok {
			address := common.HexToAddress(contractUtils.SOURCE_HYPERC_ADDRESS_MAP[uint(productId)])
			var err error
			filterers[productId], err = gen.NewHypErcFilterer(address, wf.rpcClients[sourceChainInfo.RPC_URL])
			if err != nil {
				xlog.Errorf("Failed to create new filterer: %v", err)
			}
		} else {
			xlog.Errorf("RPC client not found for product %d", productId)
		}
	}

	wf.filterers = filterers
	wf.setLastBlockFromRedis()
}

type InsertDepositStats struct {
	LatestBlock  uint64
	TotalFetched int
}

func (wf *DepositSourceTracker) Run() {
	depositSourceTrackerInterval := os.Getenv("DEPOSIT_SOURCE_INTERVAL")
	if depositSourceTrackerInterval == "" {
		depositSourceTrackerInterval = "*/30 * * * * *"
	}

	xlog.Infof("Starting deposit source tracker with interval: %s", depositSourceTrackerInterval)

	c := cron.New(cron.WithSeconds())
	for productId, _ := range contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO {
		xlog.Debugf("Adding job for product %d", productId)
		_, err := c.AddFunc(depositSourceTrackerInterval, func() {
			wf.RunJob(productId)
		})
		if err != nil {
			xlog.Errorf("Failed to add job for product %d: %v", productId, err)
		}
	}

	c.Start()
	select {}
}

type ExecuteSingleJob struct {
	StartBlock      uint64
	SkipBlockWrites bool
}

// NOTE: Init must be called before - ExecuteSingle
func (wf *DepositSourceTracker) ExecuteSingle(productId uint32, startBlock uint64) error {
	xlog.Infof("Running deposit source tracker for product %d with start block %d", productId, startBlock)
	sourceChainInfo, ok := contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId]
	if !ok {
		xlog.Errorf("Source chain info not found for product %d", productId)
		return fmt.Errorf("source chain info not found for product %d", productId)
	}

	client, err := ethclient.Dial(sourceChainInfo.RPC_URL)
	if err != nil {
		xlog.Errorf("Failed to connect to the Ethereum client: %v", err)
		return err
	}
	rpcClients := make(map[string]*ethclient.Client)
	rpcClients[sourceChainInfo.RPC_URL] = client
	wf.rpcClients = rpcClients

	filterers := make(map[uint32]*gen.HypErcFilterer)
	address := common.HexToAddress(contractUtils.SOURCE_HYPERC_ADDRESS_MAP[uint(productId)])
	filterer, err := gen.NewHypErcFilterer(address, client)
	if err != nil {
		xlog.Errorf("Failed to create new filterer: %v", err)
		return err
	}
	filterers[productId] = filterer
	wf.filterers = filterers

	wf.RunJob(productId, ExecuteSingleJob{StartBlock: startBlock, SkipBlockWrites: true})
	return nil
}

func (wf *DepositSourceTracker) writeLastBlockToRedis(productId uint32, lastBlockNumber uint64) {
	_, err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSourceChainTrackerLastBlock(), xredis.GetSourceChainTrackerLastBlockField(productId), strconv.FormatUint(lastBlockNumber, 10)).Result()
	if err != nil {
		xlog.Errorf("Failed to write last block to redis: %v", err)
	}
}

func (wf *DepositSourceTracker) RunJob(productId uint32, jobConfig ...ExecuteSingleJob) {
	defer cutils.LogTime(time.Now(), fmt.Sprintf("DEPOSIT_SOURCE_TRACKER: %v", productId))
	xlog.Debugf("Running deposit source tracker for product %d", productId)

	client := wf.rpcClients[contractUtils.PRODUCT_ID_TO_SOURCE_CHAIN_INFO[productId].RPC_URL]

	filterer, ok := wf.filterers[productId]
	if !ok {
		xlog.Errorf("Filterer not found for product %d", productId)
		return
	}

	chainId := contractUtils.ProductChainMapping[productId]

	var startBlock uint64
	var results InsertDepositStats

	if len(jobConfig) > 0 {
		startBlock = jobConfig[0].StartBlock
		results = wf.AttemptMarkDepositInsert(startBlock, filterer, client, productId, uint64(chainId), jobConfig[0].SkipBlockWrites)
	} else {
		// Regular cron job execution - refetch 4 blocks before the last fetched block to avoid missing any event
		startBlock = wf.lastFetchedBlockMap[productId] - 4
		results = wf.AttemptMarkDepositInsert(startBlock, filterer, client, productId, uint64(chainId))
	}

	xlog.Infof("Received deposits : %+v for product %d", results, productId)
	if results.TotalFetched < 0 {
		xclient.GetGlobalDiscordClient().SendWebhookMessage(fmt.Sprintf("Deposit source  %+v deposits for product %d", results, productId))
	}
}

func (*DepositSourceTracker) GetEventSignature() common.Hash {
	// Generate the event signature hash for "ProcessId(bytes32)"
	eventSignature := []byte("ProcessId(bytes32)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(eventSignature)
	return common.BytesToHash(hash.Sum(nil))
}

func (wf *DepositSourceTracker) AttemptMarkDepositInsert(_startBlock uint64, filterer *gen.HypErcFilterer, client *ethclient.Client, productId uint32, chainId uint64, skipBlockWrites ...bool) InsertDepositStats {
	// Get the latest block and set the start and end blocks accordingly
	currentBlock, err := client.BlockNumber(context.Background())
	if err != nil {
		xlog.Errorf("Failed to get the latest block number for deposit source indexer.. something wrong with rpc of product: %v | err: %v", productId, err)
		// Making TotalFetched to -1 to undertand that this was a major failure
		return InsertDepositStats{LatestBlock: _startBlock - 1, TotalFetched: -1}
	}

	if _startBlock > currentBlock {
		xlog.Warnf("Deposit source Indexer has reached the max block number for product id: %v", productId)
		return InsertDepositStats{LatestBlock: _startBlock - 1, TotalFetched: 0}
	}

	// Process up to 10,000 blocks at a time to avoid RPC timeouts
	endBlockIncrement := uint64(1000)
	if productId == 36 && os.Getenv("ENV") == "TESTNET" {
		endBlockIncrement = 100
	}

	endBlock := min(_startBlock+endBlockIncrement, currentBlock)
	startBlock := _startBlock

	// Query logs
	itr, err := filterer.FilterDepositToken(&bind.FilterOpts{Start: startBlock, End: &endBlock}, nil, nil, nil)
	if err != nil {
		xlog.Errorf("Failed to filter logs for product id: %v | from start block: %v | to end block: %v | err : %v", productId, startBlock, endBlock, err)
		// Making TotalFetched to -1 to undertand that this was a major failure
		return InsertDepositStats{LatestBlock: startBlock - 1, TotalFetched: -1}
	}

	defer itr.Close()

	count := 0
	for itr.Next() {
		event := itr.Event
		txnHash := event.Raw.TxHash.String()
		txnIndex := uint64(event.Raw.TxIndex)
		blockNumber := event.Raw.BlockNumber
		messageId := strings.ToLower("0x" + hexutils.BytesToHex(event.MessageId[:]))
		subaccountId := cutils.Bytes32ToSubaccountHex(event.SubaccountId)
		amount := event.Amount.String()

		// If the deposit already exists in the database, skip it and we won't log it in err below
		_, err := (&db.DepositWithdrawDB{}).InsertDeposit(txnHash, txnIndex, blockNumber, messageId, subaccountId, amount, productId, chainId)
		if err != nil {
			xlog.Errorf("Failed to insert deposit for txnHash: %v | txnIndex: %v | messageId: %v | blocknumber: %v | err: %v", txnHash, txnIndex, messageId, blockNumber, err)
			// WARNING: assumption made here is that logs will be in order of block number
			endBlock = blockNumber - 1
			break
		}
		count++
	}

	// Trigger finalization if deposits were found
	if count > 0 {
		err := xclient.GlobalApiServerClient.FinaliseDeposits()
		if err != nil {
			xlog.Errorf("Failed to trigger finalization: %v", err)
		}
	}

	if len(skipBlockWrites) == 0 || !skipBlockWrites[0] {
		if endBlock >= startBlock {
			wf.writeLastBlockToRedis(productId, endBlock)
		}

		wf.mapLock.Lock()
		defer wf.mapLock.Unlock()
		wf.lastFetchedBlockMap[productId] = endBlock
	}

	return InsertDepositStats{LatestBlock: endBlock, TotalFetched: count}
}
