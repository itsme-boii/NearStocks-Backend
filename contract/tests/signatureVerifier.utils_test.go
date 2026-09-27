package tests

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Trader
const MASTER_PRIVATE_KEY_HEX = "1489bb0caf91df72de55eb13a611ca377e9631570d511016f8e00875fa616287"
const SIGNING_PRIVATE_KEY_HEX = "c830f4e7df734a8febffc683c27be57a6b2b020f28891ecb02f356abf53e8123"

// Solver
// const MASTER_PRIVATE_KEY_HEX = "a489bb0caf91df72de55eb13a611ca377e9631570d511016f8e00875fa616281"
// const SIGNING_PRIVATE_KEY_HEX = "1830f4e7df734a8febffc683c27be57a6b2b020f28891ecb02f356abf53e89e1"

// TEST SOLVER MAIN
// const MASTER_PRIVATE_KEY_HEX = "ba9b61d55217fa4cb9b67beb2ed659c15d138d17ae7cb87cf0b247dd1e429433"
// const SIGNING_PRIVATE_KEY_HEX = "ab30f4e7df734a8febffc683c27be57a6b2b020f28891ecb02f356abf53e8912"

func getSubaccountId() string {
	masterPk, err := crypto.HexToECDSA(MASTER_PRIVATE_KEY_HEX)
	if err != nil {
		panic(fmt.Sprintf("Error converting master private key to ECDSA: %v", err))
	}
	masterPubKey := crypto.PubkeyToAddress(masterPk.PublicKey).Hex()
	return fmt.Sprintf("1_%s_1", masterPubKey)
}

type AuthPayload struct {
	SubaccountId     string `json:"subaccountId"`
	ChainId          uint   `json:"chainId"`
	Nonce            uint   `json:"nonce"`
	SigningKey       string `json:"signingKey"`
	SigningSignature string `json:"signingSignature"`
	EthAddress       string `json:"ethAddress"`
	EthSignature     string `json:"ethSignature"`
	ExpiryTs         uint64 `json:"expiryTs"`
}

func TestVerifySessionKeyCreation(t *testing.T) {
	subaccountId := getSubaccountId()
	fmt.Printf("Subaccount ID: %s\n", subaccountId)

	next7Day := (time.Now().UnixMilli()) + 7*cutils.DAY_MILLI
	fmt.Printf("Subaccount ID: %s\n", subaccountId)

	payload, err := contractUtils.BuildSessionKeySignaturePayload(MASTER_PRIVATE_KEY_HEX, SIGNING_PRIVATE_KEY_HEX, contractUtils.BuildSessionKeyRequest{
		SubaccountId: subaccountId,
		ExpiryTs:     next7Day,
		ChainId:      1,
		Nonce:        0,
	})

	if err != nil {
		t.Fatalf("Error building session key signature payload: %v", err)
	}

	authPayload := AuthPayload{
		SubaccountId:     subaccountId,
		ChainId:          uint(payload.ChainId),
		Nonce:            uint(payload.Nonce),
		SigningKey:       payload.SigningAddress,
		SigningSignature: payload.SigningSignature,
		EthAddress:       payload.EthAddress,
		EthSignature:     payload.EthSignature,
		ExpiryTs:         uint64(payload.ExpiryTs),
	}

	bytes, _ := json.Marshal(authPayload)
	fmt.Printf("Payload: %s\n", string(bytes))

	if err := contractUtils.VerifySessionkeyCreation(*payload); err != nil {
		t.Fatalf("Error verifying session key creation: %v", err)
	}
}

func TestVerifyOrderSignature(t *testing.T) {
	subaccountId := "1_0xa7De990d10A7D8b53476525B1212Bacf645Cb7b0_1"
	// fmt.Printf("Subaccount ID: %s\n", subaccountId)
	// next30Day := (time.Now().UnixMilli()) + 30*cutils.DAY_MILLI
	priceX18, _ := new(big.Int).SetString("3508760000000000000000", 10)
	fmt.Printf("priceX18 %v\n", priceX18)
	amountX18, _ := new(big.Int).SetString("1000000000000000000", 10)
	fmt.Printf("amountX18 %v\n", amountX18)
	productId := big.NewInt(1)
	fmt.Printf("product %v\n", productId)

	payload, err := contractUtils.BuildOrderSignaturePayload(SIGNING_PRIVATE_KEY_HEX, contractUtils.BuildOrderSigPayload{
		SubAccountId: subaccountId,
		PriceX18:     priceX18,
		Amount:       amountX18,
		Expiration:   uint64(1718353828239),
		IsReduce:     false,
		ChainId:      1,
		ProductId:    big.NewInt(1),
	})

	if err != nil {
		t.Fatalf("Error building order signature payload: %v", err)
	}

	bytes, _ := json.Marshal(payload)
	fmt.Printf("Payload: %s\n", string(bytes))

	if err := contractUtils.VerifyOrderSignature(*payload); err != nil {
		t.Fatalf("Error verifying order signature: %v", err)
	}
}

func TestVerifySettleUserPnl(t *testing.T) {
	subaccountId := getSubaccountId()
	// fmt.Printf("Subaccount ID: %s\n", subaccountId)

	payload, err := contractUtils.BuildVerifySettleUserPnlPayload(SIGNING_PRIVATE_KEY_HEX, contractUtils.BuildSettleUserPnlPayload{
		SubAccountId: subaccountId,
		Nonce:        1,
		ChainId:      1,
	})

	if err != nil {
		t.Fatalf("Error building settle user pnl payload: %v", err)
	}

	bytes, _ := json.Marshal(payload)
	fmt.Printf("Payload: %s\n", string(bytes))

	if err := contractUtils.VerifySettleUserPnl(*payload); err != nil {
		t.Fatalf("Error verifying settle user pnl: %v", err)
	}
}
