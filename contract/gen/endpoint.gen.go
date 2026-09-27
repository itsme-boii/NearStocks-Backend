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

// IEndpointRegister is an auto generated low-level Go binding around an user-defined struct.
type IEndpointRegister struct {
	SubAccountId    [32]byte
	UserAddress     common.Address
	SessionKey      common.Address
	ExpiryTimeStamp *big.Int
	Nonce           *big.Int
	ChainId         *big.Int
}

// EndpointMetaData contains all meta data concerning the Endpoint contract.
var EndpointMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"_getQuote\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"contractIERC20Base\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"_recordAdminSubaccount\",\"inputs\":[{\"name\":\"sessionRequest\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.Register\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"expiryTimeStamp\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nonce\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"accountInfo\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"bridgeInLogX\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sourceChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"depositCollateral\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"sourceChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"eip712Domain\",\"inputs\":[],\"outputs\":[{\"name\":\"fields\",\"type\":\"bytes1\",\"internalType\":\"bytes1\"},{\"name\":\"name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"version\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"verifyingContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"salt\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"extensions\",\"type\":\"uint256[]\",\"internalType\":\"uint256[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"faucetDeposit\",\"inputs\":[{\"name\":\"productIds\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"},{\"name\":\"subAccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getLinkedSigner\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getMasterWallet\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"masterWallet\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"getNonce\",\"inputs\":[{\"name\":\"_subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getNoncesOfSubaccounts\",\"inputs\":[{\"name\":\"subaccountIds\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"outputs\":[{\"name\":\"nonces\",\"type\":\"uint128[]\",\"internalType\":\"uint128[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getOffchainExchange\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getSequencer\",\"inputs\":[{\"name\":\"_sequencer\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getTime\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint128\",\"internalType\":\"uint128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_sequencer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_offchainExchange\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_clearinghouse\",\"type\":\"address\",\"internalType\":\"contractIClearinghouse\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"nSubmissions\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setSequencer\",\"inputs\":[{\"name\":\"_sequencer\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"submitTransactionsChecked\",\"inputs\":[{\"name\":\"idx\",\"type\":\"uint64\",\"internalType\":\"uint64\"},{\"name\":\"transactions\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"signatures\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"},{\"name\":\"signatures2\",\"type\":\"bytes[]\",\"internalType\":\"bytes[]\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"validateSessionkey\",\"inputs\":[{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withDrawExcessFunds\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawExcessToken\",\"inputs\":[{\"name\":\"token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ClaimStakeRewards\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"rewardAmount\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"ClaimedLogX\",\"inputs\":[{\"name\":\"accountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"int128\",\"indexed\":true,\"internalType\":\"int128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"DepositToken\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"masterWallet\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"sourceChainId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"destinationChainId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"EIP712DomainChanged\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SessionKeyCreated\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"expiryTimeStamp\",\"type\":\"uint128\",\"indexed\":false,\"internalType\":\"uint128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"StakeLogX\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"stakeAmount\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"SubmitTransactions\",\"inputs\":[],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UnstakeLogX\",\"inputs\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"stakeId\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"unstakeAmount\",\"type\":\"uint256\",\"indexed\":true,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignature\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureLength\",\"inputs\":[{\"name\":\"length\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ECDSAInvalidSignatureS\",\"inputs\":[{\"name\":\"s\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}]},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
}

// EndpointABI is the input ABI used to generate the binding from.
// Deprecated: Use EndpointMetaData.ABI instead.
var EndpointABI = EndpointMetaData.ABI

// Endpoint is an auto generated Go binding around an Ethereum contract.
type Endpoint struct {
	EndpointCaller     // Read-only binding to the contract
	EndpointTransactor // Write-only binding to the contract
	EndpointFilterer   // Log filterer for contract events
}

// EndpointCaller is an auto generated read-only Go binding around an Ethereum contract.
type EndpointCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EndpointTransactor is an auto generated write-only Go binding around an Ethereum contract.
type EndpointTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EndpointFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type EndpointFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// EndpointSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type EndpointSession struct {
	Contract     *Endpoint         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// EndpointCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type EndpointCallerSession struct {
	Contract *EndpointCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// EndpointTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type EndpointTransactorSession struct {
	Contract     *EndpointTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// EndpointRaw is an auto generated low-level Go binding around an Ethereum contract.
type EndpointRaw struct {
	Contract *Endpoint // Generic contract binding to access the raw methods on
}

// EndpointCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type EndpointCallerRaw struct {
	Contract *EndpointCaller // Generic read-only contract binding to access the raw methods on
}

// EndpointTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type EndpointTransactorRaw struct {
	Contract *EndpointTransactor // Generic write-only contract binding to access the raw methods on
}

// NewEndpoint creates a new instance of Endpoint, bound to a specific deployed contract.
func NewEndpoint(address common.Address, backend bind.ContractBackend) (*Endpoint, error) {
	contract, err := bindEndpoint(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Endpoint{EndpointCaller: EndpointCaller{contract: contract}, EndpointTransactor: EndpointTransactor{contract: contract}, EndpointFilterer: EndpointFilterer{contract: contract}}, nil
}

// NewEndpointCaller creates a new read-only instance of Endpoint, bound to a specific deployed contract.
func NewEndpointCaller(address common.Address, caller bind.ContractCaller) (*EndpointCaller, error) {
	contract, err := bindEndpoint(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &EndpointCaller{contract: contract}, nil
}

// NewEndpointTransactor creates a new write-only instance of Endpoint, bound to a specific deployed contract.
func NewEndpointTransactor(address common.Address, transactor bind.ContractTransactor) (*EndpointTransactor, error) {
	contract, err := bindEndpoint(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &EndpointTransactor{contract: contract}, nil
}

// NewEndpointFilterer creates a new log filterer instance of Endpoint, bound to a specific deployed contract.
func NewEndpointFilterer(address common.Address, filterer bind.ContractFilterer) (*EndpointFilterer, error) {
	contract, err := bindEndpoint(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &EndpointFilterer{contract: contract}, nil
}

// bindEndpoint binds a generic wrapper to an already deployed contract.
func bindEndpoint(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := EndpointMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Endpoint *EndpointRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Endpoint.Contract.EndpointCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Endpoint *EndpointRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Endpoint.Contract.EndpointTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Endpoint *EndpointRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Endpoint.Contract.EndpointTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Endpoint *EndpointCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Endpoint.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Endpoint *EndpointTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Endpoint.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Endpoint *EndpointTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Endpoint.Contract.contract.Transact(opts, method, params...)
}

// GetQuote is a free data retrieval call binding the contract method 0x0f00648e.
//
// Solidity: function _getQuote() view returns(address)
func (_Endpoint *EndpointCaller) GetQuote(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "_getQuote")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetQuote is a free data retrieval call binding the contract method 0x0f00648e.
//
// Solidity: function _getQuote() view returns(address)
func (_Endpoint *EndpointSession) GetQuote() (common.Address, error) {
	return _Endpoint.Contract.GetQuote(&_Endpoint.CallOpts)
}

// GetQuote is a free data retrieval call binding the contract method 0x0f00648e.
//
// Solidity: function _getQuote() view returns(address)
func (_Endpoint *EndpointCallerSession) GetQuote() (common.Address, error) {
	return _Endpoint.Contract.GetQuote(&_Endpoint.CallOpts)
}

// AccountInfo is a free data retrieval call binding the contract method 0x94aaf7fa.
//
// Solidity: function accountInfo(bytes32 ) view returns(bytes32 subAccountId, address userAddress, uint128 nonce)
func (_Endpoint *EndpointCaller) AccountInfo(opts *bind.CallOpts, arg0 [32]byte) (struct {
	SubAccountId [32]byte
	UserAddress  common.Address
	Nonce        *big.Int
}, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "accountInfo", arg0)

	outstruct := new(struct {
		SubAccountId [32]byte
		UserAddress  common.Address
		Nonce        *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.SubAccountId = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.UserAddress = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Nonce = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// AccountInfo is a free data retrieval call binding the contract method 0x94aaf7fa.
//
// Solidity: function accountInfo(bytes32 ) view returns(bytes32 subAccountId, address userAddress, uint128 nonce)
func (_Endpoint *EndpointSession) AccountInfo(arg0 [32]byte) (struct {
	SubAccountId [32]byte
	UserAddress  common.Address
	Nonce        *big.Int
}, error) {
	return _Endpoint.Contract.AccountInfo(&_Endpoint.CallOpts, arg0)
}

// AccountInfo is a free data retrieval call binding the contract method 0x94aaf7fa.
//
// Solidity: function accountInfo(bytes32 ) view returns(bytes32 subAccountId, address userAddress, uint128 nonce)
func (_Endpoint *EndpointCallerSession) AccountInfo(arg0 [32]byte) (struct {
	SubAccountId [32]byte
	UserAddress  common.Address
	Nonce        *big.Int
}, error) {
	return _Endpoint.Contract.AccountInfo(&_Endpoint.CallOpts, arg0)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Endpoint *EndpointCaller) Eip712Domain(opts *bind.CallOpts) (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "eip712Domain")

	outstruct := new(struct {
		Fields            [1]byte
		Name              string
		Version           string
		ChainId           *big.Int
		VerifyingContract common.Address
		Salt              [32]byte
		Extensions        []*big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Fields = *abi.ConvertType(out[0], new([1]byte)).(*[1]byte)
	outstruct.Name = *abi.ConvertType(out[1], new(string)).(*string)
	outstruct.Version = *abi.ConvertType(out[2], new(string)).(*string)
	outstruct.ChainId = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.VerifyingContract = *abi.ConvertType(out[4], new(common.Address)).(*common.Address)
	outstruct.Salt = *abi.ConvertType(out[5], new([32]byte)).(*[32]byte)
	outstruct.Extensions = *abi.ConvertType(out[6], new([]*big.Int)).(*[]*big.Int)

	return *outstruct, err

}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Endpoint *EndpointSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _Endpoint.Contract.Eip712Domain(&_Endpoint.CallOpts)
}

// Eip712Domain is a free data retrieval call binding the contract method 0x84b0196e.
//
// Solidity: function eip712Domain() view returns(bytes1 fields, string name, string version, uint256 chainId, address verifyingContract, bytes32 salt, uint256[] extensions)
func (_Endpoint *EndpointCallerSession) Eip712Domain() (struct {
	Fields            [1]byte
	Name              string
	Version           string
	ChainId           *big.Int
	VerifyingContract common.Address
	Salt              [32]byte
	Extensions        []*big.Int
}, error) {
	return _Endpoint.Contract.Eip712Domain(&_Endpoint.CallOpts)
}

// GetLinkedSigner is a free data retrieval call binding the contract method 0x91c1e3d7.
//
// Solidity: function getLinkedSigner(bytes32 subaccount) view returns(address)
func (_Endpoint *EndpointCaller) GetLinkedSigner(opts *bind.CallOpts, subaccount [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getLinkedSigner", subaccount)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetLinkedSigner is a free data retrieval call binding the contract method 0x91c1e3d7.
//
// Solidity: function getLinkedSigner(bytes32 subaccount) view returns(address)
func (_Endpoint *EndpointSession) GetLinkedSigner(subaccount [32]byte) (common.Address, error) {
	return _Endpoint.Contract.GetLinkedSigner(&_Endpoint.CallOpts, subaccount)
}

// GetLinkedSigner is a free data retrieval call binding the contract method 0x91c1e3d7.
//
// Solidity: function getLinkedSigner(bytes32 subaccount) view returns(address)
func (_Endpoint *EndpointCallerSession) GetLinkedSigner(subaccount [32]byte) (common.Address, error) {
	return _Endpoint.Contract.GetLinkedSigner(&_Endpoint.CallOpts, subaccount)
}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 subAccountId) pure returns(address masterWallet)
func (_Endpoint *EndpointCaller) GetMasterWallet(opts *bind.CallOpts, subAccountId [32]byte) (common.Address, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getMasterWallet", subAccountId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 subAccountId) pure returns(address masterWallet)
func (_Endpoint *EndpointSession) GetMasterWallet(subAccountId [32]byte) (common.Address, error) {
	return _Endpoint.Contract.GetMasterWallet(&_Endpoint.CallOpts, subAccountId)
}

// GetMasterWallet is a free data retrieval call binding the contract method 0x70bb7bc8.
//
// Solidity: function getMasterWallet(bytes32 subAccountId) pure returns(address masterWallet)
func (_Endpoint *EndpointCallerSession) GetMasterWallet(subAccountId [32]byte) (common.Address, error) {
	return _Endpoint.Contract.GetMasterWallet(&_Endpoint.CallOpts, subAccountId)
}

// GetNonce is a free data retrieval call binding the contract method 0x4136a33c.
//
// Solidity: function getNonce(bytes32 _subAccountId) view returns(uint128)
func (_Endpoint *EndpointCaller) GetNonce(opts *bind.CallOpts, _subAccountId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getNonce", _subAccountId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetNonce is a free data retrieval call binding the contract method 0x4136a33c.
//
// Solidity: function getNonce(bytes32 _subAccountId) view returns(uint128)
func (_Endpoint *EndpointSession) GetNonce(_subAccountId [32]byte) (*big.Int, error) {
	return _Endpoint.Contract.GetNonce(&_Endpoint.CallOpts, _subAccountId)
}

// GetNonce is a free data retrieval call binding the contract method 0x4136a33c.
//
// Solidity: function getNonce(bytes32 _subAccountId) view returns(uint128)
func (_Endpoint *EndpointCallerSession) GetNonce(_subAccountId [32]byte) (*big.Int, error) {
	return _Endpoint.Contract.GetNonce(&_Endpoint.CallOpts, _subAccountId)
}

// GetNoncesOfSubaccounts is a free data retrieval call binding the contract method 0xd9477825.
//
// Solidity: function getNoncesOfSubaccounts(bytes32[] subaccountIds) view returns(uint128[] nonces)
func (_Endpoint *EndpointCaller) GetNoncesOfSubaccounts(opts *bind.CallOpts, subaccountIds [][32]byte) ([]*big.Int, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getNoncesOfSubaccounts", subaccountIds)

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetNoncesOfSubaccounts is a free data retrieval call binding the contract method 0xd9477825.
//
// Solidity: function getNoncesOfSubaccounts(bytes32[] subaccountIds) view returns(uint128[] nonces)
func (_Endpoint *EndpointSession) GetNoncesOfSubaccounts(subaccountIds [][32]byte) ([]*big.Int, error) {
	return _Endpoint.Contract.GetNoncesOfSubaccounts(&_Endpoint.CallOpts, subaccountIds)
}

// GetNoncesOfSubaccounts is a free data retrieval call binding the contract method 0xd9477825.
//
// Solidity: function getNoncesOfSubaccounts(bytes32[] subaccountIds) view returns(uint128[] nonces)
func (_Endpoint *EndpointCallerSession) GetNoncesOfSubaccounts(subaccountIds [][32]byte) ([]*big.Int, error) {
	return _Endpoint.Contract.GetNoncesOfSubaccounts(&_Endpoint.CallOpts, subaccountIds)
}

// GetOffchainExchange is a free data retrieval call binding the contract method 0x8f4f8ecc.
//
// Solidity: function getOffchainExchange() view returns(address)
func (_Endpoint *EndpointCaller) GetOffchainExchange(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getOffchainExchange")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetOffchainExchange is a free data retrieval call binding the contract method 0x8f4f8ecc.
//
// Solidity: function getOffchainExchange() view returns(address)
func (_Endpoint *EndpointSession) GetOffchainExchange() (common.Address, error) {
	return _Endpoint.Contract.GetOffchainExchange(&_Endpoint.CallOpts)
}

// GetOffchainExchange is a free data retrieval call binding the contract method 0x8f4f8ecc.
//
// Solidity: function getOffchainExchange() view returns(address)
func (_Endpoint *EndpointCallerSession) GetOffchainExchange() (common.Address, error) {
	return _Endpoint.Contract.GetOffchainExchange(&_Endpoint.CallOpts)
}

// GetSequencer is a free data retrieval call binding the contract method 0xe90f218f.
//
// Solidity: function getSequencer(address _sequencer) view returns(bool)
func (_Endpoint *EndpointCaller) GetSequencer(opts *bind.CallOpts, _sequencer common.Address) (bool, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getSequencer", _sequencer)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetSequencer is a free data retrieval call binding the contract method 0xe90f218f.
//
// Solidity: function getSequencer(address _sequencer) view returns(bool)
func (_Endpoint *EndpointSession) GetSequencer(_sequencer common.Address) (bool, error) {
	return _Endpoint.Contract.GetSequencer(&_Endpoint.CallOpts, _sequencer)
}

// GetSequencer is a free data retrieval call binding the contract method 0xe90f218f.
//
// Solidity: function getSequencer(address _sequencer) view returns(bool)
func (_Endpoint *EndpointCallerSession) GetSequencer(_sequencer common.Address) (bool, error) {
	return _Endpoint.Contract.GetSequencer(&_Endpoint.CallOpts, _sequencer)
}

// GetTime is a free data retrieval call binding the contract method 0x557ed1ba.
//
// Solidity: function getTime() view returns(uint128)
func (_Endpoint *EndpointCaller) GetTime(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getTime")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTime is a free data retrieval call binding the contract method 0x557ed1ba.
//
// Solidity: function getTime() view returns(uint128)
func (_Endpoint *EndpointSession) GetTime() (*big.Int, error) {
	return _Endpoint.Contract.GetTime(&_Endpoint.CallOpts)
}

// GetTime is a free data retrieval call binding the contract method 0x557ed1ba.
//
// Solidity: function getTime() view returns(uint128)
func (_Endpoint *EndpointCallerSession) GetTime() (*big.Int, error) {
	return _Endpoint.Contract.GetTime(&_Endpoint.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Endpoint *EndpointCaller) GetVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Endpoint *EndpointSession) GetVersion() (uint64, error) {
	return _Endpoint.Contract.GetVersion(&_Endpoint.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_Endpoint *EndpointCallerSession) GetVersion() (uint64, error) {
	return _Endpoint.Contract.GetVersion(&_Endpoint.CallOpts)
}

// NSubmissions is a free data retrieval call binding the contract method 0x18ed16eb.
//
// Solidity: function nSubmissions() view returns(uint64)
func (_Endpoint *EndpointCaller) NSubmissions(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "nSubmissions")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// NSubmissions is a free data retrieval call binding the contract method 0x18ed16eb.
//
// Solidity: function nSubmissions() view returns(uint64)
func (_Endpoint *EndpointSession) NSubmissions() (uint64, error) {
	return _Endpoint.Contract.NSubmissions(&_Endpoint.CallOpts)
}

// NSubmissions is a free data retrieval call binding the contract method 0x18ed16eb.
//
// Solidity: function nSubmissions() view returns(uint64)
func (_Endpoint *EndpointCallerSession) NSubmissions() (uint64, error) {
	return _Endpoint.Contract.NSubmissions(&_Endpoint.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Endpoint *EndpointCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Endpoint *EndpointSession) Owner() (common.Address, error) {
	return _Endpoint.Contract.Owner(&_Endpoint.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Endpoint *EndpointCallerSession) Owner() (common.Address, error) {
	return _Endpoint.Contract.Owner(&_Endpoint.CallOpts)
}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_Endpoint *EndpointCaller) ValidateSessionkey(opts *bind.CallOpts, sessionKey common.Address, subAccountId [32]byte) error {
	var out []interface{}
	err := _Endpoint.contract.Call(opts, &out, "validateSessionkey", sessionKey, subAccountId)

	if err != nil {
		return err
	}

	return err

}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_Endpoint *EndpointSession) ValidateSessionkey(sessionKey common.Address, subAccountId [32]byte) error {
	return _Endpoint.Contract.ValidateSessionkey(&_Endpoint.CallOpts, sessionKey, subAccountId)
}

// ValidateSessionkey is a free data retrieval call binding the contract method 0xf0cf5ad1.
//
// Solidity: function validateSessionkey(address sessionKey, bytes32 subAccountId) view returns()
func (_Endpoint *EndpointCallerSession) ValidateSessionkey(sessionKey common.Address, subAccountId [32]byte) error {
	return _Endpoint.Contract.ValidateSessionkey(&_Endpoint.CallOpts, sessionKey, subAccountId)
}

// RecordAdminSubaccount is a paid mutator transaction binding the contract method 0x6f067c41.
//
// Solidity: function _recordAdminSubaccount((bytes32,address,address,uint128,uint128,uint256) sessionRequest) returns()
func (_Endpoint *EndpointTransactor) RecordAdminSubaccount(opts *bind.TransactOpts, sessionRequest IEndpointRegister) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "_recordAdminSubaccount", sessionRequest)
}

// RecordAdminSubaccount is a paid mutator transaction binding the contract method 0x6f067c41.
//
// Solidity: function _recordAdminSubaccount((bytes32,address,address,uint128,uint128,uint256) sessionRequest) returns()
func (_Endpoint *EndpointSession) RecordAdminSubaccount(sessionRequest IEndpointRegister) (*types.Transaction, error) {
	return _Endpoint.Contract.RecordAdminSubaccount(&_Endpoint.TransactOpts, sessionRequest)
}

// RecordAdminSubaccount is a paid mutator transaction binding the contract method 0x6f067c41.
//
// Solidity: function _recordAdminSubaccount((bytes32,address,address,uint128,uint128,uint256) sessionRequest) returns()
func (_Endpoint *EndpointTransactorSession) RecordAdminSubaccount(sessionRequest IEndpointRegister) (*types.Transaction, error) {
	return _Endpoint.Contract.RecordAdminSubaccount(&_Endpoint.TransactOpts, sessionRequest)
}

// BridgeInLogX is a paid mutator transaction binding the contract method 0xf57dd6dd.
//
// Solidity: function bridgeInLogX(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) payable returns()
func (_Endpoint *EndpointTransactor) BridgeInLogX(opts *bind.TransactOpts, subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "bridgeInLogX", subAccountId, amount, sourceChainId)
}

// BridgeInLogX is a paid mutator transaction binding the contract method 0xf57dd6dd.
//
// Solidity: function bridgeInLogX(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) payable returns()
func (_Endpoint *EndpointSession) BridgeInLogX(subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.BridgeInLogX(&_Endpoint.TransactOpts, subAccountId, amount, sourceChainId)
}

// BridgeInLogX is a paid mutator transaction binding the contract method 0xf57dd6dd.
//
// Solidity: function bridgeInLogX(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) payable returns()
func (_Endpoint *EndpointTransactorSession) BridgeInLogX(subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.BridgeInLogX(&_Endpoint.TransactOpts, subAccountId, amount, sourceChainId)
}

// DepositCollateral is a paid mutator transaction binding the contract method 0x67c07a05.
//
// Solidity: function depositCollateral(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) returns()
func (_Endpoint *EndpointTransactor) DepositCollateral(opts *bind.TransactOpts, subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "depositCollateral", subAccountId, amount, sourceChainId)
}

// DepositCollateral is a paid mutator transaction binding the contract method 0x67c07a05.
//
// Solidity: function depositCollateral(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) returns()
func (_Endpoint *EndpointSession) DepositCollateral(subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.DepositCollateral(&_Endpoint.TransactOpts, subAccountId, amount, sourceChainId)
}

// DepositCollateral is a paid mutator transaction binding the contract method 0x67c07a05.
//
// Solidity: function depositCollateral(bytes32 subAccountId, uint256 amount, uint256 sourceChainId) returns()
func (_Endpoint *EndpointTransactorSession) DepositCollateral(subAccountId [32]byte, amount *big.Int, sourceChainId *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.DepositCollateral(&_Endpoint.TransactOpts, subAccountId, amount, sourceChainId)
}

// FaucetDeposit is a paid mutator transaction binding the contract method 0x2f789154.
//
// Solidity: function faucetDeposit(uint32[] productIds, bytes32 subAccount, int128 amount) returns()
func (_Endpoint *EndpointTransactor) FaucetDeposit(opts *bind.TransactOpts, productIds []uint32, subAccount [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "faucetDeposit", productIds, subAccount, amount)
}

// FaucetDeposit is a paid mutator transaction binding the contract method 0x2f789154.
//
// Solidity: function faucetDeposit(uint32[] productIds, bytes32 subAccount, int128 amount) returns()
func (_Endpoint *EndpointSession) FaucetDeposit(productIds []uint32, subAccount [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.FaucetDeposit(&_Endpoint.TransactOpts, productIds, subAccount, amount)
}

// FaucetDeposit is a paid mutator transaction binding the contract method 0x2f789154.
//
// Solidity: function faucetDeposit(uint32[] productIds, bytes32 subAccount, int128 amount) returns()
func (_Endpoint *EndpointTransactorSession) FaucetDeposit(productIds []uint32, subAccount [32]byte, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.FaucetDeposit(&_Endpoint.TransactOpts, productIds, subAccount, amount)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _sequencer, address _offchainExchange, address _clearinghouse) returns()
func (_Endpoint *EndpointTransactor) Initialize(opts *bind.TransactOpts, _sequencer common.Address, _offchainExchange common.Address, _clearinghouse common.Address) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "initialize", _sequencer, _offchainExchange, _clearinghouse)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _sequencer, address _offchainExchange, address _clearinghouse) returns()
func (_Endpoint *EndpointSession) Initialize(_sequencer common.Address, _offchainExchange common.Address, _clearinghouse common.Address) (*types.Transaction, error) {
	return _Endpoint.Contract.Initialize(&_Endpoint.TransactOpts, _sequencer, _offchainExchange, _clearinghouse)
}

// Initialize is a paid mutator transaction binding the contract method 0xc0c53b8b.
//
// Solidity: function initialize(address _sequencer, address _offchainExchange, address _clearinghouse) returns()
func (_Endpoint *EndpointTransactorSession) Initialize(_sequencer common.Address, _offchainExchange common.Address, _clearinghouse common.Address) (*types.Transaction, error) {
	return _Endpoint.Contract.Initialize(&_Endpoint.TransactOpts, _sequencer, _offchainExchange, _clearinghouse)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Endpoint *EndpointTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Endpoint *EndpointSession) RenounceOwnership() (*types.Transaction, error) {
	return _Endpoint.Contract.RenounceOwnership(&_Endpoint.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Endpoint *EndpointTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Endpoint.Contract.RenounceOwnership(&_Endpoint.TransactOpts)
}

// SetSequencer is a paid mutator transaction binding the contract method 0x55dba28a.
//
// Solidity: function setSequencer(address _sequencer, bool value) returns()
func (_Endpoint *EndpointTransactor) SetSequencer(opts *bind.TransactOpts, _sequencer common.Address, value bool) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "setSequencer", _sequencer, value)
}

// SetSequencer is a paid mutator transaction binding the contract method 0x55dba28a.
//
// Solidity: function setSequencer(address _sequencer, bool value) returns()
func (_Endpoint *EndpointSession) SetSequencer(_sequencer common.Address, value bool) (*types.Transaction, error) {
	return _Endpoint.Contract.SetSequencer(&_Endpoint.TransactOpts, _sequencer, value)
}

// SetSequencer is a paid mutator transaction binding the contract method 0x55dba28a.
//
// Solidity: function setSequencer(address _sequencer, bool value) returns()
func (_Endpoint *EndpointTransactorSession) SetSequencer(_sequencer common.Address, value bool) (*types.Transaction, error) {
	return _Endpoint.Contract.SetSequencer(&_Endpoint.TransactOpts, _sequencer, value)
}

// SubmitTransactionsChecked is a paid mutator transaction binding the contract method 0xd2e7511d.
//
// Solidity: function submitTransactionsChecked(uint64 idx, bytes[] transactions, bytes[] signatures, bytes[] signatures2) returns()
func (_Endpoint *EndpointTransactor) SubmitTransactionsChecked(opts *bind.TransactOpts, idx uint64, transactions [][]byte, signatures [][]byte, signatures2 [][]byte) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "submitTransactionsChecked", idx, transactions, signatures, signatures2)
}

// SubmitTransactionsChecked is a paid mutator transaction binding the contract method 0xd2e7511d.
//
// Solidity: function submitTransactionsChecked(uint64 idx, bytes[] transactions, bytes[] signatures, bytes[] signatures2) returns()
func (_Endpoint *EndpointSession) SubmitTransactionsChecked(idx uint64, transactions [][]byte, signatures [][]byte, signatures2 [][]byte) (*types.Transaction, error) {
	return _Endpoint.Contract.SubmitTransactionsChecked(&_Endpoint.TransactOpts, idx, transactions, signatures, signatures2)
}

// SubmitTransactionsChecked is a paid mutator transaction binding the contract method 0xd2e7511d.
//
// Solidity: function submitTransactionsChecked(uint64 idx, bytes[] transactions, bytes[] signatures, bytes[] signatures2) returns()
func (_Endpoint *EndpointTransactorSession) SubmitTransactionsChecked(idx uint64, transactions [][]byte, signatures [][]byte, signatures2 [][]byte) (*types.Transaction, error) {
	return _Endpoint.Contract.SubmitTransactionsChecked(&_Endpoint.TransactOpts, idx, transactions, signatures, signatures2)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Endpoint *EndpointTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Endpoint *EndpointSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Endpoint.Contract.TransferOwnership(&_Endpoint.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Endpoint *EndpointTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Endpoint.Contract.TransferOwnership(&_Endpoint.TransactOpts, newOwner)
}

// WithDrawExcessFunds is a paid mutator transaction binding the contract method 0xfa37c410.
//
// Solidity: function withDrawExcessFunds(address receiver, uint256 amount) returns()
func (_Endpoint *EndpointTransactor) WithDrawExcessFunds(opts *bind.TransactOpts, receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "withDrawExcessFunds", receiver, amount)
}

// WithDrawExcessFunds is a paid mutator transaction binding the contract method 0xfa37c410.
//
// Solidity: function withDrawExcessFunds(address receiver, uint256 amount) returns()
func (_Endpoint *EndpointSession) WithDrawExcessFunds(receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.WithDrawExcessFunds(&_Endpoint.TransactOpts, receiver, amount)
}

// WithDrawExcessFunds is a paid mutator transaction binding the contract method 0xfa37c410.
//
// Solidity: function withDrawExcessFunds(address receiver, uint256 amount) returns()
func (_Endpoint *EndpointTransactorSession) WithDrawExcessFunds(receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.WithDrawExcessFunds(&_Endpoint.TransactOpts, receiver, amount)
}

// WithdrawExcessToken is a paid mutator transaction binding the contract method 0xd72a29ad.
//
// Solidity: function withdrawExcessToken(address token, address receiver, uint256 amount) returns()
func (_Endpoint *EndpointTransactor) WithdrawExcessToken(opts *bind.TransactOpts, token common.Address, receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.contract.Transact(opts, "withdrawExcessToken", token, receiver, amount)
}

// WithdrawExcessToken is a paid mutator transaction binding the contract method 0xd72a29ad.
//
// Solidity: function withdrawExcessToken(address token, address receiver, uint256 amount) returns()
func (_Endpoint *EndpointSession) WithdrawExcessToken(token common.Address, receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.WithdrawExcessToken(&_Endpoint.TransactOpts, token, receiver, amount)
}

// WithdrawExcessToken is a paid mutator transaction binding the contract method 0xd72a29ad.
//
// Solidity: function withdrawExcessToken(address token, address receiver, uint256 amount) returns()
func (_Endpoint *EndpointTransactorSession) WithdrawExcessToken(token common.Address, receiver common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Endpoint.Contract.WithdrawExcessToken(&_Endpoint.TransactOpts, token, receiver, amount)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Endpoint *EndpointTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Endpoint.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Endpoint *EndpointSession) Receive() (*types.Transaction, error) {
	return _Endpoint.Contract.Receive(&_Endpoint.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_Endpoint *EndpointTransactorSession) Receive() (*types.Transaction, error) {
	return _Endpoint.Contract.Receive(&_Endpoint.TransactOpts)
}

// EndpointClaimStakeRewardsIterator is returned from FilterClaimStakeRewards and is used to iterate over the raw logs and unpacked data for ClaimStakeRewards events raised by the Endpoint contract.
type EndpointClaimStakeRewardsIterator struct {
	Event *EndpointClaimStakeRewards // Event containing the contract specifics and raw log

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
func (it *EndpointClaimStakeRewardsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointClaimStakeRewards)
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
		it.Event = new(EndpointClaimStakeRewards)
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
func (it *EndpointClaimStakeRewardsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointClaimStakeRewardsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointClaimStakeRewards represents a ClaimStakeRewards event raised by the Endpoint contract.
type EndpointClaimStakeRewards struct {
	SubAccountId [32]byte
	UserAddress  common.Address
	RewardAmount *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterClaimStakeRewards is a free log retrieval operation binding the contract event 0xc5d0cf15c43e508778422eeba925ebbdd16e6db38f65861195983907956dfe4e.
//
// Solidity: event ClaimStakeRewards(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed rewardAmount)
func (_Endpoint *EndpointFilterer) FilterClaimStakeRewards(opts *bind.FilterOpts, subAccountId [][32]byte, userAddress []common.Address, rewardAmount []*big.Int) (*EndpointClaimStakeRewardsIterator, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var rewardAmountRule []interface{}
	for _, rewardAmountItem := range rewardAmount {
		rewardAmountRule = append(rewardAmountRule, rewardAmountItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "ClaimStakeRewards", subAccountIdRule, userAddressRule, rewardAmountRule)
	if err != nil {
		return nil, err
	}
	return &EndpointClaimStakeRewardsIterator{contract: _Endpoint.contract, event: "ClaimStakeRewards", logs: logs, sub: sub}, nil
}

// WatchClaimStakeRewards is a free log subscription operation binding the contract event 0xc5d0cf15c43e508778422eeba925ebbdd16e6db38f65861195983907956dfe4e.
//
// Solidity: event ClaimStakeRewards(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed rewardAmount)
func (_Endpoint *EndpointFilterer) WatchClaimStakeRewards(opts *bind.WatchOpts, sink chan<- *EndpointClaimStakeRewards, subAccountId [][32]byte, userAddress []common.Address, rewardAmount []*big.Int) (event.Subscription, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var rewardAmountRule []interface{}
	for _, rewardAmountItem := range rewardAmount {
		rewardAmountRule = append(rewardAmountRule, rewardAmountItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "ClaimStakeRewards", subAccountIdRule, userAddressRule, rewardAmountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointClaimStakeRewards)
				if err := _Endpoint.contract.UnpackLog(event, "ClaimStakeRewards", log); err != nil {
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

// ParseClaimStakeRewards is a log parse operation binding the contract event 0xc5d0cf15c43e508778422eeba925ebbdd16e6db38f65861195983907956dfe4e.
//
// Solidity: event ClaimStakeRewards(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed rewardAmount)
func (_Endpoint *EndpointFilterer) ParseClaimStakeRewards(log types.Log) (*EndpointClaimStakeRewards, error) {
	event := new(EndpointClaimStakeRewards)
	if err := _Endpoint.contract.UnpackLog(event, "ClaimStakeRewards", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointClaimedLogXIterator is returned from FilterClaimedLogX and is used to iterate over the raw logs and unpacked data for ClaimedLogX events raised by the Endpoint contract.
type EndpointClaimedLogXIterator struct {
	Event *EndpointClaimedLogX // Event containing the contract specifics and raw log

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
func (it *EndpointClaimedLogXIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointClaimedLogX)
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
		it.Event = new(EndpointClaimedLogX)
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
func (it *EndpointClaimedLogXIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointClaimedLogXIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointClaimedLogX represents a ClaimedLogX event raised by the Endpoint contract.
type EndpointClaimedLogX struct {
	AccountId   [32]byte
	UserAddress common.Address
	Amount      *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterClaimedLogX is a free log retrieval operation binding the contract event 0x79cebc047ee3bd160f79f9952ef3ca2d2dd3fccd26b4005c93d3dc0085b34cfe.
//
// Solidity: event ClaimedLogX(bytes32 indexed accountId, address indexed userAddress, int128 indexed amount)
func (_Endpoint *EndpointFilterer) FilterClaimedLogX(opts *bind.FilterOpts, accountId [][32]byte, userAddress []common.Address, amount []*big.Int) (*EndpointClaimedLogXIterator, error) {

	var accountIdRule []interface{}
	for _, accountIdItem := range accountId {
		accountIdRule = append(accountIdRule, accountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "ClaimedLogX", accountIdRule, userAddressRule, amountRule)
	if err != nil {
		return nil, err
	}
	return &EndpointClaimedLogXIterator{contract: _Endpoint.contract, event: "ClaimedLogX", logs: logs, sub: sub}, nil
}

// WatchClaimedLogX is a free log subscription operation binding the contract event 0x79cebc047ee3bd160f79f9952ef3ca2d2dd3fccd26b4005c93d3dc0085b34cfe.
//
// Solidity: event ClaimedLogX(bytes32 indexed accountId, address indexed userAddress, int128 indexed amount)
func (_Endpoint *EndpointFilterer) WatchClaimedLogX(opts *bind.WatchOpts, sink chan<- *EndpointClaimedLogX, accountId [][32]byte, userAddress []common.Address, amount []*big.Int) (event.Subscription, error) {

	var accountIdRule []interface{}
	for _, accountIdItem := range accountId {
		accountIdRule = append(accountIdRule, accountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var amountRule []interface{}
	for _, amountItem := range amount {
		amountRule = append(amountRule, amountItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "ClaimedLogX", accountIdRule, userAddressRule, amountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointClaimedLogX)
				if err := _Endpoint.contract.UnpackLog(event, "ClaimedLogX", log); err != nil {
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

// ParseClaimedLogX is a log parse operation binding the contract event 0x79cebc047ee3bd160f79f9952ef3ca2d2dd3fccd26b4005c93d3dc0085b34cfe.
//
// Solidity: event ClaimedLogX(bytes32 indexed accountId, address indexed userAddress, int128 indexed amount)
func (_Endpoint *EndpointFilterer) ParseClaimedLogX(log types.Log) (*EndpointClaimedLogX, error) {
	event := new(EndpointClaimedLogX)
	if err := _Endpoint.contract.UnpackLog(event, "ClaimedLogX", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointDepositTokenIterator is returned from FilterDepositToken and is used to iterate over the raw logs and unpacked data for DepositToken events raised by the Endpoint contract.
type EndpointDepositTokenIterator struct {
	Event *EndpointDepositToken // Event containing the contract specifics and raw log

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
func (it *EndpointDepositTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointDepositToken)
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
		it.Event = new(EndpointDepositToken)
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
func (it *EndpointDepositTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointDepositTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointDepositToken represents a DepositToken event raised by the Endpoint contract.
type EndpointDepositToken struct {
	SubAccountId       [32]byte
	MasterWallet       common.Address
	Amount             *big.Int
	ProductId          uint32
	SourceChainId      *big.Int
	DestinationChainId *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterDepositToken is a free log retrieval operation binding the contract event 0x4de58ce9f2fbe79c140effbdf9a8cd8aab2a6e6bef0466d2f0135ef968fda613.
//
// Solidity: event DepositToken(bytes32 indexed subAccountId, address indexed masterWallet, uint256 amount, uint32 indexed productId, uint256 sourceChainId, uint256 destinationChainId)
func (_Endpoint *EndpointFilterer) FilterDepositToken(opts *bind.FilterOpts, subAccountId [][32]byte, masterWallet []common.Address, productId []uint32) (*EndpointDepositTokenIterator, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var masterWalletRule []interface{}
	for _, masterWalletItem := range masterWallet {
		masterWalletRule = append(masterWalletRule, masterWalletItem)
	}

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "DepositToken", subAccountIdRule, masterWalletRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return &EndpointDepositTokenIterator{contract: _Endpoint.contract, event: "DepositToken", logs: logs, sub: sub}, nil
}

// WatchDepositToken is a free log subscription operation binding the contract event 0x4de58ce9f2fbe79c140effbdf9a8cd8aab2a6e6bef0466d2f0135ef968fda613.
//
// Solidity: event DepositToken(bytes32 indexed subAccountId, address indexed masterWallet, uint256 amount, uint32 indexed productId, uint256 sourceChainId, uint256 destinationChainId)
func (_Endpoint *EndpointFilterer) WatchDepositToken(opts *bind.WatchOpts, sink chan<- *EndpointDepositToken, subAccountId [][32]byte, masterWallet []common.Address, productId []uint32) (event.Subscription, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var masterWalletRule []interface{}
	for _, masterWalletItem := range masterWallet {
		masterWalletRule = append(masterWalletRule, masterWalletItem)
	}

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "DepositToken", subAccountIdRule, masterWalletRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointDepositToken)
				if err := _Endpoint.contract.UnpackLog(event, "DepositToken", log); err != nil {
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

// ParseDepositToken is a log parse operation binding the contract event 0x4de58ce9f2fbe79c140effbdf9a8cd8aab2a6e6bef0466d2f0135ef968fda613.
//
// Solidity: event DepositToken(bytes32 indexed subAccountId, address indexed masterWallet, uint256 amount, uint32 indexed productId, uint256 sourceChainId, uint256 destinationChainId)
func (_Endpoint *EndpointFilterer) ParseDepositToken(log types.Log) (*EndpointDepositToken, error) {
	event := new(EndpointDepositToken)
	if err := _Endpoint.contract.UnpackLog(event, "DepositToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointEIP712DomainChangedIterator is returned from FilterEIP712DomainChanged and is used to iterate over the raw logs and unpacked data for EIP712DomainChanged events raised by the Endpoint contract.
type EndpointEIP712DomainChangedIterator struct {
	Event *EndpointEIP712DomainChanged // Event containing the contract specifics and raw log

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
func (it *EndpointEIP712DomainChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointEIP712DomainChanged)
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
		it.Event = new(EndpointEIP712DomainChanged)
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
func (it *EndpointEIP712DomainChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointEIP712DomainChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointEIP712DomainChanged represents a EIP712DomainChanged event raised by the Endpoint contract.
type EndpointEIP712DomainChanged struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterEIP712DomainChanged is a free log retrieval operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Endpoint *EndpointFilterer) FilterEIP712DomainChanged(opts *bind.FilterOpts) (*EndpointEIP712DomainChangedIterator, error) {

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return &EndpointEIP712DomainChangedIterator{contract: _Endpoint.contract, event: "EIP712DomainChanged", logs: logs, sub: sub}, nil
}

// WatchEIP712DomainChanged is a free log subscription operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Endpoint *EndpointFilterer) WatchEIP712DomainChanged(opts *bind.WatchOpts, sink chan<- *EndpointEIP712DomainChanged) (event.Subscription, error) {

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "EIP712DomainChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointEIP712DomainChanged)
				if err := _Endpoint.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
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

// ParseEIP712DomainChanged is a log parse operation binding the contract event 0x0a6387c9ea3628b88a633bb4f3b151770f70085117a15f9bf3787cda53f13d31.
//
// Solidity: event EIP712DomainChanged()
func (_Endpoint *EndpointFilterer) ParseEIP712DomainChanged(log types.Log) (*EndpointEIP712DomainChanged, error) {
	event := new(EndpointEIP712DomainChanged)
	if err := _Endpoint.contract.UnpackLog(event, "EIP712DomainChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Endpoint contract.
type EndpointInitializedIterator struct {
	Event *EndpointInitialized // Event containing the contract specifics and raw log

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
func (it *EndpointInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointInitialized)
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
		it.Event = new(EndpointInitialized)
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
func (it *EndpointInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointInitialized represents a Initialized event raised by the Endpoint contract.
type EndpointInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Endpoint *EndpointFilterer) FilterInitialized(opts *bind.FilterOpts) (*EndpointInitializedIterator, error) {

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &EndpointInitializedIterator{contract: _Endpoint.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Endpoint *EndpointFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *EndpointInitialized) (event.Subscription, error) {

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointInitialized)
				if err := _Endpoint.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_Endpoint *EndpointFilterer) ParseInitialized(log types.Log) (*EndpointInitialized, error) {
	event := new(EndpointInitialized)
	if err := _Endpoint.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Endpoint contract.
type EndpointOwnershipTransferredIterator struct {
	Event *EndpointOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *EndpointOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointOwnershipTransferred)
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
		it.Event = new(EndpointOwnershipTransferred)
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
func (it *EndpointOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointOwnershipTransferred represents a OwnershipTransferred event raised by the Endpoint contract.
type EndpointOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Endpoint *EndpointFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*EndpointOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &EndpointOwnershipTransferredIterator{contract: _Endpoint.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Endpoint *EndpointFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *EndpointOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointOwnershipTransferred)
				if err := _Endpoint.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Endpoint *EndpointFilterer) ParseOwnershipTransferred(log types.Log) (*EndpointOwnershipTransferred, error) {
	event := new(EndpointOwnershipTransferred)
	if err := _Endpoint.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointSessionKeyCreatedIterator is returned from FilterSessionKeyCreated and is used to iterate over the raw logs and unpacked data for SessionKeyCreated events raised by the Endpoint contract.
type EndpointSessionKeyCreatedIterator struct {
	Event *EndpointSessionKeyCreated // Event containing the contract specifics and raw log

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
func (it *EndpointSessionKeyCreatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointSessionKeyCreated)
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
		it.Event = new(EndpointSessionKeyCreated)
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
func (it *EndpointSessionKeyCreatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointSessionKeyCreatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointSessionKeyCreated represents a SessionKeyCreated event raised by the Endpoint contract.
type EndpointSessionKeyCreated struct {
	SubAccountId    [32]byte
	UserAddress     common.Address
	SessionKey      common.Address
	ExpiryTimeStamp *big.Int
	Raw             types.Log // Blockchain specific contextual infos
}

// FilterSessionKeyCreated is a free log retrieval operation binding the contract event 0x7a927c3d17989c1387ceba70c2927595c5789a56dc175b71028e2d6d43599c86.
//
// Solidity: event SessionKeyCreated(bytes32 indexed subAccountId, address indexed userAddress, address indexed sessionKey, uint128 expiryTimeStamp)
func (_Endpoint *EndpointFilterer) FilterSessionKeyCreated(opts *bind.FilterOpts, subAccountId [][32]byte, userAddress []common.Address, sessionKey []common.Address) (*EndpointSessionKeyCreatedIterator, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var sessionKeyRule []interface{}
	for _, sessionKeyItem := range sessionKey {
		sessionKeyRule = append(sessionKeyRule, sessionKeyItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "SessionKeyCreated", subAccountIdRule, userAddressRule, sessionKeyRule)
	if err != nil {
		return nil, err
	}
	return &EndpointSessionKeyCreatedIterator{contract: _Endpoint.contract, event: "SessionKeyCreated", logs: logs, sub: sub}, nil
}

// WatchSessionKeyCreated is a free log subscription operation binding the contract event 0x7a927c3d17989c1387ceba70c2927595c5789a56dc175b71028e2d6d43599c86.
//
// Solidity: event SessionKeyCreated(bytes32 indexed subAccountId, address indexed userAddress, address indexed sessionKey, uint128 expiryTimeStamp)
func (_Endpoint *EndpointFilterer) WatchSessionKeyCreated(opts *bind.WatchOpts, sink chan<- *EndpointSessionKeyCreated, subAccountId [][32]byte, userAddress []common.Address, sessionKey []common.Address) (event.Subscription, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var sessionKeyRule []interface{}
	for _, sessionKeyItem := range sessionKey {
		sessionKeyRule = append(sessionKeyRule, sessionKeyItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "SessionKeyCreated", subAccountIdRule, userAddressRule, sessionKeyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointSessionKeyCreated)
				if err := _Endpoint.contract.UnpackLog(event, "SessionKeyCreated", log); err != nil {
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

// ParseSessionKeyCreated is a log parse operation binding the contract event 0x7a927c3d17989c1387ceba70c2927595c5789a56dc175b71028e2d6d43599c86.
//
// Solidity: event SessionKeyCreated(bytes32 indexed subAccountId, address indexed userAddress, address indexed sessionKey, uint128 expiryTimeStamp)
func (_Endpoint *EndpointFilterer) ParseSessionKeyCreated(log types.Log) (*EndpointSessionKeyCreated, error) {
	event := new(EndpointSessionKeyCreated)
	if err := _Endpoint.contract.UnpackLog(event, "SessionKeyCreated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointStakeLogXIterator is returned from FilterStakeLogX and is used to iterate over the raw logs and unpacked data for StakeLogX events raised by the Endpoint contract.
type EndpointStakeLogXIterator struct {
	Event *EndpointStakeLogX // Event containing the contract specifics and raw log

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
func (it *EndpointStakeLogXIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointStakeLogX)
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
		it.Event = new(EndpointStakeLogX)
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
func (it *EndpointStakeLogXIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointStakeLogXIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointStakeLogX represents a StakeLogX event raised by the Endpoint contract.
type EndpointStakeLogX struct {
	SubAccountId [32]byte
	UserAddress  common.Address
	StakeAmount  *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterStakeLogX is a free log retrieval operation binding the contract event 0x160d21fa3b8274f2cc3628e53141914e9371cfe867c25a4967e74c6f460c3de9.
//
// Solidity: event StakeLogX(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed stakeAmount)
func (_Endpoint *EndpointFilterer) FilterStakeLogX(opts *bind.FilterOpts, subAccountId [][32]byte, userAddress []common.Address, stakeAmount []*big.Int) (*EndpointStakeLogXIterator, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var stakeAmountRule []interface{}
	for _, stakeAmountItem := range stakeAmount {
		stakeAmountRule = append(stakeAmountRule, stakeAmountItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "StakeLogX", subAccountIdRule, userAddressRule, stakeAmountRule)
	if err != nil {
		return nil, err
	}
	return &EndpointStakeLogXIterator{contract: _Endpoint.contract, event: "StakeLogX", logs: logs, sub: sub}, nil
}

// WatchStakeLogX is a free log subscription operation binding the contract event 0x160d21fa3b8274f2cc3628e53141914e9371cfe867c25a4967e74c6f460c3de9.
//
// Solidity: event StakeLogX(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed stakeAmount)
func (_Endpoint *EndpointFilterer) WatchStakeLogX(opts *bind.WatchOpts, sink chan<- *EndpointStakeLogX, subAccountId [][32]byte, userAddress []common.Address, stakeAmount []*big.Int) (event.Subscription, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var stakeAmountRule []interface{}
	for _, stakeAmountItem := range stakeAmount {
		stakeAmountRule = append(stakeAmountRule, stakeAmountItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "StakeLogX", subAccountIdRule, userAddressRule, stakeAmountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointStakeLogX)
				if err := _Endpoint.contract.UnpackLog(event, "StakeLogX", log); err != nil {
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

// ParseStakeLogX is a log parse operation binding the contract event 0x160d21fa3b8274f2cc3628e53141914e9371cfe867c25a4967e74c6f460c3de9.
//
// Solidity: event StakeLogX(bytes32 indexed subAccountId, address indexed userAddress, uint256 indexed stakeAmount)
func (_Endpoint *EndpointFilterer) ParseStakeLogX(log types.Log) (*EndpointStakeLogX, error) {
	event := new(EndpointStakeLogX)
	if err := _Endpoint.contract.UnpackLog(event, "StakeLogX", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointSubmitTransactionsIterator is returned from FilterSubmitTransactions and is used to iterate over the raw logs and unpacked data for SubmitTransactions events raised by the Endpoint contract.
type EndpointSubmitTransactionsIterator struct {
	Event *EndpointSubmitTransactions // Event containing the contract specifics and raw log

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
func (it *EndpointSubmitTransactionsIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointSubmitTransactions)
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
		it.Event = new(EndpointSubmitTransactions)
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
func (it *EndpointSubmitTransactionsIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointSubmitTransactionsIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointSubmitTransactions represents a SubmitTransactions event raised by the Endpoint contract.
type EndpointSubmitTransactions struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterSubmitTransactions is a free log retrieval operation binding the contract event 0xaf5fe0f6a71d2001d425268bf9b9f279a74a1cae4d8a0871cc331dae22b76087.
//
// Solidity: event SubmitTransactions()
func (_Endpoint *EndpointFilterer) FilterSubmitTransactions(opts *bind.FilterOpts) (*EndpointSubmitTransactionsIterator, error) {

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "SubmitTransactions")
	if err != nil {
		return nil, err
	}
	return &EndpointSubmitTransactionsIterator{contract: _Endpoint.contract, event: "SubmitTransactions", logs: logs, sub: sub}, nil
}

// WatchSubmitTransactions is a free log subscription operation binding the contract event 0xaf5fe0f6a71d2001d425268bf9b9f279a74a1cae4d8a0871cc331dae22b76087.
//
// Solidity: event SubmitTransactions()
func (_Endpoint *EndpointFilterer) WatchSubmitTransactions(opts *bind.WatchOpts, sink chan<- *EndpointSubmitTransactions) (event.Subscription, error) {

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "SubmitTransactions")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointSubmitTransactions)
				if err := _Endpoint.contract.UnpackLog(event, "SubmitTransactions", log); err != nil {
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

// ParseSubmitTransactions is a log parse operation binding the contract event 0xaf5fe0f6a71d2001d425268bf9b9f279a74a1cae4d8a0871cc331dae22b76087.
//
// Solidity: event SubmitTransactions()
func (_Endpoint *EndpointFilterer) ParseSubmitTransactions(log types.Log) (*EndpointSubmitTransactions, error) {
	event := new(EndpointSubmitTransactions)
	if err := _Endpoint.contract.UnpackLog(event, "SubmitTransactions", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// EndpointUnstakeLogXIterator is returned from FilterUnstakeLogX and is used to iterate over the raw logs and unpacked data for UnstakeLogX events raised by the Endpoint contract.
type EndpointUnstakeLogXIterator struct {
	Event *EndpointUnstakeLogX // Event containing the contract specifics and raw log

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
func (it *EndpointUnstakeLogXIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(EndpointUnstakeLogX)
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
		it.Event = new(EndpointUnstakeLogX)
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
func (it *EndpointUnstakeLogXIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *EndpointUnstakeLogXIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// EndpointUnstakeLogX represents a UnstakeLogX event raised by the Endpoint contract.
type EndpointUnstakeLogX struct {
	SubAccountId  [32]byte
	UserAddress   common.Address
	StakeId       [32]byte
	UnstakeAmount *big.Int
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterUnstakeLogX is a free log retrieval operation binding the contract event 0x912fcf6b1cb49f2c230722c66135ccadbbcc71a170cc92861568e71d0e89a20e.
//
// Solidity: event UnstakeLogX(bytes32 indexed subAccountId, address indexed userAddress, bytes32 stakeId, uint256 indexed unstakeAmount)
func (_Endpoint *EndpointFilterer) FilterUnstakeLogX(opts *bind.FilterOpts, subAccountId [][32]byte, userAddress []common.Address, unstakeAmount []*big.Int) (*EndpointUnstakeLogXIterator, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}

	var unstakeAmountRule []interface{}
	for _, unstakeAmountItem := range unstakeAmount {
		unstakeAmountRule = append(unstakeAmountRule, unstakeAmountItem)
	}

	logs, sub, err := _Endpoint.contract.FilterLogs(opts, "UnstakeLogX", subAccountIdRule, userAddressRule, unstakeAmountRule)
	if err != nil {
		return nil, err
	}
	return &EndpointUnstakeLogXIterator{contract: _Endpoint.contract, event: "UnstakeLogX", logs: logs, sub: sub}, nil
}

// WatchUnstakeLogX is a free log subscription operation binding the contract event 0x912fcf6b1cb49f2c230722c66135ccadbbcc71a170cc92861568e71d0e89a20e.
//
// Solidity: event UnstakeLogX(bytes32 indexed subAccountId, address indexed userAddress, bytes32 stakeId, uint256 indexed unstakeAmount)
func (_Endpoint *EndpointFilterer) WatchUnstakeLogX(opts *bind.WatchOpts, sink chan<- *EndpointUnstakeLogX, subAccountId [][32]byte, userAddress []common.Address, unstakeAmount []*big.Int) (event.Subscription, error) {

	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}

	var unstakeAmountRule []interface{}
	for _, unstakeAmountItem := range unstakeAmount {
		unstakeAmountRule = append(unstakeAmountRule, unstakeAmountItem)
	}

	logs, sub, err := _Endpoint.contract.WatchLogs(opts, "UnstakeLogX", subAccountIdRule, userAddressRule, unstakeAmountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(EndpointUnstakeLogX)
				if err := _Endpoint.contract.UnpackLog(event, "UnstakeLogX", log); err != nil {
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

// ParseUnstakeLogX is a log parse operation binding the contract event 0x912fcf6b1cb49f2c230722c66135ccadbbcc71a170cc92861568e71d0e89a20e.
//
// Solidity: event UnstakeLogX(bytes32 indexed subAccountId, address indexed userAddress, bytes32 stakeId, uint256 indexed unstakeAmount)
func (_Endpoint *EndpointFilterer) ParseUnstakeLogX(log types.Log) (*EndpointUnstakeLogX, error) {
	event := new(EndpointUnstakeLogX)
	if err := _Endpoint.contract.UnpackLog(event, "UnstakeLogX", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
