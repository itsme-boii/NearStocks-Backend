package contract

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/spotUtils"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"os"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/contract/gen"
	"github/eugenix-io/logx-inf-backend/contract/types"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"log"
	"math/big"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Global variables for ABI types
var (
	bytesTy   abi.Type
	bytes32Ty abi.Type
	uint32Ty  abi.Type
	int128Ty  abi.Type
	int256Ty  abi.Type
	addressTy abi.Type
	uint256Ty abi.Type
	uint128Ty abi.Type
	// Array
	uint32ArrayTy  abi.Type
	int128ArrayTy  abi.Type
	bytes32ArrayTy abi.Type
	//Custom
	orderType           abi.Type
	liquidateeOrderType abi.Type
	//Bool
	boolTy abi.Type
)

// Initialize the global variables in the init function
func init() {
	var err error
	boolTy, err = abi.NewType("bool", "", nil)
	if err != nil {
		log.Fatalf("Failed to create boolTy: %v", err)
	}

	bytes32Ty, err = abi.NewType("bytes32", "", nil)
	if err != nil {
		log.Fatalf("Failed to create bytes32Ty: %v", err)
	}

	uint32Ty, err = abi.NewType("uint32", "", nil)
	if err != nil {
		log.Fatalf("Failed to create uint32Ty: %v", err)
	}

	int128Ty, err = abi.NewType("int128", "", nil)
	if err != nil {
		log.Fatalf("Failed to create int256Ty: %v", err)
	}

	int256Ty, err = abi.NewType("int256", "", nil)
	if err != nil {
		log.Fatalf("Failed to create int256Ty: %v", err)
	}

	addressTy, err = abi.NewType("address", "", nil)
	if err != nil {
		log.Fatalf("Failed to create addressTy: %v", err)
	}

	uint256Ty, err = abi.NewType("uint256", "", nil)
	if err != nil {
		log.Fatalf("Failed to create uint256Ty: %v", err)
	}

	uint128Ty, err = abi.NewType("uint128", "", nil)
	if err != nil {
		log.Fatalf("Failed to create uint128Ty: %v", err)
	}

	uint32ArrayTy, err = abi.NewType("uint32[]", "", nil)
	if err != nil {
		log.Fatalf("Failed to create uint32ArrayTy: %v", err)
	}

	int128ArrayTy, err = abi.NewType("int128[]", "", nil)
	if err != nil {
		log.Fatalf("Failed to create int128ArrayTy: %v", err)
	}

	bytes32ArrayTy, err = abi.NewType("bytes32[]", "", nil)
	if err != nil {
		log.Fatalf("Failed to create bytes32ArrayTy: %v", err)
	}

	bytesTy, err = abi.NewType("bytes", "", nil)

	orderType, err = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "subAccountId", Type: "bytes32"},
		{Name: "priceX18", Type: "int128"},
		{Name: "amount", Type: "int128"},
		{Name: "expiration", Type: "uint64"},
		{Name: "isReduce", Type: "bool"},
		{Name: "sessionKey", Type: "address"},
		{Name: "chainId", Type: "uint256"},
	})

	if err != nil {
		log.Fatalf("Failed to create taker signed order type: %v", err)
	}

	liquidateeOrderType, err = abi.NewType("tuple", "", []abi.ArgumentMarshaling{
		{Name: "subAccountId", Type: "bytes32"},
		{Name: "amount", Type: "int128"},
	})

	if err != nil {
		log.Fatalf("Failed to create liquidate subaccount type: %v", err)
	}
}

// ----------------- Types -----------------
type EndpointContract struct {
	_mutex          *sync.Mutex
	contractAddress common.Address
	RedisClient     *redis.Client
}

type PerpTick struct {
	Time          *big.Int
	AvgPriceDiffs []*big.Int
}

type Order struct {
	SubAccountId [32]byte
	PriceX18     *big.Int
	Amount       *big.Int
	Expiration   uint64
	IsReduce     bool
	SessionKey   common.Address
	ChainId      *big.Int
}

type MatchOrderPayload struct {
	ProductId        uint32
	Taker            Order
	Maker            Order
	MatchedAmountx18 *big.Int
}

type OrderRequest struct {
	SubaccountId string
	PriceX18     *big.Int
	Amount       *big.Int
	ExpiryTs     uint64
}

type MatchOrderRequest struct {
	ProductId         uint32
	TakerSubaccountId string
	MakerSubaccountId string
	TakerPriceX18     *big.Int
	MakerPriceX18     *big.Int
	MakerExpiryTs     uint64
	TakerExpiryTs     uint64
	TakerMaxAmountX18 *big.Int
	MakerMaxAmountX18 *big.Int
	TakerSide         ctypes.OrderSide
	TakerSignature    string
	MakerSignature    string
	TakerIsReduce     bool
	MakerIsReduce     bool
	TakerSessionKey   string
	MakerSessionKey   string
	TakerOrderType    ctypes.OrderType
	MatchedAmountx18  *big.Int
}

// ----------------- Create Endpoint Contract Object -----------------
func NewEndpointContract() *EndpointContract {
	return &EndpointContract{
		_mutex:          &sync.Mutex{},
		contractAddress: common.HexToAddress(contractUtils.ENDPOINT_CONTRACT_ADDRESS),
		RedisClient:     xredis.GetRedisClient(),
	}
}

// ----------------- Wrapper Functions -----------------
func (epc *EndpointContract) ClaimLogX(depositRequest contractUtils.DepositRequest, tokenAmount *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearClaimLogX(depositRequest, tokenAmount, signatureHex, transactionCounter)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		xlog.Errorf("Cannot decode signing signature: %v", err)
		return false, err
	}

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(depositRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])
	// ABI encode the ClaimLogXRequest struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: int128Ty},
		{Type: addressTy},
		{Type: uint128Ty},
		{Type: uint256Ty},
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		tokenAmount,
		common.HexToAddress(depositRequest.SessionKey),
		big.NewInt(int64(depositRequest.Nonce)),
		big.NewInt(int64(depositRequest.ChainId)),
	)
	if err != nil {
		fmt.Printf("Cannot encode ClaimLogXRequest struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.CLAIM_LOGX_TXN, encodedStruct)
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "ClaimLogX", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add ClaimLogX transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) ClaimCampaignRewards(campaignRewards contractUtils.CampaignRewardClaimRequest, rewardAmount *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear // D-4
	}
	// Decode the signing signature
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		xlog.Errorf("Cannot decode signing signature: %v", err)
		return false, err
	}

	// Convert subAccountId to bytes32
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(campaignRewards.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	// ABI encode the CampaignRewards struct
	arguments := abi.Arguments{
		{Type: bytes32Ty}, // subAccountId
		{Type: int128Ty},  // rewardAmount
		{Type: uint32Ty},  // productId
		{Type: uint32Ty},  // campaignId
		{Type: addressTy}, // sessionKey
		{Type: uint128Ty}, // nonce
		{Type: uint256Ty}, // chainId
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		rewardAmount,
		campaignRewards.ProductId,
		campaignRewards.CampaignId,
		common.HexToAddress(campaignRewards.SessionKey),
		big.NewInt(int64(campaignRewards.Nonce)),
		big.NewInt(int64(campaignRewards.ChainId)),
	)
	if err != nil {
		fmt.Printf("Cannot encode CampaignRewards struct: %v", err)
		return false, err
	}

	// Build the transaction argument for CampaignRewards
	transaction := contractUtils.BuildTransactionArg(types.CLAIM_CAMPAIGN_REWARDS_TXN, encodedStruct)

	// Add transaction to the batch database
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "ClaimCampaignRewards", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add ClaimCampaignRewards transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) PlaceOptionBet(userOptionBet contractUtils.UserOptionBet, amount *big.Int, entryPrice *big.Int, orderId uint, payout uint32, fee uint32, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearPlaceOption(userOptionBet, amount, entryPrice, orderId, payout, fee, signatureHex, transactionCounter)
		return err == nil, err
	}
	// Decode the signing signature
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		xlog.Errorf("Cannot decode signing signature: %v", err)
		return false, err
	}

	// Convert SubAccountId to bytes32
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(userOptionBet.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	// ABI encode the PlaceOptionBetRequest struct
	arguments := abi.Arguments{
		{Type: bytes32Ty}, // SubAccountId
		{Type: uint32Ty},  // ProductId
		{Type: int128Ty},  // Amount
		{Type: uint32Ty},  // Interval
		{Type: uint128Ty}, // Nonce
		{Type: addressTy}, // SessionKey
		{Type: uint256Ty}, // ChainId
		{Type: uint256Ty}, // OrderId
		{Type: int128Ty},  // EntryPriceX18
		{Type: uint32Ty},  // Payout
		{Type: uint32Ty}}  // Fee

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		userOptionBet.ProductId,
		amount,
		userOptionBet.Interval,
		big.NewInt(int64(userOptionBet.Nonce)),
		common.HexToAddress(userOptionBet.SessionKey),
		big.NewInt(int64(userOptionBet.ChainId)),
		big.NewInt(int64(orderId)),
		entryPrice,
		payout,
		fee,
	)

	if err != nil {
		xlog.Errorf("Cannot encode PlaceOptionBetRequest struct: %v", err)
		return false, err
	}

	// Build the transaction argument for PlaceOptionBet
	transaction := contractUtils.BuildTransactionArg(types.PLACE_OPTIONS_BET_TXN, encodedStruct)

	// Add transaction to the batch database
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "PlaceOptionBet", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add PlaceOptionBet transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) CloseOptionBet(orderId uint, exitPrice *big.Int, transactionCounter uint, subaccountIdHex string) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearCloseOption(orderId, exitPrice, transactionCounter, subaccountIdHex)
		return err == nil, err
	}
	arguments := abi.Arguments{
		{Type: uint256Ty},
		{Type: int128Ty},
	}

	encodedStruct, err := arguments.Pack(
		big.NewInt(int64(orderId)),
		exitPrice,
	)

	if err != nil {
		xlog.Errorf("Cannot encode CloseOptionBet struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.CLOSE_OPTIONS_BET_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subaccountIdHex, "", transaction, int64ToBytes(0), nil, "CloseOptionBet", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add CloseOptionBet transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) PreMarketOrderRequest(preMarketOrder contractUtils.PlacePreMarketOrderRequest, quoteDelta *big.Int, fees *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearPoolTrade(preMarketOrder, quoteDelta, fees, signatureHex, transactionCounter, false)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		xlog.Errorf("Cannot decode signing signature: %v", err)
		return false, err
	}

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(preMarketOrder.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	arguments := abi.Arguments{
		{Type: bytes32Ty}, // SubAccountId
		{Type: uint32Ty},  // ProductId
		{Type: int128Ty},  // Amount
		{Type: boolTy},    // IsBuy
		{Type: uint128Ty}, // Nonce
		{Type: addressTy}, // SessionKey
		{Type: uint256Ty}, // ChainId
		{Type: int128Ty},  // QuoteDelta
		{Type: int128Ty},  // Fees
	}

	// convert amount to *big.Int from string
	amount, ok := new(big.Int).SetString(preMarketOrder.Amount, 10)
	if !ok {
		xlog.Errorf("Cannot convert amount to big.Int: %v", err)
		return false, err
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		preMarketOrder.ProductId,
		amount,
		preMarketOrder.IsBuy,
		big.NewInt(int64(preMarketOrder.Nonce)),
		common.HexToAddress(preMarketOrder.SessionKey),
		big.NewInt(int64(preMarketOrder.ChainId)),
		quoteDelta,
		fees,
	)

	if err != nil {
		xlog.Errorf("Cannot encode PreMarketOrderRequest struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.PRE_MARKET_ORDER_REQUEST_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "PreMarketOrderRequest", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add PreMarketOrderRequest transaction to batch db failed with error %v", errDb)
		return false, errDb
	}

	return true, nil
}

func (epc *EndpointContract) SyntheticSpotOrderRequest(synSpotOrder contractUtils.PlacePreMarketOrderRequest, quoteDelta *big.Int, fees *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearPoolTrade(synSpotOrder, quoteDelta, fees, signatureHex, transactionCounter, true)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		xlog.Errorf("Cannot decode signing signature: %v", err)
		return false, err
	}

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(synSpotOrder.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	arguments := abi.Arguments{
		{Type: bytes32Ty}, // SubAccountId
		{Type: uint32Ty},  // ProductId
		{Type: int128Ty},  // Amount
		{Type: boolTy},    // IsBuy
		{Type: uint128Ty}, // Nonce
		{Type: addressTy}, // SessionKey
		{Type: uint256Ty}, // ChainId
		{Type: int128Ty},  // QuoteDelta
		{Type: int128Ty},  // Fees
	}

	// convert amount to *big.Int from string
	amount, ok := new(big.Int).SetString(synSpotOrder.Amount, 10)
	if !ok {
		xlog.Errorf("Cannot convert amount to big.Int: %v", err)
		return false, err
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		synSpotOrder.ProductId,
		amount,
		synSpotOrder.IsBuy,
		big.NewInt(int64(synSpotOrder.Nonce)),
		common.HexToAddress(synSpotOrder.SessionKey),
		big.NewInt(int64(synSpotOrder.ChainId)),
		quoteDelta,
		fees,
	)

	if err != nil {
		xlog.Errorf("Cannot encode SyntheticSpotOrderRequest struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.SYN_SPOT_ORDER_REQUEST_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "SyntheticSpotOrderRequest", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add SyntheticSpotOrderRequest transaction to batch db failed with error %v", errDb)
		return false, errDb
	}

	return true, nil
}

func (epc *EndpointContract) StakeLogx(stakeLogXRequest contractUtils.StakeLogXRequest, tokenAmount *big.Int, transientEarningsx18 *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		r := stakeLogXRequest
		err := epc.nearStake(true, r.SubAccountId, r.StakerContract, r.SessionKey, r.ProductId, r.Nonce, tokenAmount, signatureHex, transactionCounter)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		fmt.Printf("Cannot decode signing signature: %v", err)
		return false, err
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(stakeLogXRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	// ABI encode the StakeLogXRequest struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: uint32Ty},
		{Type: int128Ty},
		{Type: addressTy},
		{Type: addressTy},
		{Type: uint128Ty},
		{Type: uint256Ty},
		{Type: int256Ty},
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		stakeLogXRequest.ProductId,
		tokenAmount,
		common.HexToAddress(stakeLogXRequest.StakerContract),
		common.HexToAddress(stakeLogXRequest.SessionKey),
		big.NewInt(int64(stakeLogXRequest.Nonce)),
		big.NewInt(int64(stakeLogXRequest.ChainId)),
		transientEarningsx18,
	)
	if err != nil {
		fmt.Printf("Cannot encode StakeLogXRequest struct: %v", err)
		return false, err
	}

	// Create the transaction
	transaction := contractUtils.BuildTransactionArg(types.STAKE_LOGX_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "StakeLogx", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add StakeLogX transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) UnstakeLogx(unstakeLogXRequest contractUtils.UnstakeLogXRequest, amount *big.Int, transientEarningsx18 *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		r := unstakeLogXRequest
		err := epc.nearStake(false, r.SubAccountId, r.StakerContract, r.SessionKey, r.ProductId, r.Nonce, amount, signatureHex, transactionCounter)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		fmt.Printf("Cannot decode signing signature: %v", err)
		return false, err
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(unstakeLogXRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	// ABI encode the UnstakeLogXRequest struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: uint32Ty},
		{Type: int128Ty},
		{Type: addressTy},
		{Type: addressTy},
		{Type: uint128Ty},
		{Type: uint256Ty},
		{Type: int256Ty},
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		unstakeLogXRequest.ProductId,
		amount,
		common.HexToAddress(unstakeLogXRequest.StakerContract),
		common.HexToAddress(unstakeLogXRequest.SessionKey),
		big.NewInt(int64(unstakeLogXRequest.Nonce)),
		big.NewInt(int64(unstakeLogXRequest.ChainId)),
		transientEarningsx18,
	)
	if err != nil {
		fmt.Printf("Cannot encode StakeLogXRequest struct: %v", err)
		return false, err
	}

	// Create the transaction
	transaction := contractUtils.BuildTransactionArg(types.UNSTAKE_LOGX_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "UnstakeLogx", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add UnstakeLogX transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) WithdrawLogX(withdrawLogX contractUtils.WithdrawLogX, tokenAmount *big.Int, signatureHex string, transactionCounter uint) (bool, error, []byte) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear, nil // NEAR: NearWithdraw with product 0
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		fmt.Printf("Cannot decode signing signature: %v", err)
		return false, err, nil
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(withdrawLogX.SubAccountId)
	if err != nil {
		return false, err, nil
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])
	// ABI encode the WithdrawLogX struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: int128Ty},
		{Type: addressTy},
		{Type: uint256Ty},
		{Type: addressTy},
		{Type: uint128Ty},
		{Type: addressTy},
		{Type: uint256Ty},
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		tokenAmount,
		common.HexToAddress(withdrawLogX.SessionKey),
		big.NewInt(int64(withdrawLogX.DestinationChainId)),
		common.HexToAddress(withdrawLogX.BridgeOutContract),
		big.NewInt(int64(withdrawLogX.Nonce)),
		common.HexToAddress(withdrawLogX.Receiver),
		big.NewInt(int64(withdrawLogX.ChainId)),
	)

	if err != nil {
		fmt.Printf("Cannot encode StakeLogXRequest struct: %v", err)
		return false, err, nil
	}

	// Create the transaction
	transaction := contractUtils.BuildTransactionArg(types.WITHDRAW_LOGX_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "WithdrawLogX", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add WithdrawLogX transaction to batch db failed with error %v", errDb)
		return false, errDb, nil
	}
	return true, nil, encodedStruct
}

// claimablex18 is the LogX credited to the user (TotalClaimableEarningsx18); NEAR carries it in
// the transaction (D-6), the appchain recomputed it from transientEarningsx18.
func (epc *EndpointContract) ClaimRewards(claimRewards contractUtils.ClaimRewards, transientEarningsx18 *big.Int, claimablex18 *big.Int, signatureHex string, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		err := epc.nearClaimRewards(claimRewards, claimablex18, signatureHex, transactionCounter)
		return err == nil, err
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		fmt.Printf("Cannot decode signing signature: %v", err)
		return false, err
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(claimRewards.SubAccountId)
	if err != nil {
		return false, err
	}

	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	// ABI encode the ClaimRewards struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: addressTy},
		{Type: addressTy},
		{Type: uint32Ty},
		{Type: uint128Ty},
		{Type: uint256Ty},
		{Type: int256Ty},
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		common.HexToAddress(claimRewards.SessionKey),
		common.HexToAddress(claimRewards.StakerContract),
		claimRewards.ProductId,
		big.NewInt(int64(claimRewards.Nonce)),
		big.NewInt(int64(claimRewards.ChainId)),
		transientEarningsx18,
	)
	if err != nil {
		fmt.Printf("Cannot encode ClaimRewards struct: %v", err)
		return false, err
	}

	// Create the transaction
	transaction := contractUtils.BuildTransactionArg(types.CLAIM_REWARDS_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "ClaimRewards", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add ClaimRewards transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) WithdrawCollateral(withdrawCollateral contractUtils.WithdrawCollateral, amount *big.Int, signatureHex string, transactionCounter uint) (bool, error, []byte) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear, nil // NEAR: NearWithdraw binds the NEAR receiver
	}
	signature, err := hexutil.Decode(signatureHex)
	if err != nil {
		fmt.Printf("Cannot decode signing signature: %v", err)
		return false, err, nil
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(withdrawCollateral.SubAccountId)
	if err != nil {
		return false, err, nil
	}

	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])
	// ABI encode the ClaimRewards struct
	arguments := abi.Arguments{
		{Type: bytes32Ty},
		{Type: addressTy},
		{Type: uint32Ty},
		{Type: uint128Ty},
		{Type: uint128Ty},
		{Type: uint256Ty},
		{Type: addressTy},
		{Type: uint256Ty},
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		common.HexToAddress(withdrawCollateral.SessionKey),
		withdrawCollateral.ProductId,
		amount,
		big.NewInt(int64(withdrawCollateral.Nonce)),
		big.NewInt(int64(withdrawCollateral.DestinationChainId)),
		common.HexToAddress(withdrawCollateral.Receiver),
		big.NewInt(int64(withdrawCollateral.ChainId)),
	)

	if err != nil {
		fmt.Printf("Cannot encode WithdrawCollateral struct: %v", err)
		return false, err, nil
	}

	transaction := contractUtils.BuildTransactionArg(types.WITHDRAW_COLLATERAL_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, nil, "WithdrawCollateral", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add WithdrawCollateral transaction to batch db failed with error %v", errDb)
		return false, errDb, nil
	}
	return true, nil, encodedStruct
}

func (epc *EndpointContract) RewardRateTick(stakerContract string, earningRate *big.Int, transactionCounter uint) {
	if contractUtils.NearSettlement() {
		if err := epc.nearRewardRateTick(earningRate, transactionCounter); err != nil {
			xlog.Errorf("Add RewardRateTick to batch db failed: %v", err)
		}
		return
	}
	stakerAddress := common.HexToAddress(stakerContract)
	//ToDo - check if arguments' types are correct
	arguments := abi.Arguments{
		{
			Type: addressTy,
		},
		{
			Type: uint256Ty,
		},
	}
	encodedStruct, err := arguments.Pack(stakerAddress, earningRate)
	if err != nil {
		xlog.Errorf("Error packing struct: %v", err)
	}

	transactions := contractUtils.BuildTransactionArg(types.REWARD_RATE_TICK_TXN, encodedStruct)

	//ToDo - add correct function name
	errDb := AddTransactionToBatchDb("", "", transactions, int64ToBytes(0), nil, "RewardRateTick", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add EarnTick transaction to batch db failed with error %v", errDb)
	}
}

func (epc *EndpointContract) PerpTick(fundingRates []*big.Int, fundingTimestamp int64, transactionCounter uint) {
	if contractUtils.NearSettlement() {
		xlog.Errorf("PerpTick without product ids is not valid on NEAR (H-2): use FundingTick")
		return
	}
	arguments := abi.Arguments{
		{
			Type: uint128Ty,
		},
		{
			Type: int128ArrayTy,
		},
	}

	timeBigInt := new(big.Int).SetUint64(uint64(fundingTimestamp))

	perpTickBytes, err := arguments.Pack(
		timeBigInt,
		fundingRates,
	)
	if err != nil {
		log.Fatalf("Error packing struct: %v", err)
	}

	// enum number is 0 for PerpTick
	header := []byte{0x00}
	padding := make([]byte, 32)
	// Appending '32'
	padding[31] = 0x20

	// Combine header, padding, and packed payload
	finalPayload := append(header, padding...)
	finalPayload = append(finalPayload, perpTickBytes...)

	errDb := AddTransactionToBatchDb("", "", finalPayload, int64ToBytes(0), nil, "PerpTick", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add PerpTick transaction to batch db failed with error %v", errDb)
	}
}

// Contains locking mechanism to have sequential order matching
func (epc *EndpointContract) MatchOrders(matchOrderRequest MatchOrderRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		epc._mutex.Lock()
		defer epc._mutex.Unlock()
		return epc.nearMatchOrders(matchOrderRequest, transactionCounter)
	}
	epc._mutex.Lock()
	defer epc._mutex.Unlock()

	takerSubaccountIdBytes, err := cutils.SubaccountIdToBytes32(matchOrderRequest.TakerSubaccountId)
	if err != nil {
		return err
	}
	takerSubAccountIdHex := "0x" + hex.EncodeToString(takerSubaccountIdBytes[:])

	makerSubaccountIdBytes, err := cutils.SubaccountIdToBytes32(matchOrderRequest.MakerSubaccountId)
	if err != nil {
		return err
	}
	makerSubAccountIdHex := "0x" + hex.EncodeToString(makerSubaccountIdBytes[:])
	makerSignature := common.FromHex(matchOrderRequest.MakerSignature)
	takerSignature := common.FromHex(matchOrderRequest.TakerSignature)

	// These amounts are in absolute value and we are converting them to negative if the side is sell
	takerAmount := matchOrderRequest.TakerMaxAmountX18
	makerAmount := matchOrderRequest.MakerMaxAmountX18
	matchedAmount := matchOrderRequest.MatchedAmountx18
	if matchOrderRequest.TakerSide == ctypes.ORDER_SIDE_SELL {
		takerAmount = new(big.Int).Mul(takerAmount, big.NewInt(-1))
	} else {
		// Sign of matched amount is same as maker side
		makerAmount = new(big.Int).Mul(makerAmount, big.NewInt(-1))
		matchedAmount = new(big.Int).Mul(matchedAmount, big.NewInt(-1))
	}

	matchOrderPayload := MatchOrderPayload{
		ProductId: matchOrderRequest.ProductId,
		Taker: Order{
			SubAccountId: takerSubaccountIdBytes,
			PriceX18:     matchOrderRequest.TakerPriceX18,
			Amount:       takerAmount,
			Expiration:   matchOrderRequest.TakerExpiryTs,
			IsReduce:     matchOrderRequest.TakerIsReduce,
			SessionKey:   common.HexToAddress(matchOrderRequest.TakerSessionKey),
			ChainId:      big.NewInt(contractUtils.SESSION_KEY_CHAIN_ID),
		},
		Maker: Order{
			SubAccountId: makerSubaccountIdBytes,
			PriceX18:     matchOrderRequest.MakerPriceX18,
			Amount:       makerAmount,
			Expiration:   matchOrderRequest.MakerExpiryTs,
			IsReduce:     matchOrderRequest.MakerIsReduce,
			SessionKey:   common.HexToAddress(matchOrderRequest.MakerSessionKey),
			ChainId:      big.NewInt(contractUtils.SESSION_KEY_CHAIN_ID),
		},
		MatchedAmountx18: matchedAmount,
	}

	bytes, err := json.Marshal(matchOrderPayload)
	if err != nil {
		fmt.Printf("Cannot marshal match order request: %v", err)
		return err
	}
	fmt.Printf("Match Order Request: %v\n\n", string(bytes))

	var arguments = abi.Arguments{}
	var encodedStruct []byte

	if os.Getenv("UPGRADED_CONTRACT") != "1" {
		// This will be same as existing match orders struct. We don't want to change this before testing on testnet
		arguments = abi.Arguments{
			{Type: uint32Ty},
			{Type: orderType},
			{Type: orderType},
		}

		encodedStruct, err = arguments.Pack(
			matchOrderPayload.ProductId,
			matchOrderPayload.Taker,
			matchOrderPayload.Maker,
		)
	} else {
		arguments = abi.Arguments{
			{Type: uint32Ty},
			{Type: orderType},
			{Type: orderType},
			{Type: int128Ty},
		}

		encodedStruct, err = arguments.Pack(
			matchOrderPayload.ProductId,
			matchOrderPayload.Taker,
			matchOrderPayload.Maker,
			matchOrderPayload.MatchedAmountx18,
		)
	}

	if err != nil {
		fmt.Printf("Cannot encode struct: %v", err)
		return err
	}

	// fmt.Printf("Encoded struct: %v\n", hexutil.Encode(encodedStruct))
	transaction := contractUtils.BuildTransactionArg(types.MATCH_ORDERS_TXN, encodedStruct)

	errDb := AddTransactionToBatchDb(takerSubAccountIdHex, makerSubAccountIdHex, transaction, takerSignature, makerSignature, "MatchOrders", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add MatchOrders transaction to batch db failed with error %v", errDb)
		return errDb
	}
	return nil
}

// Convert subaccount id to bytes32
func (epc *EndpointContract) SettleUserPnl(settleUserPnlRequest contractUtils.SettleUserPnlRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return epc.nearSettleUserPnl(settleUserPnlRequest, transactionCounter)
	}
	var subaccounts [][32]byte
	for _, subaccount := range settleUserPnlRequest.SubaccountIds {

		subaccountIdBytes, err := cutils.SubAccountIdStrToBytes32(subaccount)

		if err != nil {
			xlog.Errorf("SettlePnl - Error while converting subaccountId to bytes. Err: %v", err)
		}

		subaccounts = append(subaccounts, subaccountIdBytes)
	}

	arguments := abi.Arguments{
		{Type: bytes32ArrayTy},
		{Type: uint32ArrayTy},
		{Type: int128ArrayTy},
	}

	encodedStruct, err := arguments.Pack(
		subaccounts,
		settleUserPnlRequest.ProductIds,
		settleUserPnlRequest.OraclePrices,
	)

	if err != nil {
		fmt.Printf("Settle Pnl - Cannot encode settle-pnl struct: %v", err)
		return err
	}

	bytes, _ := json.Marshal(settleUserPnlRequest)
	xlog.Infof("Settle Pnl - Settle User Pnl Request: %v\n", string(bytes))

	seqTransaction := contractUtils.BuildDynamicTransactionArg(types.SETTLE_USER_PNL_TXN, encodedStruct)

	var subaccountId1 = ""

	if len(subaccounts) == 1 {
		subaccountId1 = settleUserPnlRequest.SubaccountIds[0]
	}

	errDb := AddTransactionToBatchDb(subaccountId1, "", seqTransaction, int64ToBytes(0), nil, transaction.SETTLE_USER_PNL_FN, transactionCounter)
	if errDb != nil {
		xlog.Errorf("Settle Pnl - Add SettleUserPnl transaction to batch db failed with error %v", errDb)
		return errDb
	}

	return nil
}

func (epc *EndpointContract) ShiftBalance(settleBroker2PnLRequest contractUtils.SettleBroker2PnL, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(settleBroker2PnLRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	arguments := abi.Arguments{
		{Type: bytes32Ty},
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
	)
	if err != nil {
		xlog.Errorf("Cannot encode ShiftBalance struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.SHIFT_BALANCE_TXN, encodedStruct)

	// No signatures are required for this transaction
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, int64ToBytes(0), nil, "ShiftBalance", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add ShiftBalance transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) ShiftBalanceKroma(shiftBalanceKromaRequest contractUtils.ShiftBalanceKroma, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(shiftBalanceKromaRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	arguments := abi.Arguments{
		{Type: bytes32Ty},
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
	)
	if err != nil {
		xlog.Errorf("Cannot encode ShiftBalanceKroma struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.SHIFT_BALANCE_KROMA_TXN, encodedStruct)

	// No signatures are required for this transaction
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, int64ToBytes(0), nil, "ShiftBalanceKroma", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add ShiftBalanceKroma transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) BurnBalanceKroma(burnBalanceKromaRequest contractUtils.BurnBalanceKroma, transactionCounter uint) (bool, error) {
	if contractUtils.NearSettlement() {
		return false, ErrNotOnNear
	}
	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(burnBalanceKromaRequest.SubAccountId)
	if err != nil {
		return false, err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	arguments := abi.Arguments{
		{Type: bytes32Ty},
	}
	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
	)
	if err != nil {
		xlog.Errorf("Cannot encode BurnBalanceKroma struct: %v", err)
		return false, err
	}

	transaction := contractUtils.BuildTransactionArg(types.BURN_BALANCE_KROMA_TXN, encodedStruct)

	// No signatures are required for this transaction
	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, int64ToBytes(0), nil, "BurnBalanceKroma", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add BurnBalanceKroma transaction to batch db failed with error %v", errDb)
		return false, errDb
	}
	return true, nil
}

func (epc *EndpointContract) RegisterSessionKey(sessionRequest contractUtils.SessionRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return ErrNotOnNear // users call register_session_key on the contract (§6.3)
	}
	epc._mutex.Lock()
	defer epc._mutex.Unlock()

	subaccountIdBytes, err := cutils.SubaccountIdToBytes32(sessionRequest.SubaccountId)
	if err != nil {
		return err
	}
	subAccountIdHex := "0x" + hex.EncodeToString(subaccountIdBytes[:])

	contractSessionRequest := gen.AccountsRegister{
		UserAddress:     common.HexToAddress(sessionRequest.EthAddress),
		SubAccountId:    subaccountIdBytes,
		SessionKey:      common.HexToAddress(sessionRequest.SigningAddress),
		ExpiryTimeStamp: big.NewInt(sessionRequest.ExpiryTs),
		Nonce:           big.NewInt(sessionRequest.Nonce),
		ChainId:         big.NewInt(sessionRequest.ChainId),
	}

	signature, err1 := hexutil.Decode(sessionRequest.SigningSignature)
	masterSignature, err2 := hexutil.Decode(sessionRequest.EthSignature)
	if err1 != nil {
		fmt.Printf("Cannot decode signing signature: %v", err1)
		return err1
	}
	if err2 != nil {
		fmt.Printf("Cannot decode master: %v", err2)
		return err2
	}

	arguments := abi.Arguments{
		{Type: bytes32Ty}, // subAccountId
		{Type: addressTy}, // userAddress
		{Type: addressTy}, // sessionKey
		{Type: uint128Ty}, // expiryTimeStamp
		{Type: uint128Ty}, // nonce
		{Type: uint256Ty}, // chainId
	}

	encodedStruct, err := arguments.Pack(
		contractSessionRequest.SubAccountId,
		contractSessionRequest.UserAddress,
		contractSessionRequest.SessionKey,
		contractSessionRequest.ExpiryTimeStamp,
		contractSessionRequest.Nonce,
		contractSessionRequest.ChainId,
	)
	if err != nil {
		fmt.Printf("Cannot encode struct: %v", err)
		return err
	}

	transaction := contractUtils.BuildTransactionArg(types.REGISTER_TXN, encodedStruct)
	//ToDo - double check if signature and master signature are being passed in the right orde

	errDb := AddTransactionToBatchDb(subAccountIdHex, "", transaction, signature, masterSignature, "RegisterSessionKey", transactionCounter)
	if errDb != nil {
		xlog.Errorf("Add RegisterSessionKey transaction to batch db failed with error %v", errDb)
		return errDb
	}
	return nil

}

// ----------------- Contract Call Functions (Read) -----------------

// Following are templates for balance server which have not been fully tested yet.
// GetSubmissionIndex fetches the current submission index from the contract
// func (epc *EndpointContract) GetSubmissionIndex() (uint64, error) {
// 	idx, err := epc._endpoint.NSubmissions(&bind.CallOpts{})
// 	if err != nil {
// 		return 0, err
// 	}
// 	return idx, nil
// }

// func (ecp *EndpointContract) GetNonceForSubaccount(subaccountId string) (uint64, error) {
// 	subaccountIdBytes, err := cutils.SubAccountIdStrToBytes32(subaccountId)
// 	if err != nil {
// 		return 0, err
// 	}

// 	nonce, err := ecp._endpoint.GetNonce(&bind.CallOpts{}, subaccountIdBytes)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return nonce.Uint64(), nil
// }

// func (ecp *EndpointContract) GetNoncesForSubaccounts(subaccountIds []string) ([]uint64, error) {
// 	// Convert subaccount ID strings to bytes32 format
// 	subaccountIdBytes := make([][32]byte, len(subaccountIds))
// 	for i, id := range subaccountIds {
// 		bytes32Id, err := cutils.SubAccountIdStrToBytes32(id)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to convert subaccountId %s to bytes32: %v", id, err)
// 		}
// 		subaccountIdBytes[i] = bytes32Id
// 	}

// 	// Call the contract's getNoncesOfSubaccounts function
// 	nonces, err := ecp._endpoint.GetNoncesOfSubaccounts(&bind.CallOpts{}, subaccountIdBytes)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to retrieve nonces for subaccount IDs: %v", err)
// 	}

// 	convertedNonces := make([]uint64, len(nonces))
// 	for i, nonce := range nonces {
// 		convertedNonces[i] = nonce.Uint64()
// 	}

// 	return convertedNonces, nil
// }

// func (epc *EndpointContract) GetNSubmissions() (uint64, error) {
// 	return epc._endpoint.NSubmissions(&bind.CallOpts{})
// }

// ----------------- Contract Call Functions (Write) -----------------
// func (epc *EndpointContract) SubmitBatchTransactionsChecked(transactions, signatures1, signatures2 [][]byte) (bool, uint64, error) {
// 	// Get the current submission index
// 	idx, err := epc._endpoint.NSubmissions(&bind.CallOpts{})
// 	if err != nil {
// 		xlog.Errorf("Failed to get submission index: %v", err)
// 		return false, 0, err
// 	}

// 	// Submit the batch of transactions with signatures
// 	txn, err := epc._endpoint.SubmitTransactionsChecked(epc._txnOpts, idx, transactions, signatures1, signatures2)
// 	if err != nil {
// 		xlog.Errorf("Failed to submit transactions: %v", err)
// 		return false, 0, err
// 	}

// 	xlog.Infof("Submit Batch Transactions Hash: %v and nSubIdx: %v\n", txn.Hash().Hex(), idx)

// 	maxRetries := 2

// 	for attempt := 1; attempt <= maxRetries; attempt++ {
// 		// Check the transaction status
// 		txnCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 		defer cancel()

// 		success, err := WaitnCheckTxnStatus(txnCtx, epc._client, txn.Hash())
// 		if err != nil {
// 			xlog.Errorf("Attempt %d: Error checking transaction status: %v", attempt, err)
// 			if attempt == maxRetries {
// 				return false, 0, err
// 			}
// 			continue
// 		}

// 		if success {
// 			return true, idx, nil
// 		}
// 	}

// 	// If the transaction was not successful after retries
// 	return false, idx, fmt.Errorf("transaction failed after %d attempts", maxRetries)
// }

// ----------------- Estimate Gas Functions -----------------
// func (epc *EndpointContract) EstimateGasFeesForSubmitBatchTransactionsChecked(transactions, signatures1, signatures2 [][]byte) (uint64, error) {
// 	// Get the current submission index
// 	idx, err := epc._endpoint.NSubmissions(&bind.CallOpts{})
// 	if err != nil {
// 		return 0, err
// 	}

// 	// Pack the arguments for the SubmitTransactionsChecked function
// 	contractABI, err := abi.JSON(strings.NewReader(string(gen.EndpointABI)))
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to parse contract ABI: %v", err)
// 	}

// 	packedData, err := contractABI.Pack("submitTransactionsChecked", idx, transactions, signatures1, signatures2)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to pack arguments: %v", err)
// 	}

// 	// Create a call message for estimating gas
// 	msg := ethereum.CallMsg{
// 		From: epc._txnOpts.From,
// 		To:   &epc.contractAddress,
// 		Data: packedData,
// 	}

// 	// Estimate the gas required for the transaction
// 	gasLimit, err := epc._client.EstimateGas(context.Background(), msg)
// 	if err != nil {
// 		return 0, fmt.Errorf("failed to estimate gas: %v", err)
// 	}

// 	return gasLimit, nil
// }

// ----------------- Helper Functions -----------------
func int64ToBytes(value int64) []byte {
	bi := big.NewInt(value)
	b := bi.Bytes()
	// Ensure the byte array is 32 bytes long
	padded := make([]byte, 32-len(b))
	padded = append(padded, b...)
	return padded
}

type LiquidationRequest struct {
	ProductId              uint32
	LiquidatorSubaccountId string
	LiquidatorPriceX18     *big.Int
	LiquidatorAmount       *big.Int
	LiquidatorExpiryTs     uint64
	LiquidatorIsReduce     bool
	LiquidatorSessionKey   string
	LiquidatorSide         ctypes.OrderSide

	LiquidateeSubaccountId string
	LiquidateeAmount       *big.Int
	PerpOraclePricesX18    map[uint32]*big.Int
	SpotOraclePricesX18    map[uint32]*big.Int
	LiquidateeSide         ctypes.OrderSide

	LiquidatorSignature string
	MatchedAmountx18    *big.Int
}

type LiquidateeOrder struct {
	SubAccountId [32]byte
	Amount       *big.Int
}

type LiquidationPayload struct {
	ProductId           uint32
	Liquidator          Order
	Liquidaatee         LiquidateeOrder
	PerpOraclePricesX18 []*big.Int
	SpotOraclePricesX18 []*big.Int
	MatchedAmountX18    *big.Int
}

func (epc *EndpointContract) LiquidateSubaccount(liquidationRequest LiquidationRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return epc.nearLiquidate(liquidationRequest, transactionCounter)
	}

	liquidatorSignature := common.FromHex(liquidationRequest.LiquidatorSignature)
	// Liquidator
	liquidatorSubaccountBytes, err := cutils.SubaccountIdToBytes32(liquidationRequest.LiquidatorSubaccountId)
	if err != nil {
		return err
	}

	liquidateeSubacountBytes, err := cutils.SubaccountIdToBytes32(liquidationRequest.LiquidateeSubaccountId)
	if err != nil {
		return err
	}

	liquidateeAmountX18 := liquidationRequest.LiquidateeAmount
	liquidatorAmountX18 := liquidationRequest.LiquidatorAmount
	matchedAmountX18 := liquidationRequest.MatchedAmountx18

	if liquidationRequest.LiquidateeSide == ctypes.ORDER_SIDE_SELL {
		liquidateeAmountX18 = new(big.Int).Mul(liquidateeAmountX18, big.NewInt(-1))
	} else {
		// Sign of matched amount is same as liquidator amount sign
		liquidatorAmountX18 = new(big.Int).Mul(liquidatorAmountX18, big.NewInt(-1))
		matchedAmountX18 = new(big.Int).Mul(matchedAmountX18, big.NewInt(-1))
	}

	liquidationPayload := LiquidationPayload{
		ProductId: liquidationRequest.ProductId,
		Liquidator: Order{
			SubAccountId: liquidatorSubaccountBytes,
			PriceX18:     liquidationRequest.LiquidatorPriceX18,
			Amount:       liquidatorAmountX18,
			Expiration:   liquidationRequest.LiquidatorExpiryTs,
			IsReduce:     liquidationRequest.LiquidatorIsReduce,
			SessionKey:   common.HexToAddress(liquidationRequest.LiquidatorSessionKey),
			ChainId:      big.NewInt(contractUtils.SESSION_KEY_CHAIN_ID),
		},
		Liquidaatee: LiquidateeOrder{
			SubAccountId: liquidateeSubacountBytes,
			Amount:       liquidateeAmountX18,
		},
		PerpOraclePricesX18: GetPerpPricesX18SliceForContract(liquidationRequest.PerpOraclePricesX18),
		SpotOraclePricesX18: spotUtils.GetSpotPricesX18SliceForContract(liquidationRequest.SpotOraclePricesX18),
		MatchedAmountX18:    matchedAmountX18,
	}
	// xlog.Debugf("Liquidation Request: %+v", liquidationRequest)
	jsonPayload, e := json.Marshal(liquidationPayload)
	if e == nil {
		xlog.Debugf("Liquidation Payload - %v: %+v", liquidationRequest.LiquidateeSubaccountId, string(jsonPayload))
	}

	var encodedStruct []byte

	if os.Getenv("UPGRADED_CONTRACT") != "1" {
		// This will be same as existing struct. We don't want to change this before testing on testnet

		// struct LiquidateSubaccount on the contract
		arguments := abi.Arguments{
			{Type: uint32Ty},            // productId
			{Type: int128ArrayTy},       // perpOraclePricesX18
			{Type: int128ArrayTy},       // spotOraclePricesX18
			{Type: orderType},           // liquidator
			{Type: liquidateeOrderType}, // liquidatee
		}

		encodedStruct, err = arguments.Pack(
			liquidationPayload.ProductId,
			liquidationPayload.PerpOraclePricesX18,
			liquidationPayload.SpotOraclePricesX18,
			liquidationPayload.Liquidator,
			liquidationPayload.Liquidaatee,
		)
	} else {
		// struct LiquidateSubaccount on the contract
		arguments := abi.Arguments{
			{Type: uint32Ty},            // productId
			{Type: int128ArrayTy},       // perpOraclePricesX18
			{Type: int128ArrayTy},       // spotOraclePricesX18
			{Type: orderType},           // liquidator
			{Type: liquidateeOrderType}, // liquidatee
			{Type: int128Ty},            // matchedAmountX18
		}

		encodedStruct, err = arguments.Pack(
			liquidationPayload.ProductId,
			liquidationPayload.PerpOraclePricesX18,
			liquidationPayload.SpotOraclePricesX18,
			liquidationPayload.Liquidator,
			liquidationPayload.Liquidaatee,
			liquidationPayload.MatchedAmountX18,
		)
	}

	if err != nil {
		fmt.Printf("Cannot encode struct: %v", err)
		return err
	}

	liquidateeSubaccountIdHex := cutils.Bytes32ToSubaccountHex(liquidateeSubacountBytes)
	liquidatorSubaccountIdHex := cutils.Bytes32ToSubaccountHex(liquidatorSubaccountBytes)

	transaction := contractUtils.BuildDynamicTransactionArg(types.LIQUIDATE_SUBACCOUNT_TXN, encodedStruct)

	// xlog.Debugf("LiquidateSubaccount transaction: %v", hexutil.Encode(transaction))

	err = AddTransactionToBatchDb(liquidateeSubaccountIdHex, liquidatorSubaccountIdHex, transaction, liquidatorSignature, nil, "LiquidateSubaccount", transactionCounter)
	if err != nil {
		xlog.Errorf("Add LiquidateSubaccount transaction to batch db failed with error %v", err)
		return err
	}

	return nil
}

type SocialiseSubaccountRequest struct {
	ProductIds      []uint32
	SubaccountId    string
	OraclePricesX18 []*big.Int
}

var DUMMY_SIGNATURE = common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000")

func (epc *EndpointContract) SocialiseSubaccount(socialiseSubaccountRequest SocialiseSubaccountRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return epc.nearSocialise(socialiseSubaccountRequest, transactionCounter)
	}
	subaccountBytes, err := cutils.SubaccountIdToBytes32(socialiseSubaccountRequest.SubaccountId)
	if err != nil {
		return err
	}

	arguments := abi.Arguments{
		{Type: bytes32Ty},     // SubaccountId
		{Type: uint32ArrayTy}, // SpotProductIds
		{Type: int128ArrayTy}, // SpotPricesPricesX18
	}

	encodedStruct, err := arguments.Pack(
		subaccountBytes,
		socialiseSubaccountRequest.ProductIds,
		socialiseSubaccountRequest.OraclePricesX18,
	)

	if err != nil {
		xlog.Errorf("Cannot encode struct: %v", err)
		return err
	}

	transaction := contractUtils.BuildDynamicTransactionArg(types.SOCIALISE_SUBACCOUNT_TXN, encodedStruct)

	subaccountIdHex := cutils.Bytes32ToSubaccountHex(subaccountBytes)

	err = AddTransactionToBatchDb(subaccountIdHex, "", transaction, DUMMY_SIGNATURE, nil, "SocialiseSubaccount", transactionCounter)
	if err != nil {
		xlog.Errorf("Add SocialiseSubaccount transaction to batch db failed with error %v", err)
		return err
	}
	return nil
}

// Keeping delta as uint64 for now so, we only increase by positive values
func (epc *EndpointContract) SetNonce(subaccountIdHex string, delta uint64, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return epc.nearSetNonce(subaccountIdHex, delta, transactionCounter)
	}
	subaccountBytes, err := cutils.HexToSubaccountBytes32(subaccountIdHex)
	if err != nil {
		return err
	}

	arguments := abi.Arguments{
		{Type: bytes32Ty}, // SubaccountId
		{Type: int128Ty},  // Delta
	}

	encodedStruct, err := arguments.Pack(
		subaccountBytes,
		big.NewInt(int64(delta)),
	)

	if err != nil {
		xlog.Errorf("Cannot encode struct: %v", err)
		return err
	}

	transaction := contractUtils.BuildTransactionArg(types.SET_NONCE_TXN, encodedStruct)
	err = AddTransactionToBatchDb(subaccountIdHex, "", transaction, DUMMY_SIGNATURE, nil, "SetNonce", transactionCounter, true)
	if err != nil {
		xlog.Errorf("Add SetNonce transaction to batch db failed with error %v", err)
		return err
	}
	return nil
}

type FinaliseDepositRequest struct {
	SubaccountId  string
	ProductId     uint32
	Amount        *big.Int
	SourceChainId *big.Int
}

func (epc *EndpointContract) FinalisePendingDeposits(finaliseDepositRequest FinaliseDepositRequest, transactionCounter uint) error {
	if contractUtils.NearSettlement() {
		return ErrNotOnNear // deposits arrive through ft_on_transfer (§7.1)
	}

	subaccountIdBytes, err := cutils.HexToBytes32(finaliseDepositRequest.SubaccountId)
	if err != nil {
		return err
	}

	// struct FinaliseDeposit on the contract
	arguments := abi.Arguments{
		{Type: bytes32Ty}, // subAccountId
		{Type: uint32Ty},  // productId
		{Type: int128Ty},  // amount
		{Type: uint256Ty}, // sourceChainId
	}

	encodedStruct, err := arguments.Pack(
		subaccountIdBytes,
		finaliseDepositRequest.ProductId,
		finaliseDepositRequest.Amount,
		finaliseDepositRequest.SourceChainId,
	)

	if err != nil {
		xlog.Errorf("Cannot encode struct: %v", err)
		return err
	}

	subaccountIdHex := finaliseDepositRequest.SubaccountId
	transaction := contractUtils.BuildTransactionArg(types.FINALISE_DEPOSIT_TXN, encodedStruct)

	err = AddTransactionToBatchDb(subaccountIdHex, "", transaction, int64ToBytes(0), nil, "FinaliseDeposit", transactionCounter)
	if err != nil {
		xlog.Errorf("Add FinaliseDeposit transaction to batch db failed with error %v", err)
		return err
	}

	return nil
}

// hexToDecimal converts a hexadecimal string to a decimal string.
func hexToDecimal(hexStr string) string {
	// Remove the "0x" prefix if present
	if len(hexStr) > 1 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	decimalValue := new(big.Int)

	// SetString with base 16 (hexadecimal)
	decimalValue.SetString(hexStr, 16)

	// Return the decimal value as a string
	return decimalValue.String()
}

// Function to convert hexadecimal string subAccountId to [32]byte
func HexStringToBytes32(hexStr string) ([32]byte, error) {
	var result [32]byte

	// Decode hexadecimal string to bytes
	bytes, err := hex.DecodeString(strings.TrimPrefix(hexStr, "0x"))
	if err != nil {
		return result, fmt.Errorf("failed to decode hexadecimal string: %v", err)
	}

	// Copy bytes into result array
	copy(result[:], bytes)

	return result, nil
}

func GetPerpPricesX18SliceForContract(perpOraclePricesX18 map[uint32]*big.Int) []*big.Int {
	perpPricesX18 := make([]*big.Int, 0)
	for _, productId := range contractUtils.ALL_PERPS_ON_CONTRACT {
		if perpOraclePricesX18[productId] == nil {
			perpPricesX18 = append(perpPricesX18, big.NewInt(0))
		} else {
			perpPricesX18 = append(perpPricesX18, perpOraclePricesX18[productId])
		}
	}
	return perpPricesX18
}
