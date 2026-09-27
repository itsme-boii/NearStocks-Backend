// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package gen

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// IProductEngineCoreRisk is an auto generated low-level Go binding around an user-defined struct.
type IProductEngineCoreRisk struct {
	Amount     *big.Int
	Price      *big.Int
	LongWeight *big.Int
}

// ISpotEngineBalances is an auto generated low-level Go binding around an user-defined struct.
type ISpotEngineBalances struct {
	Amount *big.Int
}

// ISpotEngineTokenInfo is an auto generated low-level Go binding around an user-defined struct.
type ISpotEngineTokenInfo struct {
	Token                  common.Address
	WithdrawFeeBasisPoints *big.Int
	TotalDeposits          *big.Int
	SizeIncrement          *big.Int
	MinSize                *big.Int
	AssetWeight            uint32
	AvailableSettle        *big.Int
}

// ISpotEngineUpdateProductTx is an auto generated low-level Go binding around an user-defined struct.
type ISpotEngineUpdateProductTx struct {
	ProductId              uint32
	SizeIncrement          *big.Int
	MinSize                *big.Int
	Token                  common.Address
	WithdrawFeeBasisPoints *big.Int
	AssetWeight            uint32
	AvailableSettle        *big.Int
}

// RiskHelperRisk is an auto generated low-level Go binding around an user-defined struct.
type RiskHelperRisk struct {
	LongWeightInitialX18      *big.Int
	ShortWeightInitialX18     *big.Int
	LongWeightMaintenanceX18  *big.Int
	ShortWeightMaintenanceX18 *big.Int
	PriceX18                  *big.Int
}

// RiskHelperRiskStore is an auto generated low-level Go binding around an user-defined struct.
type RiskHelperRiskStore struct {
	LongWeightInitial      int32
	ShortWeightInitial     int32
	LongWeightMaintenance  int32
	ShortWeightMaintenance int32
	PriceX18               *big.Int
}

// SpotMetaData contains all meta data concerning the Spot contract.
var SpotMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"InvalidInitialization\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotInitializing\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnableInvalidOwner\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"OwnableUnauthorizedAccount\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"AddProduct\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"}],\"name\":\"BalanceUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint64\",\"name\":\"version\",\"type\":\"uint64\"}],\"name\":\"Initialized\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"ProductUpdate\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint32\",\"name\":\"isoGroup\",\"type\":\"uint32\"}],\"name\":\"QuoteProductUpdate\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"address\",\"name\":\"book\",\"type\":\"address\"},{\"internalType\":\"int128\",\"name\":\"sizeIncrement\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"minSize\",\"type\":\"int128\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"},{\"internalType\":\"int128\",\"name\":\"withdrawFeeBasisPoints\",\"type\":\"int128\"},{\"internalType\":\"uint32\",\"name\":\"assetWeight\",\"type\":\"uint32\"}],\"name\":\"addProduct\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_addr\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"value\",\"type\":\"bool\"}],\"name\":\"configureAssertInternal\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"name\":\"defaultFeeRates\",\"outputs\":[{\"internalType\":\"int64\",\"name\":\"makerRateX18\",\"type\":\"int64\"},{\"internalType\":\"int64\",\"name\":\"takerRateX18\",\"type\":\"int64\"},{\"internalType\":\"uint8\",\"name\":\"isNonDefault\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"subaccounts\",\"type\":\"bytes32[]\"}],\"name\":\"getAllBalancesOfSubaccounts\",\"outputs\":[{\"components\":[{\"internalType\":\"int128\",\"name\":\"amount\",\"type\":\"int128\"}],\"internalType\":\"structISpotEngine.Balances[][]\",\"name\":\"\",\"type\":\"tuple[][]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getAvailableSettle\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"}],\"name\":\"getBalance\",\"outputs\":[{\"components\":[{\"internalType\":\"int128\",\"name\":\"amount\",\"type\":\"int128\"}],\"internalType\":\"structISpotEngine.Balances\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getClearinghouse\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"enumIProductEngine.HealthType\",\"name\":\"healthType\",\"type\":\"uint8\"}],\"name\":\"getCoreRisk\",\"outputs\":[{\"components\":[{\"internalType\":\"int128\",\"name\":\"amount\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"price\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"longWeight\",\"type\":\"int128\"}],\"internalType\":\"structIProductEngine.CoreRisk\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"startAt\",\"type\":\"uint32\"},{\"internalType\":\"uint32\",\"name\":\"limit\",\"type\":\"uint32\"}],\"name\":\"getCustomFeeSubAccounts\",\"outputs\":[{\"internalType\":\"bytes32[]\",\"name\":\"\",\"type\":\"bytes32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getEndpoint\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getEngineType\",\"outputs\":[{\"internalType\":\"enumIProductEngine.EngineType\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"bool\",\"name\":\"taker\",\"type\":\"bool\"}],\"name\":\"getFeeFractionX18\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getFeeRatesX18\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"int128[]\",\"name\":\"spotPricesX18\",\"type\":\"int128[]\"}],\"name\":\"getHealthContribution\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"health\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getProductIds\",\"outputs\":[{\"internalType\":\"uint32[]\",\"name\":\"\",\"type\":\"uint32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"isoGroup\",\"type\":\"uint32\"}],\"name\":\"getProductIds\",\"outputs\":[{\"internalType\":\"uint32[]\",\"name\":\"\",\"type\":\"uint32[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getRisk\",\"outputs\":[{\"components\":[{\"internalType\":\"int128\",\"name\":\"longWeightInitialX18\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"shortWeightInitialX18\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"longWeightMaintenanceX18\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"shortWeightMaintenanceX18\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"priceX18\",\"type\":\"int128\"}],\"internalType\":\"structRiskHelper.Risk\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getToken\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getTokenAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"}],\"name\":\"getTokenInfoAndBalance\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"int128\",\"name\":\"withdrawFeeBasisPoints\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"totalDeposits\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"sizeIncrement\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"minSize\",\"type\":\"int128\"},{\"internalType\":\"uint32\",\"name\":\"assetWeight\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"availableSettle\",\"type\":\"int128\"}],\"internalType\":\"structISpotEngine.TokenInfo\",\"name\":\"\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"int128\",\"name\":\"amount\",\"type\":\"int128\"}],\"internalType\":\"structISpotEngine.Balances\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getVersion\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getWithdrawFee\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"getWithdrawFees\",\"outputs\":[{\"internalType\":\"int128\",\"name\":\"\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_clearinghouse\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_offchainExchange\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_quote\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_endpoint\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"_admin\",\"type\":\"address\"}],\"name\":\"initialize\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int128[]\",\"name\":\"totalDeposits\",\"type\":\"int128[]\"}],\"name\":\"manualAssert\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"migrationFlag\",\"outputs\":[{\"internalType\":\"uint64\",\"name\":\"\",\"type\":\"uint64\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"int64\",\"name\":\"makerRateX18\",\"type\":\"int64\"},{\"internalType\":\"int64\",\"name\":\"takerRateX18\",\"type\":\"int64\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"setDefaultFeeRates\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_endpoint\",\"type\":\"address\"}],\"name\":\"setEndpoint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"name\":\"tokenInfo\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"int128\",\"name\":\"withdrawFeeBasisPoints\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"totalDeposits\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"sizeIncrement\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"minSize\",\"type\":\"int128\"},{\"internalType\":\"uint32\",\"name\":\"assetWeight\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"availableSettle\",\"type\":\"int128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"tokenProductId\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"availableSettleDelta\",\"type\":\"int128\"}],\"name\":\"updateAvailableSettle\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"int128\",\"name\":\"amountDelta\",\"type\":\"int128\"}],\"name\":\"updateBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"bytes32\",\"name\":\"subaccount\",\"type\":\"bytes32\"},{\"internalType\":\"int128\",\"name\":\"amountDelta\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"quoteDelta\",\"type\":\"int128\"}],\"name\":\"updateBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"int64\",\"name\":\"makerRateX18\",\"type\":\"int64\"},{\"internalType\":\"int64\",\"name\":\"takerRateX18\",\"type\":\"int64\"}],\"name\":\"updateFeeRates\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"priceX18\",\"type\":\"int128\"}],\"name\":\"updatePrice\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"sizeIncrement\",\"type\":\"int128\"},{\"internalType\":\"int128\",\"name\":\"minSize\",\"type\":\"int128\"},{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"int128\",\"name\":\"withdrawFeeBasisPoints\",\"type\":\"int128\"},{\"internalType\":\"uint32\",\"name\":\"assetWeight\",\"type\":\"uint32\"},{\"internalType\":\"int128\",\"name\":\"availableSettle\",\"type\":\"int128\"}],\"internalType\":\"structISpotEngine.UpdateProductTx\",\"name\":\"txn\",\"type\":\"tuple\"}],\"name\":\"updateProduct\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"},{\"components\":[{\"internalType\":\"int32\",\"name\":\"longWeightInitial\",\"type\":\"int32\"},{\"internalType\":\"int32\",\"name\":\"shortWeightInitial\",\"type\":\"int32\"},{\"internalType\":\"int32\",\"name\":\"longWeightMaintenance\",\"type\":\"int32\"},{\"internalType\":\"int32\",\"name\":\"shortWeightMaintenance\",\"type\":\"int32\"},{\"internalType\":\"int128\",\"name\":\"priceX18\",\"type\":\"int128\"}],\"internalType\":\"structRiskHelper.RiskStore\",\"name\":\"riskStore\",\"type\":\"tuple\"}],\"name\":\"updateRisk\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"token\",\"type\":\"address\"},{\"internalType\":\"uint32\",\"name\":\"productId\",\"type\":\"uint32\"}],\"name\":\"updateTokenProductIdMapping\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// SpotABI is the input ABI used to generate the binding from.
// Deprecated: Use SpotMetaData.ABI instead.
var SpotABI = SpotMetaData.ABI

// Spot is an auto generated Go binding around an Ethereum contract.
type Spot struct {
	SpotCaller     // Read-only binding to the contract
	SpotTransactor // Write-only binding to the contract
	SpotFilterer   // Log filterer for contract events
}

// SpotCaller is an auto generated read-only Go binding around an Ethereum contract.
type SpotCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SpotTransactor is an auto generated write-only Go binding around an Ethereum contract.
type SpotTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SpotFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type SpotFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// SpotSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type SpotSession struct {
	Contract     *Spot             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SpotCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type SpotCallerSession struct {
	Contract *SpotCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// SpotTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type SpotTransactorSession struct {
	Contract     *SpotTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// SpotRaw is an auto generated low-level Go binding around an Ethereum contract.
type SpotRaw struct {
	Contract *Spot // Generic contract binding to access the raw methods on
}

// SpotCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type SpotCallerRaw struct {
	Contract *SpotCaller // Generic read-only contract binding to access the raw methods on
}

// SpotTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type SpotTransactorRaw struct {
	Contract *SpotTransactor // Generic write-only contract binding to access the raw methods on
}

// NewSpot creates a new instance of Spot, bound to a specific deployed contract.
func NewSpot(address common.Address, backend bind.ContractBackend) (*Spot, error) {
	contract, err := bindSpot(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Spot{SpotCaller: SpotCaller{contract: contract}, SpotTransactor: SpotTransactor{contract: contract}, SpotFilterer: SpotFilterer{contract: contract}}, nil
}

// NewSpotCaller creates a new read-only instance of Spot, bound to a specific deployed contract.
func NewSpotCaller(address common.Address, caller bind.ContractCaller) (*SpotCaller, error) {
	contract, err := bindSpot(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &SpotCaller{contract: contract}, nil
}

// NewSpotTransactor creates a new write-only instance of Spot, bound to a specific deployed contract.
func NewSpotTransactor(address common.Address, transactor bind.ContractTransactor) (*SpotTransactor, error) {
	contract, err := bindSpot(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &SpotTransactor{contract: contract}, nil
}

// NewSpotFilterer creates a new log filterer instance of Spot, bound to a specific deployed contract.
func NewSpotFilterer(address common.Address, filterer bind.ContractFilterer) (*SpotFilterer, error) {
	contract, err := bindSpot(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &SpotFilterer{contract: contract}, nil
}

// bindSpot binds a generic wrapper to an already deployed contract.
func bindSpot(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := SpotMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Spot *SpotRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Spot.Contract.SpotCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Spot *SpotRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Spot.Contract.SpotTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Spot *SpotRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Spot.Contract.SpotTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Spot *SpotCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Spot.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Spot *SpotTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Spot.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Spot *SpotTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Spot.Contract.contract.Transact(opts, method, params...)
}

// DefaultFeeRates is a free data retrieval call binding the contract method 0x443389bd.
//
// Solidity: function defaultFeeRates(uint32 ) view returns(int64 makerRateX18, int64 takerRateX18, uint8 isNonDefault)
func (_Spot *SpotCaller) DefaultFeeRates(opts *bind.CallOpts, arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "defaultFeeRates", arg0)

	outstruct := new(struct {
		MakerRateX18 int64
		TakerRateX18 int64
		IsNonDefault uint8
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.MakerRateX18 = *abi.ConvertType(out[0], new(int64)).(*int64)
	outstruct.TakerRateX18 = *abi.ConvertType(out[1], new(int64)).(*int64)
	outstruct.IsNonDefault = *abi.ConvertType(out[2], new(uint8)).(*uint8)

	return *outstruct, err

}

// DefaultFeeRates is a free data retrieval call binding the contract method 0x443389bd.
//
// Solidity: function defaultFeeRates(uint32 ) view returns(int64 makerRateX18, int64 takerRateX18, uint8 isNonDefault)
func (_Spot *SpotSession) DefaultFeeRates(arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	return _Spot.Contract.DefaultFeeRates(&_Spot.CallOpts, arg0)
}

// DefaultFeeRates is a free data retrieval call binding the contract method 0x443389bd.
//
// Solidity: function defaultFeeRates(uint32 ) view returns(int64 makerRateX18, int64 takerRateX18, uint8 isNonDefault)
func (_Spot *SpotCallerSession) DefaultFeeRates(arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	return _Spot.Contract.DefaultFeeRates(&_Spot.CallOpts, arg0)
}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128)[][])
func (_Spot *SpotCaller) GetAllBalancesOfSubaccounts(opts *bind.CallOpts, subaccounts [][32]byte) ([][]ISpotEngineBalances, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getAllBalancesOfSubaccounts", subaccounts)

	if err != nil {
		return *new([][]ISpotEngineBalances), err
	}

	out0 := *abi.ConvertType(out[0], new([][]ISpotEngineBalances)).(*[][]ISpotEngineBalances)

	return out0, err

}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128)[][])
func (_Spot *SpotSession) GetAllBalancesOfSubaccounts(subaccounts [][32]byte) ([][]ISpotEngineBalances, error) {
	return _Spot.Contract.GetAllBalancesOfSubaccounts(&_Spot.CallOpts, subaccounts)
}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128)[][])
func (_Spot *SpotCallerSession) GetAllBalancesOfSubaccounts(subaccounts [][32]byte) ([][]ISpotEngineBalances, error) {
	return _Spot.Contract.GetAllBalancesOfSubaccounts(&_Spot.CallOpts, subaccounts)
}

// GetAvailableSettle is a free data retrieval call binding the contract method 0x3056f78f.
//
// Solidity: function getAvailableSettle(uint32 productId) view returns(int128)
func (_Spot *SpotCaller) GetAvailableSettle(opts *bind.CallOpts, productId uint32) (*big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getAvailableSettle", productId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAvailableSettle is a free data retrieval call binding the contract method 0x3056f78f.
//
// Solidity: function getAvailableSettle(uint32 productId) view returns(int128)
func (_Spot *SpotSession) GetAvailableSettle(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetAvailableSettle(&_Spot.CallOpts, productId)
}

// GetAvailableSettle is a free data retrieval call binding the contract method 0x3056f78f.
//
// Solidity: function getAvailableSettle(uint32 productId) view returns(int128)
func (_Spot *SpotCallerSession) GetAvailableSettle(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetAvailableSettle(&_Spot.CallOpts, productId)
}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128))
func (_Spot *SpotCaller) GetBalance(opts *bind.CallOpts, productId uint32, subaccount [32]byte) (ISpotEngineBalances, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getBalance", productId, subaccount)

	if err != nil {
		return *new(ISpotEngineBalances), err
	}

	out0 := *abi.ConvertType(out[0], new(ISpotEngineBalances)).(*ISpotEngineBalances)

	return out0, err

}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128))
func (_Spot *SpotSession) GetBalance(productId uint32, subaccount [32]byte) (ISpotEngineBalances, error) {
	return _Spot.Contract.GetBalance(&_Spot.CallOpts, productId, subaccount)
}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128))
func (_Spot *SpotCallerSession) GetBalance(productId uint32, subaccount [32]byte) (ISpotEngineBalances, error) {
	return _Spot.Contract.GetBalance(&_Spot.CallOpts, productId, subaccount)
}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Spot *SpotCaller) GetClearinghouse(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getClearinghouse")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Spot *SpotSession) GetClearinghouse() (common.Address, error) {
	return _Spot.Contract.GetClearinghouse(&_Spot.CallOpts)
}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Spot *SpotCallerSession) GetClearinghouse() (common.Address, error) {
	return _Spot.Contract.GetClearinghouse(&_Spot.CallOpts)
}

// GetCoreRisk is a free data retrieval call binding the contract method 0x8a1d43c9.
//
// Solidity: function getCoreRisk(bytes32 subaccount, uint32 productId, uint8 healthType) view returns((int128,int128,int128))
func (_Spot *SpotCaller) GetCoreRisk(opts *bind.CallOpts, subaccount [32]byte, productId uint32, healthType uint8) (IProductEngineCoreRisk, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getCoreRisk", subaccount, productId, healthType)

	if err != nil {
		return *new(IProductEngineCoreRisk), err
	}

	out0 := *abi.ConvertType(out[0], new(IProductEngineCoreRisk)).(*IProductEngineCoreRisk)

	return out0, err

}

// GetCoreRisk is a free data retrieval call binding the contract method 0x8a1d43c9.
//
// Solidity: function getCoreRisk(bytes32 subaccount, uint32 productId, uint8 healthType) view returns((int128,int128,int128))
func (_Spot *SpotSession) GetCoreRisk(subaccount [32]byte, productId uint32, healthType uint8) (IProductEngineCoreRisk, error) {
	return _Spot.Contract.GetCoreRisk(&_Spot.CallOpts, subaccount, productId, healthType)
}

// GetCoreRisk is a free data retrieval call binding the contract method 0x8a1d43c9.
//
// Solidity: function getCoreRisk(bytes32 subaccount, uint32 productId, uint8 healthType) view returns((int128,int128,int128))
func (_Spot *SpotCallerSession) GetCoreRisk(subaccount [32]byte, productId uint32, healthType uint8) (IProductEngineCoreRisk, error) {
	return _Spot.Contract.GetCoreRisk(&_Spot.CallOpts, subaccount, productId, healthType)
}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Spot *SpotCaller) GetCustomFeeSubAccounts(opts *bind.CallOpts, startAt uint32, limit uint32) ([][32]byte, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getCustomFeeSubAccounts", startAt, limit)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Spot *SpotSession) GetCustomFeeSubAccounts(startAt uint32, limit uint32) ([][32]byte, error) {
	return _Spot.Contract.GetCustomFeeSubAccounts(&_Spot.CallOpts, startAt, limit)
}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Spot *SpotCallerSession) GetCustomFeeSubAccounts(startAt uint32, limit uint32) ([][32]byte, error) {
	return _Spot.Contract.GetCustomFeeSubAccounts(&_Spot.CallOpts, startAt, limit)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Spot *SpotCaller) GetEndpoint(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getEndpoint")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Spot *SpotSession) GetEndpoint() (common.Address, error) {
	return _Spot.Contract.GetEndpoint(&_Spot.CallOpts)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Spot *SpotCallerSession) GetEndpoint() (common.Address, error) {
	return _Spot.Contract.GetEndpoint(&_Spot.CallOpts)
}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Spot *SpotCaller) GetEngineType(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getEngineType")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Spot *SpotSession) GetEngineType() (uint8, error) {
	return _Spot.Contract.GetEngineType(&_Spot.CallOpts)
}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Spot *SpotCallerSession) GetEngineType() (uint8, error) {
	return _Spot.Contract.GetEngineType(&_Spot.CallOpts)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Spot *SpotCaller) GetFeeFractionX18(opts *bind.CallOpts, subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getFeeFractionX18", subaccount, productId, taker)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Spot *SpotSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _Spot.Contract.GetFeeFractionX18(&_Spot.CallOpts, subaccount, productId, taker)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Spot *SpotCallerSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _Spot.Contract.GetFeeFractionX18(&_Spot.CallOpts, subaccount, productId, taker)
}

// GetFeeRatesX18 is a free data retrieval call binding the contract method 0x0f2c878e.
//
// Solidity: function getFeeRatesX18(bytes32 subaccount, uint32 productId) view returns(int128, int128)
func (_Spot *SpotCaller) GetFeeRatesX18(opts *bind.CallOpts, subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getFeeRatesX18", subaccount, productId)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetFeeRatesX18 is a free data retrieval call binding the contract method 0x0f2c878e.
//
// Solidity: function getFeeRatesX18(bytes32 subaccount, uint32 productId) view returns(int128, int128)
func (_Spot *SpotSession) GetFeeRatesX18(subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	return _Spot.Contract.GetFeeRatesX18(&_Spot.CallOpts, subaccount, productId)
}

// GetFeeRatesX18 is a free data retrieval call binding the contract method 0x0f2c878e.
//
// Solidity: function getFeeRatesX18(bytes32 subaccount, uint32 productId) view returns(int128, int128)
func (_Spot *SpotCallerSession) GetFeeRatesX18(subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	return _Spot.Contract.GetFeeRatesX18(&_Spot.CallOpts, subaccount, productId)
}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] spotPricesX18) view returns(int128 health)
func (_Spot *SpotCaller) GetHealthContribution(opts *bind.CallOpts, subaccount [32]byte, spotPricesX18 []*big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getHealthContribution", subaccount, spotPricesX18)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] spotPricesX18) view returns(int128 health)
func (_Spot *SpotSession) GetHealthContribution(subaccount [32]byte, spotPricesX18 []*big.Int) (*big.Int, error) {
	return _Spot.Contract.GetHealthContribution(&_Spot.CallOpts, subaccount, spotPricesX18)
}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] spotPricesX18) view returns(int128 health)
func (_Spot *SpotCallerSession) GetHealthContribution(subaccount [32]byte, spotPricesX18 []*big.Int) (*big.Int, error) {
	return _Spot.Contract.GetHealthContribution(&_Spot.CallOpts, subaccount, spotPricesX18)
}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Spot *SpotCaller) GetProductIds(opts *bind.CallOpts) ([]uint32, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getProductIds")

	if err != nil {
		return *new([]uint32), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint32)).(*[]uint32)

	return out0, err

}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Spot *SpotSession) GetProductIds() ([]uint32, error) {
	return _Spot.Contract.GetProductIds(&_Spot.CallOpts)
}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Spot *SpotCallerSession) GetProductIds() ([]uint32, error) {
	return _Spot.Contract.GetProductIds(&_Spot.CallOpts)
}

// GetProductIds0 is a free data retrieval call binding the contract method 0xf4c8c58d.
//
// Solidity: function getProductIds(uint32 isoGroup) view returns(uint32[])
func (_Spot *SpotCaller) GetProductIds0(opts *bind.CallOpts, isoGroup uint32) ([]uint32, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getProductIds0", isoGroup)

	if err != nil {
		return *new([]uint32), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint32)).(*[]uint32)

	return out0, err

}

// GetProductIds0 is a free data retrieval call binding the contract method 0xf4c8c58d.
//
// Solidity: function getProductIds(uint32 isoGroup) view returns(uint32[])
func (_Spot *SpotSession) GetProductIds0(isoGroup uint32) ([]uint32, error) {
	return _Spot.Contract.GetProductIds0(&_Spot.CallOpts, isoGroup)
}

// GetProductIds0 is a free data retrieval call binding the contract method 0xf4c8c58d.
//
// Solidity: function getProductIds(uint32 isoGroup) view returns(uint32[])
func (_Spot *SpotCallerSession) GetProductIds0(isoGroup uint32) ([]uint32, error) {
	return _Spot.Contract.GetProductIds0(&_Spot.CallOpts, isoGroup)
}

// GetRisk is a free data retrieval call binding the contract method 0xecd9cba8.
//
// Solidity: function getRisk(uint32 productId) view returns((int128,int128,int128,int128,int128))
func (_Spot *SpotCaller) GetRisk(opts *bind.CallOpts, productId uint32) (RiskHelperRisk, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getRisk", productId)

	if err != nil {
		return *new(RiskHelperRisk), err
	}

	out0 := *abi.ConvertType(out[0], new(RiskHelperRisk)).(*RiskHelperRisk)

	return out0, err

}

// GetRisk is a free data retrieval call binding the contract method 0xecd9cba8.
//
// Solidity: function getRisk(uint32 productId) view returns((int128,int128,int128,int128,int128))
func (_Spot *SpotSession) GetRisk(productId uint32) (RiskHelperRisk, error) {
	return _Spot.Contract.GetRisk(&_Spot.CallOpts, productId)
}

// GetRisk is a free data retrieval call binding the contract method 0xecd9cba8.
//
// Solidity: function getRisk(uint32 productId) view returns((int128,int128,int128,int128,int128))
func (_Spot *SpotCallerSession) GetRisk(productId uint32) (RiskHelperRisk, error) {
	return _Spot.Contract.GetRisk(&_Spot.CallOpts, productId)
}

// GetToken is a free data retrieval call binding the contract method 0x45be7ed6.
//
// Solidity: function getToken(uint32 productId) view returns(address)
func (_Spot *SpotCaller) GetToken(opts *bind.CallOpts, productId uint32) (common.Address, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getToken", productId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetToken is a free data retrieval call binding the contract method 0x45be7ed6.
//
// Solidity: function getToken(uint32 productId) view returns(address)
func (_Spot *SpotSession) GetToken(productId uint32) (common.Address, error) {
	return _Spot.Contract.GetToken(&_Spot.CallOpts, productId)
}

// GetToken is a free data retrieval call binding the contract method 0x45be7ed6.
//
// Solidity: function getToken(uint32 productId) view returns(address)
func (_Spot *SpotCallerSession) GetToken(productId uint32) (common.Address, error) {
	return _Spot.Contract.GetToken(&_Spot.CallOpts, productId)
}

// GetTokenAddress is a free data retrieval call binding the contract method 0x543d7a19.
//
// Solidity: function getTokenAddress(uint32 productId) view returns(address)
func (_Spot *SpotCaller) GetTokenAddress(opts *bind.CallOpts, productId uint32) (common.Address, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getTokenAddress", productId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetTokenAddress is a free data retrieval call binding the contract method 0x543d7a19.
//
// Solidity: function getTokenAddress(uint32 productId) view returns(address)
func (_Spot *SpotSession) GetTokenAddress(productId uint32) (common.Address, error) {
	return _Spot.Contract.GetTokenAddress(&_Spot.CallOpts, productId)
}

// GetTokenAddress is a free data retrieval call binding the contract method 0x543d7a19.
//
// Solidity: function getTokenAddress(uint32 productId) view returns(address)
func (_Spot *SpotCallerSession) GetTokenAddress(productId uint32) (common.Address, error) {
	return _Spot.Contract.GetTokenAddress(&_Spot.CallOpts, productId)
}

// GetTokenInfoAndBalance is a free data retrieval call binding the contract method 0x7da21105.
//
// Solidity: function getTokenInfoAndBalance(uint32 productId, bytes32 subaccount) view returns((address,int128,int128,int128,int128,uint32,int128), (int128))
func (_Spot *SpotCaller) GetTokenInfoAndBalance(opts *bind.CallOpts, productId uint32, subaccount [32]byte) (ISpotEngineTokenInfo, ISpotEngineBalances, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getTokenInfoAndBalance", productId, subaccount)

	if err != nil {
		return *new(ISpotEngineTokenInfo), *new(ISpotEngineBalances), err
	}

	out0 := *abi.ConvertType(out[0], new(ISpotEngineTokenInfo)).(*ISpotEngineTokenInfo)
	out1 := *abi.ConvertType(out[1], new(ISpotEngineBalances)).(*ISpotEngineBalances)

	return out0, out1, err

}

// GetTokenInfoAndBalance is a free data retrieval call binding the contract method 0x7da21105.
//
// Solidity: function getTokenInfoAndBalance(uint32 productId, bytes32 subaccount) view returns((address,int128,int128,int128,int128,uint32,int128), (int128))
func (_Spot *SpotSession) GetTokenInfoAndBalance(productId uint32, subaccount [32]byte) (ISpotEngineTokenInfo, ISpotEngineBalances, error) {
	return _Spot.Contract.GetTokenInfoAndBalance(&_Spot.CallOpts, productId, subaccount)
}

// GetTokenInfoAndBalance is a free data retrieval call binding the contract method 0x7da21105.
//
// Solidity: function getTokenInfoAndBalance(uint32 productId, bytes32 subaccount) view returns((address,int128,int128,int128,int128,uint32,int128), (int128))
func (_Spot *SpotCallerSession) GetTokenInfoAndBalance(productId uint32, subaccount [32]byte) (ISpotEngineTokenInfo, ISpotEngineBalances, error) {
	return _Spot.Contract.GetTokenInfoAndBalance(&_Spot.CallOpts, productId, subaccount)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Spot *SpotCaller) GetVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Spot *SpotSession) GetVersion() (uint64, error) {
	return _Spot.Contract.GetVersion(&_Spot.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Spot *SpotCallerSession) GetVersion() (uint64, error) {
	return _Spot.Contract.GetVersion(&_Spot.CallOpts)
}

// GetWithdrawFee is a free data retrieval call binding the contract method 0xfdf4a0c0.
//
// Solidity: function getWithdrawFee(uint32 productId) view returns(int128)
func (_Spot *SpotCaller) GetWithdrawFee(opts *bind.CallOpts, productId uint32) (*big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getWithdrawFee", productId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawFee is a free data retrieval call binding the contract method 0xfdf4a0c0.
//
// Solidity: function getWithdrawFee(uint32 productId) view returns(int128)
func (_Spot *SpotSession) GetWithdrawFee(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetWithdrawFee(&_Spot.CallOpts, productId)
}

// GetWithdrawFee is a free data retrieval call binding the contract method 0xfdf4a0c0.
//
// Solidity: function getWithdrawFee(uint32 productId) view returns(int128)
func (_Spot *SpotCallerSession) GetWithdrawFee(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetWithdrawFee(&_Spot.CallOpts, productId)
}

// GetWithdrawFees is a free data retrieval call binding the contract method 0x18a69ef1.
//
// Solidity: function getWithdrawFees(uint32 productId) view returns(int128)
func (_Spot *SpotCaller) GetWithdrawFees(opts *bind.CallOpts, productId uint32) (*big.Int, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "getWithdrawFees", productId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawFees is a free data retrieval call binding the contract method 0x18a69ef1.
//
// Solidity: function getWithdrawFees(uint32 productId) view returns(int128)
func (_Spot *SpotSession) GetWithdrawFees(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetWithdrawFees(&_Spot.CallOpts, productId)
}

// GetWithdrawFees is a free data retrieval call binding the contract method 0x18a69ef1.
//
// Solidity: function getWithdrawFees(uint32 productId) view returns(int128)
func (_Spot *SpotCallerSession) GetWithdrawFees(productId uint32) (*big.Int, error) {
	return _Spot.Contract.GetWithdrawFees(&_Spot.CallOpts, productId)
}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] totalDeposits) view returns()
func (_Spot *SpotCaller) ManualAssert(opts *bind.CallOpts, totalDeposits []*big.Int) error {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "manualAssert", totalDeposits)

	if err != nil {
		return err
	}

	return err

}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] totalDeposits) view returns()
func (_Spot *SpotSession) ManualAssert(totalDeposits []*big.Int) error {
	return _Spot.Contract.ManualAssert(&_Spot.CallOpts, totalDeposits)
}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] totalDeposits) view returns()
func (_Spot *SpotCallerSession) ManualAssert(totalDeposits []*big.Int) error {
	return _Spot.Contract.ManualAssert(&_Spot.CallOpts, totalDeposits)
}

// MigrationFlag is a free data retrieval call binding the contract method 0xc362d19e.
//
// Solidity: function migrationFlag() view returns(uint64)
func (_Spot *SpotCaller) MigrationFlag(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "migrationFlag")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// MigrationFlag is a free data retrieval call binding the contract method 0xc362d19e.
//
// Solidity: function migrationFlag() view returns(uint64)
func (_Spot *SpotSession) MigrationFlag() (uint64, error) {
	return _Spot.Contract.MigrationFlag(&_Spot.CallOpts)
}

// MigrationFlag is a free data retrieval call binding the contract method 0xc362d19e.
//
// Solidity: function migrationFlag() view returns(uint64)
func (_Spot *SpotCallerSession) MigrationFlag() (uint64, error) {
	return _Spot.Contract.MigrationFlag(&_Spot.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Spot *SpotCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Spot *SpotSession) Owner() (common.Address, error) {
	return _Spot.Contract.Owner(&_Spot.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Spot *SpotCallerSession) Owner() (common.Address, error) {
	return _Spot.Contract.Owner(&_Spot.CallOpts)
}

// TokenInfo is a free data retrieval call binding the contract method 0x68fb1109.
//
// Solidity: function tokenInfo(uint32 ) view returns(address token, int128 withdrawFeeBasisPoints, int128 totalDeposits, int128 sizeIncrement, int128 minSize, uint32 assetWeight, int128 availableSettle)
func (_Spot *SpotCaller) TokenInfo(opts *bind.CallOpts, arg0 uint32) (struct {
	Token                  common.Address
	WithdrawFeeBasisPoints *big.Int
	TotalDeposits          *big.Int
	SizeIncrement          *big.Int
	MinSize                *big.Int
	AssetWeight            uint32
	AvailableSettle        *big.Int
}, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "tokenInfo", arg0)

	outstruct := new(struct {
		Token                  common.Address
		WithdrawFeeBasisPoints *big.Int
		TotalDeposits          *big.Int
		SizeIncrement          *big.Int
		MinSize                *big.Int
		AssetWeight            uint32
		AvailableSettle        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Token = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.WithdrawFeeBasisPoints = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.TotalDeposits = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.SizeIncrement = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.MinSize = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)
	outstruct.AssetWeight = *abi.ConvertType(out[5], new(uint32)).(*uint32)
	outstruct.AvailableSettle = *abi.ConvertType(out[6], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// TokenInfo is a free data retrieval call binding the contract method 0x68fb1109.
//
// Solidity: function tokenInfo(uint32 ) view returns(address token, int128 withdrawFeeBasisPoints, int128 totalDeposits, int128 sizeIncrement, int128 minSize, uint32 assetWeight, int128 availableSettle)
func (_Spot *SpotSession) TokenInfo(arg0 uint32) (struct {
	Token                  common.Address
	WithdrawFeeBasisPoints *big.Int
	TotalDeposits          *big.Int
	SizeIncrement          *big.Int
	MinSize                *big.Int
	AssetWeight            uint32
	AvailableSettle        *big.Int
}, error) {
	return _Spot.Contract.TokenInfo(&_Spot.CallOpts, arg0)
}

// TokenInfo is a free data retrieval call binding the contract method 0x68fb1109.
//
// Solidity: function tokenInfo(uint32 ) view returns(address token, int128 withdrawFeeBasisPoints, int128 totalDeposits, int128 sizeIncrement, int128 minSize, uint32 assetWeight, int128 availableSettle)
func (_Spot *SpotCallerSession) TokenInfo(arg0 uint32) (struct {
	Token                  common.Address
	WithdrawFeeBasisPoints *big.Int
	TotalDeposits          *big.Int
	SizeIncrement          *big.Int
	MinSize                *big.Int
	AssetWeight            uint32
	AvailableSettle        *big.Int
}, error) {
	return _Spot.Contract.TokenInfo(&_Spot.CallOpts, arg0)
}

// TokenProductId is a free data retrieval call binding the contract method 0x2679b68d.
//
// Solidity: function tokenProductId(address ) view returns(uint32)
func (_Spot *SpotCaller) TokenProductId(opts *bind.CallOpts, arg0 common.Address) (uint32, error) {
	var out []interface{}
	err := _Spot.contract.Call(opts, &out, "tokenProductId", arg0)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// TokenProductId is a free data retrieval call binding the contract method 0x2679b68d.
//
// Solidity: function tokenProductId(address ) view returns(uint32)
func (_Spot *SpotSession) TokenProductId(arg0 common.Address) (uint32, error) {
	return _Spot.Contract.TokenProductId(&_Spot.CallOpts, arg0)
}

// TokenProductId is a free data retrieval call binding the contract method 0x2679b68d.
//
// Solidity: function tokenProductId(address ) view returns(uint32)
func (_Spot *SpotCallerSession) TokenProductId(arg0 common.Address) (uint32, error) {
	return _Spot.Contract.TokenProductId(&_Spot.CallOpts, arg0)
}

// AddProduct is a paid mutator transaction binding the contract method 0xc26b4928.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, address tokenAddress, int128 withdrawFeeBasisPoints, uint32 assetWeight) returns()
func (_Spot *SpotTransactor) AddProduct(opts *bind.TransactOpts, productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, tokenAddress common.Address, withdrawFeeBasisPoints *big.Int, assetWeight uint32) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "addProduct", productId, book, sizeIncrement, minSize, tokenAddress, withdrawFeeBasisPoints, assetWeight)
}

// AddProduct is a paid mutator transaction binding the contract method 0xc26b4928.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, address tokenAddress, int128 withdrawFeeBasisPoints, uint32 assetWeight) returns()
func (_Spot *SpotSession) AddProduct(productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, tokenAddress common.Address, withdrawFeeBasisPoints *big.Int, assetWeight uint32) (*types.Transaction, error) {
	return _Spot.Contract.AddProduct(&_Spot.TransactOpts, productId, book, sizeIncrement, minSize, tokenAddress, withdrawFeeBasisPoints, assetWeight)
}

// AddProduct is a paid mutator transaction binding the contract method 0xc26b4928.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, address tokenAddress, int128 withdrawFeeBasisPoints, uint32 assetWeight) returns()
func (_Spot *SpotTransactorSession) AddProduct(productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, tokenAddress common.Address, withdrawFeeBasisPoints *big.Int, assetWeight uint32) (*types.Transaction, error) {
	return _Spot.Contract.AddProduct(&_Spot.TransactOpts, productId, book, sizeIncrement, minSize, tokenAddress, withdrawFeeBasisPoints, assetWeight)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Spot *SpotTransactor) ConfigureAssertInternal(opts *bind.TransactOpts, _addr common.Address, value bool) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "configureAssertInternal", _addr, value)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Spot *SpotSession) ConfigureAssertInternal(_addr common.Address, value bool) (*types.Transaction, error) {
	return _Spot.Contract.ConfigureAssertInternal(&_Spot.TransactOpts, _addr, value)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Spot *SpotTransactorSession) ConfigureAssertInternal(_addr common.Address, value bool) (*types.Transaction, error) {
	return _Spot.Contract.ConfigureAssertInternal(&_Spot.TransactOpts, _addr, value)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address _quote, address _endpoint, address _admin) returns()
func (_Spot *SpotTransactor) Initialize(opts *bind.TransactOpts, _clearinghouse common.Address, _offchainExchange common.Address, _quote common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "initialize", _clearinghouse, _offchainExchange, _quote, _endpoint, _admin)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address _quote, address _endpoint, address _admin) returns()
func (_Spot *SpotSession) Initialize(_clearinghouse common.Address, _offchainExchange common.Address, _quote common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Spot.Contract.Initialize(&_Spot.TransactOpts, _clearinghouse, _offchainExchange, _quote, _endpoint, _admin)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address _quote, address _endpoint, address _admin) returns()
func (_Spot *SpotTransactorSession) Initialize(_clearinghouse common.Address, _offchainExchange common.Address, _quote common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Spot.Contract.Initialize(&_Spot.TransactOpts, _clearinghouse, _offchainExchange, _quote, _endpoint, _admin)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Spot *SpotTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Spot *SpotSession) RenounceOwnership() (*types.Transaction, error) {
	return _Spot.Contract.RenounceOwnership(&_Spot.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Spot *SpotTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Spot.Contract.RenounceOwnership(&_Spot.TransactOpts)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Spot *SpotTransactor) SetDefaultFeeRates(opts *bind.TransactOpts, makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "setDefaultFeeRates", makerRateX18, takerRateX18, productId)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Spot *SpotSession) SetDefaultFeeRates(makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Spot.Contract.SetDefaultFeeRates(&_Spot.TransactOpts, makerRateX18, takerRateX18, productId)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Spot *SpotTransactorSession) SetDefaultFeeRates(makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Spot.Contract.SetDefaultFeeRates(&_Spot.TransactOpts, makerRateX18, takerRateX18, productId)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Spot *SpotTransactor) SetEndpoint(opts *bind.TransactOpts, _endpoint common.Address) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "setEndpoint", _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Spot *SpotSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _Spot.Contract.SetEndpoint(&_Spot.TransactOpts, _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Spot *SpotTransactorSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _Spot.Contract.SetEndpoint(&_Spot.TransactOpts, _endpoint)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Spot *SpotTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Spot *SpotSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Spot.Contract.TransferOwnership(&_Spot.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Spot *SpotTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Spot.Contract.TransferOwnership(&_Spot.TransactOpts, newOwner)
}

// UpdateAvailableSettle is a paid mutator transaction binding the contract method 0x7fbaee8e.
//
// Solidity: function updateAvailableSettle(uint32 productId, int128 availableSettleDelta) returns()
func (_Spot *SpotTransactor) UpdateAvailableSettle(opts *bind.TransactOpts, productId uint32, availableSettleDelta *big.Int) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateAvailableSettle", productId, availableSettleDelta)
}

// UpdateAvailableSettle is a paid mutator transaction binding the contract method 0x7fbaee8e.
//
// Solidity: function updateAvailableSettle(uint32 productId, int128 availableSettleDelta) returns()
func (_Spot *SpotSession) UpdateAvailableSettle(productId uint32, availableSettleDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateAvailableSettle(&_Spot.TransactOpts, productId, availableSettleDelta)
}

// UpdateAvailableSettle is a paid mutator transaction binding the contract method 0x7fbaee8e.
//
// Solidity: function updateAvailableSettle(uint32 productId, int128 availableSettleDelta) returns()
func (_Spot *SpotTransactorSession) UpdateAvailableSettle(productId uint32, availableSettleDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateAvailableSettle(&_Spot.TransactOpts, productId, availableSettleDelta)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xe0b0621f.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta) returns()
func (_Spot *SpotTransactor) UpdateBalance(opts *bind.TransactOpts, productId uint32, subaccount [32]byte, amountDelta *big.Int) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateBalance", productId, subaccount, amountDelta)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xe0b0621f.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta) returns()
func (_Spot *SpotSession) UpdateBalance(productId uint32, subaccount [32]byte, amountDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateBalance(&_Spot.TransactOpts, productId, subaccount, amountDelta)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xe0b0621f.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta) returns()
func (_Spot *SpotTransactorSession) UpdateBalance(productId uint32, subaccount [32]byte, amountDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateBalance(&_Spot.TransactOpts, productId, subaccount, amountDelta)
}

// UpdateBalance0 is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 quoteDelta) returns()
func (_Spot *SpotTransactor) UpdateBalance0(opts *bind.TransactOpts, productId uint32, subaccount [32]byte, amountDelta *big.Int, quoteDelta *big.Int) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateBalance0", productId, subaccount, amountDelta, quoteDelta)
}

// UpdateBalance0 is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 quoteDelta) returns()
func (_Spot *SpotSession) UpdateBalance0(productId uint32, subaccount [32]byte, amountDelta *big.Int, quoteDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateBalance0(&_Spot.TransactOpts, productId, subaccount, amountDelta, quoteDelta)
}

// UpdateBalance0 is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 quoteDelta) returns()
func (_Spot *SpotTransactorSession) UpdateBalance0(productId uint32, subaccount [32]byte, amountDelta *big.Int, quoteDelta *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdateBalance0(&_Spot.TransactOpts, productId, subaccount, amountDelta, quoteDelta)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Spot *SpotTransactor) UpdateFeeRates(opts *bind.TransactOpts, subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateFeeRates", subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Spot *SpotSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Spot.Contract.UpdateFeeRates(&_Spot.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Spot *SpotTransactorSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Spot.Contract.UpdateFeeRates(&_Spot.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdatePrice is a paid mutator transaction binding the contract method 0x153ca6c0.
//
// Solidity: function updatePrice(uint32 productId, int128 priceX18) returns()
func (_Spot *SpotTransactor) UpdatePrice(opts *bind.TransactOpts, productId uint32, priceX18 *big.Int) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updatePrice", productId, priceX18)
}

// UpdatePrice is a paid mutator transaction binding the contract method 0x153ca6c0.
//
// Solidity: function updatePrice(uint32 productId, int128 priceX18) returns()
func (_Spot *SpotSession) UpdatePrice(productId uint32, priceX18 *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdatePrice(&_Spot.TransactOpts, productId, priceX18)
}

// UpdatePrice is a paid mutator transaction binding the contract method 0x153ca6c0.
//
// Solidity: function updatePrice(uint32 productId, int128 priceX18) returns()
func (_Spot *SpotTransactorSession) UpdatePrice(productId uint32, priceX18 *big.Int) (*types.Transaction, error) {
	return _Spot.Contract.UpdatePrice(&_Spot.TransactOpts, productId, priceX18)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0xf5947556.
//
// Solidity: function updateProduct((uint32,int128,int128,address,int128,uint32,int128) txn) returns()
func (_Spot *SpotTransactor) UpdateProduct(opts *bind.TransactOpts, txn ISpotEngineUpdateProductTx) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateProduct", txn)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0xf5947556.
//
// Solidity: function updateProduct((uint32,int128,int128,address,int128,uint32,int128) txn) returns()
func (_Spot *SpotSession) UpdateProduct(txn ISpotEngineUpdateProductTx) (*types.Transaction, error) {
	return _Spot.Contract.UpdateProduct(&_Spot.TransactOpts, txn)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0xf5947556.
//
// Solidity: function updateProduct((uint32,int128,int128,address,int128,uint32,int128) txn) returns()
func (_Spot *SpotTransactorSession) UpdateProduct(txn ISpotEngineUpdateProductTx) (*types.Transaction, error) {
	return _Spot.Contract.UpdateProduct(&_Spot.TransactOpts, txn)
}

// UpdateRisk is a paid mutator transaction binding the contract method 0xc55607b5.
//
// Solidity: function updateRisk(uint32 productId, (int32,int32,int32,int32,int128) riskStore) returns()
func (_Spot *SpotTransactor) UpdateRisk(opts *bind.TransactOpts, productId uint32, riskStore RiskHelperRiskStore) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateRisk", productId, riskStore)
}

// UpdateRisk is a paid mutator transaction binding the contract method 0xc55607b5.
//
// Solidity: function updateRisk(uint32 productId, (int32,int32,int32,int32,int128) riskStore) returns()
func (_Spot *SpotSession) UpdateRisk(productId uint32, riskStore RiskHelperRiskStore) (*types.Transaction, error) {
	return _Spot.Contract.UpdateRisk(&_Spot.TransactOpts, productId, riskStore)
}

// UpdateRisk is a paid mutator transaction binding the contract method 0xc55607b5.
//
// Solidity: function updateRisk(uint32 productId, (int32,int32,int32,int32,int128) riskStore) returns()
func (_Spot *SpotTransactorSession) UpdateRisk(productId uint32, riskStore RiskHelperRiskStore) (*types.Transaction, error) {
	return _Spot.Contract.UpdateRisk(&_Spot.TransactOpts, productId, riskStore)
}

// UpdateTokenProductIdMapping is a paid mutator transaction binding the contract method 0x736e4c3d.
//
// Solidity: function updateTokenProductIdMapping(address token, uint32 productId) returns()
func (_Spot *SpotTransactor) UpdateTokenProductIdMapping(opts *bind.TransactOpts, token common.Address, productId uint32) (*types.Transaction, error) {
	return _Spot.contract.Transact(opts, "updateTokenProductIdMapping", token, productId)
}

// UpdateTokenProductIdMapping is a paid mutator transaction binding the contract method 0x736e4c3d.
//
// Solidity: function updateTokenProductIdMapping(address token, uint32 productId) returns()
func (_Spot *SpotSession) UpdateTokenProductIdMapping(token common.Address, productId uint32) (*types.Transaction, error) {
	return _Spot.Contract.UpdateTokenProductIdMapping(&_Spot.TransactOpts, token, productId)
}

// UpdateTokenProductIdMapping is a paid mutator transaction binding the contract method 0x736e4c3d.
//
// Solidity: function updateTokenProductIdMapping(address token, uint32 productId) returns()
func (_Spot *SpotTransactorSession) UpdateTokenProductIdMapping(token common.Address, productId uint32) (*types.Transaction, error) {
	return _Spot.Contract.UpdateTokenProductIdMapping(&_Spot.TransactOpts, token, productId)
}

// SpotAddProductIterator is returned from FilterAddProduct and is used to iterate over the raw logs and unpacked data for AddProduct events raised by the Spot contract.
type SpotAddProductIterator struct {
	Event *SpotAddProduct // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotAddProductIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotAddProduct)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotAddProduct)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotAddProductIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotAddProductIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotAddProduct represents a AddProduct event raised by the Spot contract.
type SpotAddProduct struct {
	ProductId uint32
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAddProduct is a free log retrieval operation binding the contract event 0x3286b0394bf1350245290b7226c92ed186bd716f28938e62dbb895298f018172.
//
// Solidity: event AddProduct(uint32 productId)
func (_Spot *SpotFilterer) FilterAddProduct(opts *bind.FilterOpts) (*SpotAddProductIterator, error) {

	logs, sub, err := _Spot.contract.FilterLogs(opts, "AddProduct")
	if err != nil {
		return nil, err
	}
	return &SpotAddProductIterator{contract: _Spot.contract, event: "AddProduct", logs: logs, sub: sub}, nil
}

// WatchAddProduct is a free log subscription operation binding the contract event 0x3286b0394bf1350245290b7226c92ed186bd716f28938e62dbb895298f018172.
//
// Solidity: event AddProduct(uint32 productId)
func (_Spot *SpotFilterer) WatchAddProduct(opts *bind.WatchOpts, sink chan<- *SpotAddProduct) (event.Subscription, error) {

	logs, sub, err := _Spot.contract.WatchLogs(opts, "AddProduct")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotAddProduct)
				if err := _Spot.contract.UnpackLog(event, "AddProduct", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAddProduct is a log parse operation binding the contract event 0x3286b0394bf1350245290b7226c92ed186bd716f28938e62dbb895298f018172.
//
// Solidity: event AddProduct(uint32 productId)
func (_Spot *SpotFilterer) ParseAddProduct(log types.Log) (*SpotAddProduct, error) {
	event := new(SpotAddProduct)
	if err := _Spot.contract.UnpackLog(event, "AddProduct", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SpotBalanceUpdateIterator is returned from FilterBalanceUpdate and is used to iterate over the raw logs and unpacked data for BalanceUpdate events raised by the Spot contract.
type SpotBalanceUpdateIterator struct {
	Event *SpotBalanceUpdate // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotBalanceUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotBalanceUpdate)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotBalanceUpdate)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotBalanceUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotBalanceUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotBalanceUpdate represents a BalanceUpdate event raised by the Spot contract.
type SpotBalanceUpdate struct {
	ProductId  uint32
	Subaccount [32]byte
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterBalanceUpdate is a free log retrieval operation binding the contract event 0x6f7b1abe76aa89745b8bf26b9cd9a8c5b1951ab2b57969bd7a091cde2225c940.
//
// Solidity: event BalanceUpdate(uint32 productId, bytes32 subaccount)
func (_Spot *SpotFilterer) FilterBalanceUpdate(opts *bind.FilterOpts) (*SpotBalanceUpdateIterator, error) {

	logs, sub, err := _Spot.contract.FilterLogs(opts, "BalanceUpdate")
	if err != nil {
		return nil, err
	}
	return &SpotBalanceUpdateIterator{contract: _Spot.contract, event: "BalanceUpdate", logs: logs, sub: sub}, nil
}

// WatchBalanceUpdate is a free log subscription operation binding the contract event 0x6f7b1abe76aa89745b8bf26b9cd9a8c5b1951ab2b57969bd7a091cde2225c940.
//
// Solidity: event BalanceUpdate(uint32 productId, bytes32 subaccount)
func (_Spot *SpotFilterer) WatchBalanceUpdate(opts *bind.WatchOpts, sink chan<- *SpotBalanceUpdate) (event.Subscription, error) {

	logs, sub, err := _Spot.contract.WatchLogs(opts, "BalanceUpdate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotBalanceUpdate)
				if err := _Spot.contract.UnpackLog(event, "BalanceUpdate", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseBalanceUpdate is a log parse operation binding the contract event 0x6f7b1abe76aa89745b8bf26b9cd9a8c5b1951ab2b57969bd7a091cde2225c940.
//
// Solidity: event BalanceUpdate(uint32 productId, bytes32 subaccount)
func (_Spot *SpotFilterer) ParseBalanceUpdate(log types.Log) (*SpotBalanceUpdate, error) {
	event := new(SpotBalanceUpdate)
	if err := _Spot.contract.UnpackLog(event, "BalanceUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SpotInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Spot contract.
type SpotInitializedIterator struct {
	Event *SpotInitialized // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotInitialized)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotInitialized)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotInitialized represents a Initialized event raised by the Spot contract.
type SpotInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Spot *SpotFilterer) FilterInitialized(opts *bind.FilterOpts) (*SpotInitializedIterator, error) {

	logs, sub, err := _Spot.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &SpotInitializedIterator{contract: _Spot.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Spot *SpotFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *SpotInitialized) (event.Subscription, error) {

	logs, sub, err := _Spot.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotInitialized)
				if err := _Spot.contract.UnpackLog(event, "Initialized", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseInitialized is a log parse operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Spot *SpotFilterer) ParseInitialized(log types.Log) (*SpotInitialized, error) {
	event := new(SpotInitialized)
	if err := _Spot.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SpotOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Spot contract.
type SpotOwnershipTransferredIterator struct {
	Event *SpotOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotOwnershipTransferred represents a OwnershipTransferred event raised by the Spot contract.
type SpotOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Spot *SpotFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*SpotOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Spot.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &SpotOwnershipTransferredIterator{contract: _Spot.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Spot *SpotFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *SpotOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Spot.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotOwnershipTransferred)
				if err := _Spot.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Spot *SpotFilterer) ParseOwnershipTransferred(log types.Log) (*SpotOwnershipTransferred, error) {
	event := new(SpotOwnershipTransferred)
	if err := _Spot.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SpotProductUpdateIterator is returned from FilterProductUpdate and is used to iterate over the raw logs and unpacked data for ProductUpdate events raised by the Spot contract.
type SpotProductUpdateIterator struct {
	Event *SpotProductUpdate // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotProductUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotProductUpdate)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotProductUpdate)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotProductUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotProductUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotProductUpdate represents a ProductUpdate event raised by the Spot contract.
type SpotProductUpdate struct {
	ProductId uint32
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterProductUpdate is a free log retrieval operation binding the contract event 0xe6195122b31334b8a2bd5ec64f0dd6ac3ab865ac54c2a0413fb82dfb22ad6432.
//
// Solidity: event ProductUpdate(uint32 productId)
func (_Spot *SpotFilterer) FilterProductUpdate(opts *bind.FilterOpts) (*SpotProductUpdateIterator, error) {

	logs, sub, err := _Spot.contract.FilterLogs(opts, "ProductUpdate")
	if err != nil {
		return nil, err
	}
	return &SpotProductUpdateIterator{contract: _Spot.contract, event: "ProductUpdate", logs: logs, sub: sub}, nil
}

// WatchProductUpdate is a free log subscription operation binding the contract event 0xe6195122b31334b8a2bd5ec64f0dd6ac3ab865ac54c2a0413fb82dfb22ad6432.
//
// Solidity: event ProductUpdate(uint32 productId)
func (_Spot *SpotFilterer) WatchProductUpdate(opts *bind.WatchOpts, sink chan<- *SpotProductUpdate) (event.Subscription, error) {

	logs, sub, err := _Spot.contract.WatchLogs(opts, "ProductUpdate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotProductUpdate)
				if err := _Spot.contract.UnpackLog(event, "ProductUpdate", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseProductUpdate is a log parse operation binding the contract event 0xe6195122b31334b8a2bd5ec64f0dd6ac3ab865ac54c2a0413fb82dfb22ad6432.
//
// Solidity: event ProductUpdate(uint32 productId)
func (_Spot *SpotFilterer) ParseProductUpdate(log types.Log) (*SpotProductUpdate, error) {
	event := new(SpotProductUpdate)
	if err := _Spot.contract.UnpackLog(event, "ProductUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// SpotQuoteProductUpdateIterator is returned from FilterQuoteProductUpdate and is used to iterate over the raw logs and unpacked data for QuoteProductUpdate events raised by the Spot contract.
type SpotQuoteProductUpdateIterator struct {
	Event *SpotQuoteProductUpdate // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *SpotQuoteProductUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(SpotQuoteProductUpdate)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(SpotQuoteProductUpdate)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *SpotQuoteProductUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *SpotQuoteProductUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// SpotQuoteProductUpdate represents a QuoteProductUpdate event raised by the Spot contract.
type SpotQuoteProductUpdate struct {
	IsoGroup uint32
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterQuoteProductUpdate is a free log retrieval operation binding the contract event 0x3abf10de00d6fa2e77927fcd6332af4365205ad32a82245390c817a179acae23.
//
// Solidity: event QuoteProductUpdate(uint32 isoGroup)
func (_Spot *SpotFilterer) FilterQuoteProductUpdate(opts *bind.FilterOpts) (*SpotQuoteProductUpdateIterator, error) {

	logs, sub, err := _Spot.contract.FilterLogs(opts, "QuoteProductUpdate")
	if err != nil {
		return nil, err
	}
	return &SpotQuoteProductUpdateIterator{contract: _Spot.contract, event: "QuoteProductUpdate", logs: logs, sub: sub}, nil
}

// WatchQuoteProductUpdate is a free log subscription operation binding the contract event 0x3abf10de00d6fa2e77927fcd6332af4365205ad32a82245390c817a179acae23.
//
// Solidity: event QuoteProductUpdate(uint32 isoGroup)
func (_Spot *SpotFilterer) WatchQuoteProductUpdate(opts *bind.WatchOpts, sink chan<- *SpotQuoteProductUpdate) (event.Subscription, error) {

	logs, sub, err := _Spot.contract.WatchLogs(opts, "QuoteProductUpdate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(SpotQuoteProductUpdate)
				if err := _Spot.contract.UnpackLog(event, "QuoteProductUpdate", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseQuoteProductUpdate is a log parse operation binding the contract event 0x3abf10de00d6fa2e77927fcd6332af4365205ad32a82245390c817a179acae23.
//
// Solidity: event QuoteProductUpdate(uint32 isoGroup)
func (_Spot *SpotFilterer) ParseQuoteProductUpdate(log types.Log) (*SpotQuoteProductUpdate, error) {
	event := new(SpotQuoteProductUpdate)
	if err := _Spot.contract.UnpackLog(event, "QuoteProductUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
