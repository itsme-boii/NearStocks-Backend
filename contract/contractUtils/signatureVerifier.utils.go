package contractUtils

import (
	"crypto/subtle"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type SessionRequest struct {
	SubaccountId     string
	SigningAddress   string
	EthAddress       string
	ExpiryTs         int64
	ChainId          int64
	SigningSignature string
	EthSignature     string
	Nonce            int64
}

type DepositRequest struct {
	SubAccountId string `json:"subAccountId"`
	TokenAmount  string `json:"tokenAmount"`
	SessionKey   string `json:"sessionKey"`
	Nonce        int64  `json:"nonce"`
	ChainId      int64  `json:"chainId"`
}

type CampaignRewardClaimRequest struct {
	SubAccountId string `json:"subAccountId"`
	Amount       string `json:"amount"`
	ProductId    uint32 `json:"productId"`
	CampaignId   uint32 `json:"campaignId"`
	SessionKey   string `json:"sessionKey"`
	Nonce        int64  `json:"nonce"`
	ChainId      int64  `json:"chainId"`
}

type StakeLogXRequest struct {
	SubAccountId   string `json:"subAccountId"`
	ProductId      uint32 `json:"productId"`
	TokenAmount    string `json:"tokenAmount"`
	StakerContract string `json:"stakerContract"`
	SessionKey     string `json:"sessionKey"`
	Nonce          int64  `json:"nonce"`
	ChainId        int64  `json:"chainId"`
}

type UnstakeLogXRequest struct {
	SubAccountId   string `json:"subAccountId"`
	ProductId      uint32 `json:"productId"`
	Amount         string `json:"amount"`
	StakerContract string `json:"stakerContract"`
	SessionKey     string `json:"sessionKey"`
	Nonce          int64  `json:"nonce"`
	ChainId        int64  `json:"chainId"`
}

type WithdrawLogX struct {
	SubAccountId       string `json:"subAccountId"`
	TokenAmount        string `json:"tokenAmount"`
	SessionKey         string `json:"sessionKey"`
	DestinationChainId int64  `json:"destinationChainId"`
	BridgeOutContract  string `json:"bridgeOutContract"`
	Nonce              int64  `json:"nonce"`
	Receiver           string `json:"receiver"`
	ChainId            int64  `json:"chainId"`
}

type ClaimRewards struct {
	SubAccountId   string `json:"subAccountId"`
	SessionKey     string `json:"sessionKey"`
	StakerContract string `json:"stakerContract"`
	ProductId      uint32 `json:"productId"`
	Nonce          int64  `json:"nonce"`
	ChainId        int64  `json:"chainId"`
}

type WithdrawCollateral struct {
	SubAccountId       string `json:"subAccountId"`
	SessionKey         string `json:"sessionKey"`
	ProductId          uint32 `json:"productId"`
	Amount             string `json:"amount"`
	Nonce              int64  `json:"nonce"`
	DestinationChainId int64  `json:"destinationChainId"`
	Receiver           string `json:"receiver"`
	ChainId            int64  `json:"chainId"`
}

type SettleBroker2PnL struct {
	SubAccountId string `json:"subAccountId"`
}

type ShiftBalanceKroma struct {
	SubAccountId string `json:"subAccountId"`
}

type BurnBalanceKroma struct {
	SubAccountId string `json:"subAccountId"`
}

type SettleUserPnl struct {
	SubAccountId string `json:"subAccountId"`
	SessionKey   string `json:"sessionKey"`
	Nonce        int64  `json:"nonce"`
	ChainId      int64  `json:"chainId"`
	Signature    string `json:"signature"`
}

type OrderSigRequest struct {
	SubAccountId string   `json:"subAccountId"`
	PriceX18     *big.Int `json:"priceX18"`
	Amount       *big.Int `json:"amount"`
	Expiration   uint64   `json:"expiration"`
	IsReduce     bool     `json:"isReduce"`
	SessionKey   string   `json:"sessionKey"`
	ChainId      int64    `json:"chainId"`
	Signature    string   `json:"signature"`
	ProductId    *big.Int `json:"productId"`
}

type CantonAuthRegistration struct {
	SigningAddress string `json:"signingAddress"`
	CantonPartyId  string `json:"cantonPartyId"`
	Timestamp      int64  `json:"timestamp"`
	Nonce          int64  `json:"nonce"`
	ChainId        int64  `json:"chainId"`
}

type SettleUserPnlRequest struct {
	SubaccountIds []string
	ProductIds    []uint32
	OraclePrices  []*big.Int
}

type UserOptionBet struct {
	SubAccountId string `json:"subAccountId"`
	ProductId    uint32 `json:"productId"`
	Amount       string `json:"amount"`
	Interval     uint32 `json:"interval"`
	Nonce        int64  `json:"nonce"`
	SessionKey   string `json:"sessionKey"`
	ChainId      int64  `json:"chainId"`
}

type PlaceOptionBetRequest struct {
	UserOptionBet UserOptionBet `json:"userOptionBet"`
	OrderId       int64         `json:"orderId"`
	EntryPriceX18 string        `json:"entryPrice"`
	Payout        uint32        `json:"payout"`
	Fee           uint32        `json:"fee"`
}

type PlaceOptionBetRedis struct {
	OrderId      uint   `json:"orderId"`
	SubaccountId string `json:"subaccountId"`
	ProductId    uint32 `json:"productId"`
	QuoteDelta   string `json:"quoteDelta"`
	EntryPrice   string `json:"entryPrice"`
	Payout       uint32 `json:"payout"`
	Fees         uint32 `json:"fees"`
}

type CloseOptionBetRequest struct {
	OrderId      int64  `json:"orderId"`
	ExitPriceX18 string `json:"exitPrice"`
}

type PlacePreMarketOrderRequest struct {
	SubAccountId string `json:"subAccountId"`
	ProductId    uint32 `json:"productId"`
	Amount       string `json:"amount"`
	IsBuy        bool   `json:"isBuy"`
	Nonce        int64  `json:"nonce"`
	SessionKey   string `json:"sessionKey"`
	ChainId      int64  `json:"chainId"`
}

type PreMarketPricingRedis struct {
	XReserve string `json:"xReserve"`
	YReserve string `json:"yReserve"`
}

type AddPreMarketProductRequest struct {
	ProductId         uint32  `json:"product_id"`
	MaxSupply         string  `json:"max_supply"`
	ClosingTimestamp  int64   `json:"closing_timestamp"`
	DeliveryTimestamp *int64  `json:"delivery_timestamp"`
	StartingPrice     string  `json:"starting_price"`
	IsEnabled         bool    `json:"is_enabled"`
	ToBeDelivered     *string `json:"to_be_delivered"`
	Details           string  `json:"details"`
}

type ClosePreMarketProductRequest struct {
	ProductId uint32 `json:"product_id"`
}

var EIP712_DOMAIN_TYPE = []apitypes.Type{
	{Name: "name", Type: "string"},
	{Name: "version", Type: "string"},
	{Name: "chainId", Type: "uint256"},
	{Name: "verifyingContract", Type: "address"},
}

func getDomainDataForEndpoint(chainId int64) apitypes.TypedDataDomain {
	if NearSettlement() {
		// the configured chain id, not the request's: a request signed for another chain then
		// fails here instead of failing on-chain
		return NearStocksDomain(NearStocksAccount(), NearStocksChainId())
	}
	return apitypes.TypedDataDomain{
		Name:              "LogX",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(chainId),
		VerifyingContract: ENDPOINT_CONTRACT_ADDRESS,
	}
}

func getDomainDataForOffchainExchange(chainId int64) apitypes.TypedDataDomain {
	if NearSettlement() {
		// the configured chain id, not the request's: a request signed for another chain then
		// fails here instead of failing on-chain
		return NearStocksDomain(NearStocksAccount(), NearStocksChainId())
	}
	return apitypes.TypedDataDomain{
		Name:              "LogX",
		Version:           "1",
		ChainId:           math.NewHexOrDecimal256(chainId),
		VerifyingContract: OFFCHAIN_CONTRACT_ADDRESS,
	}
}

func VerifySessionkeyCreation(sessionRequest SessionRequest) error {
	chainId := math.NewHexOrDecimal256(sessionRequest.ChainId)
	expiryTimestamp := math.NewHexOrDecimal256(sessionRequest.ExpiryTs)
	nonce := math.NewHexOrDecimal256(sessionRequest.Nonce)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(sessionRequest.SubaccountId)
	if err != nil {
		return err
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
			"userAddress":     sessionRequest.EthAddress,
			"sessionKey":      sessionRequest.SigningAddress,
			"expiryTimeStamp": expiryTimestamp,
			"nonce":           nonce,
			"chainId":         chainId,
		},
	}

	hash, _, err := apitypes.TypedDataAndHash(typedData)

	if err != nil {
		return fmt.Errorf("1. verification failed: %w", err)
	}

	fmt.Printf("Hash checker: %s\n", hexutil.Encode(hash))

	err1 := _VerifySignature(hash, sessionRequest.SigningSignature, sessionRequest.SigningAddress)
	if err1 != nil {
		return fmt.Errorf("2. verification failed: %w", err1)
	}
	err2 := _VerifySignature(hash, sessionRequest.EthSignature, sessionRequest.EthAddress)
	if err2 != nil {
		return fmt.Errorf("3. verification failed: %w", err2)
	}

	return nil
}

func VerifyDepositRequest(depositRequest DepositRequest, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(depositRequest.ChainId)

	nonce := math.NewHexOrDecimal256(depositRequest.Nonce)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(depositRequest.SubAccountId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"ClaimLogX": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "tokenAmount", Type: "int128"},
				{Name: "sessionKey", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "ClaimLogX",
		Domain:      getDomainDataForEndpoint(depositRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"tokenAmount":  depositRequest.TokenAmount,
			"sessionKey":   depositRequest.SessionKey,
			"nonce":        nonce,
			"chainId":      chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyCampaignRewards(campaignRewards CampaignRewardClaimRequest, signatureHex string, signerHex string) error {
	// Convert the fields to the correct types for EIP-712
	chainId := math.NewHexOrDecimal256(campaignRewards.ChainId)

	nonce := math.NewHexOrDecimal256(campaignRewards.Nonce)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(campaignRewards.SubAccountId)
	if err != nil {
		return err
	}
	productId := math.NewHexOrDecimal256(int64(campaignRewards.ProductId))
	campaignId := math.NewHexOrDecimal256(int64(campaignRewards.CampaignId))

	// Define the EIP-712 typed data structure
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"CampaignRewards": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "amount", Type: "int128"},
				{Name: "productId", Type: "uint32"},
				{Name: "campaignId", Type: "uint32"},
				{Name: "sessionKey", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "CampaignRewards",
		Domain:      getDomainDataForEndpoint(campaignRewards.ChainId), // Assume same domain function is used
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"amount":       campaignRewards.Amount,
			"productId":    productId,
			"campaignId":   campaignId,
			"sessionKey":   campaignRewards.SessionKey,
			"nonce":        nonce,
			"chainId":      chainId,
		},
	}

	// Create the hash for the typed data
	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	// Verify the signature
	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyPlaceOptionsBet(userOptionBet UserOptionBet, signatureHex string, signerHex string) error {
	// Convert the fields to the correct types for EIP-712
	chainId := math.NewHexOrDecimal256(userOptionBet.ChainId)
	nonce := math.NewHexOrDecimal256(userOptionBet.Nonce)
	interval := math.NewHexOrDecimal256(int64(userOptionBet.Interval))
	productId := math.NewHexOrDecimal256(int64(userOptionBet.ProductId))

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(userOptionBet.SubAccountId)
	if err != nil {
		return err
	}

	// Define the EIP-712 typed data structure
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"UserOptionBet": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "productId", Type: "uint32"},
				{Name: "amount", Type: "int128"},
				{Name: "interval", Type: "uint32"},
				{Name: "nonce", Type: "uint128"},
				{Name: "sessionKey", Type: "address"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "UserOptionBet",
		Domain:      getDomainDataForEndpoint(userOptionBet.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"productId":    productId,
			"amount":       userOptionBet.Amount,
			"interval":     interval,
			"nonce":        nonce,
			"sessionKey":   userOptionBet.SessionKey,
			"chainId":      chainId,
		},
	}

	// Create the hash for the typed data
	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	// Verify the signature
	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyPlacePreMarketOrder(placePreMarketOrder PlacePreMarketOrderRequest, signatureHex string, signerHex string) error {
	// Convert the fields to the correct types for EIP-712
	chainId := math.NewHexOrDecimal256(placePreMarketOrder.ChainId)
	nonce := math.NewHexOrDecimal256(placePreMarketOrder.Nonce)
	productId := math.NewHexOrDecimal256(int64(placePreMarketOrder.ProductId))

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(placePreMarketOrder.SubAccountId)
	if err != nil {
		return err
	}

	// Define the EIP-712 typed data structure
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"PlacePreMarketOrderRequest": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "productId", Type: "uint32"},
				{Name: "amount", Type: "int128"},
				{Name: "isBuy", Type: "bool"},
				{Name: "nonce", Type: "uint128"},
				{Name: "sessionKey", Type: "address"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "PlacePreMarketOrderRequest",
		Domain:      getDomainDataForEndpoint(placePreMarketOrder.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"productId":    productId,
			"amount":       placePreMarketOrder.Amount,
			"isBuy":        placePreMarketOrder.IsBuy,
			"nonce":        nonce,
			"sessionKey":   placePreMarketOrder.SessionKey,
			"chainId":      chainId,
		},
	}

	// Create the hash for the typed data
	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	// Verify the signature
	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyPlaceSyntheticSpotOrder(placePreMarketOrder PlacePreMarketOrderRequest, signatureHex string, signerHex string) error {
	// Convert the fields to the correct types for EIP-712
	chainId := math.NewHexOrDecimal256(placePreMarketOrder.ChainId)
	nonce := math.NewHexOrDecimal256(placePreMarketOrder.Nonce)
	productId := math.NewHexOrDecimal256(int64(placePreMarketOrder.ProductId))

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(placePreMarketOrder.SubAccountId)
	if err != nil {
		return err
	}

	// Define the EIP-712 typed data structure
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"PlaceSyntheticSpotOrderRequest": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "productId", Type: "uint32"},
				{Name: "amount", Type: "int128"},
				{Name: "isBuy", Type: "bool"},
				{Name: "nonce", Type: "uint128"},
				{Name: "sessionKey", Type: "address"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "PlaceSyntheticSpotOrderRequest",
		Domain:      getDomainDataForEndpoint(placePreMarketOrder.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"productId":    productId,
			"amount":       placePreMarketOrder.Amount,
			"isBuy":        placePreMarketOrder.IsBuy,
			"nonce":        nonce,
			"sessionKey":   placePreMarketOrder.SessionKey,
			"chainId":      chainId,
		},
	}

	// Create the hash for the typed data
	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	// Verify the signature
	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyStakeRequest(stakeRequest StakeLogXRequest, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(stakeRequest.ChainId)
	nonce := math.NewHexOrDecimal256(stakeRequest.Nonce)
	productId := math.NewHexOrDecimal256(int64(stakeRequest.ProductId))
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(stakeRequest.SubAccountId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"StakeLogXRequest": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "productId", Type: "uint32"},
				{Name: "tokenAmount", Type: "int128"},
				{Name: "stakerContract", Type: "address"},
				{Name: "sessionKey", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "StakeLogXRequest",
		Domain:      getDomainDataForEndpoint(stakeRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":   subaccountIdBytes,
			"productId":      productId,
			"tokenAmount":    stakeRequest.TokenAmount,
			"stakerContract": stakeRequest.StakerContract,
			"sessionKey":     stakeRequest.SessionKey,
			"nonce":          nonce,
			"chainId":        chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyUnstakeRequest(unstakeRequest UnstakeLogXRequest, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(unstakeRequest.ChainId)
	nonce := math.NewHexOrDecimal256(unstakeRequest.Nonce)
	productId := math.NewHexOrDecimal256(int64(unstakeRequest.ProductId))
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(unstakeRequest.SubAccountId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"UnstakeLogXRequest": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "productId", Type: "uint32"},
				{Name: "amount", Type: "int128"},
				{Name: "stakerContract", Type: "address"},
				{Name: "sessionKey", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "UnstakeLogXRequest",
		Domain:      getDomainDataForEndpoint(unstakeRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":   subaccountIdBytes,
			"productId":      productId,
			"amount":         unstakeRequest.Amount,
			"stakerContract": unstakeRequest.StakerContract,
			"sessionKey":     unstakeRequest.SessionKey,
			"nonce":          nonce,
			"chainId":        chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyWithdrawLogX(withdrawRequest WithdrawLogX, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(withdrawRequest.ChainId)
	nonce := math.NewHexOrDecimal256(withdrawRequest.Nonce)
	destinationChainId := math.NewHexOrDecimal256(withdrawRequest.DestinationChainId)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(withdrawRequest.SubAccountId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"WithdrawLogX": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "tokenAmount", Type: "int128"},
				{Name: "sessionKey", Type: "address"},
				{Name: "destinationChainId", Type: "uint256"},
				{Name: "bridgeOutContract", Type: "address"},
				{Name: "nonce", Type: "uint128"},
				{Name: "receiver", Type: "address"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "WithdrawLogX",
		Domain:      getDomainDataForEndpoint(withdrawRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":       subaccountIdBytes,
			"tokenAmount":        withdrawRequest.TokenAmount,
			"sessionKey":         withdrawRequest.SessionKey,
			"destinationChainId": destinationChainId,
			"bridgeOutContract":  withdrawRequest.BridgeOutContract,
			"nonce":              nonce,
			"receiver":           withdrawRequest.Receiver,
			"chainId":            chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyClaimRewards(claimRewardsRequest ClaimRewards, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(claimRewardsRequest.ChainId)
	nonce := math.NewHexOrDecimal256(claimRewardsRequest.Nonce)
	productId := math.NewHexOrDecimal256(int64(claimRewardsRequest.ProductId))
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(claimRewardsRequest.SubAccountId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"ClaimRewards": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "sessionKey", Type: "address"},
				{Name: "stakerContract", Type: "address"},
				{Name: "productId", Type: "uint32"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "ClaimRewards",
		Domain:      getDomainDataForEndpoint(claimRewardsRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":   subaccountIdBytes,
			"sessionKey":     claimRewardsRequest.SessionKey,
			"stakerContract": claimRewardsRequest.StakerContract,
			"productId":      productId,
			"nonce":          nonce,
			"chainId":        chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyWithdrawCollateral(withdrawCollateralRequest WithdrawCollateral, signatureHex string, signerHex string) error {
	chainId := math.NewHexOrDecimal256(withdrawCollateralRequest.ChainId)
	nonce := math.NewHexOrDecimal256(withdrawCollateralRequest.Nonce)
	productId := math.NewHexOrDecimal256(int64(withdrawCollateralRequest.ProductId))
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(withdrawCollateralRequest.SubAccountId)
	destinationChainId := math.NewHexOrDecimal256(withdrawCollateralRequest.DestinationChainId)
	if err != nil {
		return err
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"WithdrawCollateral": []apitypes.Type{
				{Name: "subAccountId", Type: "bytes32"},
				{Name: "sessionKey", Type: "address"},
				{Name: "productId", Type: "uint32"},
				{Name: "amount", Type: "uint128"},
				{Name: "nonce", Type: "uint128"},
				{Name: "destinationChainId", Type: "uint256"},
				{Name: "receiver", Type: "address"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "WithdrawCollateral",
		Domain:      getDomainDataForEndpoint(withdrawCollateralRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId":       subaccountIdBytes,
			"sessionKey":         withdrawCollateralRequest.SessionKey,
			"productId":          productId,
			"amount":             withdrawCollateralRequest.Amount,
			"nonce":              nonce,
			"destinationChainId": destinationChainId,
			"receiver":           withdrawCollateralRequest.Receiver,
			"chainId":            chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, signerHex)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifySettleUserPnl(settlePnlRequest SettleUserPnl) error {
	chainId := math.NewHexOrDecimal256(settlePnlRequest.ChainId)
	nonce := math.NewHexOrDecimal256(settlePnlRequest.Nonce)
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(settlePnlRequest.SubAccountId)
	if err != nil {
		return err
	}

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
		Domain:      getDomainDataForEndpoint(settlePnlRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"subAccountId": subaccountIdBytes,
			"sessionKey":   settlePnlRequest.SessionKey,
			"nonce":        nonce,
			"chainId":      chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, settlePnlRequest.Signature, settlePnlRequest.SessionKey)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func VerifyOrderSignature(orderRequest OrderSigRequest) error {
	chainId := math.NewHexOrDecimal256(orderRequest.ChainId)
	expiration := math.NewHexOrDecimal256(int64(orderRequest.Expiration))
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(orderRequest.SubAccountId)
	if err != nil {
		return err
	}

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
			"sessionKey":   orderRequest.SessionKey,
			"chainId":      chainId,
			"productId":    orderRequest.ProductId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, orderRequest.Signature, orderRequest.SessionKey)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}

func _VerifySignature(hash hexutil.Bytes, signatureHex string, signerHex string) error {
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	// A short signature used to panic on signature[64]; wallets always produce 65 bytes with v 27/28.
	if len(signature) != 65 || (signature[64] != 27 && signature[64] != 28) {
		return fmt.Errorf("invalid signature: expected 65 bytes with v 27 or 28")
	}

	// update the recovery id
	// https://github.com/ethereum/go-ethereum/blob/55599ee95d4151a2502465e0afc7c47bd1acba77/internal/ethapi/api.go#L442
	signature[64] -= 27

	// get the pubkey used to sign this signature
	sigPubkey, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		return fmt.Errorf("ecrecover: %w", err)
	}

	pubkey, err := crypto.UnmarshalPubkey(sigPubkey)
	if err != nil {
		return err
	}

	address := crypto.PubkeyToAddress(*pubkey)
	// fmt.Println("ADDRESS:", address.Hex())

	tokenAddress := common.HexToAddress(signerHex)
	if subtle.ConstantTimeCompare(address.Bytes(), tokenAddress.Bytes()) == 0 {
		return fmt.Errorf("address mismatch")
	}

	return nil
}

func VerifyCantonAuthRegistration(authRequest CantonAuthRegistration, signatureHex string) error {
	chainId := math.NewHexOrDecimal256(authRequest.ChainId)
	timestamp := math.NewHexOrDecimal256(authRequest.Timestamp)
	nonce := math.NewHexOrDecimal256(authRequest.Nonce)

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": EIP712_DOMAIN_TYPE,
			"CantonAuthRegistration": []apitypes.Type{
				{Name: "signingAddress", Type: "address"},
				{Name: "cantonPartyId", Type: "string"},
				{Name: "timestamp", Type: "uint128"},
				{Name: "nonce", Type: "uint128"},
				{Name: "chainId", Type: "uint256"},
			},
		},
		PrimaryType: "CantonAuthRegistration",
		Domain:      getDomainDataForEndpoint(authRequest.ChainId),
		Message: apitypes.TypedDataMessage{
			"signingAddress": authRequest.SigningAddress,
			"cantonPartyId":  authRequest.CantonPartyId,
			"timestamp":      timestamp,
			"nonce":          nonce,
			"chainId":        chainId,
		},
	}

	hash, _, err1 := apitypes.TypedDataAndHash(typedData)
	if err1 != nil {
		return fmt.Errorf("1. verification failed: %w", err1)
	}

	err2 := _VerifySignature(hash, signatureHex, authRequest.SigningAddress)
	if err2 != nil {
		return fmt.Errorf("2. verification failed: %w", err2)
	}

	return nil
}
