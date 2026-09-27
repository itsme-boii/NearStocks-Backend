package tests

import (
	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"
)

func TestClientConnect(t *testing.T) {
	testutils.SetupContractEnv()
	contractUtils.Init()
	contract.Init()

	// nSubmission, err := contract.GlobalContracts.EndpointContract.GetNSubmissions()
	// if err != nil {
	// 	t.Fatalf("Error getting nSubmission: %v", err)
	// }
	// fmt.Printf("nSubmission: %v\n", nSubmission)

	// // GET NONCE:
	// nonce, err := contract.GlobalContracts.EndpointContract.GetNonceForSubaccount("0x000000000001d37eD507cA37Faa17079bFf5e46DcedE951577dB000000000001")
	// if err != nil {
	// 	t.Fatalf("Error getting nonce: %v", err)
	// }

	// fmt.Printf("nonce: %v\n", nonce)
}
