package contractUtils

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func BuildTransactionArg(functionNumber uint32, encodedStruct []byte) []byte {
	transaction := make([]byte, 1+len(encodedStruct))
	transaction[0] = byte(functionNumber)
	copy(transaction[1:], encodedStruct)

	return transaction
}

func BuildDynamicTransactionArg(functionNumber uint32, encodedStruct []byte) []byte {
	transaction := make([]byte, 1+32+len(encodedStruct))
	transaction[0] = byte(functionNumber)
	padding := make([]byte, 32)
	padding[31] = 0x20
	copy(transaction[1:], padding)
	copy(transaction[1+32:], encodedStruct)

	return transaction
}

func GetAddressFromPrivateKey(privateKey string) string {
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(fmt.Sprintf("Error converting private key to ECDSA: %v", err))
	}
	return crypto.PubkeyToAddress(pk.PublicKey).Hex()
}

func IsAMMAccount(subaccountHex string) bool {
	return strings.EqualFold(AMM_SUBACCOUNT_ID, subaccountHex)
}
