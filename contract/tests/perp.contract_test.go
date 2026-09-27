package tests

import (
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/testutils"
)

func setup() {
	testutils.SetupContractEnv()
	contract.Init()
}

// func TestGetProductIds(t *testing.T) {
// 	setup()

// 	// Mock the expected product IDs
// 	expectedProductIds := contractUtils.ALL_PERPS_ON_CONTRACT

// 	// Mock the error to be returned by the GetProductIds function
// 	var expectedError error

// 	// Call the GetProductIds function
// 	productIds, err := contract.GlobalContracts.PerpContract.GetProductIds()

// 	// Assert that the returned product IDs match the expected product IDs
// 	assert.Equal(t, expectedProductIds, productIds)

// 	// Assert that the returned error matches the expected error
// 	assert.Equal(t, expectedError, err)
// }

// This is just for manual fetching
// func TestGetBalance(t *testing.T) {
// 	setup()

// 	// Call the GetBalance function
// 	balance, err := contract.GlobalContracts.PerpContract.GetAllBalancesOfSubaccounts([]string{"0x0000000000010de8653062ce5698bdaea713eb785cc096e39131000000000001"})
// 	if err != nil {
// 		t.Errorf("Failed to get balance: %v", err)
// 	}
// 	str, _ := json.Marshal(balance)

// 	xlog.Debugf("Balance: %v", string(str))
// }
