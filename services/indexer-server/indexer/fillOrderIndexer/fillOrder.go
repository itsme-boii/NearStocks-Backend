package fillOrderIndexer

// NOTE NOTE NOTE NOTE NOTE: This is the newer version

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"strconv"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/contract/gen2"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type UserPnListener struct {
	client           *ethclient.Client
	fillOrderDB      *db.FillOrderDB
	maxBlockLocal    uint64
	retries          int
	isError          bool
	errorBlock       uint64
	maxRetries       int
	lazyTimeout      time.Duration
	historyBatchSize uint64
	startBlock       uint64
	_filterer        *gen2.OffChainExchangeFilterer
}

func NewUserFillOrderListener(client *ethclient.Client) *UserPnListener {
	fillOrderDB := &db.FillOrderDB{}
	_filterer, err := gen2.NewOffChainExchangeFilterer(common.HexToAddress(contractUtils.OFFCHAIN_CONTRACT_ADDRESS), client)
	if err != nil {
		log.Fatalf("Failed to create new off-chain exchange filterer: %v", err)
	}

	indexer := &UserPnListener{
		client:           client,
		fillOrderDB:      fillOrderDB,
		retries:          5,
		maxRetries:       3,
		lazyTimeout:      10 * time.Second,
		historyBatchSize: 100,
		startBlock:       0,
		_filterer:        _filterer,
	}
	if os.Getenv("START_BLOCK_NUMBER") != "" {
		indexer.maxBlockLocal, _ = strconv.ParseUint(os.Getenv("START_BLOCK_NUMBER"), 10, 64)
	} else {
		indexer.maxBlockLocal = fillOrderDB.GetMaxBlockNumber()
	}

	return indexer
}

func (indexer *UserPnListener) StartFillOrderSettledJob() {
	for {
		if indexer.retries > 0 {
			indexer.fetchAndStoreEvents()
			time.Sleep(indexer.lazyTimeout)
		} else {
			log.Println("Max retries exceeded. Stopping indexer.")
			break
		}
	}
}

func (indexer *UserPnListener) fetchAndStoreEvents() {
	log.Println("Starting to fetch and store UserPnLSettled events...")

	var currentBlock uint64
	if os.Getenv("MAX_BLOCK_NUMBER") != "" {
		currentBlock, _ = strconv.ParseUint(os.Getenv("MAX_BLOCK_NUMBER"), 10, 64)
	} else {
		var err error
		currentBlock, err = indexer.client.BlockNumber(context.Background())

		if err != nil {
			xlog.Errorf("Failed to get current block number: %v. Not events will be fetched.", err)
			return
		}
	}

	startBlock := indexer.maxBlockLocal
	if indexer.isError {
		startBlock = indexer.errorBlock
		indexer.isError = false
	}

	endBlock := startBlock + indexer.historyBatchSize - 1
	endBlock = min(endBlock, currentBlock)
	startBlock = min(startBlock, endBlock)

	if startBlock < 1 {
		xlog.Infof("No start block set. Not events will be fetched.")
		return
	}

	log.Printf("Fetching events from block %d to block %d...", startBlock, endBlock)

	itr, err := indexer._filterer.FilterFillOrder(&bind.FilterOpts{Context: context.Background(), Start: startBlock, End: &endBlock}, nil, nil, nil)
	if err != nil {
		xlog.Errorf("Failed to filter fill orders due to error %v", err)
		indexer.handleError(startBlock)
		return
	}

	defer itr.Close()

	count := 0

	for itr.Next() {
		event := itr.Event
		settlePnLEvent := db.FillOrderTable{
			BrokerID:     cutils.ExtractBrokerIdFromSubaccountHex("0x" + hex.EncodeToString(event.Subaccount[:])),
			TxnHash:      event.Raw.TxHash.String(),
			Index:        event.Raw.Index,
			UserAddress:  event.User.String(),
			ProductID:    uint(event.ProductId),
			SubAccountID: "0x" + hex.EncodeToString(event.Subaccount[:]),
			PriceX18:     ctypes.NewBigInt(event.PriceX18),
			Amount:       ctypes.NewBigInt(event.Amount),
			IsTaker:      event.IsTaker,
			FeeAmount:    ctypes.NewBigInt(event.FeeAmount),
			BaseDelta:    ctypes.NewBigInt(event.BaseDelta),
			QuoteDelta:   ctypes.NewBigInt(event.QuoteDelta),
			RealisedPnl:  ctypes.NewBigInt(event.RealisedPnl),
			FundingFees:  ctypes.NewBigInt(event.FundingFees),
			BlockNumber:  event.Raw.BlockNumber,
			Signature:    event.Signature,
		}

		v, err := indexer.fillOrderDB.CreateFillOrder(settlePnLEvent)
		if err != nil {
			xlog.Errorf("Error storing Fills order event: %v", err)
			indexer.handleError(startBlock)
			return
		}
		if v != nil {
			count++
		}
	}

	if itr.Error() != nil {
		xlog.Errorf("Failed to query fills order events: %v | Processed %v events", itr.Error(), count)
		indexer.handleError(startBlock)
		return
	}

	xlog.Infof("Successfully processed %v events from block %d to block %d.", count, startBlock, endBlock)

	indexer.maxBlockLocal = endBlock
	indexer.isError = false
	indexer.retries = indexer.maxRetries
	xlog.Infof("Events stored successfully.")
}

func (indexer *UserPnListener) handleError(startBlock uint64) {
	indexer.isError = true
	indexer.errorBlock = startBlock
	indexer.retries--
	if indexer.retries > 0 {
		log.Printf("Retrying... attempts left: %d", indexer.retries)
		time.Sleep(indexer.lazyTimeout)
	}
}
