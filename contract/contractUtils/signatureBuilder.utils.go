package contractUtils

import (
	"crypto/ecdsa"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// Create a function that takes privateKeyHex and message body and create signature based on EIP712

type BuildSessionKeyRequest struct {
	SubaccountId string
	ExpiryTs     int64
	ChainId      int64
	Nonce        int64
}

type BuildOrderSigPayload struct {
	SubAccountId string   `json:"subAccountId"`
	PriceX18     *big.Int `json:"priceX18"`
	Amount       *big.Int `json:"amount"`
	Expiration   uint64   `json:"expiration"`
	IsReduce     bool     `json:"isReduce"`
	ChainId      int64    `json:"chainId"`
	ProductId    *big.Int `json:"productId"`
}

type BuildSettleUserPnlPayload struct {
	SubAccountId string `json:"subAccountId"`
	Nonce        int64  `json:"nonce"`
	ChainId      int64  `json:"chainId"`
}

func _buildSignature(privateKey *ecdsa.PrivateKey, hash []byte) (string, error) {
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return "", err
	}
	signature[64] += 27
	return hexutil.Encode(signature), nil
}

func BuildSessionKeySignaturePayload(masterPrivateKeyHex string, signingPrivateKeyHex string, sessionRequest BuildSessionKeyRequest) (*SessionRequest, error) {
	masterPrivateKey, err := crypto.HexToECDSA(masterPrivateKeyHex)
	if err != nil {
		fmt.Printf("Error converting masterPrivateKeyHex to ECDSA: %v\n", err)
		return nil, err
	}

	signingPrivateKey, err := crypto.HexToECDSA(signingPrivateKeyHex)
	if err != nil {
		fmt.Printf("Error converting signingPrivateKeyHex to ECDSA: %v\n", err)
		return nil, err
	}

	masterAddress := crypto.PubkeyToAddress(masterPrivateKey.PublicKey).Hex()
	signingAddress := crypto.PubkeyToAddress(signingPrivateKey.PublicKey).Hex()

	chainId := math.NewHexOrDecimal256(sessionRequest.ChainId)
	expiryTimestamp := math.NewHexOrDecimal256(sessionRequest.ExpiryTs)
	nonce := math.NewHexOrDecimal256(sessionRequest.Nonce)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(sessionRequest.SubaccountId)
	if err != nil {
		return nil, err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"Register": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "userAddress", Type: "address"},
				{Name: "sessionKey", Type: "address"},
				{Name: "expiryTimeStamp", Type: "uint128"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
			"EIP712Domain": EIP712_DOMAIN_TYPE,
		},
		PrimaryType: "Register",
		Domain:      getDomainDataForEndpoint(sessionRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":    subaccountIdBytes,
			"userAddress":     masterAddress,
			"sessionKey":      signingAddress,
			"expiryTimeStamp": expiryTimestamp,
			"nonce":           nonce,
			"chainId":         chainId,
		},
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)

	if err != nil {
		return nil, err
	}

	fmt.Printf("Hash maker: %s\n", hexutil.Encode(hash))

	masterSignatureHex, err := _buildSignature(masterPrivateKey, hash)
	if err != nil {
		return nil, err
	}

	signingSignatureHex, err := _buildSignature(signingPrivateKey, hash)
	if err != nil {
		return nil, err
	}

	return &SessionRequest{
		SubaccountId:     sessionRequest.SubaccountId,
		SigningAddress:   signingAddress,
		SigningSignature: signingSignatureHex,
		EthAddress:       masterAddress,
		EthSignature:     masterSignatureHex,
		ExpiryTs:         sessionRequest.ExpiryTs,
		Nonce:            sessionRequest.Nonce,
		ChainId:          sessionRequest.ChainId,
	}, err
}

func BuildOrderSignaturePayload(signingPrivateKeyHex string, orderRequest BuildOrderSigPayload) (*OrderSigRequest, error) {
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(orderRequest.SubAccountId)
	if err != nil {
		return nil, err
	}

	signingPrivateKey, err := crypto.HexToECDSA(signingPrivateKeyHex)
	if err != nil {
		return nil, err
	}
	signingKey := crypto.PubkeyToAddress(signingPrivateKey.PublicKey).Hex()
	chainId := math.NewHexOrDecimal256(orderRequest.ChainId)
	expiration := math.NewHexOrDecimal256(int64(orderRequest.Expiration))

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"Order": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "priceX18", Type: "int128"},
				{Name: "amount", Type: "int128"},
				{Name: "expiration", Type: "uint64"},
				{Name: "isReduce", Type: "bool"},
				{Name: "sessionKey", Type: "address"},
				{Name: "chainId", Type: "uint256"},
				{Name: "productId", Type: "uint32"},
			},
		},
		PrimaryType: "Order",
		Domain:      getDomainDataForOffchainExchange(orderRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"priceX18":     orderRequest.PriceX18,
			"amount":       new(big.Int).Set(orderRequest.Amount), // Creating copy as typedata modifies negative values to its 2's complement
			"expiration":   expiration,
			"isReduce":     orderRequest.IsReduce,
			"sessionKey":   signingKey,
			"chainId":      chainId,
			"productId":    orderRequest.ProductId,
		},
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		fmt.Printf("Error hashing typed data: %v\n", err)
		return nil, err
	}

	signature, err := _buildSignature(signingPrivateKey, hash)
	if err != nil {
		fmt.Printf("Error building signature: %v\n", err)
		return nil, err
	}

	return &OrderSigRequest{
		SubAccountId: orderRequest.SubAccountId,
		PriceX18:     orderRequest.PriceX18,
		Amount:       orderRequest.Amount,
		Expiration:   orderRequest.Expiration,
		IsReduce:     orderRequest.IsReduce,
		ChainId:      orderRequest.ChainId,
		ProductId:    orderRequest.ProductId,
		SessionKey:   signingKey,
		Signature:    signature,
	}, nil
}

func BuildVerifySettleUserPnlPayload(signingPrivateKeyHex string, settleUserPnlRequest BuildSettleUserPnlPayload) (*SettleUserPnl, error) {
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(settleUserPnlRequest.SubAccountId)
	if err != nil {
		return nil, err
	}

	signingPrivateKey, err := crypto.HexToECDSA(signingPrivateKeyHex)
	if err != nil {
		return nil, err
	}
	signingKey := crypto.PubkeyToAddress(signingPrivateKey.PublicKey).Hex()
	chainId := math.NewHexOrDecimal256(settleUserPnlRequest.ChainId)
	nonce := math.NewHexOrDecimal256(settleUserPnlRequest.Nonce)

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"SettleUserPnl": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "sessionKey", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "SettleUserPnl",
		Domain:      getDomainDataForEndpoint(settleUserPnlRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"sessionKey":   signingKey,
			"nonce":        nonce,
			"chainId":      chainId,
		},
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)
	if err != nil {
		fmt.Printf("Error hashing typed data: %v\n", err)
		return nil, err
	}

	signature, err := _buildSignature(signingPrivateKey, hash)
	if err != nil {
		fmt.Printf("Error building signature: %v\n", err)
		return nil, err
	}

	return &SettleUserPnl{
		SubAccountId: settleUserPnlRequest.SubAccountId,
		SessionKey:   signingKey,
		Nonce:        settleUserPnlRequest.Nonce,
		ChainId:      settleUserPnlRequest.ChainId,
		Signature:    signature,
	}, nil
}
