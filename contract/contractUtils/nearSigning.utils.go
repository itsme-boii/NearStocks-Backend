package contractUtils

import (
	"fmt"
	"math/big"

	"github/eugenix-io/logx-inf-backend/libs/cutils"

	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// EIP-712 types signed by the browser session key under the near-stocks domain
// (Development.md §5.5). Pinned in vectors/eip712.json and checked in Go, Rust and TS.

var NearRegisterType = []apitypes.Type{
	{Name: "subAccountId", Type: "bytes32"},
	{Name: "userAddress", Type: "address"},
	{Name: "sessionKey", Type: "address"},
	{Name: "expiryTimeStamp", Type: "uint128"},
	{Name: "nonce", Type: "uint128"},
	{Name: "chainId", Type: "uint256"},
}

// NearWithdrawType binds the exact NEAR receiver (an account id, or a 1Click deposit address).
var NearWithdrawType = []apitypes.Type{
	{Name: "subAccountId", Type: "bytes32"},
	{Name: "sessionKey", Type: "address"},
	{Name: "productId", Type: "uint32"},
	{Name: "amount", Type: "uint128"},
	{Name: "nonce", Type: "uint128"},
	{Name: "receiver", Type: "string"},
	{Name: "chainId", Type: "uint256"},
}

type NearRegister struct {
	SubaccountId string // "broker_addr20_n"
	UserAddress  string // addr20
	SessionKey   string
	ExpiryTs     int64
	Nonce        int64
}

type NearWithdraw struct {
	SubaccountId string
	SessionKey   string
	ProductId    uint32
	Amount       string // x18
	Nonce        int64
	Receiver     string
}

func nearTypedDataHash(contractAccount string, chainId int64, primary string, fields []apitypes.Type, msg apitypes.TypedDataMessage) ([]byte, error) {
	td := apitypes.TypedData{
		Types:       apitypes.Types{"EIP712Domain": EIP712_DOMAIN_TYPE, primary: fields},
		PrimaryType: primary,
		Domain:      NearStocksDomain(contractAccount, chainId),
		Message:     msg,
	}
	hash, _, err := apitypes.TypedDataAndHash(td)
	return hash, err
}

func NearRegisterHash(contractAccount string, chainId int64, r NearRegister) ([]byte, error) {
	sub, err := cutils.SubaccountIdToBytes32(r.SubaccountId)
	if err != nil {
		return nil, err
	}
	return nearTypedDataHash(contractAccount, chainId, "Register", NearRegisterType, apitypes.TypedDataMessage{
		"subAccountId":    sub,
		"userAddress":     r.UserAddress,
		"sessionKey":      r.SessionKey,
		"expiryTimeStamp": math.NewHexOrDecimal256(r.ExpiryTs),
		"nonce":           math.NewHexOrDecimal256(r.Nonce),
		"chainId":         math.NewHexOrDecimal256(chainId),
	})
}

func NearWithdrawHash(contractAccount string, chainId int64, w NearWithdraw) ([]byte, error) {
	sub, err := cutils.SubaccountIdToBytes32(w.SubaccountId)
	if err != nil {
		return nil, err
	}
	amount, ok := new(big.Int).SetString(w.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return nil, fmt.Errorf("invalid amount %q", w.Amount)
	}
	return nearTypedDataHash(contractAccount, chainId, "NearWithdraw", NearWithdrawType, apitypes.TypedDataMessage{
		"subAccountId": sub,
		"sessionKey":   w.SessionKey,
		"productId":    math.NewHexOrDecimal256(int64(w.ProductId)),
		"amount":       amount.String(),
		"nonce":        math.NewHexOrDecimal256(w.Nonce),
		"receiver":     w.Receiver,
		"chainId":      math.NewHexOrDecimal256(chainId),
	})
}

// VerifyNearRegister proves the caller holds the session key it asks to register.
func VerifyNearRegister(contractAccount string, chainId int64, r NearRegister, signatureHex string) error {
	hash, err := NearRegisterHash(contractAccount, chainId, r)
	if err != nil {
		return err
	}
	return _VerifySignature(hash, signatureHex, r.SessionKey)
}

func VerifyNearWithdraw(contractAccount string, chainId int64, w NearWithdraw, signatureHex string) error {
	hash, err := NearWithdrawHash(contractAccount, chainId, w)
	if err != nil {
		return err
	}
	return _VerifySignature(hash, signatureHex, w.SessionKey)
}

// Request types kept from the current system (same field lists as the Verify* functions above);
// on NEAR they are signed under the near-stocks domain and hashed by the contract too
// (near-stocks-contracts/core/src/eip712.rs). Pinned in vectors/nearsign.json.
var (
	NearUserOptionBetType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "productId", Type: "uint32"}, {Name: "amount", Type: "int128"},
		{Name: "interval", Type: "uint32"}, {Name: "nonce", Type: "uint128"}, {Name: "sessionKey", Type: "address"}, {Name: "chainId", Type: "uint256"},
	}
	NearPoolOrderType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "productId", Type: "uint32"}, {Name: "amount", Type: "int128"},
		{Name: "isBuy", Type: "bool"}, {Name: "nonce", Type: "uint128"}, {Name: "sessionKey", Type: "address"}, {Name: "chainId", Type: "uint256"},
	}
	NearStakeType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "productId", Type: "uint32"}, {Name: "tokenAmount", Type: "int128"},
		{Name: "stakerContract", Type: "address"}, {Name: "sessionKey", Type: "address"}, {Name: "nonce", Type: "uint128"}, {Name: "chainId", Type: "uint256"},
	}
	NearUnstakeType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "productId", Type: "uint32"}, {Name: "amount", Type: "int128"},
		{Name: "stakerContract", Type: "address"}, {Name: "sessionKey", Type: "address"}, {Name: "nonce", Type: "uint128"}, {Name: "chainId", Type: "uint256"},
	}
	NearClaimRewardsType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "sessionKey", Type: "address"}, {Name: "stakerContract", Type: "address"},
		{Name: "productId", Type: "uint32"}, {Name: "nonce", Type: "uint128"}, {Name: "chainId", Type: "uint256"},
	}
	NearClaimLogXType = []apitypes.Type{
		{Name: "subAccountId", Type: "bytes32"}, {Name: "tokenAmount", Type: "int128"}, {Name: "sessionKey", Type: "address"},
		{Name: "nonce", Type: "uint128"}, {Name: "chainId", Type: "uint256"},
	}
)

// NearTypedHash is the EIP-712 digest of a request under the near-stocks domain. Integer message
// values are passed as decimal strings (apitypes mutates *big.Int arguments).
func NearTypedHash(contractAccount string, chainId int64, primary string, fields []apitypes.Type, msg apitypes.TypedDataMessage) ([]byte, error) {
	return nearTypedDataHash(contractAccount, chainId, primary, fields, msg)
}

// VerifyNearTyped checks that signatureHex over NearTypedHash(...) was made by signerHex.
func VerifyNearTyped(contractAccount string, chainId int64, primary string, fields []apitypes.Type, msg apitypes.TypedDataMessage, signatureHex, signerHex string) error {
	hash, err := NearTypedHash(contractAccount, chainId, primary, fields, msg)
	if err != nil {
		return err
	}
	return _VerifySignature(hash, signatureHex, signerHex)
}

// NearOrderType is the Order the engine's orders are signed with (VerifyOrderSignature's fields).
var NearOrderType = []apitypes.Type{
	{Name: "subAccountId", Type: "bytes32"}, {Name: "priceX18", Type: "int128"}, {Name: "amount", Type: "int128"},
	{Name: "expiration", Type: "uint64"}, {Name: "isReduce", Type: "bool"}, {Name: "sessionKey", Type: "address"},
	{Name: "chainId", Type: "uint256"}, {Name: "productId", Type: "uint32"},
}

// NearOrderDigest is the order's EIP-712 digest under the near-stocks domain: the key of the
// contract's filled_amounts (replay protection), which prune_filled removes once the order expires.
func NearOrderDigest(contractAccount string, chainId int64, sub [32]byte, priceX18, amount *big.Int, expiration uint64, isReduce bool, sessionKey string, productId uint32) ([]byte, error) {
	return nearTypedDataHash(contractAccount, chainId, "Order", NearOrderType, apitypes.TypedDataMessage{
		"subAccountId": sub[:],
		"priceX18":     priceX18.String(),
		"amount":       amount.String(),
		"expiration":   math.NewHexOrDecimal256(int64(expiration)),
		"isReduce":     isReduce,
		"sessionKey":   sessionKey,
		"chainId":      math.NewHexOrDecimal256(chainId),
		"productId":    math.NewHexOrDecimal256(int64(productId)),
	})
}
