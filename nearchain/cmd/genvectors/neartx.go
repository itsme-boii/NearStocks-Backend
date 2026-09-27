package main

// Signed NEAR transaction vector: Go nearchain encoding must equal @near-js/transactions.

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"

	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/mr-tron/base58"
)

func genNearTx(dir string) {
	seed := sha256.Sum256([]byte("near-stocks vector treasury key 1")) // throwaway test key
	signer := nearchain.NewLocalSigner(ed25519.NewKeyFromSeed(seed[:]))
	block := sha256.Sum256([]byte("block"))
	storageArgs, _ := json.Marshal(map[string]any{"account_id": "alice.near", "registration_only": true})
	ftArgs, _ := json.Marshal(map[string]string{"receiver_id": "alice.near", "amount": "1234567"})
	minStorage, _ := new(big.Int).SetString("1250000000000000000000", 10)

	tx := &nearchain.Transaction{
		SignerId: "treasury.near-stocks.near", PublicKey: signer.PublicKey(), Nonce: 123456789012,
		ReceiverId: nearchain.USDCMainnet, BlockHash: block,
		Actions: []nearchain.Action{
			nearchain.FunctionCall{MethodName: "storage_deposit", Args: storageArgs, Gas: 10 * nearchain.TGas, Deposit: minStorage},
			nearchain.FunctionCall{MethodName: "ft_transfer", Args: ftArgs, Gas: 20 * nearchain.TGas, Deposit: big.NewInt(1)},
		},
	}
	signed, hash, err := nearchain.SignTransaction(tx, signer)
	if err != nil {
		log.Fatal(err)
	}
	write(dir, "neartx.json", map[string]any{
		"spec":                "borsh(TransactionV0) per nearcore transaction.rs; FunctionCall = action 2",
		"privateSeedTestOnly": hx(seed[:]),
		"publicKey":           nearchain.PublicKeyString(signer.PublicKey()),
		"signerId":            tx.SignerId,
		"receiverId":          tx.ReceiverId,
		"nonce":               "123456789012",
		"blockHash":           base58.Encode(block[:]),
		"actions": []map[string]string{
			{"methodName": "storage_deposit", "argsJson": string(storageArgs), "gas": "10000000000000", "deposit": minStorage.String()},
			{"methodName": "ft_transfer", "argsJson": string(ftArgs), "gas": "20000000000000", "deposit": "1"},
		},
		"txBorshHex":   hx(tx.Encode()),
		"txHash":       base58.Encode(hash[:]),
		"signedBase64": base64.StdEncoding.EncodeToString(signed),
	})
}
