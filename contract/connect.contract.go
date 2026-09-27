package contract

import (
	"context"
	"errors"

	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Contracts struct {
	EndpointContract EndpointContract
}

var GlobalContracts Contracts

func Init() {
	GlobalContracts.EndpointContract = *NewEndpointContract()
}

// WaitnCheckTxnStatus checks the status of a transaction given its hash and an Ethereum client
func WaitnCheckTxnStatus(ctx context.Context, client *ethclient.Client, txnHash common.Hash) (bool, error) {
	for {
		receipt, err := client.TransactionReceipt(ctx, txnHash)
		if err == ethereum.NotFound {
			// Transaction is not yet mined, wait and retry
			select {
			case <-time.After(1 * time.Second):
				// continue loop
				continue
			case <-ctx.Done():
				// context timeout or cancellation
				return false, ctx.Err()
			}
		} else if err != nil {
			return false, err
		}

		if receipt == nil {
			return false, errors.New("Contract - received nil transaction receipt")
		}

		if receipt.Status == types.ReceiptStatusSuccessful {
			return true, nil
		} else {
			return false, nil
		}
	}
}
