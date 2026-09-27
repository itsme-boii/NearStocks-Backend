package contract

import (
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"time"
)

func AddTransactionToBatchDb(subAccountHex1 string, subAccountHex2 string, transaction []byte, signature1 []byte, signature2 []byte, functionName string, transactionCounter uint, skipMaxTxnCounterCheck ...bool) error {
	start := time.Now()

	brokerId := getBrokerIdFromSubaccountId(subAccountHex1, subAccountHex2)

	// If simulation is successful, proceed to add the transaction to the actual database
	var insertedBatch *db.BatchTable
	if len(skipMaxTxnCounterCheck) > 0 && skipMaxTxnCounterCheck[0] {
		insertedBatch = (&db.BatchDB{}).Insert(subAccountHex1, subAccountHex2, transaction, signature1, signature2, functionName, transactionCounter, brokerId, true)
	} else {
		insertedBatch = (&db.BatchDB{}).Insert(subAccountHex1, subAccountHex2, transaction, signature1, signature2, functionName, transactionCounter, brokerId)
	}

	elapsed := time.Since(start)
	xlog.Infof("Batching - Time taken: %v", elapsed.Seconds())
	if insertedBatch == nil {
		return fmt.Errorf("failed to insert transaction into the database")
	}

	return nil
}

// If both subaccountIDs are present then take non amm subaccount and extract brokerId from it
// If no subaccountID is present then take brokerId as 1
func getBrokerIdFromSubaccountId(subAccountHex1 string, subAccountHex2 string) uint {
	if subAccountHex1 != contractUtils.AMM_SUBACCOUNT_ID {
		return cutils.ExtractBrokerIdFromSubaccountHex(subAccountHex1)
	}
	return cutils.ExtractBrokerIdFromSubaccountHex(subAccountHex2)
}
