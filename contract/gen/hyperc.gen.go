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

// GasRouterGasRouterConfig is an auto generated low-level Go binding around an user-defined struct.
type GasRouterGasRouterConfig struct {
	Domain uint32
	Gas    *big.Int
}

// HypErcMetaData contains all meta data concerning the HypErc contract.
var HypErcMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"erc20\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_mailbox\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"_account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"destinationGas\",\"inputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"domains\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"enrollRemoteRouter\",\"inputs\":[{\"name\":\"_domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_router\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"enrollRemoteRouters\",\"inputs\":[{\"name\":\"_domains\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"},{\"name\":\"_addresses\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"handle\",\"inputs\":[{\"name\":\"_origin\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_message\",\"type\":\"bytes\",\"internalType\":\"bytes\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"hook\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIPostDispatchHook\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_hook\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_interchainSecurityModule\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_owner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"interchainSecurityModule\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIInterchainSecurityModule\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"localDomain\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"mailbox\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIMailbox\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"quoteGasPayment\",\"inputs\":[{\"name\":\"_destinationDomain\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"_gasPayment\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"routers\",\"inputs\":[{\"name\":\"_domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"setDestinationGas\",\"inputs\":[{\"name\":\"domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"gas\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setDestinationGas\",\"inputs\":[{\"name\":\"gasConfigs\",\"type\":\"tuple[]\",\"internalType\":\"structGasRouter.GasRouterConfig[]\",\"components\":[{\"name\":\"domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"gas\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setHook\",\"inputs\":[{\"name\":\"_hook\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInterchainSecurityModule\",\"inputs\":[{\"name\":\"_module\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferRemote\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_amountOrId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_hookMetadata\",\"type\":\"bytes\",\"internalType\":\"bytes\"},{\"name\":\"_hook\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"messageId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"transferRemote\",\"inputs\":[{\"name\":\"_destination\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"_recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_amountOrId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"messageId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"unenrollRemoteRouter\",\"inputs\":[{\"name\":\"_domain\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unenrollRemoteRouters\",\"inputs\":[{\"name\":\"_domains\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"wrappedToken\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIERC20\"}],\"stateMutability\":\"view\"},{\"type\":\"event\",\"name\":\"DepositToken\",\"inputs\":[{\"name\":\"messageId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"subaccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint8\",\"indexed\":false,\"internalType\":\"uint8\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ReceivedTransferRemote\",\"inputs\":[{\"name\":\"origin\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SentTransferRemote\",\"inputs\":[{\"name\":\"destination\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false}]",
}

// HypErcABI is the input ABI used to generate the binding from.
// Deprecated: Use HypErcMetaData.ABI instead.
var HypErcABI = HypErcMetaData.ABI

// HypErc is an auto generated Go binding around an Ethereum contract.
type HypErc struct {
	HypErcCaller     // Read-only binding to the contract
	HypErcTransactor // Write-only binding to the contract
	HypErcFilterer   // Log filterer for contract events
}

// HypErcCaller is an auto generated read-only Go binding around an Ethereum contract.
type HypErcCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HypErcTransactor is an auto generated write-only Go binding around an Ethereum contract.
type HypErcTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HypErcFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type HypErcFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// HypErcSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type HypErcSession struct {
	Contract     *HypErc           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HypErcCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type HypErcCallerSession struct {
	Contract *HypErcCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// HypErcTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type HypErcTransactorSession struct {
	Contract     *HypErcTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// HypErcRaw is an auto generated low-level Go binding around an Ethereum contract.
type HypErcRaw struct {
	Contract *HypErc // Generic contract binding to access the raw methods on
}

// HypErcCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type HypErcCallerRaw struct {
	Contract *HypErcCaller // Generic read-only contract binding to access the raw methods on
}

// HypErcTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type HypErcTransactorRaw struct {
	Contract *HypErcTransactor // Generic write-only contract binding to access the raw methods on
}

// NewHypErc creates a new instance of HypErc, bound to a specific deployed contract.
func NewHypErc(address common.Address, backend bind.ContractBackend) (*HypErc, error) {
	contract, err := bindHypErc(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &HypErc{HypErcCaller: HypErcCaller{contract: contract}, HypErcTransactor: HypErcTransactor{contract: contract}, HypErcFilterer: HypErcFilterer{contract: contract}}, nil
}

// NewHypErcCaller creates a new read-only instance of HypErc, bound to a specific deployed contract.
func NewHypErcCaller(address common.Address, caller bind.ContractCaller) (*HypErcCaller, error) {
	contract, err := bindHypErc(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &HypErcCaller{contract: contract}, nil
}

// NewHypErcTransactor creates a new write-only instance of HypErc, bound to a specific deployed contract.
func NewHypErcTransactor(address common.Address, transactor bind.ContractTransactor) (*HypErcTransactor, error) {
	contract, err := bindHypErc(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &HypErcTransactor{contract: contract}, nil
}

// NewHypErcFilterer creates a new log filterer instance of HypErc, bound to a specific deployed contract.
func NewHypErcFilterer(address common.Address, filterer bind.ContractFilterer) (*HypErcFilterer, error) {
	contract, err := bindHypErc(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &HypErcFilterer{contract: contract}, nil
}

// bindHypErc binds a generic wrapper to an already deployed contract.
func bindHypErc(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := HypErcMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HypErc *HypErcRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HypErc.Contract.HypErcCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HypErc *HypErcRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HypErc.Contract.HypErcTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HypErc *HypErcRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HypErc.Contract.HypErcTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_HypErc *HypErcCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _HypErc.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_HypErc *HypErcTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HypErc.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_HypErc *HypErcTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _HypErc.Contract.contract.Transact(opts, method, params...)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_HypErc *HypErcCaller) BalanceOf(opts *bind.CallOpts, _account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "balanceOf", _account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_HypErc *HypErcSession) BalanceOf(_account common.Address) (*big.Int, error) {
	return _HypErc.Contract.BalanceOf(&_HypErc.CallOpts, _account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_HypErc *HypErcCallerSession) BalanceOf(_account common.Address) (*big.Int, error) {
	return _HypErc.Contract.BalanceOf(&_HypErc.CallOpts, _account)
}

// DestinationGas is a free data retrieval call binding the contract method 0x775313a1.
//
// Solidity: function destinationGas(uint32 ) view returns(uint256)
func (_HypErc *HypErcCaller) DestinationGas(opts *bind.CallOpts, arg0 uint32) (*big.Int, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "destinationGas", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DestinationGas is a free data retrieval call binding the contract method 0x775313a1.
//
// Solidity: function destinationGas(uint32 ) view returns(uint256)
func (_HypErc *HypErcSession) DestinationGas(arg0 uint32) (*big.Int, error) {
	return _HypErc.Contract.DestinationGas(&_HypErc.CallOpts, arg0)
}

// DestinationGas is a free data retrieval call binding the contract method 0x775313a1.
//
// Solidity: function destinationGas(uint32 ) view returns(uint256)
func (_HypErc *HypErcCallerSession) DestinationGas(arg0 uint32) (*big.Int, error) {
	return _HypErc.Contract.DestinationGas(&_HypErc.CallOpts, arg0)
}

// Domains is a free data retrieval call binding the contract method 0x440df4f4.
//
// Solidity: function domains() view returns(uint32[])
func (_HypErc *HypErcCaller) Domains(opts *bind.CallOpts) ([]uint32, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "domains")

	if err != nil {
		return *new([]uint32), err
	}

	out0 := *abi.ConvertType(out[0], new([]uint32)).(*[]uint32)

	return out0, err

}

// Domains is a free data retrieval call binding the contract method 0x440df4f4.
//
// Solidity: function domains() view returns(uint32[])
func (_HypErc *HypErcSession) Domains() ([]uint32, error) {
	return _HypErc.Contract.Domains(&_HypErc.CallOpts)
}

// Domains is a free data retrieval call binding the contract method 0x440df4f4.
//
// Solidity: function domains() view returns(uint32[])
func (_HypErc *HypErcCallerSession) Domains() ([]uint32, error) {
	return _HypErc.Contract.Domains(&_HypErc.CallOpts)
}

// Hook is a free data retrieval call binding the contract method 0x7f5a7c7b.
//
// Solidity: function hook() view returns(address)
func (_HypErc *HypErcCaller) Hook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "hook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Hook is a free data retrieval call binding the contract method 0x7f5a7c7b.
//
// Solidity: function hook() view returns(address)
func (_HypErc *HypErcSession) Hook() (common.Address, error) {
	return _HypErc.Contract.Hook(&_HypErc.CallOpts)
}

// Hook is a free data retrieval call binding the contract method 0x7f5a7c7b.
//
// Solidity: function hook() view returns(address)
func (_HypErc *HypErcCallerSession) Hook() (common.Address, error) {
	return _HypErc.Contract.Hook(&_HypErc.CallOpts)
}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_HypErc *HypErcCaller) InterchainSecurityModule(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "interchainSecurityModule")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_HypErc *HypErcSession) InterchainSecurityModule() (common.Address, error) {
	return _HypErc.Contract.InterchainSecurityModule(&_HypErc.CallOpts)
}

// InterchainSecurityModule is a free data retrieval call binding the contract method 0xde523cf3.
//
// Solidity: function interchainSecurityModule() view returns(address)
func (_HypErc *HypErcCallerSession) InterchainSecurityModule() (common.Address, error) {
	return _HypErc.Contract.InterchainSecurityModule(&_HypErc.CallOpts)
}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_HypErc *HypErcCaller) LocalDomain(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "localDomain")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_HypErc *HypErcSession) LocalDomain() (uint32, error) {
	return _HypErc.Contract.LocalDomain(&_HypErc.CallOpts)
}

// LocalDomain is a free data retrieval call binding the contract method 0x8d3638f4.
//
// Solidity: function localDomain() view returns(uint32)
func (_HypErc *HypErcCallerSession) LocalDomain() (uint32, error) {
	return _HypErc.Contract.LocalDomain(&_HypErc.CallOpts)
}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_HypErc *HypErcCaller) Mailbox(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "mailbox")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_HypErc *HypErcSession) Mailbox() (common.Address, error) {
	return _HypErc.Contract.Mailbox(&_HypErc.CallOpts)
}

// Mailbox is a free data retrieval call binding the contract method 0xd5438eae.
//
// Solidity: function mailbox() view returns(address)
func (_HypErc *HypErcCallerSession) Mailbox() (common.Address, error) {
	return _HypErc.Contract.Mailbox(&_HypErc.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_HypErc *HypErcCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_HypErc *HypErcSession) Owner() (common.Address, error) {
	return _HypErc.Contract.Owner(&_HypErc.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_HypErc *HypErcCallerSession) Owner() (common.Address, error) {
	return _HypErc.Contract.Owner(&_HypErc.CallOpts)
}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xf2ed8c53.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain) view returns(uint256 _gasPayment)
func (_HypErc *HypErcCaller) QuoteGasPayment(opts *bind.CallOpts, _destinationDomain uint32) (*big.Int, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "quoteGasPayment", _destinationDomain)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xf2ed8c53.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain) view returns(uint256 _gasPayment)
func (_HypErc *HypErcSession) QuoteGasPayment(_destinationDomain uint32) (*big.Int, error) {
	return _HypErc.Contract.QuoteGasPayment(&_HypErc.CallOpts, _destinationDomain)
}

// QuoteGasPayment is a free data retrieval call binding the contract method 0xf2ed8c53.
//
// Solidity: function quoteGasPayment(uint32 _destinationDomain) view returns(uint256 _gasPayment)
func (_HypErc *HypErcCallerSession) QuoteGasPayment(_destinationDomain uint32) (*big.Int, error) {
	return _HypErc.Contract.QuoteGasPayment(&_HypErc.CallOpts, _destinationDomain)
}

// Routers is a free data retrieval call binding the contract method 0x2ead72f6.
//
// Solidity: function routers(uint32 _domain) view returns(bytes32)
func (_HypErc *HypErcCaller) Routers(opts *bind.CallOpts, _domain uint32) ([32]byte, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "routers", _domain)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// Routers is a free data retrieval call binding the contract method 0x2ead72f6.
//
// Solidity: function routers(uint32 _domain) view returns(bytes32)
func (_HypErc *HypErcSession) Routers(_domain uint32) ([32]byte, error) {
	return _HypErc.Contract.Routers(&_HypErc.CallOpts, _domain)
}

// Routers is a free data retrieval call binding the contract method 0x2ead72f6.
//
// Solidity: function routers(uint32 _domain) view returns(bytes32)
func (_HypErc *HypErcCallerSession) Routers(_domain uint32) ([32]byte, error) {
	return _HypErc.Contract.Routers(&_HypErc.CallOpts, _domain)
}

// WrappedToken is a free data retrieval call binding the contract method 0x996c6cc3.
//
// Solidity: function wrappedToken() view returns(address)
func (_HypErc *HypErcCaller) WrappedToken(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _HypErc.contract.Call(opts, &out, "wrappedToken")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// WrappedToken is a free data retrieval call binding the contract method 0x996c6cc3.
//
// Solidity: function wrappedToken() view returns(address)
func (_HypErc *HypErcSession) WrappedToken() (common.Address, error) {
	return _HypErc.Contract.WrappedToken(&_HypErc.CallOpts)
}

// WrappedToken is a free data retrieval call binding the contract method 0x996c6cc3.
//
// Solidity: function wrappedToken() view returns(address)
func (_HypErc *HypErcCallerSession) WrappedToken() (common.Address, error) {
	return _HypErc.Contract.WrappedToken(&_HypErc.CallOpts)
}

// EnrollRemoteRouter is a paid mutator transaction binding the contract method 0xb49c53a7.
//
// Solidity: function enrollRemoteRouter(uint32 _domain, bytes32 _router) returns()
func (_HypErc *HypErcTransactor) EnrollRemoteRouter(opts *bind.TransactOpts, _domain uint32, _router [32]byte) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "enrollRemoteRouter", _domain, _router)
}

// EnrollRemoteRouter is a paid mutator transaction binding the contract method 0xb49c53a7.
//
// Solidity: function enrollRemoteRouter(uint32 _domain, bytes32 _router) returns()
func (_HypErc *HypErcSession) EnrollRemoteRouter(_domain uint32, _router [32]byte) (*types.Transaction, error) {
	return _HypErc.Contract.EnrollRemoteRouter(&_HypErc.TransactOpts, _domain, _router)
}

// EnrollRemoteRouter is a paid mutator transaction binding the contract method 0xb49c53a7.
//
// Solidity: function enrollRemoteRouter(uint32 _domain, bytes32 _router) returns()
func (_HypErc *HypErcTransactorSession) EnrollRemoteRouter(_domain uint32, _router [32]byte) (*types.Transaction, error) {
	return _HypErc.Contract.EnrollRemoteRouter(&_HypErc.TransactOpts, _domain, _router)
}

// EnrollRemoteRouters is a paid mutator transaction binding the contract method 0xe9198bf9.
//
// Solidity: function enrollRemoteRouters(uint32[] _domains, bytes32[] _addresses) returns()
func (_HypErc *HypErcTransactor) EnrollRemoteRouters(opts *bind.TransactOpts, _domains []uint32, _addresses [][32]byte) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "enrollRemoteRouters", _domains, _addresses)
}

// EnrollRemoteRouters is a paid mutator transaction binding the contract method 0xe9198bf9.
//
// Solidity: function enrollRemoteRouters(uint32[] _domains, bytes32[] _addresses) returns()
func (_HypErc *HypErcSession) EnrollRemoteRouters(_domains []uint32, _addresses [][32]byte) (*types.Transaction, error) {
	return _HypErc.Contract.EnrollRemoteRouters(&_HypErc.TransactOpts, _domains, _addresses)
}

// EnrollRemoteRouters is a paid mutator transaction binding the contract method 0xe9198bf9.
//
// Solidity: function enrollRemoteRouters(uint32[] _domains, bytes32[] _addresses) returns()
func (_HypErc *HypErcTransactorSession) EnrollRemoteRouters(_domains []uint32, _addresses [][32]byte) (*types.Transaction, error) {
	return _HypErc.Contract.EnrollRemoteRouters(&_HypErc.TransactOpts, _domains, _addresses)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_HypErc *HypErcTransactor) Handle(opts *bind.TransactOpts, _origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "handle", _origin, _sender, _message)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_HypErc *HypErcSession) Handle(_origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _HypErc.Contract.Handle(&_HypErc.TransactOpts, _origin, _sender, _message)
}

// Handle is a paid mutator transaction binding the contract method 0x56d5d475.
//
// Solidity: function handle(uint32 _origin, bytes32 _sender, bytes _message) payable returns()
func (_HypErc *HypErcTransactorSession) Handle(_origin uint32, _sender [32]byte, _message []byte) (*types.Transaction, error) {
	return _HypErc.Contract.Handle(&_HypErc.TransactOpts, _origin, _sender, _message)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _hook, address _interchainSecurityModule, address _owner) returns()
func (_HypErc *HypErcTransactor) Initialize(opts *bind.TransactOpts, _hook common.Address, _interchainSecurityModule common.Address, _owner common.Address) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "initialize", _hook, _interchainSecurityModule, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _hook, address _interchainSecurityModule, address _owner) returns()
func (_HypErc *HypErcSession) Initialize(_hook common.Address, _interchainSecurityModule common.Address, _owner common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.Initialize(&_HypErc.TransactOpts, _hook, _interchainSecurityModule, _owner)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _hook, address _interchainSecurityModule, address _owner) returns()
func (_HypErc *HypErcTransactorSession) Initialize(_hook common.Address, _interchainSecurityModule common.Address, _owner common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.Initialize(&_HypErc.TransactOpts, _hook, _interchainSecurityModule, _owner)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_HypErc *HypErcTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_HypErc *HypErcSession) RenounceOwnership() (*types.Transaction, error) {
	return _HypErc.Contract.RenounceOwnership(&_HypErc.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_HypErc *HypErcTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _HypErc.Contract.RenounceOwnership(&_HypErc.TransactOpts)
}

// SetDestinationGas is a paid mutator transaction binding the contract method 0x49d462ef.
//
// Solidity: function setDestinationGas(uint32 domain, uint256 gas) returns()
func (_HypErc *HypErcTransactor) SetDestinationGas(opts *bind.TransactOpts, domain uint32, gas *big.Int) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "setDestinationGas", domain, gas)
}

// SetDestinationGas is a paid mutator transaction binding the contract method 0x49d462ef.
//
// Solidity: function setDestinationGas(uint32 domain, uint256 gas) returns()
func (_HypErc *HypErcSession) SetDestinationGas(domain uint32, gas *big.Int) (*types.Transaction, error) {
	return _HypErc.Contract.SetDestinationGas(&_HypErc.TransactOpts, domain, gas)
}

// SetDestinationGas is a paid mutator transaction binding the contract method 0x49d462ef.
//
// Solidity: function setDestinationGas(uint32 domain, uint256 gas) returns()
func (_HypErc *HypErcTransactorSession) SetDestinationGas(domain uint32, gas *big.Int) (*types.Transaction, error) {
	return _HypErc.Contract.SetDestinationGas(&_HypErc.TransactOpts, domain, gas)
}

// SetDestinationGas0 is a paid mutator transaction binding the contract method 0xb1bd6436.
//
// Solidity: function setDestinationGas((uint32,uint256)[] gasConfigs) returns()
func (_HypErc *HypErcTransactor) SetDestinationGas0(opts *bind.TransactOpts, gasConfigs []GasRouterGasRouterConfig) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "setDestinationGas0", gasConfigs)
}

// SetDestinationGas0 is a paid mutator transaction binding the contract method 0xb1bd6436.
//
// Solidity: function setDestinationGas((uint32,uint256)[] gasConfigs) returns()
func (_HypErc *HypErcSession) SetDestinationGas0(gasConfigs []GasRouterGasRouterConfig) (*types.Transaction, error) {
	return _HypErc.Contract.SetDestinationGas0(&_HypErc.TransactOpts, gasConfigs)
}

// SetDestinationGas0 is a paid mutator transaction binding the contract method 0xb1bd6436.
//
// Solidity: function setDestinationGas((uint32,uint256)[] gasConfigs) returns()
func (_HypErc *HypErcTransactorSession) SetDestinationGas0(gasConfigs []GasRouterGasRouterConfig) (*types.Transaction, error) {
	return _HypErc.Contract.SetDestinationGas0(&_HypErc.TransactOpts, gasConfigs)
}

// SetHook is a paid mutator transaction binding the contract method 0x3dfd3873.
//
// Solidity: function setHook(address _hook) returns()
func (_HypErc *HypErcTransactor) SetHook(opts *bind.TransactOpts, _hook common.Address) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "setHook", _hook)
}

// SetHook is a paid mutator transaction binding the contract method 0x3dfd3873.
//
// Solidity: function setHook(address _hook) returns()
func (_HypErc *HypErcSession) SetHook(_hook common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.SetHook(&_HypErc.TransactOpts, _hook)
}

// SetHook is a paid mutator transaction binding the contract method 0x3dfd3873.
//
// Solidity: function setHook(address _hook) returns()
func (_HypErc *HypErcTransactorSession) SetHook(_hook common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.SetHook(&_HypErc.TransactOpts, _hook)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _module) returns()
func (_HypErc *HypErcTransactor) SetInterchainSecurityModule(opts *bind.TransactOpts, _module common.Address) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "setInterchainSecurityModule", _module)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _module) returns()
func (_HypErc *HypErcSession) SetInterchainSecurityModule(_module common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.SetInterchainSecurityModule(&_HypErc.TransactOpts, _module)
}

// SetInterchainSecurityModule is a paid mutator transaction binding the contract method 0x0e72cc06.
//
// Solidity: function setInterchainSecurityModule(address _module) returns()
func (_HypErc *HypErcTransactorSession) SetInterchainSecurityModule(_module common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.SetInterchainSecurityModule(&_HypErc.TransactOpts, _module)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_HypErc *HypErcTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_HypErc *HypErcSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.TransferOwnership(&_HypErc.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_HypErc *HypErcTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.TransferOwnership(&_HypErc.TransactOpts, newOwner)
}

// TransferRemote is a paid mutator transaction binding the contract method 0x51debffc.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId, bytes _hookMetadata, address _hook) payable returns(bytes32 messageId)
func (_HypErc *HypErcTransactor) TransferRemote(opts *bind.TransactOpts, _destination uint32, _recipient [32]byte, _amountOrId *big.Int, _hookMetadata []byte, _hook common.Address) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "transferRemote", _destination, _recipient, _amountOrId, _hookMetadata, _hook)
}

// TransferRemote is a paid mutator transaction binding the contract method 0x51debffc.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId, bytes _hookMetadata, address _hook) payable returns(bytes32 messageId)
func (_HypErc *HypErcSession) TransferRemote(_destination uint32, _recipient [32]byte, _amountOrId *big.Int, _hookMetadata []byte, _hook common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.TransferRemote(&_HypErc.TransactOpts, _destination, _recipient, _amountOrId, _hookMetadata, _hook)
}

// TransferRemote is a paid mutator transaction binding the contract method 0x51debffc.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId, bytes _hookMetadata, address _hook) payable returns(bytes32 messageId)
func (_HypErc *HypErcTransactorSession) TransferRemote(_destination uint32, _recipient [32]byte, _amountOrId *big.Int, _hookMetadata []byte, _hook common.Address) (*types.Transaction, error) {
	return _HypErc.Contract.TransferRemote(&_HypErc.TransactOpts, _destination, _recipient, _amountOrId, _hookMetadata, _hook)
}

// TransferRemote0 is a paid mutator transaction binding the contract method 0x81b4e8b4.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId) payable returns(bytes32 messageId)
func (_HypErc *HypErcTransactor) TransferRemote0(opts *bind.TransactOpts, _destination uint32, _recipient [32]byte, _amountOrId *big.Int) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "transferRemote0", _destination, _recipient, _amountOrId)
}

// TransferRemote0 is a paid mutator transaction binding the contract method 0x81b4e8b4.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId) payable returns(bytes32 messageId)
func (_HypErc *HypErcSession) TransferRemote0(_destination uint32, _recipient [32]byte, _amountOrId *big.Int) (*types.Transaction, error) {
	return _HypErc.Contract.TransferRemote0(&_HypErc.TransactOpts, _destination, _recipient, _amountOrId)
}

// TransferRemote0 is a paid mutator transaction binding the contract method 0x81b4e8b4.
//
// Solidity: function transferRemote(uint32 _destination, bytes32 _recipient, uint256 _amountOrId) payable returns(bytes32 messageId)
func (_HypErc *HypErcTransactorSession) TransferRemote0(_destination uint32, _recipient [32]byte, _amountOrId *big.Int) (*types.Transaction, error) {
	return _HypErc.Contract.TransferRemote0(&_HypErc.TransactOpts, _destination, _recipient, _amountOrId)
}

// UnenrollRemoteRouter is a paid mutator transaction binding the contract method 0xefae508a.
//
// Solidity: function unenrollRemoteRouter(uint32 _domain) returns()
func (_HypErc *HypErcTransactor) UnenrollRemoteRouter(opts *bind.TransactOpts, _domain uint32) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "unenrollRemoteRouter", _domain)
}

// UnenrollRemoteRouter is a paid mutator transaction binding the contract method 0xefae508a.
//
// Solidity: function unenrollRemoteRouter(uint32 _domain) returns()
func (_HypErc *HypErcSession) UnenrollRemoteRouter(_domain uint32) (*types.Transaction, error) {
	return _HypErc.Contract.UnenrollRemoteRouter(&_HypErc.TransactOpts, _domain)
}

// UnenrollRemoteRouter is a paid mutator transaction binding the contract method 0xefae508a.
//
// Solidity: function unenrollRemoteRouter(uint32 _domain) returns()
func (_HypErc *HypErcTransactorSession) UnenrollRemoteRouter(_domain uint32) (*types.Transaction, error) {
	return _HypErc.Contract.UnenrollRemoteRouter(&_HypErc.TransactOpts, _domain)
}

// UnenrollRemoteRouters is a paid mutator transaction binding the contract method 0x71a15b38.
//
// Solidity: function unenrollRemoteRouters(uint32[] _domains) returns()
func (_HypErc *HypErcTransactor) UnenrollRemoteRouters(opts *bind.TransactOpts, _domains []uint32) (*types.Transaction, error) {
	return _HypErc.contract.Transact(opts, "unenrollRemoteRouters", _domains)
}

// UnenrollRemoteRouters is a paid mutator transaction binding the contract method 0x71a15b38.
//
// Solidity: function unenrollRemoteRouters(uint32[] _domains) returns()
func (_HypErc *HypErcSession) UnenrollRemoteRouters(_domains []uint32) (*types.Transaction, error) {
	return _HypErc.Contract.UnenrollRemoteRouters(&_HypErc.TransactOpts, _domains)
}

// UnenrollRemoteRouters is a paid mutator transaction binding the contract method 0x71a15b38.
//
// Solidity: function unenrollRemoteRouters(uint32[] _domains) returns()
func (_HypErc *HypErcTransactorSession) UnenrollRemoteRouters(_domains []uint32) (*types.Transaction, error) {
	return _HypErc.Contract.UnenrollRemoteRouters(&_HypErc.TransactOpts, _domains)
}

// HypErcDepositTokenIterator is returned from FilterDepositToken and is used to iterate over the raw logs and unpacked data for DepositToken events raised by the HypErc contract.
type HypErcDepositTokenIterator struct {
	Event *HypErcDepositToken // Event containing the contract specifics and raw log

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
func (it *HypErcDepositTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HypErcDepositToken)
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
		it.Event = new(HypErcDepositToken)
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
func (it *HypErcDepositTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HypErcDepositTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HypErcDepositToken represents a DepositToken event raised by the HypErc contract.
type HypErcDepositToken struct {
	MessageId    [32]byte
	SubaccountId [32]byte
	Amount       *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterDepositToken is a free log retrieval operation binding the contract event 0xfec3f2a1e8041e26666ae560885ebd9c5f856494b9fabe7e39473da2cdd77a7e.
//
// Solidity: event DepositToken(bytes32 indexed messageId, bytes32 indexed subaccountId, uint256 indexed amount)
func (_HypErc *HypErcFilterer) FilterDepositToken(opts *bind.FilterOpts, messageId [][32]byte, subaccountId [][32]byte, amount []*big.Int) (*HypErcDepositTokenIterator, error) {

	var messageIdRule []interface{}
	for _, messageIdItem := range messageId {
		messageIdRule = append(messageIdRule, messageIdItem)
	}
	var subaccountIdRule []interface{}
	for _, subaccountIdItem := range subaccountId {
		subaccountIdRule = append(subaccountIdRule, subaccountIdItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _HypErc.contract.FilterLogs(opts, "DepositToken", messageIdRule, subaccountIdRule, amountRule)
	if err != nil {
		return nil, err
	}
	return &HypErcDepositTokenIterator{contract: _HypErc.contract, event: "DepositToken", logs: logs, sub: sub}, nil
}

// WatchDepositToken is a free log subscription operation binding the contract event 0xfec3f2a1e8041e26666ae560885ebd9c5f856494b9fabe7e39473da2cdd77a7e.
//
// Solidity: event DepositToken(bytes32 indexed messageId, bytes32 indexed subaccountId, uint256 indexed amount)
func (_HypErc *HypErcFilterer) WatchDepositToken(opts *bind.WatchOpts, sink chan<- *HypErcDepositToken, messageId [][32]byte, subaccountId [][32]byte, amount []*big.Int) (event.Subscription, error) {

	var messageIdRule []interface{}
	for _, messageIdItem := range messageId {
		messageIdRule = append(messageIdRule, messageIdItem)
	}
	var subaccountIdRule []interface{}
	for _, subaccountIdItem := range subaccountId {
		subaccountIdRule = append(subaccountIdRule, subaccountIdItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _HypErc.contract.WatchLogs(opts, "DepositToken", messageIdRule, subaccountIdRule, amountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HypErcDepositToken)
				if err := _HypErc.contract.UnpackLog(event, "DepositToken", log); err != nil {
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

// ParseDepositToken is a log parse operation binding the contract event 0xfec3f2a1e8041e26666ae560885ebd9c5f856494b9fabe7e39473da2cdd77a7e.
//
// Solidity: event DepositToken(bytes32 indexed messageId, bytes32 indexed subaccountId, uint256 indexed amount)
func (_HypErc *HypErcFilterer) ParseDepositToken(log types.Log) (*HypErcDepositToken, error) {
	event := new(HypErcDepositToken)
	if err := _HypErc.contract.UnpackLog(event, "DepositToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HypErcInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the HypErc contract.
type HypErcInitializedIterator struct {
	Event *HypErcInitialized // Event containing the contract specifics and raw log

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
func (it *HypErcInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HypErcInitialized)
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
		it.Event = new(HypErcInitialized)
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
func (it *HypErcInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HypErcInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HypErcInitialized represents a Initialized event raised by the HypErc contract.
type HypErcInitialized struct {
	Version uint8
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_HypErc *HypErcFilterer) FilterInitialized(opts *bind.FilterOpts) (*HypErcInitializedIterator, error) {

	logs, sub, err := _HypErc.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &HypErcInitializedIterator{contract: _HypErc.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_HypErc *HypErcFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *HypErcInitialized) (event.Subscription, error) {

	logs, sub, err := _HypErc.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HypErcInitialized)
				if err := _HypErc.contract.UnpackLog(event, "Initialized", log); err != nil {
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

// ParseInitialized is a log parse operation binding the contract event 0x7f26b83ff96e1f2b6a682f133852f6798a09c465da95921460cefb3847402498.
//
// Solidity: event Initialized(uint8 version)
func (_HypErc *HypErcFilterer) ParseInitialized(log types.Log) (*HypErcInitialized, error) {
	event := new(HypErcInitialized)
	if err := _HypErc.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HypErcOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the HypErc contract.
type HypErcOwnershipTransferredIterator struct {
	Event *HypErcOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *HypErcOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HypErcOwnershipTransferred)
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
		it.Event = new(HypErcOwnershipTransferred)
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
func (it *HypErcOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HypErcOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HypErcOwnershipTransferred represents a OwnershipTransferred event raised by the HypErc contract.
type HypErcOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_HypErc *HypErcFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*HypErcOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _HypErc.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &HypErcOwnershipTransferredIterator{contract: _HypErc.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_HypErc *HypErcFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *HypErcOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _HypErc.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HypErcOwnershipTransferred)
				if err := _HypErc.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_HypErc *HypErcFilterer) ParseOwnershipTransferred(log types.Log) (*HypErcOwnershipTransferred, error) {
	event := new(HypErcOwnershipTransferred)
	if err := _HypErc.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HypErcReceivedTransferRemoteIterator is returned from FilterReceivedTransferRemote and is used to iterate over the raw logs and unpacked data for ReceivedTransferRemote events raised by the HypErc contract.
type HypErcReceivedTransferRemoteIterator struct {
	Event *HypErcReceivedTransferRemote // Event containing the contract specifics and raw log

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
func (it *HypErcReceivedTransferRemoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HypErcReceivedTransferRemote)
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
		it.Event = new(HypErcReceivedTransferRemote)
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
func (it *HypErcReceivedTransferRemoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HypErcReceivedTransferRemoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HypErcReceivedTransferRemote represents a ReceivedTransferRemote event raised by the HypErc contract.
type HypErcReceivedTransferRemote struct {
	Origin    uint32
	Recipient [32]byte
	Amount    *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterReceivedTransferRemote is a free log retrieval operation binding the contract event 0xba20947a325f450d232530e5f5fce293e7963499d5309a07cee84a269f2f15a6.
//
// Solidity: event ReceivedTransferRemote(uint32 indexed origin, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) FilterReceivedTransferRemote(opts *bind.FilterOpts, origin []uint32, recipient [][32]byte) (*HypErcReceivedTransferRemoteIterator, error) {

	var originRule []interface{}
	for _, originItem := range origin {
		originRule = append(originRule, originItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HypErc.contract.FilterLogs(opts, "ReceivedTransferRemote", originRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HypErcReceivedTransferRemoteIterator{contract: _HypErc.contract, event: "ReceivedTransferRemote", logs: logs, sub: sub}, nil
}

// WatchReceivedTransferRemote is a free log subscription operation binding the contract event 0xba20947a325f450d232530e5f5fce293e7963499d5309a07cee84a269f2f15a6.
//
// Solidity: event ReceivedTransferRemote(uint32 indexed origin, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) WatchReceivedTransferRemote(opts *bind.WatchOpts, sink chan<- *HypErcReceivedTransferRemote, origin []uint32, recipient [][32]byte) (event.Subscription, error) {

	var originRule []interface{}
	for _, originItem := range origin {
		originRule = append(originRule, originItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HypErc.contract.WatchLogs(opts, "ReceivedTransferRemote", originRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HypErcReceivedTransferRemote)
				if err := _HypErc.contract.UnpackLog(event, "ReceivedTransferRemote", log); err != nil {
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

// ParseReceivedTransferRemote is a log parse operation binding the contract event 0xba20947a325f450d232530e5f5fce293e7963499d5309a07cee84a269f2f15a6.
//
// Solidity: event ReceivedTransferRemote(uint32 indexed origin, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) ParseReceivedTransferRemote(log types.Log) (*HypErcReceivedTransferRemote, error) {
	event := new(HypErcReceivedTransferRemote)
	if err := _HypErc.contract.UnpackLog(event, "ReceivedTransferRemote", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// HypErcSentTransferRemoteIterator is returned from FilterSentTransferRemote and is used to iterate over the raw logs and unpacked data for SentTransferRemote events raised by the HypErc contract.
type HypErcSentTransferRemoteIterator struct {
	Event *HypErcSentTransferRemote // Event containing the contract specifics and raw log

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
func (it *HypErcSentTransferRemoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(HypErcSentTransferRemote)
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
		it.Event = new(HypErcSentTransferRemote)
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
func (it *HypErcSentTransferRemoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *HypErcSentTransferRemoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// HypErcSentTransferRemote represents a SentTransferRemote event raised by the HypErc contract.
type HypErcSentTransferRemote struct {
	Destination uint32
	Recipient   [32]byte
	Amount      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterSentTransferRemote is a free log retrieval operation binding the contract event 0xd229aacb94204188fe8042965fa6b269c62dc5818b21238779ab64bdd17efeec.
//
// Solidity: event SentTransferRemote(uint32 indexed destination, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) FilterSentTransferRemote(opts *bind.FilterOpts, destination []uint32, recipient [][32]byte) (*HypErcSentTransferRemoteIterator, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HypErc.contract.FilterLogs(opts, "SentTransferRemote", destinationRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return &HypErcSentTransferRemoteIterator{contract: _HypErc.contract, event: "SentTransferRemote", logs: logs, sub: sub}, nil
}

// WatchSentTransferRemote is a free log subscription operation binding the contract event 0xd229aacb94204188fe8042965fa6b269c62dc5818b21238779ab64bdd17efeec.
//
// Solidity: event SentTransferRemote(uint32 indexed destination, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) WatchSentTransferRemote(opts *bind.WatchOpts, sink chan<- *HypErcSentTransferRemote, destination []uint32, recipient [][32]byte) (event.Subscription, error) {

	var destinationRule []interface{}
	for _, destinationItem := range destination {
		destinationRule = append(destinationRule, destinationItem)
	}
	var recipientRule []interface{}
	for _, recipientItem := range recipient {
		recipientRule = append(recipientRule, recipientItem)
	}

	logs, sub, err := _HypErc.contract.WatchLogs(opts, "SentTransferRemote", destinationRule, recipientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(HypErcSentTransferRemote)
				if err := _HypErc.contract.UnpackLog(event, "SentTransferRemote", log); err != nil {
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

// ParseSentTransferRemote is a log parse operation binding the contract event 0xd229aacb94204188fe8042965fa6b269c62dc5818b21238779ab64bdd17efeec.
//
// Solidity: event SentTransferRemote(uint32 indexed destination, bytes32 indexed recipient, uint256 amount)
func (_HypErc *HypErcFilterer) ParseSentTransferRemote(log types.Log) (*HypErcSentTransferRemote, error) {
	event := new(HypErcSentTransferRemote)
	if err := _HypErc.contract.UnpackLog(event, "SentTransferRemote", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
