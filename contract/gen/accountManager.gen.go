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

// AccountsRegister is an auto generated low-level Go binding around an user-defined struct.
type AccountsRegister struct {
	SubAccountId    [32]byte
	UserAddress     common.Address
	SessionKey      common.Address
	ExpiryTimeStamp *big.Int
	Nonce           *big.Int
	ChainId         *big.Int
}

// AccountManagerMetaData contains all meta data concerning the AccountManager contract.
var AccountManagerMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"ECDSAInvalidSignature\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"ECDSAInvalidSignatureLength\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"s\",\"type\":\"bytes32\"}],\"name\":\"ECDSAInvalidSignatureS\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"},{\"indexed\":false,\"internalType\":\"address\",\"name\":\"sessionKey\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint128\",\"name\":\"expiryTimeStamp\",\"type\":\"uint128\"}],\"name\":\"SessionKeyCreated\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"accountId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenAmount\",\"type\":\"uint256\"}],\"name\":\"decreaseTokenBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"}],\"name\":\"fetchNonce\",\"outputs\":[{\"internalType\":\"uint128\",\"name\":\"\",\"type\":\"uint128\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getDomainSeparator\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"userAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sessionKey\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"expiryTimeStamp\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"nonce\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"internalType\":\"structAccounts.Register\",\"name\":\"sessionRequest\",\"type\":\"tuple\"}],\"name\":\"getHashStruct\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"accountId\",\"type\":\"bytes32\"}],\"name\":\"getMasterWallet\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"accountId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"tokenAmount\",\"type\":\"uint256\"}],\"name\":\"increaseTokenBalance\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"userAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"sessionKey\",\"type\":\"address\"},{\"internalType\":\"uint128\",\"name\":\"expiryTimeStamp\",\"type\":\"uint128\"},{\"internalType\":\"uint128\",\"name\":\"nonce\",\"type\":\"uint128\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"internalType\":\"structAccounts.Register\",\"name\":\"sessionRequest\",\"type\":\"tuple\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"bytes\",\"name\":\"masterSignature\",\"type\":\"bytes\"}],\"name\":\"registerSessionKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"},{\"internalType\":\"uint128\",\"name\":\"nonce\",\"type\":\"uint128\"}],\"name\":\"setNonce\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"testTxn\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"accountId\",\"type\":\"bytes32\"},{\"internalType\":\"address\",\"name\":\"tokenAddress\",\"type\":\"address\"}],\"name\":\"tokenBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sessionKey\",\"type\":\"address\"},{\"internalType\":\"bytes32\",\"name\":\"subAccountId\",\"type\":\"bytes32\"}],\"name\":\"validateSessionkey\",\"outputs\":[],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"hashStruct\",\"type\":\"bytes32\"},{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"},{\"internalType\":\"bytes\",\"name\":\"signature\",\"type\":\"bytes\"},{\"internalType\":\"address\",\"name\":\"signer\",\"type\":\"address\"}],\"name\":\"verifySignature\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
}

// AccountManagerABI is the input ABI used to generate the binding from.
// Deprecated: Use AccountManagerMetaData.ABI instead.
var AccountManagerABI = AccountManagerMetaData.ABI

// AccountManager is an auto generated Go binding around an Ethereum contract.
type AccountManager struct {
	AccountManagerCaller     // Read-only binding to the contract
	AccountManagerTransactor // Write-only binding to the contract
	AccountManagerFilterer   // Log filterer for contract events
}

// AccountManagerCaller is an auto generated read-only Go binding around an Ethereum contract.
type AccountManagerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountManagerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AccountManagerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountManagerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AccountManagerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountManagerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AccountManagerSession struct {
	Contract     *AccountManager   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AccountManagerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AccountManagerCallerSession struct {
	Contract *AccountManagerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// AccountManagerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AccountManagerTransactorSession struct {
	Contract     *AccountManagerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// AccountManagerRaw is an auto generated low-level Go binding around an Ethereum contract.
type AccountManagerRaw struct {
	Contract *AccountManager // Generic contract binding to access the raw methods on
}

// AccountManagerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AccountManagerCallerRaw struct {
	Contract *AccountManagerCaller // Generic read-only contract binding to access the raw methods on
}

// AccountManagerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AccountManagerTransactorRaw struct {
	Contract *AccountManagerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAccountManager creates a new instance of AccountManager, bound to a specific deployed contract.
func NewAccountManager(address common.Address, backend bind.ContractBackend) (*AccountManager, error) {
	contract, err := bindAccountManager(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &AccountManager{AccountManagerCaller: AccountManagerCaller{contract: contract}, AccountManagerTransactor: AccountManagerTransactor{contract: contract}, AccountManagerFilterer: AccountManagerFilterer{contract: contract}}, nil
}

// NewAccountManagerCaller creates a new read-only instance of AccountManager, bound to a specific deployed contract.
func NewAccountManagerCaller(address common.Address, caller bind.ContractCaller) (*AccountManagerCaller, error) {
	contract, err := bindAccountManager(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AccountManagerCaller{contract: contract}, nil
}

// NewAccountManagerTransactor creates a new write-only instance of AccountManager, bound to a specific deployed contract.
func NewAccountManagerTransactor(address common.Address, transactor bind.ContractTransactor) (*AccountManagerTransactor, error) {
	contract, err := bindAccountManager(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AccountManagerTransactor{contract: contract}, nil
}

// NewAccountManagerFilterer creates a new log filterer instance of AccountManager, bound to a specific deployed contract.
func NewAccountManagerFilterer(address common.Address, filterer bind.ContractFilterer) (*AccountManagerFilterer, error) {
	contract, err := bindAccountManager(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AccountManagerFilterer{contract: contract}, nil
}

// bindAccountManager binds a generic wrapper to an already deployed contract.
func bindAccountManager(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := AccountManagerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AccountManager *AccountManagerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AccountManager.Contract.AccountManagerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AccountManager *AccountManagerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountManager.Contract.AccountManagerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AccountManager *AccountManagerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AccountManager.Contract.AccountManagerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_AccountManager *AccountManagerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _AccountManager.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_AccountManager *AccountManagerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountManager.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_AccountManager *AccountManagerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _AccountManager.Contract.contract.Transact(opts, method, params...)
}

// FetchNonce is a free data retrieval call binding the contract method 0xde0af820.
//
// Solidity: function fetchNonce(bytes32 subAccountId) view returns(uint128)
func (_AccountManager *AccountManagerCaller) FetchNonce(opts *bind.CallOpts, subAccountId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "fetchNonce", subAccountId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// FetchNonce is a free data retrieval call binding the contract method 0xde0af820.
//
// Solidity: function fetchNonce(bytes32 subAccountId) view returns(uint128)
func (_AccountManager *AccountManagerSession) FetchNonce(subAccountId [32]byte) (*big.Int, error) {
	return _AccountManager.Contract.FetchNonce(&_AccountManager.CallOpts, subAccountId)
}

// FetchNonce is a free data retrieval call binding the contract method 0xde0af820.
//
// Solidity: function fetchNonce(bytes32 subAccountId) view returns(uint128)
func (_AccountManager *AccountManagerCallerSession) FetchNonce(subAccountId [32]byte) (*big.Int, error) {
	return _AccountManager.Contract.FetchNonce(&_AccountManager.CallOpts, subAccountId)
}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_AccountManager *AccountManagerCaller) GetDomainSeparator(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "getDomainSeparator")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_AccountManager *AccountManagerSession) GetDomainSeparator() ([32]byte, error) {
	return _AccountManager.Contract.GetDomainSeparator(&_AccountManager.CallOpts)
}

// GetDomainSeparator is a free data retrieval call binding the contract method 0xed24911d.
//
// Solidity: function getDomainSeparator() view returns(bytes32)
func (_AccountManager *AccountManagerCallerSession) GetDomainSeparator() ([32]byte, error) {
	return _AccountManager.Contract.GetDomainSeparator(&_AccountManager.CallOpts)
}

// GetHashStruct is a free data retrieval call binding the contract method 0x140e6a43.
//
// Solidity: function getHashStruct((bytes32,address,address,uint128,uint128,uint256) sessionRequest) pure returns(bytes32)
func (_AccountManager *AccountManagerCaller) GetHashStruct(opts *bind.CallOpts, sessionRequest AccountsRegister) ([32]byte, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "getHashStruct", sessionRequest)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetHashStruct is a free data retrieval call binding the contract method 0x140e6a43.
//
// Solidity: function getHashStruct((bytes32,address,address,uint128,uint128,uint256) sessionRequest) pure returns(bytes32)
func (_AccountManager *AccountManagerSession) GetHashStruct(sessionRequest AccountsRegister) ([32]byte, error) {
	return _AccountManager.Contract.GetHashStruct(&_AccountManager.CallOpts, sessionRequest)
}

// GetHashStruct is a free data retrieval call binding the contract method 0x140e6a43.
//
// Solidity: function getHashStruct((bytes32,address,address,uint128,uint128,uint256) sessionRequest) pure returns(bytes32)
func (_AccountManager *AccountManagerCallerSession) GetHashStruct(sessionRequest AccountsRegister) ([32]byte, error) {
	return _AccountManager.Contract.GetHashStruct(&_AccountManager.CallOpts, sessionRequest)
}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 accountId) view returns(address)
func (_AccountManager *AccountManagerCaller) GetMasterWallet(opts *bind.CallOpts, accountId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "getMasterWallet", accountId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 accountId) view returns(address)
func (_AccountManager *AccountManagerSession) GetMasterWallet(accountId [32]byte) (common.Address, error) {
	return _AccountManager.Contract.GetMasterWallet(&_AccountManager.CallOpts, accountId)
}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 accountId) view returns(address)
func (_AccountManager *AccountManagerCallerSession) GetMasterWallet(accountId [32]byte) (common.Address, error) {
	return _AccountManager.Contract.GetMasterWallet(&_AccountManager.CallOpts, accountId)
}

// TokenBalance is a free data retrieval call binding the contract method 0x379f75b4.
//
// Solidity: function tokenBalance(bytes32 accountId, address tokenAddress) view returns(uint256)
func (_AccountManager *AccountManagerCaller) TokenBalance(opts *bind.CallOpts, accountId [32]byte, tokenAddress common.Address) (*big.Int, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "tokenBalance", accountId, tokenAddress)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TokenBalance is a free data retrieval call binding the contract method 0x379f75b4.
//
// Solidity: function tokenBalance(bytes32 accountId, address tokenAddress) view returns(uint256)
func (_AccountManager *AccountManagerSession) TokenBalance(accountId [32]byte, tokenAddress common.Address) (*big.Int, error) {
	return _AccountManager.Contract.TokenBalance(&_AccountManager.CallOpts, accountId, tokenAddress)
}

// TokenBalance is a free data retrieval call binding the contract method 0x379f75b4.
//
// Solidity: function tokenBalance(bytes32 accountId, address tokenAddress) view returns(uint256)
func (_AccountManager *AccountManagerCallerSession) TokenBalance(accountId [32]byte, tokenAddress common.Address) (*big.Int, error) {
	return _AccountManager.Contract.TokenBalance(&_AccountManager.CallOpts, accountId, tokenAddress)
}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_AccountManager *AccountManagerCaller) ValidateSessionkey(opts *bind.CallOpts, sessionKey common.Address, subAccountId [32]byte) error {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "validateSessionkey", sessionKey, subAccountId)

	if err != nil {
		return err
	}

	return err

}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_AccountManager *AccountManagerSession) ValidateSessionkey(sessionKey common.Address, subAccountId [32]byte) error {
	return _AccountManager.Contract.ValidateSessionkey(&_AccountManager.CallOpts, sessionKey, subAccountId)
}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_AccountManager *AccountManagerCallerSession) ValidateSessionkey(sessionKey common.Address, subAccountId [32]byte) error {
	return _AccountManager.Contract.ValidateSessionkey(&_AccountManager.CallOpts, sessionKey, subAccountId)
}

// VerifySignature is a free data retrieval call binding the contract method 0x52af8ac8.
//
// Solidity: function verifySignature(bytes32 hashStruct, uint256 chainId, bytes signature, address signer) view returns(bool)
func (_AccountManager *AccountManagerCaller) VerifySignature(opts *bind.CallOpts, hashStruct [32]byte, chainId *big.Int, signature []byte, signer common.Address) (bool, error) {
	var out []interface{}
	err := _AccountManager.contract.Call(opts, &out, "verifySignature", hashStruct, chainId, signature, signer)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// VerifySignature is a free data retrieval call binding the contract method 0x52af8ac8.
//
// Solidity: function verifySignature(bytes32 hashStruct, uint256 chainId, bytes signature, address signer) view returns(bool)
func (_AccountManager *AccountManagerSession) VerifySignature(hashStruct [32]byte, chainId *big.Int, signature []byte, signer common.Address) (bool, error) {
	return _AccountManager.Contract.VerifySignature(&_AccountManager.CallOpts, hashStruct, chainId, signature, signer)
}

// VerifySignature is a free data retrieval call binding the contract method 0x52af8ac8.
//
// Solidity: function verifySignature(bytes32 hashStruct, uint256 chainId, bytes signature, address signer) view returns(bool)
func (_AccountManager *AccountManagerCallerSession) VerifySignature(hashStruct [32]byte, chainId *big.Int, signature []byte, signer common.Address) (bool, error) {
	return _AccountManager.Contract.VerifySignature(&_AccountManager.CallOpts, hashStruct, chainId, signature, signer)
}

// DecreaseTokenBalance is a paid mutator transaction binding the contract method 0x14a54616.
//
// Solidity: function decreaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerTransactor) DecreaseTokenBalance(opts *bind.TransactOpts, accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.contract.Transact(opts, "decreaseTokenBalance", accountId, tokenAddress, tokenAmount)
}

// DecreaseTokenBalance is a paid mutator transaction binding the contract method 0x14a54616.
//
// Solidity: function decreaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerSession) DecreaseTokenBalance(accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.DecreaseTokenBalance(&_AccountManager.TransactOpts, accountId, tokenAddress, tokenAmount)
}

// DecreaseTokenBalance is a paid mutator transaction binding the contract method 0x14a54616.
//
// Solidity: function decreaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerTransactorSession) DecreaseTokenBalance(accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.DecreaseTokenBalance(&_AccountManager.TransactOpts, accountId, tokenAddress, tokenAmount)
}

// IncreaseTokenBalance is a paid mutator transaction binding the contract method 0x6b106fc5.
//
// Solidity: function increaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerTransactor) IncreaseTokenBalance(opts *bind.TransactOpts, accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.contract.Transact(opts, "increaseTokenBalance", accountId, tokenAddress, tokenAmount)
}

// IncreaseTokenBalance is a paid mutator transaction binding the contract method 0x6b106fc5.
//
// Solidity: function increaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerSession) IncreaseTokenBalance(accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.IncreaseTokenBalance(&_AccountManager.TransactOpts, accountId, tokenAddress, tokenAmount)
}

// IncreaseTokenBalance is a paid mutator transaction binding the contract method 0x6b106fc5.
//
// Solidity: function increaseTokenBalance(bytes32 accountId, address tokenAddress, uint256 tokenAmount) returns()
func (_AccountManager *AccountManagerTransactorSession) IncreaseTokenBalance(accountId [32]byte, tokenAddress common.Address, tokenAmount *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.IncreaseTokenBalance(&_AccountManager.TransactOpts, accountId, tokenAddress, tokenAmount)
}

// RegisterSessionKey is a paid mutator transaction binding the contract method 0xdfe4d427.
//
// Solidity: function registerSessionKey((bytes32,address,address,uint128,uint128,uint256) sessionRequest, bytes signature, bytes masterSignature) returns()
func (_AccountManager *AccountManagerTransactor) RegisterSessionKey(opts *bind.TransactOpts, sessionRequest AccountsRegister, signature []byte, masterSignature []byte) (*types.Transaction, error) {
	return _AccountManager.contract.Transact(opts, "registerSessionKey", sessionRequest, signature, masterSignature)
}

// RegisterSessionKey is a paid mutator transaction binding the contract method 0xdfe4d427.
//
// Solidity: function registerSessionKey((bytes32,address,address,uint128,uint128,uint256) sessionRequest, bytes signature, bytes masterSignature) returns()
func (_AccountManager *AccountManagerSession) RegisterSessionKey(sessionRequest AccountsRegister, signature []byte, masterSignature []byte) (*types.Transaction, error) {
	return _AccountManager.Contract.RegisterSessionKey(&_AccountManager.TransactOpts, sessionRequest, signature, masterSignature)
}

// RegisterSessionKey is a paid mutator transaction binding the contract method 0xdfe4d427.
//
// Solidity: function registerSessionKey((bytes32,address,address,uint128,uint128,uint256) sessionRequest, bytes signature, bytes masterSignature) returns()
func (_AccountManager *AccountManagerTransactorSession) RegisterSessionKey(sessionRequest AccountsRegister, signature []byte, masterSignature []byte) (*types.Transaction, error) {
	return _AccountManager.Contract.RegisterSessionKey(&_AccountManager.TransactOpts, sessionRequest, signature, masterSignature)
}

// SetNonce is a paid mutator transaction binding the contract method 0x3acb7797.
//
// Solidity: function setNonce(bytes32 subAccountId, uint128 nonce) returns()
func (_AccountManager *AccountManagerTransactor) SetNonce(opts *bind.TransactOpts, subAccountId [32]byte, nonce *big.Int) (*types.Transaction, error) {
	return _AccountManager.contract.Transact(opts, "setNonce", subAccountId, nonce)
}

// SetNonce is a paid mutator transaction binding the contract method 0x3acb7797.
//
// Solidity: function setNonce(bytes32 subAccountId, uint128 nonce) returns()
func (_AccountManager *AccountManagerSession) SetNonce(subAccountId [32]byte, nonce *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.SetNonce(&_AccountManager.TransactOpts, subAccountId, nonce)
}

// SetNonce is a paid mutator transaction binding the contract method 0x3acb7797.
//
// Solidity: function setNonce(bytes32 subAccountId, uint128 nonce) returns()
func (_AccountManager *AccountManagerTransactorSession) SetNonce(subAccountId [32]byte, nonce *big.Int) (*types.Transaction, error) {
	return _AccountManager.Contract.SetNonce(&_AccountManager.TransactOpts, subAccountId, nonce)
}

// TestTxn is a paid mutator transaction binding the contract method 0x59d54d57.
//
// Solidity: function testTxn() returns(bool)
func (_AccountManager *AccountManagerTransactor) TestTxn(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _AccountManager.contract.Transact(opts, "testTxn")
}

// TestTxn is a paid mutator transaction binding the contract method 0x59d54d57.
//
// Solidity: function testTxn() returns(bool)
func (_AccountManager *AccountManagerSession) TestTxn() (*types.Transaction, error) {
	return _AccountManager.Contract.TestTxn(&_AccountManager.TransactOpts)
}

// TestTxn is a paid mutator transaction binding the contract method 0x59d54d57.
//
// Solidity: function testTxn() returns(bool)
func (_AccountManager *AccountManagerTransactorSession) TestTxn() (*types.Transaction, error) {
	return _AccountManager.Contract.TestTxn(&_AccountManager.TransactOpts)
}

// AccountManagerSessionKeyCreatedIterator is returned from FilterSessionKeyCreated and is used to iterate over the raw logs and unpacked data for SessionKeyCreated events raised by the AccountManager contract.
type AccountManagerSessionKeyCreatedIterator struct {
	Event *AccountManagerSessionKeyCreated // Event containing the contract specifics and raw log

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
func (it *AccountManagerSessionKeyCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountManagerSessionKeyCreated)
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
		it.Event = new(AccountManagerSessionKeyCreated)
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
func (it *AccountManagerSessionKeyCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountManagerSessionKeyCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountManagerSessionKeyCreated represents a SessionKeyCreated event raised by the AccountManager contract.
type AccountManagerSessionKeyCreated struct {
	SubAccountId    [32]byte
	SessionKey      common.Address
	ExpiryTimeStamp *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSessionKeyCreated is a free log retrieval operation binding the contract event 0xa952475419f5f382d1792a66c04465288e4fdde2479de093488c40ec69ae8e44.
//
// Solidity: event SessionKeyCreated(bytes32 subAccountId, address sessionKey, uint128 expiryTimeStamp)
func (_AccountManager *AccountManagerFilterer) FilterSessionKeyCreated(opts *bind.FilterOpts) (*AccountManagerSessionKeyCreatedIterator, error) {

	logs, sub, err := _AccountManager.contract.FilterLogs(opts, "SessionKeyCreated")
	if err != nil {
		return nil, err
	}
	return &AccountManagerSessionKeyCreatedIterator{contract: _AccountManager.contract, event: "SessionKeyCreated", logs: logs, sub: sub}, nil
}

// WatchSessionKeyCreated is a free log subscription operation binding the contract event 0xa952475419f5f382d1792a66c04465288e4fdde2479de093488c40ec69ae8e44.
//
// Solidity: event SessionKeyCreated(bytes32 subAccountId, address sessionKey, uint128 expiryTimeStamp)
func (_AccountManager *AccountManagerFilterer) WatchSessionKeyCreated(opts *bind.WatchOpts, sink chan<- *AccountManagerSessionKeyCreated) (event.Subscription, error) {

	logs, sub, err := _AccountManager.contract.WatchLogs(opts, "SessionKeyCreated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountManagerSessionKeyCreated)
				if err := _AccountManager.contract.UnpackLog(event, "SessionKeyCreated", log); err != nil {
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

// ParseSessionKeyCreated is a log parse operation binding the contract event 0xa952475419f5f382d1792a66c04465288e4fdde2479de093488c40ec69ae8e44.
//
// Solidity: event SessionKeyCreated(bytes32 subAccountId, address sessionKey, uint128 expiryTimeStamp)
func (_AccountManager *AccountManagerFilterer) ParseSessionKeyCreated(log types.Log) (*AccountManagerSessionKeyCreated, error) {
	event := new(AccountManagerSessionKeyCreated)
	if err := _AccountManager.contract.UnpackLog(event, "SessionKeyCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
