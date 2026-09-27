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

// IEndpointMatchOrders is an auto generated low-level Go binding around an user-defined struct.
type IEndpointMatchOrders struct {
	ProductId uint32
	Taker     IEndpointOrder
	Maker     IEndpointOrder
}

// IEndpointMatchOrdersWithSigner is an auto generated low-level Go binding around an user-defined struct.
type IEndpointMatchOrdersWithSigner struct {
	MatchOrders       IEndpointMatchOrders
	TakerLinkedSigner common.Address
	MakerLinkedSigner common.Address
	TakerSignature    []byte
	MakerSignature    []byte
}

// IEndpointOrder is an auto generated low-level Go binding around an user-defined struct.
type IEndpointOrder struct {
	SubAccountId [32]byte
	PriceX18     *big.Int
	Amount       *big.Int
	Expiration   uint64
	IsReduce     bool
	SessionKey   common.Address
	ChainId      *big.Int
}

// IOffchainExchangeMarketInfo is an auto generated low-level Go binding around an user-defined struct.
type IOffchainExchangeMarketInfo struct {
	MinSize       *big.Int
	SizeIncrement *big.Int
}

// OffChainExchangeMetaData contains all meta data concerning the OffChainExchange contract.
var OffChainExchangeMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"collectFees\",\"inputs\":[{\"name\":\"fees\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"feesAccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"feeAmount\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"matchQuote\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"taker\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"filledAmounts\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAllVirtualBooks\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address[]\",\"internalType\":\"address[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getDigest\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"order\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.Order\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"expiration\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"isReduce\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEndpoint\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getFeeFractionX18\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"taker\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getMarketInfo\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"m\",\"type\":\"tuple\",\"internalType\":\"structIOffchainExchange.MarketInfo\",\"components\":[{\"name\":\"minSize\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"sizeIncrement\",\"type\":\"int128\",\"internalType\":\"int128\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getMinSize\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getSizeIncrement\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getVirtualBook\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_clearinghouse\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"matchOrders\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.MatchOrdersWithSigner\",\"components\":[{\"name\":\"matchOrders\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.MatchOrders\",\"components\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"taker\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.Order\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"expiration\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"isReduce\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"name\":\"maker\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.Order\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"priceX18\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"expiration\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"isReduce\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}]},{\"name\":\"takerLinkedSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"makerLinkedSigner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"takerSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"makerSignature\",\"type\":\"bytes\",\"internalType\":\"bytes\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setEndpoint\",\"inputs\":[{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateFeeRates\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"makerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"},{\"name\":\"takerRateX18\",\"type\":\"int64\",\"internalType\":\"int64\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateMarket\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"virtualBook\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"sizeIncrement\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"minSize\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"updateMinSizes\",\"inputs\":[{\"name\":\"productIds\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"},{\"name\":\"minSizes\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"FillOrder\",\"inputs\":[{\"name\":\"user\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"digest\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"subaccount\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"priceX18\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"amount\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"expiration\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"},{\"name\":\"isTaker\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"},{\"name\":\"feeAmount\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"baseDelta\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"quoteDelta\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"realisedPnl\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"fundingFees\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
}

// OffChainExchangeABI is the input ABI used to generate the binding from.
// Deprecated: Use OffChainExchangeMetaData.ABI instead.
var OffChainExchangeABI = OffChainExchangeMetaData.ABI

// OffChainExchange is an auto generated Go binding around an Ethereum contract.
type OffChainExchange struct {
	OffChainExchangeCaller     // Read-only binding to the contract
	OffChainExchangeTransactor // Write-only binding to the contract
	OffChainExchangeFilterer   // Log filterer for contract events
}

// OffChainExchangeCaller is an auto generated read-only Go binding around an Ethereum contract.
type OffChainExchangeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OffChainExchangeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OffChainExchangeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OffChainExchangeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OffChainExchangeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OffChainExchangeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OffChainExchangeSession struct {
	Contract     *OffChainExchange // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OffChainExchangeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OffChainExchangeCallerSession struct {
	Contract *OffChainExchangeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts           // Call options to use throughout this session
}

// OffChainExchangeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OffChainExchangeTransactorSession struct {
	Contract     *OffChainExchangeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts           // Transaction auth options to use throughout this session
}

// OffChainExchangeRaw is an auto generated low-level Go binding around an Ethereum contract.
type OffChainExchangeRaw struct {
	Contract *OffChainExchange // Generic contract binding to access the raw methods on
}

// OffChainExchangeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OffChainExchangeCallerRaw struct {
	Contract *OffChainExchangeCaller // Generic read-only contract binding to access the raw methods on
}

// OffChainExchangeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OffChainExchangeTransactorRaw struct {
	Contract *OffChainExchangeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOffChainExchange creates a new instance of OffChainExchange, bound to a specific deployed contract.
func NewOffChainExchange(address common.Address, backend bind.ContractBackend) (*OffChainExchange, error) {
	contract, err := bindOffChainExchange(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &OffChainExchange{OffChainExchangeCaller: OffChainExchangeCaller{contract: contract}, OffChainExchangeTransactor: OffChainExchangeTransactor{contract: contract}, OffChainExchangeFilterer: OffChainExchangeFilterer{contract: contract}}, nil
}

// NewOffChainExchangeCaller creates a new read-only instance of OffChainExchange, bound to a specific deployed contract.
func NewOffChainExchangeCaller(address common.Address, caller bind.ContractCaller) (*OffChainExchangeCaller, error) {
	contract, err := bindOffChainExchange(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeCaller{contract: contract}, nil
}

// NewOffChainExchangeTransactor creates a new write-only instance of OffChainExchange, bound to a specific deployed contract.
func NewOffChainExchangeTransactor(address common.Address, transactor bind.ContractTransactor) (*OffChainExchangeTransactor, error) {
	contract, err := bindOffChainExchange(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeTransactor{contract: contract}, nil
}

// NewOffChainExchangeFilterer creates a new log filterer instance of OffChainExchange, bound to a specific deployed contract.
func NewOffChainExchangeFilterer(address common.Address, filterer bind.ContractFilterer) (*OffChainExchangeFilterer, error) {
	contract, err := bindOffChainExchange(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeFilterer{contract: contract}, nil
}

// bindOffChainExchange binds a generic wrapper to an already deployed contract.
func bindOffChainExchange(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := OffChainExchangeMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OffChainExchange *OffChainExchangeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OffChainExchange.Contract.OffChainExchangeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OffChainExchange *OffChainExchangeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OffChainExchange.Contract.OffChainExchangeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OffChainExchange *OffChainExchangeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OffChainExchange.Contract.OffChainExchangeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_OffChainExchange *OffChainExchangeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _OffChainExchange.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_OffChainExchange *OffChainExchangeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OffChainExchange.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_OffChainExchange *OffChainExchangeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _OffChainExchange.Contract.contract.Transact(opts, method, params...)
}

// FilledAmounts is a free data retrieval call binding the contract method 0x40f1a34d.
//
// Solidity: function filledAmounts(bytes32 ) view returns(int128)
func (_OffChainExchange *OffChainExchangeCaller) FilledAmounts(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "filledAmounts", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FilledAmounts is a free data retrieval call binding the contract method 0x40f1a34d.
//
// Solidity: function filledAmounts(bytes32 ) view returns(int128)
func (_OffChainExchange *OffChainExchangeSession) FilledAmounts(arg0 [32]byte) (*big.Int, error) {
	return _OffChainExchange.Contract.FilledAmounts(&_OffChainExchange.CallOpts, arg0)
}

// FilledAmounts is a free data retrieval call binding the contract method 0x40f1a34d.
//
// Solidity: function filledAmounts(bytes32 ) view returns(int128)
func (_OffChainExchange *OffChainExchangeCallerSession) FilledAmounts(arg0 [32]byte) (*big.Int, error) {
	return _OffChainExchange.Contract.FilledAmounts(&_OffChainExchange.CallOpts, arg0)
}

// GetAllVirtualBooks is a free data retrieval call binding the contract method 0xce933e59.
//
// Solidity: function getAllVirtualBooks() view returns(address[])
func (_OffChainExchange *OffChainExchangeCaller) GetAllVirtualBooks(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getAllVirtualBooks")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetAllVirtualBooks is a free data retrieval call binding the contract method 0xce933e59.
//
// Solidity: function getAllVirtualBooks() view returns(address[])
func (_OffChainExchange *OffChainExchangeSession) GetAllVirtualBooks() ([]common.Address, error) {
	return _OffChainExchange.Contract.GetAllVirtualBooks(&_OffChainExchange.CallOpts)
}

// GetAllVirtualBooks is a free data retrieval call binding the contract method 0xce933e59.
//
// Solidity: function getAllVirtualBooks() view returns(address[])
func (_OffChainExchange *OffChainExchangeCallerSession) GetAllVirtualBooks() ([]common.Address, error) {
	return _OffChainExchange.Contract.GetAllVirtualBooks(&_OffChainExchange.CallOpts)
}

// GetDigest is a free data retrieval call binding the contract method 0xcd74c620.
//
// Solidity: function getDigest(uint32 productId, (bytes32,int128,int128,uint64,bool,address,uint256) order) view returns(bytes32)
func (_OffChainExchange *OffChainExchangeCaller) GetDigest(opts *bind.CallOpts, productId uint32, order IEndpointOrder) ([32]byte, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getDigest", productId, order)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetDigest is a free data retrieval call binding the contract method 0xcd74c620.
//
// Solidity: function getDigest(uint32 productId, (bytes32,int128,int128,uint64,bool,address,uint256) order) view returns(bytes32)
func (_OffChainExchange *OffChainExchangeSession) GetDigest(productId uint32, order IEndpointOrder) ([32]byte, error) {
	return _OffChainExchange.Contract.GetDigest(&_OffChainExchange.CallOpts, productId, order)
}

// GetDigest is a free data retrieval call binding the contract method 0xcd74c620.
//
// Solidity: function getDigest(uint32 productId, (bytes32,int128,int128,uint64,bool,address,uint256) order) view returns(bytes32)
func (_OffChainExchange *OffChainExchangeCallerSession) GetDigest(productId uint32, order IEndpointOrder) ([32]byte, error) {
	return _OffChainExchange.Contract.GetDigest(&_OffChainExchange.CallOpts, productId, order)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_OffChainExchange *OffChainExchangeCaller) GetEndpoint(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getEndpoint")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_OffChainExchange *OffChainExchangeSession) GetEndpoint() (common.Address, error) {
	return _OffChainExchange.Contract.GetEndpoint(&_OffChainExchange.CallOpts)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_OffChainExchange *OffChainExchangeCallerSession) GetEndpoint() (common.Address, error) {
	return _OffChainExchange.Contract.GetEndpoint(&_OffChainExchange.CallOpts)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_OffChainExchange *OffChainExchangeCaller) GetFeeFractionX18(opts *bind.CallOpts, subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getFeeFractionX18", subaccount, productId, taker)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_OffChainExchange *OffChainExchangeSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _OffChainExchange.Contract.GetFeeFractionX18(&_OffChainExchange.CallOpts, subaccount, productId, taker)
}

// GetFeeFractionX18 is a free data retrieval call binding the contract method 0xb5cbd70e.
//
// Solidity: function getFeeFractionX18(bytes32 subaccount, uint32 productId, bool taker) view returns(int128)
func (_OffChainExchange *OffChainExchangeCallerSession) GetFeeFractionX18(subaccount [32]byte, productId uint32, taker bool) (*big.Int, error) {
	return _OffChainExchange.Contract.GetFeeFractionX18(&_OffChainExchange.CallOpts, subaccount, productId, taker)
}

// GetMarketInfo is a free data retrieval call binding the contract method 0x1d029b4d.
//
// Solidity: function getMarketInfo(uint32 productId) view returns((int128,int128) m)
func (_OffChainExchange *OffChainExchangeCaller) GetMarketInfo(opts *bind.CallOpts, productId uint32) (IOffchainExchangeMarketInfo, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getMarketInfo", productId)

	if err != nil {
		return *new(IOffchainExchangeMarketInfo), err
	}

	out0 := *abi.ConvertType(out[0], new(IOffchainExchangeMarketInfo)).(*IOffchainExchangeMarketInfo)

	return out0, err

}

// GetMarketInfo is a free data retrieval call binding the contract method 0x1d029b4d.
//
// Solidity: function getMarketInfo(uint32 productId) view returns((int128,int128) m)
func (_OffChainExchange *OffChainExchangeSession) GetMarketInfo(productId uint32) (IOffchainExchangeMarketInfo, error) {
	return _OffChainExchange.Contract.GetMarketInfo(&_OffChainExchange.CallOpts, productId)
}

// GetMarketInfo is a free data retrieval call binding the contract method 0x1d029b4d.
//
// Solidity: function getMarketInfo(uint32 productId) view returns((int128,int128) m)
func (_OffChainExchange *OffChainExchangeCallerSession) GetMarketInfo(productId uint32) (IOffchainExchangeMarketInfo, error) {
	return _OffChainExchange.Contract.GetMarketInfo(&_OffChainExchange.CallOpts, productId)
}

// GetMinSize is a free data retrieval call binding the contract method 0xb60aaa7c.
//
// Solidity: function getMinSize(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeCaller) GetMinSize(opts *bind.CallOpts, productId uint32) (*big.Int, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getMinSize", productId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetMinSize is a free data retrieval call binding the contract method 0xb60aaa7c.
//
// Solidity: function getMinSize(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeSession) GetMinSize(productId uint32) (*big.Int, error) {
	return _OffChainExchange.Contract.GetMinSize(&_OffChainExchange.CallOpts, productId)
}

// GetMinSize is a free data retrieval call binding the contract method 0xb60aaa7c.
//
// Solidity: function getMinSize(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeCallerSession) GetMinSize(productId uint32) (*big.Int, error) {
	return _OffChainExchange.Contract.GetMinSize(&_OffChainExchange.CallOpts, productId)
}

// GetSizeIncrement is a free data retrieval call binding the contract method 0xf2b26331.
//
// Solidity: function getSizeIncrement(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeCaller) GetSizeIncrement(opts *bind.CallOpts, productId uint32) (*big.Int, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getSizeIncrement", productId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSizeIncrement is a free data retrieval call binding the contract method 0xf2b26331.
//
// Solidity: function getSizeIncrement(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeSession) GetSizeIncrement(productId uint32) (*big.Int, error) {
	return _OffChainExchange.Contract.GetSizeIncrement(&_OffChainExchange.CallOpts, productId)
}

// GetSizeIncrement is a free data retrieval call binding the contract method 0xf2b26331.
//
// Solidity: function getSizeIncrement(uint32 productId) view returns(int128)
func (_OffChainExchange *OffChainExchangeCallerSession) GetSizeIncrement(productId uint32) (*big.Int, error) {
	return _OffChainExchange.Contract.GetSizeIncrement(&_OffChainExchange.CallOpts, productId)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_OffChainExchange *OffChainExchangeCaller) GetVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_OffChainExchange *OffChainExchangeSession) GetVersion() (uint64, error) {
	return _OffChainExchange.Contract.GetVersion(&_OffChainExchange.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_OffChainExchange *OffChainExchangeCallerSession) GetVersion() (uint64, error) {
	return _OffChainExchange.Contract.GetVersion(&_OffChainExchange.CallOpts)
}

// GetVirtualBook is a free data retrieval call binding the contract method 0x66f87bd1.
//
// Solidity: function getVirtualBook(uint32 productId) view returns(address)
func (_OffChainExchange *OffChainExchangeCaller) GetVirtualBook(opts *bind.CallOpts, productId uint32) (common.Address, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "getVirtualBook", productId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetVirtualBook is a free data retrieval call binding the contract method 0x66f87bd1.
//
// Solidity: function getVirtualBook(uint32 productId) view returns(address)
func (_OffChainExchange *OffChainExchangeSession) GetVirtualBook(productId uint32) (common.Address, error) {
	return _OffChainExchange.Contract.GetVirtualBook(&_OffChainExchange.CallOpts, productId)
}

// GetVirtualBook is a free data retrieval call binding the contract method 0x66f87bd1.
//
// Solidity: function getVirtualBook(uint32 productId) view returns(address)
func (_OffChainExchange *OffChainExchangeCallerSession) GetVirtualBook(productId uint32) (common.Address, error) {
	return _OffChainExchange.Contract.GetVirtualBook(&_OffChainExchange.CallOpts, productId)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OffChainExchange *OffChainExchangeCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _OffChainExchange.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OffChainExchange *OffChainExchangeSession) Owner() (common.Address, error) {
	return _OffChainExchange.Contract.Owner(&_OffChainExchange.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_OffChainExchange *OffChainExchangeCallerSession) Owner() (common.Address, error) {
	return _OffChainExchange.Contract.Owner(&_OffChainExchange.CallOpts)
}

// CollectFees is a paid mutator transaction binding the contract method 0x90265b7e.
//
// Solidity: function collectFees(int128 fees, bytes32 feesAccount) returns()
func (_OffChainExchange *OffChainExchangeTransactor) CollectFees(opts *bind.TransactOpts, fees *big.Int, feesAccount [32]byte) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "collectFees", fees, feesAccount)
}

// CollectFees is a paid mutator transaction binding the contract method 0x90265b7e.
//
// Solidity: function collectFees(int128 fees, bytes32 feesAccount) returns()
func (_OffChainExchange *OffChainExchangeSession) CollectFees(fees *big.Int, feesAccount [32]byte) (*types.Transaction, error) {
	return _OffChainExchange.Contract.CollectFees(&_OffChainExchange.TransactOpts, fees, feesAccount)
}

// CollectFees is a paid mutator transaction binding the contract method 0x90265b7e.
//
// Solidity: function collectFees(int128 fees, bytes32 feesAccount) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) CollectFees(fees *big.Int, feesAccount [32]byte) (*types.Transaction, error) {
	return _OffChainExchange.Contract.CollectFees(&_OffChainExchange.TransactOpts, fees, feesAccount)
}

// FeeAmount is a paid mutator transaction binding the contract method 0x7a3b6cd6.
//
// Solidity: function feeAmount(uint32 productId, bytes32 subaccount, int128 matchQuote, bool taker) returns(int128)
func (_OffChainExchange *OffChainExchangeTransactor) FeeAmount(opts *bind.TransactOpts, productId uint32, subaccount [32]byte, matchQuote *big.Int, taker bool) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "feeAmount", productId, subaccount, matchQuote, taker)
}

// FeeAmount is a paid mutator transaction binding the contract method 0x7a3b6cd6.
//
// Solidity: function feeAmount(uint32 productId, bytes32 subaccount, int128 matchQuote, bool taker) returns(int128)
func (_OffChainExchange *OffChainExchangeSession) FeeAmount(productId uint32, subaccount [32]byte, matchQuote *big.Int, taker bool) (*types.Transaction, error) {
	return _OffChainExchange.Contract.FeeAmount(&_OffChainExchange.TransactOpts, productId, subaccount, matchQuote, taker)
}

// FeeAmount is a paid mutator transaction binding the contract method 0x7a3b6cd6.
//
// Solidity: function feeAmount(uint32 productId, bytes32 subaccount, int128 matchQuote, bool taker) returns(int128)
func (_OffChainExchange *OffChainExchangeTransactorSession) FeeAmount(productId uint32, subaccount [32]byte, matchQuote *big.Int, taker bool) (*types.Transaction, error) {
	return _OffChainExchange.Contract.FeeAmount(&_OffChainExchange.TransactOpts, productId, subaccount, matchQuote, taker)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _clearinghouse, address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeTransactor) Initialize(opts *bind.TransactOpts, _clearinghouse common.Address, _endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "initialize", _clearinghouse, _endpoint)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _clearinghouse, address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeSession) Initialize(_clearinghouse common.Address, _endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.Initialize(&_OffChainExchange.TransactOpts, _clearinghouse, _endpoint)
}

// Initialize is a paid mutator transaction binding the contract method 0x485cc955.
//
// Solidity: function initialize(address _clearinghouse, address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) Initialize(_clearinghouse common.Address, _endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.Initialize(&_OffChainExchange.TransactOpts, _clearinghouse, _endpoint)
}

// MatchOrders is a paid mutator transaction binding the contract method 0x027b4f5c.
//
// Solidity: function matchOrders(((uint32,(bytes32,int128,int128,uint64,bool,address,uint256),(bytes32,int128,int128,uint64,bool,address,uint256)),address,address,bytes,bytes) txn) returns()
func (_OffChainExchange *OffChainExchangeTransactor) MatchOrders(opts *bind.TransactOpts, txn IEndpointMatchOrdersWithSigner) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "matchOrders", txn)
}

// MatchOrders is a paid mutator transaction binding the contract method 0x027b4f5c.
//
// Solidity: function matchOrders(((uint32,(bytes32,int128,int128,uint64,bool,address,uint256),(bytes32,int128,int128,uint64,bool,address,uint256)),address,address,bytes,bytes) txn) returns()
func (_OffChainExchange *OffChainExchangeSession) MatchOrders(txn IEndpointMatchOrdersWithSigner) (*types.Transaction, error) {
	return _OffChainExchange.Contract.MatchOrders(&_OffChainExchange.TransactOpts, txn)
}

// MatchOrders is a paid mutator transaction binding the contract method 0x027b4f5c.
//
// Solidity: function matchOrders(((uint32,(bytes32,int128,int128,uint64,bool,address,uint256),(bytes32,int128,int128,uint64,bool,address,uint256)),address,address,bytes,bytes) txn) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) MatchOrders(txn IEndpointMatchOrdersWithSigner) (*types.Transaction, error) {
	return _OffChainExchange.Contract.MatchOrders(&_OffChainExchange.TransactOpts, txn)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OffChainExchange *OffChainExchangeTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OffChainExchange *OffChainExchangeSession) RenounceOwnership() (*types.Transaction, error) {
	return _OffChainExchange.Contract.RenounceOwnership(&_OffChainExchange.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _OffChainExchange.Contract.RenounceOwnership(&_OffChainExchange.TransactOpts)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeTransactor) SetEndpoint(opts *bind.TransactOpts, _endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "setEndpoint", _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.SetEndpoint(&_OffChainExchange.TransactOpts, _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.SetEndpoint(&_OffChainExchange.TransactOpts, _endpoint)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OffChainExchange *OffChainExchangeTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OffChainExchange *OffChainExchangeSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.TransferOwnership(&_OffChainExchange.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _OffChainExchange.Contract.TransferOwnership(&_OffChainExchange.TransactOpts, newOwner)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_OffChainExchange *OffChainExchangeTransactor) UpdateFeeRates(opts *bind.TransactOpts, subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "updateFeeRates", subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_OffChainExchange *OffChainExchangeSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateFeeRates(&_OffChainExchange.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateFeeRates is a paid mutator transaction binding the contract method 0xea40c80e.
//
// Solidity: function updateFeeRates(bytes32 subAccountId, uint32 productId, int64 makerRateX18, int64 takerRateX18) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) UpdateFeeRates(subAccountId [32]byte, productId uint32, makerRateX18 int64, takerRateX18 int64) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateFeeRates(&_OffChainExchange.TransactOpts, subAccountId, productId, makerRateX18, takerRateX18)
}

// UpdateMarket is a paid mutator transaction binding the contract method 0xdd02f28b.
//
// Solidity: function updateMarket(uint32 productId, address virtualBook, int128 sizeIncrement, int128 minSize) returns()
func (_OffChainExchange *OffChainExchangeTransactor) UpdateMarket(opts *bind.TransactOpts, productId uint32, virtualBook common.Address, sizeIncrement *big.Int, minSize *big.Int) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "updateMarket", productId, virtualBook, sizeIncrement, minSize)
}

// UpdateMarket is a paid mutator transaction binding the contract method 0xdd02f28b.
//
// Solidity: function updateMarket(uint32 productId, address virtualBook, int128 sizeIncrement, int128 minSize) returns()
func (_OffChainExchange *OffChainExchangeSession) UpdateMarket(productId uint32, virtualBook common.Address, sizeIncrement *big.Int, minSize *big.Int) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateMarket(&_OffChainExchange.TransactOpts, productId, virtualBook, sizeIncrement, minSize)
}

// UpdateMarket is a paid mutator transaction binding the contract method 0xdd02f28b.
//
// Solidity: function updateMarket(uint32 productId, address virtualBook, int128 sizeIncrement, int128 minSize) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) UpdateMarket(productId uint32, virtualBook common.Address, sizeIncrement *big.Int, minSize *big.Int) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateMarket(&_OffChainExchange.TransactOpts, productId, virtualBook, sizeIncrement, minSize)
}

// UpdateMinSizes is a paid mutator transaction binding the contract method 0xe77bce84.
//
// Solidity: function updateMinSizes(uint32[] productIds, int128[] minSizes) returns()
func (_OffChainExchange *OffChainExchangeTransactor) UpdateMinSizes(opts *bind.TransactOpts, productIds []uint32, minSizes []*big.Int) (*types.Transaction, error) {
	return _OffChainExchange.contract.Transact(opts, "updateMinSizes", productIds, minSizes)
}

// UpdateMinSizes is a paid mutator transaction binding the contract method 0xe77bce84.
//
// Solidity: function updateMinSizes(uint32[] productIds, int128[] minSizes) returns()
func (_OffChainExchange *OffChainExchangeSession) UpdateMinSizes(productIds []uint32, minSizes []*big.Int) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateMinSizes(&_OffChainExchange.TransactOpts, productIds, minSizes)
}

// UpdateMinSizes is a paid mutator transaction binding the contract method 0xe77bce84.
//
// Solidity: function updateMinSizes(uint32[] productIds, int128[] minSizes) returns()
func (_OffChainExchange *OffChainExchangeTransactorSession) UpdateMinSizes(productIds []uint32, minSizes []*big.Int) (*types.Transaction, error) {
	return _OffChainExchange.Contract.UpdateMinSizes(&_OffChainExchange.TransactOpts, productIds, minSizes)
}

// OffChainExchangeFillOrderIterator is returned from FilterFillOrder and is used to iterate over the raw logs and unpacked data for FillOrder events raised by the OffChainExchange contract.
type OffChainExchangeFillOrderIterator struct {
	Event *OffChainExchangeFillOrder // Event containing the contract specifics and raw log

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
func (it *OffChainExchangeFillOrderIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OffChainExchangeFillOrder)
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
		it.Event = new(OffChainExchangeFillOrder)
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
func (it *OffChainExchangeFillOrderIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OffChainExchangeFillOrderIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OffChainExchangeFillOrder represents a FillOrder event raised by the OffChainExchange contract.
type OffChainExchangeFillOrder struct {
	User        common.Address
	ProductId   uint32
	Digest      [32]byte
	Subaccount  [32]byte
	PriceX18    *big.Int
	Amount      *big.Int
	Expiration  uint64
	IsTaker     bool
	FeeAmount   *big.Int
	BaseDelta   *big.Int
	QuoteDelta  *big.Int
	RealisedPnl *big.Int
	FundingFees *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterFillOrder is a free log retrieval operation binding the contract event 0xee9e9d22798a7382f25061d0ef84368ab48298fcddd49bb4a7f5f87fe7766eb9.
//
// Solidity: event FillOrder(address user, uint32 indexed productId, bytes32 indexed digest, bytes32 indexed subaccount, int128 priceX18, int128 amount, uint64 expiration, bool isTaker, int128 feeAmount, int128 baseDelta, int128 quoteDelta, int128 realisedPnl, int128 fundingFees)
func (_OffChainExchange *OffChainExchangeFilterer) FilterFillOrder(opts *bind.FilterOpts, productId []uint32, digest [][32]byte, subaccount [][32]byte) (*OffChainExchangeFillOrderIterator, error) {

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}
	var digestRule []interface{}
	for _, digestItem := range digest {
		digestRule = append(digestRule, digestItem)
	}
	var subaccountRule []interface{}
	for _, subaccountItem := range subaccount {
		subaccountRule = append(subaccountRule, subaccountItem)
	}

	logs, sub, err := _OffChainExchange.contract.FilterLogs(opts, "FillOrder", productIdRule, digestRule, subaccountRule)
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeFillOrderIterator{contract: _OffChainExchange.contract, event: "FillOrder", logs: logs, sub: sub}, nil
}

// WatchFillOrder is a free log subscription operation binding the contract event 0xee9e9d22798a7382f25061d0ef84368ab48298fcddd49bb4a7f5f87fe7766eb9.
//
// Solidity: event FillOrder(address user, uint32 indexed productId, bytes32 indexed digest, bytes32 indexed subaccount, int128 priceX18, int128 amount, uint64 expiration, bool isTaker, int128 feeAmount, int128 baseDelta, int128 quoteDelta, int128 realisedPnl, int128 fundingFees)
func (_OffChainExchange *OffChainExchangeFilterer) WatchFillOrder(opts *bind.WatchOpts, sink chan<- *OffChainExchangeFillOrder, productId []uint32, digest [][32]byte, subaccount [][32]byte) (event.Subscription, error) {

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}
	var digestRule []interface{}
	for _, digestItem := range digest {
		digestRule = append(digestRule, digestItem)
	}
	var subaccountRule []interface{}
	for _, subaccountItem := range subaccount {
		subaccountRule = append(subaccountRule, subaccountItem)
	}

	logs, sub, err := _OffChainExchange.contract.WatchLogs(opts, "FillOrder", productIdRule, digestRule, subaccountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OffChainExchangeFillOrder)
				if err := _OffChainExchange.contract.UnpackLog(event, "FillOrder", log); err != nil {
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

// ParseFillOrder is a log parse operation binding the contract event 0xee9e9d22798a7382f25061d0ef84368ab48298fcddd49bb4a7f5f87fe7766eb9.
//
// Solidity: event FillOrder(address user, uint32 indexed productId, bytes32 indexed digest, bytes32 indexed subaccount, int128 priceX18, int128 amount, uint64 expiration, bool isTaker, int128 feeAmount, int128 baseDelta, int128 quoteDelta, int128 realisedPnl, int128 fundingFees)
func (_OffChainExchange *OffChainExchangeFilterer) ParseFillOrder(log types.Log) (*OffChainExchangeFillOrder, error) {
	event := new(OffChainExchangeFillOrder)
	if err := _OffChainExchange.contract.UnpackLog(event, "FillOrder", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OffChainExchangeInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the OffChainExchange contract.
type OffChainExchangeInitializedIterator struct {
	Event *OffChainExchangeInitialized // Event containing the contract specifics and raw log

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
func (it *OffChainExchangeInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OffChainExchangeInitialized)
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
		it.Event = new(OffChainExchangeInitialized)
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
func (it *OffChainExchangeInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OffChainExchangeInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OffChainExchangeInitialized represents a Initialized event raised by the OffChainExchange contract.
type OffChainExchangeInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_OffChainExchange *OffChainExchangeFilterer) FilterInitialized(opts *bind.FilterOpts) (*OffChainExchangeInitializedIterator, error) {

	logs, sub, err := _OffChainExchange.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeInitializedIterator{contract: _OffChainExchange.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_OffChainExchange *OffChainExchangeFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *OffChainExchangeInitialized) (event.Subscription, error) {

	logs, sub, err := _OffChainExchange.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OffChainExchangeInitialized)
				if err := _OffChainExchange.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_OffChainExchange *OffChainExchangeFilterer) ParseInitialized(log types.Log) (*OffChainExchangeInitialized, error) {
	event := new(OffChainExchangeInitialized)
	if err := _OffChainExchange.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OffChainExchangeOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the OffChainExchange contract.
type OffChainExchangeOwnershipTransferredIterator struct {
	Event *OffChainExchangeOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *OffChainExchangeOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(OffChainExchangeOwnershipTransferred)
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
		it.Event = new(OffChainExchangeOwnershipTransferred)
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
func (it *OffChainExchangeOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *OffChainExchangeOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// OffChainExchangeOwnershipTransferred represents a OwnershipTransferred event raised by the OffChainExchange contract.
type OffChainExchangeOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OffChainExchange *OffChainExchangeFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*OffChainExchangeOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OffChainExchange.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &OffChainExchangeOwnershipTransferredIterator{contract: _OffChainExchange.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_OffChainExchange *OffChainExchangeFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *OffChainExchangeOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _OffChainExchange.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(OffChainExchangeOwnershipTransferred)
				if err := _OffChainExchange.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_OffChainExchange *OffChainExchangeFilterer) ParseOwnershipTransferred(log types.Log) (*OffChainExchangeOwnershipTransferred, error) {
	event := new(OffChainExchangeOwnershipTransferred)
	if err := _OffChainExchange.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
