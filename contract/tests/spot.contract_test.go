package tests

import (
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/testutils"
)

func setupSpot() {
	testutils.SetupContractEnv()
	contract.Init()
}

// func TestGetSpotProductIds(t *testing.T) {
// 	t.Skip("Skipping test for now")
// 	setupSpot()

// 	expectedProductIds := contractUtils.ALL_SPOTS_ON_CONTRACT
// 	var expectedError error

// 	productIds, err := contract.GlobalContracts.SpotContract.GetProductIds()

// 	assert.Equal(t, expectedProductIds, productIds)
// 	assert.Equal(t, expectedError, err)
// }
