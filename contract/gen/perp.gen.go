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

// IPerpEngineBalance is an auto generated low-level Go binding around an user-defined struct.
type IPerpEngineBalance struct {
	Amount                   *big.Int
	VQuoteBalance            *big.Int
	LastCumulativeFundingX18 *big.Int
}

// IPerpEngineState is an auto generated low-level Go binding around an user-defined struct.
type IPerpEngineState struct {
	CumulativeFundingLongX18  *big.Int
	CumulativeFundingShortX18 *big.Int
	LongOpenInterest          *big.Int
	ShortOpenInterest         *big.Int
}

// IPerpEngineUpdateProductTx is an auto generated low-level Go binding around an user-defined struct.
type IPerpEngineUpdateProductTx struct {
	ProductId            uint32
	SizeIncrement        *big.Int
	MinSize              *big.Int
	MaintainanceFraction uint32
}

// PerpMetaData contains all meta data concerning the Perp contract.
var PerpMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"addProduct\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"book\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"sizeIncrement\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"minSize\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"maintainanceFraction\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"configureAssertInternal\",\"inputs\":[{\"name\":\"_addr\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"defaultFeeRates\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"makerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"takerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"isNonDefault\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAllBalancesOfSubaccounts\",\"inputs\":[{\"name\":\"subaccounts\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple[][]\",\"internalType\":\"structIPerpEngine.Balance[][]\",\"components\":[{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"lastCumulativeFundingX18\",\"type\":\"int128\",\"internalType\":\"int128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getBalance\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.Balance\",\"components\":[{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"lastCumulativeFundingX18\",\"type\":\"int128\",\"internalType\":\"int128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getClearinghouse\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCumulativeFundingRates\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"},{\"name\":\"\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getCustomFeeSubAccounts\",\"inputs\":[{\"name\":\"startAt\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"limit\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEndpoint\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEngineType\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"enumIProductEngine.EngineType\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getFeeFractionX18\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"taker\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getFeeRatesX18\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getHealthContribution\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"perpPricesX18\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}],\"outputs\":[{\"name\":\"health\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"maintainanceMargin\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getPositionPnl\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceX18\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getProductIds\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getStateAndBalance\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.State\",\"components\":[{\"name\":\"cumulativeFundingLongX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"cumulativeFundingShortX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"longOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"shortOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"}]},{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.Balance\",\"components\":[{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"lastCumulativeFundingX18\",\"type\":\"int128\",\"internalType\":\"int128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getStatesAndBalances\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.State\",\"components\":[{\"name\":\"cumulativeFundingLongX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"cumulativeFundingShortX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"longOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"shortOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"}]},{\"name\":\"\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.Balance\",\"components\":[{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"lastCumulativeFundingX18\",\"type\":\"int128\",\"internalType\":\"int128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getUpdatedVquote\",\"inputs\":[{\"name\":\"existingAmount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"amountDelta\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"existingQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"quoteBalanceDelta\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[{\"name\":\"realisedPnl\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteBalance\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_clearinghouse\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_offchainExchange\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_admin\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"maintainanceMarginFractions\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"manualAssert\",\"inputs\":[{\"name\":\"openInterests\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}],\"outputs\":[],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setDefaultFeeRates\",\"inputs\":[{\"name\":\"makerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"takerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setEndpoint\",\"inputs\":[{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"socializeSubaccount\",\"inputs\":[{\"name\":\"insurance\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"pnl\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"states\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"cumulativeFundingLongX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"cumulativeFundingShortX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"longOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"shortOpenInterest\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalFudingFee\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateBalance\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amountDelta\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"vQuoteDelta\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[{\"name\":\"fundingFees\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"realisedPnl\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateFeeRates\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"makerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"takerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateProduct\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIPerpEngine.UpdateProductTx\",\"components\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"sizeIncrement\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"minSize\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"maintainanceFraction\",\"type\":\"uint32\",\"internalType\":\"uint32\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateStates\",\"inputs\":[{\"name\":\"dt\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"fundingRateUpdates\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"AddProduct\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"FundingRateUpdated\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"newCumulativeFundingLongX18\",\"type\":\"int128\",\"indexed\":true,\"internalType\":\"int128\"},{\"name\":\"newCumulativeFundingShortX18\",\"type\":\"int128\",\"indexed\":true,\"internalType\":\"int128\"},{\"name\":\"delta\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ProductUpdate\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
}

// PerpABI is the input ABI used to generate the binding from.
// Deprecated: Use PerpMetaData.ABI instead.
var PerpABI = PerpMetaData.ABI

// Perp is an auto generated Go binding around an Ethereum contract.
type Perp struct {
	PerpCaller     // Read-only binding to the contract
	PerpTransactor // Write-only binding to the contract
	PerpFilterer   // Log filterer for contract events
}

// PerpCaller is an auto generated read-only Go binding around an Ethereum contract.
type PerpCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PerpTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PerpTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PerpFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PerpFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PerpSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PerpSession struct {
	Contract     *Perp             // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PerpCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PerpCallerSession struct {
	Contract *PerpCaller   // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// PerpTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PerpTransactorSession struct {
	Contract     *PerpTransactor   // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PerpRaw is an auto generated low-level Go binding around an Ethereum contract.
type PerpRaw struct {
	Contract *Perp // Generic contract binding to access the raw methods on
}

// PerpCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PerpCallerRaw struct {
	Contract *PerpCaller // Generic read-only contract binding to access the raw methods on
}

// PerpTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PerpTransactorRaw struct {
	Contract *PerpTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPerp creates a new instance of Perp, bound to a specific deployed contract.
func NewPerp(address common.Address, backend bind.ContractBackend) (*Perp, error) {
	contract, err := bindPerp(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Perp{PerpCaller: PerpCaller{contract: contract}, PerpTransactor: PerpTransactor{contract: contract}, PerpFilterer: PerpFilterer{contract: contract}}, nil
}

// NewPerpCaller creates a new read-only instance of Perp, bound to a specific deployed contract.
func NewPerpCaller(address common.Address, caller bind.ContractCaller) (*PerpCaller, error) {
	contract, err := bindPerp(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PerpCaller{contract: contract}, nil
}

// NewPerpTransactor creates a new write-only instance of Perp, bound to a specific deployed contract.
func NewPerpTransactor(address common.Address, transactor bind.ContractTransactor) (*PerpTransactor, error) {
	contract, err := bindPerp(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PerpTransactor{contract: contract}, nil
}

// NewPerpFilterer creates a new log filterer instance of Perp, bound to a specific deployed contract.
func NewPerpFilterer(address common.Address, filterer bind.ContractFilterer) (*PerpFilterer, error) {
	contract, err := bindPerp(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PerpFilterer{contract: contract}, nil
}

// bindPerp binds a generic wrapper to an already deployed contract.
func bindPerp(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PerpMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Perp *PerpRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Perp.Contract.PerpCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Perp *PerpRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Perp.Contract.PerpTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Perp *PerpRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Perp.Contract.PerpTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Perp *PerpCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Perp.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Perp *PerpTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Perp.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Perp *PerpTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Perp.Contract.contract.Transact(opts, method, params...)
}

// DefaultFeeRates is a free data retrieval call binding the contract method 0x443389bd.
//
// Solidity: function defaultFeeRates(uint32 ) view returns(int64 makerRateX18, int64 takerRateX18, uint8 isNonDefault)
func (_Perp *PerpCaller) DefaultFeeRates(opts *bind.CallOpts, arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "defaultFeeRates", arg0)

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
func (_Perp *PerpSession) DefaultFeeRates(arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	return _Perp.Contract.DefaultFeeRates(&_Perp.CallOpts, arg0)
}

// DefaultFeeRates is a free data retrieval call binding the contract method 0x443389bd.
//
// Solidity: function defaultFeeRates(uint32 ) view returns(int64 makerRateX18, int64 takerRateX18, uint8 isNonDefault)
func (_Perp *PerpCallerSession) DefaultFeeRates(arg0 uint32) (struct {
	MakerRateX18 int64
	TakerRateX18 int64
	IsNonDefault uint8
}, error) {
	return _Perp.Contract.DefaultFeeRates(&_Perp.CallOpts, arg0)
}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128,int128,int128)[][])
func (_Perp *PerpCaller) GetAllBalancesOfSubaccounts(opts *bind.CallOpts, subaccounts [][32]byte) ([][]IPerpEngineBalance, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getAllBalancesOfSubaccounts", subaccounts)

	if err != nil {
		return *new([][]IPerpEngineBalance), err
	}

	out0 := *abi.ConvertType(out[0], new([][]IPerpEngineBalance)).(*[][]IPerpEngineBalance)

	return out0, err

}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128,int128,int128)[][])
func (_Perp *PerpSession) GetAllBalancesOfSubaccounts(subaccounts [][32]byte) ([][]IPerpEngineBalance, error) {
	return _Perp.Contract.GetAllBalancesOfSubaccounts(&_Perp.CallOpts, subaccounts)
}

// GetAllBalancesOfSubaccounts is a free data retrieval call binding the contract method 0x1cc89f4c.
//
// Solidity: function getAllBalancesOfSubaccounts(bytes32[] subaccounts) view returns((int128,int128,int128)[][])
func (_Perp *PerpCallerSession) GetAllBalancesOfSubaccounts(subaccounts [][32]byte) ([][]IPerpEngineBalance, error) {
	return _Perp.Contract.GetAllBalancesOfSubaccounts(&_Perp.CallOpts, subaccounts)
}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128))
func (_Perp *PerpCaller) GetBalance(opts *bind.CallOpts, productId uint32, subaccount [32]byte) (IPerpEngineBalance, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getBalance", productId, subaccount)

	if err != nil {
		return *new(IPerpEngineBalance), err
	}

	out0 := *abi.ConvertType(out[0], new(IPerpEngineBalance)).(*IPerpEngineBalance)

	return out0, err

}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128))
func (_Perp *PerpSession) GetBalance(productId uint32, subaccount [32]byte) (IPerpEngineBalance, error) {
	return _Perp.Contract.GetBalance(&_Perp.CallOpts, productId, subaccount)
}

// GetBalance is a free data retrieval call binding the contract method 0x7c1e1487.
//
// Solidity: function getBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128))
func (_Perp *PerpCallerSession) GetBalance(productId uint32, subaccount [32]byte) (IPerpEngineBalance, error) {
	return _Perp.Contract.GetBalance(&_Perp.CallOpts, productId, subaccount)
}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Perp *PerpCaller) GetClearinghouse(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getClearinghouse")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Perp *PerpSession) GetClearinghouse() (common.Address, error) {
	return _Perp.Contract.GetClearinghouse(&_Perp.CallOpts)
}

// GetClearinghouse is a free data retrieval call binding the contract method 0xb1cb0f42.
//
// Solidity: function getClearinghouse() view returns(address)
func (_Perp *PerpCallerSession) GetClearinghouse() (common.Address, error) {
	return _Perp.Contract.GetClearinghouse(&_Perp.CallOpts)
}

// GetCumulativeFundingRates is a free data retrieval call binding the contract method 0x23efdc14.
//
// Solidity: function getCumulativeFundingRates() view returns(int128[], int128[])
func (_Perp *PerpCaller) GetCumulativeFundingRates(opts *bind.CallOpts) ([]*big.Int, []*big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getCumulativeFundingRates")

	if err != nil {
		return *new([]*big.Int), *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)
	out1 := *abi.ConvertType(out[1], new([]*big.Int)).(*[]*big.Int)

	return out0, out1, err

}

// GetCumulativeFundingRates is a free data retrieval call binding the contract method 0x23efdc14.
//
// Solidity: function getCumulativeFundingRates() view returns(int128[], int128[])
func (_Perp *PerpSession) GetCumulativeFundingRates() ([]*big.Int, []*big.Int, error) {
	return _Perp.Contract.GetCumulativeFundingRates(&_Perp.CallOpts)
}

// GetCumulativeFundingRates is a free data retrieval call binding the contract method 0x23efdc14.
//
// Solidity: function getCumulativeFundingRates() view returns(int128[], int128[])
func (_Perp *PerpCallerSession) GetCumulativeFundingRates() ([]*big.Int, []*big.Int, error) {
	return _Perp.Contract.GetCumulativeFundingRates(&_Perp.CallOpts)
}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Perp *PerpCaller) GetCustomFeeSubAccounts(opts *bind.CallOpts, startAt uint32, limit uint32) ([][32]byte, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getCustomFeeSubAccounts", startAt, limit)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Perp *PerpSession) GetCustomFeeSubAccounts(startAt uint32, limit uint32) ([][32]byte, error) {
	return _Perp.Contract.GetCustomFeeSubAccounts(&_Perp.CallOpts, startAt, limit)
}

// GetCustomFeeSubAccounts is a free data retrieval call binding the contract method 0xd964fefc.
//
// Solidity: function getCustomFeeSubAccounts(uint32 startAt, uint32 limit) view returns(bytes32[])
func (_Perp *PerpCallerSession) GetCustomFeeSubAccounts(startAt uint32, limit uint32) ([][32]byte, error) {
	return _Perp.Contract.GetCustomFeeSubAccounts(&_Perp.CallOpts, startAt, limit)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Perp *PerpCaller) GetEndpoint(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getEndpoint")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Perp *PerpSession) GetEndpoint() (common.Address, error) {
	return _Perp.Contract.GetEndpoint(&_Perp.CallOpts)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_Perp *PerpCallerSession) GetEndpoint() (common.Address, error) {
	return _Perp.Contract.GetEndpoint(&_Perp.CallOpts)
}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Perp *PerpCaller) GetEngineType(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getEngineType")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Perp *PerpSession) GetEngineType() (uint8, error) {
	return _Perp.Contract.GetEngineType(&_Perp.CallOpts)
}

// GetEngineType is a free data retrieval call binding the contract method 0x4604d19b.
//
// Solidity: function getEngineType() pure returns(uint8)
func (_Perp *PerpCallerSession) GetEngineType() (uint8, error) {
	return _Perp.Contract.GetEngineType(&_Perp.CallOpts)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Perp *PerpCaller) GetFeeFractionX18(opts *bind.CallOpts, subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getFeeFractionX18", subaccount, productId, taker)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Perp *PerpSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _Perp.Contract.GetFeeFractionX18(&_Perp.CallOpts, subaccount, productId, taker)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_Perp *PerpCallerSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _Perp.Contract.GetFeeFractionX18(&_Perp.CallOpts, subaccount, productId, taker)
}

// GetFeeRatesX18 is a free data retrieval call binding the contract method 0x0f2c878e.
//
// Solidity: function getFeeRatesX18(bytes32 subaccount, uint32 productId) view returns(int128, int128)
func (_Perp *PerpCaller) GetFeeRatesX18(opts *bind.CallOpts, subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getFeeRatesX18", subaccount, productId)

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
func (_Perp *PerpSession) GetFeeRatesX18(subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	return _Perp.Contract.GetFeeRatesX18(&_Perp.CallOpts, subaccount, productId)
}

// GetFeeRatesX18 is a free data retrieval call binding the contract method 0x0f2c878e.
//
// Solidity: function getFeeRatesX18(bytes32 subaccount, uint32 productId) view returns(int128, int128)
func (_Perp *PerpCallerSession) GetFeeRatesX18(subaccount [32]byte, productId uint32) (*big.Int, *big.Int, error) {
	return _Perp.Contract.GetFeeRatesX18(&_Perp.CallOpts, subaccount, productId)
}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] perpPricesX18) view returns(int128 health, int128 maintainanceMargin)
func (_Perp *PerpCaller) GetHealthContribution(opts *bind.CallOpts, subaccount [32]byte, perpPricesX18 []*big.Int) (struct {
	Health             *big.Int
	MaintainanceMargin *big.Int
}, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getHealthContribution", subaccount, perpPricesX18)

	outstruct := new(struct {
		Health             *big.Int
		MaintainanceMargin *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Health = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.MaintainanceMargin = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] perpPricesX18) view returns(int128 health, int128 maintainanceMargin)
func (_Perp *PerpSession) GetHealthContribution(subaccount [32]byte, perpPricesX18 []*big.Int) (struct {
	Health             *big.Int
	MaintainanceMargin *big.Int
}, error) {
	return _Perp.Contract.GetHealthContribution(&_Perp.CallOpts, subaccount, perpPricesX18)
}

// GetHealthContribution is a free data retrieval call binding the contract method 0xb162a4ac.
//
// Solidity: function getHealthContribution(bytes32 subaccount, int128[] perpPricesX18) view returns(int128 health, int128 maintainanceMargin)
func (_Perp *PerpCallerSession) GetHealthContribution(subaccount [32]byte, perpPricesX18 []*big.Int) (struct {
	Health             *big.Int
	MaintainanceMargin *big.Int
}, error) {
	return _Perp.Contract.GetHealthContribution(&_Perp.CallOpts, subaccount, perpPricesX18)
}

// GetPositionPnl is a free data retrieval call binding the contract method 0x42d14fd6.
//
// Solidity: function getPositionPnl(uint32 productId, bytes32 subaccount, int128 priceX18) view returns(int128, int128)
func (_Perp *PerpCaller) GetPositionPnl(opts *bind.CallOpts, productId uint32, subaccount [32]byte, priceX18 *big.Int) (*big.Int, *big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getPositionPnl", productId, subaccount, priceX18)

	if err != nil {
		return *new(*big.Int), *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	out1 := *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return out0, out1, err

}

// GetPositionPnl is a free data retrieval call binding the contract method 0x42d14fd6.
//
// Solidity: function getPositionPnl(uint32 productId, bytes32 subaccount, int128 priceX18) view returns(int128, int128)
func (_Perp *PerpSession) GetPositionPnl(productId uint32, subaccount [32]byte, priceX18 *big.Int) (*big.Int, *big.Int, error) {
	return _Perp.Contract.GetPositionPnl(&_Perp.CallOpts, productId, subaccount, priceX18)
}

// GetPositionPnl is a free data retrieval call binding the contract method 0x42d14fd6.
//
// Solidity: function getPositionPnl(uint32 productId, bytes32 subaccount, int128 priceX18) view returns(int128, int128)
func (_Perp *PerpCallerSession) GetPositionPnl(productId uint32, subaccount [32]byte, priceX18 *big.Int) (*big.Int, *big.Int, error) {
	return _Perp.Contract.GetPositionPnl(&_Perp.CallOpts, productId, subaccount, priceX18)
}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Perp *PerpCaller) GetProductIds(opts *bind.CallOpts) ([]uint32, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getProductIds")

	if err != nil {
		return *new([]uint32), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint32)).(*[]uint32)

	return out0, err

}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Perp *PerpSession) GetProductIds() ([]uint32, error) {
	return _Perp.Contract.GetProductIds(&_Perp.CallOpts)
}

// GetProductIds is a free data retrieval call binding the contract method 0x47428e7b.
//
// Solidity: function getProductIds() view returns(uint32[])
func (_Perp *PerpCallerSession) GetProductIds() ([]uint32, error) {
	return _Perp.Contract.GetProductIds(&_Perp.CallOpts)
}

// GetStateAndBalance is a free data retrieval call binding the contract method 0xe334be33.
//
// Solidity: function getStateAndBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpCaller) GetStateAndBalance(opts *bind.CallOpts, productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getStateAndBalance", productId, subaccount)

	if err != nil {
		return *new(IPerpEngineState), *new(IPerpEngineBalance), err
	}

	out0 := *abi.ConvertType(out[0], new(IPerpEngineState)).(*IPerpEngineState)
	out1 := *abi.ConvertType(out[1], new(IPerpEngineBalance)).(*IPerpEngineBalance)

	return out0, out1, err

}

// GetStateAndBalance is a free data retrieval call binding the contract method 0xe334be33.
//
// Solidity: function getStateAndBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpSession) GetStateAndBalance(productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	return _Perp.Contract.GetStateAndBalance(&_Perp.CallOpts, productId, subaccount)
}

// GetStateAndBalance is a free data retrieval call binding the contract method 0xe334be33.
//
// Solidity: function getStateAndBalance(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpCallerSession) GetStateAndBalance(productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	return _Perp.Contract.GetStateAndBalance(&_Perp.CallOpts, productId, subaccount)
}

// GetStatesAndBalances is a free data retrieval call binding the contract method 0x3d5cc9dc.
//
// Solidity: function getStatesAndBalances(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpCaller) GetStatesAndBalances(opts *bind.CallOpts, productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getStatesAndBalances", productId, subaccount)

	if err != nil {
		return *new(IPerpEngineState), *new(IPerpEngineBalance), err
	}

	out0 := *abi.ConvertType(out[0], new(IPerpEngineState)).(*IPerpEngineState)
	out1 := *abi.ConvertType(out[1], new(IPerpEngineBalance)).(*IPerpEngineBalance)

	return out0, out1, err

}

// GetStatesAndBalances is a free data retrieval call binding the contract method 0x3d5cc9dc.
//
// Solidity: function getStatesAndBalances(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpSession) GetStatesAndBalances(productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	return _Perp.Contract.GetStatesAndBalances(&_Perp.CallOpts, productId, subaccount)
}

// GetStatesAndBalances is a free data retrieval call binding the contract method 0x3d5cc9dc.
//
// Solidity: function getStatesAndBalances(uint32 productId, bytes32 subaccount) view returns((int128,int128,int128,int128), (int128,int128,int128))
func (_Perp *PerpCallerSession) GetStatesAndBalances(productId uint32, subaccount [32]byte) (IPerpEngineState, IPerpEngineBalance, error) {
	return _Perp.Contract.GetStatesAndBalances(&_Perp.CallOpts, productId, subaccount)
}

// GetUpdatedVquote is a free data retrieval call binding the contract method 0xd23bcee5.
//
// Solidity: function getUpdatedVquote(int128 existingAmount, int128 amountDelta, int128 existingQuoteBalance, int128 quoteBalanceDelta) pure returns(int128 realisedPnl, int128 vQuoteBalance)
func (_Perp *PerpCaller) GetUpdatedVquote(opts *bind.CallOpts, existingAmount *big.Int, amountDelta *big.Int, existingQuoteBalance *big.Int, quoteBalanceDelta *big.Int) (struct {
	RealisedPnl   *big.Int
	VQuoteBalance *big.Int
}, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getUpdatedVquote", existingAmount, amountDelta, existingQuoteBalance, quoteBalanceDelta)

	outstruct := new(struct {
		RealisedPnl   *big.Int
		VQuoteBalance *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.RealisedPnl = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.VQuoteBalance = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// GetUpdatedVquote is a free data retrieval call binding the contract method 0xd23bcee5.
//
// Solidity: function getUpdatedVquote(int128 existingAmount, int128 amountDelta, int128 existingQuoteBalance, int128 quoteBalanceDelta) pure returns(int128 realisedPnl, int128 vQuoteBalance)
func (_Perp *PerpSession) GetUpdatedVquote(existingAmount *big.Int, amountDelta *big.Int, existingQuoteBalance *big.Int, quoteBalanceDelta *big.Int) (struct {
	RealisedPnl   *big.Int
	VQuoteBalance *big.Int
}, error) {
	return _Perp.Contract.GetUpdatedVquote(&_Perp.CallOpts, existingAmount, amountDelta, existingQuoteBalance, quoteBalanceDelta)
}

// GetUpdatedVquote is a free data retrieval call binding the contract method 0xd23bcee5.
//
// Solidity: function getUpdatedVquote(int128 existingAmount, int128 amountDelta, int128 existingQuoteBalance, int128 quoteBalanceDelta) pure returns(int128 realisedPnl, int128 vQuoteBalance)
func (_Perp *PerpCallerSession) GetUpdatedVquote(existingAmount *big.Int, amountDelta *big.Int, existingQuoteBalance *big.Int, quoteBalanceDelta *big.Int) (struct {
	RealisedPnl   *big.Int
	VQuoteBalance *big.Int
}, error) {
	return _Perp.Contract.GetUpdatedVquote(&_Perp.CallOpts, existingAmount, amountDelta, existingQuoteBalance, quoteBalanceDelta)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Perp *PerpCaller) GetVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Perp *PerpSession) GetVersion() (uint64, error) {
	return _Perp.Contract.GetVersion(&_Perp.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Perp *PerpCallerSession) GetVersion() (uint64, error) {
	return _Perp.Contract.GetVersion(&_Perp.CallOpts)
}

// MaintainanceMarginFractions is a free data retrieval call binding the contract method 0x3e2a3a24.
//
// Solidity: function maintainanceMarginFractions(uint32 ) view returns(uint32)
func (_Perp *PerpCaller) MaintainanceMarginFractions(opts *bind.CallOpts, arg0 uint32) (uint32, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "maintainanceMarginFractions", arg0)

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// MaintainanceMarginFractions is a free data retrieval call binding the contract method 0x3e2a3a24.
//
// Solidity: function maintainanceMarginFractions(uint32 ) view returns(uint32)
func (_Perp *PerpSession) MaintainanceMarginFractions(arg0 uint32) (uint32, error) {
	return _Perp.Contract.MaintainanceMarginFractions(&_Perp.CallOpts, arg0)
}

// MaintainanceMarginFractions is a free data retrieval call binding the contract method 0x3e2a3a24.
//
// Solidity: function maintainanceMarginFractions(uint32 ) view returns(uint32)
func (_Perp *PerpCallerSession) MaintainanceMarginFractions(arg0 uint32) (uint32, error) {
	return _Perp.Contract.MaintainanceMarginFractions(&_Perp.CallOpts, arg0)
}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] openInterests) view returns()
func (_Perp *PerpCaller) ManualAssert(opts *bind.CallOpts, openInterests []*big.Int) error {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "manualAssert", openInterests)

	if err != nil {
		return err
	}

	return err

}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] openInterests) view returns()
func (_Perp *PerpSession) ManualAssert(openInterests []*big.Int) error {
	return _Perp.Contract.ManualAssert(&_Perp.CallOpts, openInterests)
}

// ManualAssert is a free data retrieval call binding the contract method 0x9b6f762b.
//
// Solidity: function manualAssert(int128[] openInterests) view returns()
func (_Perp *PerpCallerSession) ManualAssert(openInterests []*big.Int) error {
	return _Perp.Contract.ManualAssert(&_Perp.CallOpts, openInterests)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Perp *PerpCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Perp *PerpSession) Owner() (common.Address, error) {
	return _Perp.Contract.Owner(&_Perp.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Perp *PerpCallerSession) Owner() (common.Address, error) {
	return _Perp.Contract.Owner(&_Perp.CallOpts)
}

// SocializeSubaccount is a free data retrieval call binding the contract method 0x76c8d413.
//
// Solidity: function socializeSubaccount(int128 insurance, int128 pnl) view returns(int128)
func (_Perp *PerpCaller) SocializeSubaccount(opts *bind.CallOpts, insurance *big.Int, pnl *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "socializeSubaccount", insurance, pnl)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SocializeSubaccount is a free data retrieval call binding the contract method 0x76c8d413.
//
// Solidity: function socializeSubaccount(int128 insurance, int128 pnl) view returns(int128)
func (_Perp *PerpSession) SocializeSubaccount(insurance *big.Int, pnl *big.Int) (*big.Int, error) {
	return _Perp.Contract.SocializeSubaccount(&_Perp.CallOpts, insurance, pnl)
}

// SocializeSubaccount is a free data retrieval call binding the contract method 0x76c8d413.
//
// Solidity: function socializeSubaccount(int128 insurance, int128 pnl) view returns(int128)
func (_Perp *PerpCallerSession) SocializeSubaccount(insurance *big.Int, pnl *big.Int) (*big.Int, error) {
	return _Perp.Contract.SocializeSubaccount(&_Perp.CallOpts, insurance, pnl)
}

// States is a free data retrieval call binding the contract method 0x7f17baad.
//
// Solidity: function states(uint32 ) view returns(int128 cumulativeFundingLongX18, int128 cumulativeFundingShortX18, int128 longOpenInterest, int128 shortOpenInterest)
func (_Perp *PerpCaller) States(opts *bind.CallOpts, arg0 uint32) (struct {
	CumulativeFundingLongX18  *big.Int
	CumulativeFundingShortX18 *big.Int
	LongOpenInterest          *big.Int
	ShortOpenInterest         *big.Int
}, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "states", arg0)

	outstruct := new(struct {
		CumulativeFundingLongX18  *big.Int
		CumulativeFundingShortX18 *big.Int
		LongOpenInterest          *big.Int
		ShortOpenInterest         *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.CumulativeFundingLongX18 = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.CumulativeFundingShortX18 = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.LongOpenInterest = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.ShortOpenInterest = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// States is a free data retrieval call binding the contract method 0x7f17baad.
//
// Solidity: function states(uint32 ) view returns(int128 cumulativeFundingLongX18, int128 cumulativeFundingShortX18, int128 longOpenInterest, int128 shortOpenInterest)
func (_Perp *PerpSession) States(arg0 uint32) (struct {
	CumulativeFundingLongX18  *big.Int
	CumulativeFundingShortX18 *big.Int
	LongOpenInterest          *big.Int
	ShortOpenInterest         *big.Int
}, error) {
	return _Perp.Contract.States(&_Perp.CallOpts, arg0)
}

// States is a free data retrieval call binding the contract method 0x7f17baad.
//
// Solidity: function states(uint32 ) view returns(int128 cumulativeFundingLongX18, int128 cumulativeFundingShortX18, int128 longOpenInterest, int128 shortOpenInterest)
func (_Perp *PerpCallerSession) States(arg0 uint32) (struct {
	CumulativeFundingLongX18  *big.Int
	CumulativeFundingShortX18 *big.Int
	LongOpenInterest          *big.Int
	ShortOpenInterest         *big.Int
}, error) {
	return _Perp.Contract.States(&_Perp.CallOpts, arg0)
}

// TotalFudingFee is a free data retrieval call binding the contract method 0xdbed873c.
//
// Solidity: function totalFudingFee() view returns(int128)
func (_Perp *PerpCaller) TotalFudingFee(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Perp.contract.Call(opts, &out, "totalFudingFee")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalFudingFee is a free data retrieval call binding the contract method 0xdbed873c.
//
// Solidity: function totalFudingFee() view returns(int128)
func (_Perp *PerpSession) TotalFudingFee() (*big.Int, error) {
	return _Perp.Contract.TotalFudingFee(&_Perp.CallOpts)
}

// TotalFudingFee is a free data retrieval call binding the contract method 0xdbed873c.
//
// Solidity: function totalFudingFee() view returns(int128)
func (_Perp *PerpCallerSession) TotalFudingFee() (*big.Int, error) {
	return _Perp.Contract.TotalFudingFee(&_Perp.CallOpts)
}

// AddProduct is a paid mutator transaction binding the contract method 0xcdbca189.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, uint32 maintainanceFraction) returns()
func (_Perp *PerpTransactor) AddProduct(opts *bind.TransactOpts, productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, maintainanceFraction uint32) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "addProduct", productId, book, sizeIncrement, minSize, maintainanceFraction)
}

// AddProduct is a paid mutator transaction binding the contract method 0xcdbca189.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, uint32 maintainanceFraction) returns()
func (_Perp *PerpSession) AddProduct(productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, maintainanceFraction uint32) (*types.Transaction, error) {
	return _Perp.Contract.AddProduct(&_Perp.TransactOpts, productId, book, sizeIncrement, minSize, maintainanceFraction)
}

// AddProduct is a paid mutator transaction binding the contract method 0xcdbca189.
//
// Solidity: function addProduct(uint32 productId, address book, int128 sizeIncrement, int128 minSize, uint32 maintainanceFraction) returns()
func (_Perp *PerpTransactorSession) AddProduct(productId uint32, book common.Address, sizeIncrement *big.Int, minSize *big.Int, maintainanceFraction uint32) (*types.Transaction, error) {
	return _Perp.Contract.AddProduct(&_Perp.TransactOpts, productId, book, sizeIncrement, minSize, maintainanceFraction)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Perp *PerpTransactor) ConfigureAssertInternal(opts *bind.TransactOpts, _addr common.Address, value bool) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "configureAssertInternal", _addr, value)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Perp *PerpSession) ConfigureAssertInternal(_addr common.Address, value bool) (*types.Transaction, error) {
	return _Perp.Contract.ConfigureAssertInternal(&_Perp.TransactOpts, _addr, value)
}

// ConfigureAssertInternal is a paid mutator transaction binding the contract method 0x7ac35335.
//
// Solidity: function configureAssertInternal(address _addr, bool value) returns()
func (_Perp *PerpTransactorSession) ConfigureAssertInternal(_addr common.Address, value bool) (*types.Transaction, error) {
	return _Perp.Contract.ConfigureAssertInternal(&_Perp.TransactOpts, _addr, value)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address , address _endpoint, address _admin) returns()
func (_Perp *PerpTransactor) Initialize(opts *bind.TransactOpts, _clearinghouse common.Address, _offchainExchange common.Address, arg2 common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "initialize", _clearinghouse, _offchainExchange, arg2, _endpoint, _admin)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address , address _endpoint, address _admin) returns()
func (_Perp *PerpSession) Initialize(_clearinghouse common.Address, _offchainExchange common.Address, arg2 common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Perp.Contract.Initialize(&_Perp.TransactOpts, _clearinghouse, _offchainExchange, arg2, _endpoint, _admin)
}

// Initialize is a paid mutator transaction binding the contract method 0x1459457a.
//
// Solidity: function initialize(address _clearinghouse, address _offchainExchange, address , address _endpoint, address _admin) returns()
func (_Perp *PerpTransactorSession) Initialize(_clearinghouse common.Address, _offchainExchange common.Address, arg2 common.Address, _endpoint common.Address, _admin common.Address) (*types.Transaction, error) {
	return _Perp.Contract.Initialize(&_Perp.TransactOpts, _clearinghouse, _offchainExchange, arg2, _endpoint, _admin)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Perp *PerpTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Perp *PerpSession) RenounceOwnership() (*types.Transaction, error) {
	return _Perp.Contract.RenounceOwnership(&_Perp.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Perp *PerpTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Perp.Contract.RenounceOwnership(&_Perp.TransactOpts)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Perp *PerpTransactor) SetDefaultFeeRates(opts *bind.TransactOpts, makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "setDefaultFeeRates", makerRateX18, takerRateX18, productId)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Perp *PerpSession) SetDefaultFeeRates(makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Perp.Contract.SetDefaultFeeRates(&_Perp.TransactOpts, makerRateX18, takerRateX18, productId)
}

// SetDefaultFeeRates is a paid mutator transaction binding the contract method 0xba6148de.
//
// Solidity: function setDefaultFeeRates(int64 makerRateX18, int64 takerRateX18, uint32 productId) returns()
func (_Perp *PerpTransactorSession) SetDefaultFeeRates(makerRateX18 int64, takerRateX18 int64, productId uint32) (*types.Transaction, error) {
	return _Perp.Contract.SetDefaultFeeRates(&_Perp.TransactOpts, makerRateX18, takerRateX18, productId)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Perp *PerpTransactor) SetEndpoint(opts *bind.TransactOpts, _endpoint common.Address) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "setEndpoint", _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Perp *PerpSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _Perp.Contract.SetEndpoint(&_Perp.TransactOpts, _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_Perp *PerpTransactorSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _Perp.Contract.SetEndpoint(&_Perp.TransactOpts, _endpoint)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Perp *PerpTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Perp *PerpSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Perp.Contract.TransferOwnership(&_Perp.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Perp *PerpTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Perp.Contract.TransferOwnership(&_Perp.TransactOpts, newOwner)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 vQuoteDelta) returns(int128 fundingFees, int128 realisedPnl)
func (_Perp *PerpTransactor) UpdateBalance(opts *bind.TransactOpts, productId uint32, subaccount [32]byte, amountDelta *big.Int, vQuoteDelta *big.Int) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "updateBalance", productId, subaccount, amountDelta, vQuoteDelta)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 vQuoteDelta) returns(int128 fundingFees, int128 realisedPnl)
func (_Perp *PerpSession) UpdateBalance(productId uint32, subaccount [32]byte, amountDelta *big.Int, vQuoteDelta *big.Int) (*types.Transaction, error) {
	return _Perp.Contract.UpdateBalance(&_Perp.TransactOpts, productId, subaccount, amountDelta, vQuoteDelta)
}

// UpdateBalance is a paid mutator transaction binding the contract method 0xf8a42e51.
//
// Solidity: function updateBalance(uint32 productId, bytes32 subaccount, int128 amountDelta, int128 vQuoteDelta) returns(int128 fundingFees, int128 realisedPnl)
func (_Perp *PerpTransactorSession) UpdateBalance(productId uint32, subaccount [32]byte, amountDelta *big.Int, vQuoteDelta *big.Int) (*types.Transaction, error) {
	return _Perp.Contract.UpdateBalance(&_Perp.TransactOpts, productId, subaccount, amountDelta, vQuoteDelta)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Perp *PerpTransactor) UpdateFeeRates(opts *bind.TransactOpts, subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "updateFeeRates", subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Perp *PerpSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Perp.Contract.UpdateFeeRates(&_Perp.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_Perp *PerpTransactorSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _Perp.Contract.UpdateFeeRates(&_Perp.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0x85c01f89.
//
// Solidity: function updateProduct((uint32,int128,int128,uint32) txn) returns()
func (_Perp *PerpTransactor) UpdateProduct(opts *bind.TransactOpts, txn IPerpEngineUpdateProductTx) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "updateProduct", txn)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0x85c01f89.
//
// Solidity: function updateProduct((uint32,int128,int128,uint32) txn) returns()
func (_Perp *PerpSession) UpdateProduct(txn IPerpEngineUpdateProductTx) (*types.Transaction, error) {
	return _Perp.Contract.UpdateProduct(&_Perp.TransactOpts, txn)
}

// UpdateProduct is a paid mutator transaction binding the contract method 0x85c01f89.
//
// Solidity: function updateProduct((uint32,int128,int128,uint32) txn) returns()
func (_Perp *PerpTransactorSession) UpdateProduct(txn IPerpEngineUpdateProductTx) (*types.Transaction, error) {
	return _Perp.Contract.UpdateProduct(&_Perp.TransactOpts, txn)
}

// UpdateStates is a paid mutator transaction binding the contract method 0x6736f5da.
//
// Solidity: function updateStates(uint128 dt, int128[] fundingRateUpdates) returns()
func (_Perp *PerpTransactor) UpdateStates(opts *bind.TransactOpts, dt *big.Int, fundingRateUpdates []*big.Int) (*types.Transaction, error) {
	return _Perp.contract.Transact(opts, "updateStates", dt, fundingRateUpdates)
}

// UpdateStates is a paid mutator transaction binding the contract method 0x6736f5da.
//
// Solidity: function updateStates(uint128 dt, int128[] fundingRateUpdates) returns()
func (_Perp *PerpSession) UpdateStates(dt *big.Int, fundingRateUpdates []*big.Int) (*types.Transaction, error) {
	return _Perp.Contract.UpdateStates(&_Perp.TransactOpts, dt, fundingRateUpdates)
}

// UpdateStates is a paid mutator transaction binding the contract method 0x6736f5da.
//
// Solidity: function updateStates(uint128 dt, int128[] fundingRateUpdates) returns()
func (_Perp *PerpTransactorSession) UpdateStates(dt *big.Int, fundingRateUpdates []*big.Int) (*types.Transaction, error) {
	return _Perp.Contract.UpdateStates(&_Perp.TransactOpts, dt, fundingRateUpdates)
}

// PerpAddProductIterator is returned from FilterAddProduct and is used to iterate over the raw logs and unpacked data for AddProduct events raised by the Perp contract.
type PerpAddProductIterator struct {
	Event *PerpAddProduct // Event containing the contract specifics and raw log

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
func (it *PerpAddProductIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PerpAddProduct)
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
		it.Event = new(PerpAddProduct)
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
func (it *PerpAddProductIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PerpAddProductIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PerpAddProduct represents a AddProduct event raised by the Perp contract.
type PerpAddProduct struct {
	ProductId uint32
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAddProduct is a free log retrieval operation binding the contract event 0x3286b0394bf1350245290b7226c92ed186bd716f28938e62dbb895298f018172.
//
// Solidity: event AddProduct(uint32 productId)
func (_Perp *PerpFilterer) FilterAddProduct(opts *bind.FilterOpts) (*PerpAddProductIterator, error) {

	logs, sub, err := _Perp.contract.FilterLogs(opts, "AddProduct")
	if err != nil {
		return nil, err
	}
	return &PerpAddProductIterator{contract: _Perp.contract, event: "AddProduct", logs: logs, sub: sub}, nil
}

// WatchAddProduct is a free log subscription operation binding the contract event 0x3286b0394bf1350245290b7226c92ed186bd716f28938e62dbb895298f018172.
//
// Solidity: event AddProduct(uint32 productId)
func (_Perp *PerpFilterer) WatchAddProduct(opts *bind.WatchOpts, sink chan<- *PerpAddProduct) (event.Subscription, error) {

	logs, sub, err := _Perp.contract.WatchLogs(opts, "AddProduct")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PerpAddProduct)
				if err := _Perp.contract.UnpackLog(event, "AddProduct", log); err != nil {
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
func (_Perp *PerpFilterer) ParseAddProduct(log types.Log) (*PerpAddProduct, error) {
	event := new(PerpAddProduct)
	if err := _Perp.contract.UnpackLog(event, "AddProduct", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PerpFundingRateUpdatedIterator is returned from FilterFundingRateUpdated and is used to iterate over the raw logs and unpacked data for FundingRateUpdated events raised by the Perp contract.
type PerpFundingRateUpdatedIterator struct {
	Event *PerpFundingRateUpdated // Event containing the contract specifics and raw log

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
func (it *PerpFundingRateUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PerpFundingRateUpdated)
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
		it.Event = new(PerpFundingRateUpdated)
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
func (it *PerpFundingRateUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PerpFundingRateUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PerpFundingRateUpdated represents a FundingRateUpdated event raised by the Perp contract.
type PerpFundingRateUpdated struct {
	ProductId                    uint32
	NewCumulativeFundingLongX18  *big.Int
	NewCumulativeFundingShortX18 *big.Int
	Delta                        *big.Int
	Raw                          types.Log // Blockchain specific contextual infos
}

// FilterFundingRateUpdated is a free log retrieval operation binding the contract event 0x9eafee6cde0010e066d6939eb239ae2d3e786be333bcba08f4ce0e3c05ee462c.
//
// Solidity: event FundingRateUpdated(uint32 indexed productId, int128 indexed newCumulativeFundingLongX18, int128 indexed newCumulativeFundingShortX18, int128 delta)
func (_Perp *PerpFilterer) FilterFundingRateUpdated(opts *bind.FilterOpts, productId []uint32, newCumulativeFundingLongX18 []*big.Int, newCumulativeFundingShortX18 []*big.Int) (*PerpFundingRateUpdatedIterator, error) {

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}
	var newCumulativeFundingLongX18Rule []interface{}
	for _, newCumulativeFundingLongX18Item := range newCumulativeFundingLongX18 {
		newCumulativeFundingLongX18Rule = append(newCumulativeFundingLongX18Rule, newCumulativeFundingLongX18Item)
	}
	var newCumulativeFundingShortX18Rule []interface{}
	for _, newCumulativeFundingShortX18Item := range newCumulativeFundingShortX18 {
		newCumulativeFundingShortX18Rule = append(newCumulativeFundingShortX18Rule, newCumulativeFundingShortX18Item)
	}

	logs, sub, err := _Perp.contract.FilterLogs(opts, "FundingRateUpdated", productIdRule, newCumulativeFundingLongX18Rule, newCumulativeFundingShortX18Rule)
	if err != nil {
		return nil, err
	}
	return &PerpFundingRateUpdatedIterator{contract: _Perp.contract, event: "FundingRateUpdated", logs: logs, sub: sub}, nil
}

// WatchFundingRateUpdated is a free log subscription operation binding the contract event 0x9eafee6cde0010e066d6939eb239ae2d3e786be333bcba08f4ce0e3c05ee462c.
//
// Solidity: event FundingRateUpdated(uint32 indexed productId, int128 indexed newCumulativeFundingLongX18, int128 indexed newCumulativeFundingShortX18, int128 delta)
func (_Perp *PerpFilterer) WatchFundingRateUpdated(opts *bind.WatchOpts, sink chan<- *PerpFundingRateUpdated, productId []uint32, newCumulativeFundingLongX18 []*big.Int, newCumulativeFundingShortX18 []*big.Int) (event.Subscription, error) {

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}
	var newCumulativeFundingLongX18Rule []interface{}
	for _, newCumulativeFundingLongX18Item := range newCumulativeFundingLongX18 {
		newCumulativeFundingLongX18Rule = append(newCumulativeFundingLongX18Rule, newCumulativeFundingLongX18Item)
	}
	var newCumulativeFundingShortX18Rule []interface{}
	for _, newCumulativeFundingShortX18Item := range newCumulativeFundingShortX18 {
		newCumulativeFundingShortX18Rule = append(newCumulativeFundingShortX18Rule, newCumulativeFundingShortX18Item)
	}

	logs, sub, err := _Perp.contract.WatchLogs(opts, "FundingRateUpdated", productIdRule, newCumulativeFundingLongX18Rule, newCumulativeFundingShortX18Rule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PerpFundingRateUpdated)
				if err := _Perp.contract.UnpackLog(event, "FundingRateUpdated", log); err != nil {
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

// ParseFundingRateUpdated is a log parse operation binding the contract event 0x9eafee6cde0010e066d6939eb239ae2d3e786be333bcba08f4ce0e3c05ee462c.
//
// Solidity: event FundingRateUpdated(uint32 indexed productId, int128 indexed newCumulativeFundingLongX18, int128 indexed newCumulativeFundingShortX18, int128 delta)
func (_Perp *PerpFilterer) ParseFundingRateUpdated(log types.Log) (*PerpFundingRateUpdated, error) {
	event := new(PerpFundingRateUpdated)
	if err := _Perp.contract.UnpackLog(event, "FundingRateUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PerpInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Perp contract.
type PerpInitializedIterator struct {
	Event *PerpInitialized // Event containing the contract specifics and raw log

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
func (it *PerpInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PerpInitialized)
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
		it.Event = new(PerpInitialized)
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
func (it *PerpInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PerpInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PerpInitialized represents a Initialized event raised by the Perp contract.
type PerpInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Perp *PerpFilterer) FilterInitialized(opts *bind.FilterOpts) (*PerpInitializedIterator, error) {

	logs, sub, err := _Perp.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &PerpInitializedIterator{contract: _Perp.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Perp *PerpFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *PerpInitialized) (event.Subscription, error) {

	logs, sub, err := _Perp.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PerpInitialized)
				if err := _Perp.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_Perp *PerpFilterer) ParseInitialized(log types.Log) (*PerpInitialized, error) {
	event := new(PerpInitialized)
	if err := _Perp.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PerpOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Perp contract.
type PerpOwnershipTransferredIterator struct {
	Event *PerpOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *PerpOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PerpOwnershipTransferred)
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
		it.Event = new(PerpOwnershipTransferred)
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
func (it *PerpOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PerpOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PerpOwnershipTransferred represents a OwnershipTransferred event raised by the Perp contract.
type PerpOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Perp *PerpFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*PerpOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Perp.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &PerpOwnershipTransferredIterator{contract: _Perp.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Perp *PerpFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *PerpOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Perp.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PerpOwnershipTransferred)
				if err := _Perp.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Perp *PerpFilterer) ParseOwnershipTransferred(log types.Log) (*PerpOwnershipTransferred, error) {
	event := new(PerpOwnershipTransferred)
	if err := _Perp.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PerpProductUpdateIterator is returned from FilterProductUpdate and is used to iterate over the raw logs and unpacked data for ProductUpdate events raised by the Perp contract.
type PerpProductUpdateIterator struct {
	Event *PerpProductUpdate // Event containing the contract specifics and raw log

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
func (it *PerpProductUpdateIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PerpProductUpdate)
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
		it.Event = new(PerpProductUpdate)
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
func (it *PerpProductUpdateIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PerpProductUpdateIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PerpProductUpdate represents a ProductUpdate event raised by the Perp contract.
type PerpProductUpdate struct {
	ProductId uint32
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterProductUpdate is a free log retrieval operation binding the contract event 0xe6195122b31334b8a2bd5ec64f0dd6ac3ab865ac54c2a0413fb82dfb22ad6432.
//
// Solidity: event ProductUpdate(uint32 productId)
func (_Perp *PerpFilterer) FilterProductUpdate(opts *bind.FilterOpts) (*PerpProductUpdateIterator, error) {

	logs, sub, err := _Perp.contract.FilterLogs(opts, "ProductUpdate")
	if err != nil {
		return nil, err
	}
	return &PerpProductUpdateIterator{contract: _Perp.contract, event: "ProductUpdate", logs: logs, sub: sub}, nil
}

// WatchProductUpdate is a free log subscription operation binding the contract event 0xe6195122b31334b8a2bd5ec64f0dd6ac3ab865ac54c2a0413fb82dfb22ad6432.
//
// Solidity: event ProductUpdate(uint32 productId)
func (_Perp *PerpFilterer) WatchProductUpdate(opts *bind.WatchOpts, sink chan<- *PerpProductUpdate) (event.Subscription, error) {

	logs, sub, err := _Perp.contract.WatchLogs(opts, "ProductUpdate")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PerpProductUpdate)
				if err := _Perp.contract.UnpackLog(event, "ProductUpdate", log); err != nil {
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
func (_Perp *PerpFilterer) ParseProductUpdate(log types.Log) (*PerpProductUpdate, error) {
	event := new(PerpProductUpdate)
	if err := _Perp.contract.UnpackLog(event, "ProductUpdate", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
