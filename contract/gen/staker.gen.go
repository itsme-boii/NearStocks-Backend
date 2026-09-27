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

// StakerMetaData contains all meta data concerning the Staker contract.
var StakerMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"BASIS_POINTS_DIVISOR\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"PRECISION\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"apyForDuration\",\"inputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"_account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"balances\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"claimTokens\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"claimTokensForAccount\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_receiver\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"claimableTokens\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"cumulativeFeeRewardPerToken\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"cumulativeTokens\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAccountForStakeId\",\"inputs\":[{\"name\":\"stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getAmountForStakeId\",\"inputs\":[{\"name\":\"stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getStakeIdRewards\",\"inputs\":[{\"name\":\"stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getUnclaimedUserRewards\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getUserIds\",\"inputs\":[{\"name\":\"_user\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\",\"internalType\":\"bytes32[]\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"inPrivateClaimingMode\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"inPrivateStakingMode\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"inPrivateTransferMode\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_name\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"_symbol\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"isHandler\",\"inputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"isStakeActive\",\"inputs\":[{\"name\":\"stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"receive\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"restake\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"restakeForAccount\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setAPRForDurationInDays\",\"inputs\":[{\"name\":\"_duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_apy\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setHandler\",\"inputs\":[{\"name\":\"_handler\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_isActive\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInPrivateClaimingMode\",\"inputs\":[{\"name\":\"_inPrivateClaimingMode\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInPrivateStakingMode\",\"inputs\":[{\"name\":\"_inPrivateStakingMode\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setInPrivateTransferMode\",\"inputs\":[{\"name\":\"_inPrivateTransferMode\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stake\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stakeForAccount\",\"inputs\":[{\"name\":\"_fundingAccount\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"_duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stakedAmounts\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"stakes\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"apy\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"startTime\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalDepositSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"recipient\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unstake\",\"inputs\":[{\"name\":\"account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unstakeForAccount\",\"inputs\":[{\"name\":\"_account\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"_receiver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"userIds\",\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"withdrawToken\",\"inputs\":[{\"name\":\"_token\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_account\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Claim\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"bytes32\",\"indexed\":false,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ReentrancyGuardReentrantCall\",\"inputs\":[]}]",
}

// StakerABI is the input ABI used to generate the binding from.
// Deprecated: Use StakerMetaData.ABI instead.
var StakerABI = StakerMetaData.ABI

// Staker is an auto generated Go binding around an Ethereum contract.
type Staker struct {
	StakerCaller     // Read-only binding to the contract
	StakerTransactor // Write-only binding to the contract
	StakerFilterer   // Log filterer for contract events
}

// StakerCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakerCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakerTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakerFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakerSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakerSession struct {
	Contract     *Staker           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakerCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakerCallerSession struct {
	Contract *StakerCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// StakerTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakerTransactorSession struct {
	Contract     *StakerTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakerRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakerRaw struct {
	Contract *Staker // Generic contract binding to access the raw methods on
}

// StakerCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakerCallerRaw struct {
	Contract *StakerCaller // Generic read-only contract binding to access the raw methods on
}

// StakerTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakerTransactorRaw struct {
	Contract *StakerTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStaker creates a new instance of Staker, bound to a specific deployed contract.
func NewStaker(address common.Address, backend bind.ContractBackend) (*Staker, error) {
	contract, err := bindStaker(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Staker{StakerCaller: StakerCaller{contract: contract}, StakerTransactor: StakerTransactor{contract: contract}, StakerFilterer: StakerFilterer{contract: contract}}, nil
}

// NewStakerCaller creates a new read-only instance of Staker, bound to a specific deployed contract.
func NewStakerCaller(address common.Address, caller bind.ContractCaller) (*StakerCaller, error) {
	contract, err := bindStaker(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakerCaller{contract: contract}, nil
}

// NewStakerTransactor creates a new write-only instance of Staker, bound to a specific deployed contract.
func NewStakerTransactor(address common.Address, transactor bind.ContractTransactor) (*StakerTransactor, error) {
	contract, err := bindStaker(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakerTransactor{contract: contract}, nil
}

// NewStakerFilterer creates a new log filterer instance of Staker, bound to a specific deployed contract.
func NewStakerFilterer(address common.Address, filterer bind.ContractFilterer) (*StakerFilterer, error) {
	contract, err := bindStaker(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakerFilterer{contract: contract}, nil
}

// bindStaker binds a generic wrapper to an already deployed contract.
func bindStaker(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StakerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staker *StakerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staker.Contract.StakerCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staker *StakerRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.Contract.StakerTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staker *StakerRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staker.Contract.StakerTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Staker *StakerCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Staker.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Staker *StakerTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Staker *StakerTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Staker.Contract.contract.Transact(opts, method, params...)
}

// BASISPOINTSDIVISOR is a free data retrieval call binding the contract method 0x126082cf.
//
// Solidity: function BASIS_POINTS_DIVISOR() view returns(uint256)
func (_Staker *StakerCaller) BASISPOINTSDIVISOR(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "BASIS_POINTS_DIVISOR")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BASISPOINTSDIVISOR is a free data retrieval call binding the contract method 0x126082cf.
//
// Solidity: function BASIS_POINTS_DIVISOR() view returns(uint256)
func (_Staker *StakerSession) BASISPOINTSDIVISOR() (*big.Int, error) {
	return _Staker.Contract.BASISPOINTSDIVISOR(&_Staker.CallOpts)
}

// BASISPOINTSDIVISOR is a free data retrieval call binding the contract method 0x126082cf.
//
// Solidity: function BASIS_POINTS_DIVISOR() view returns(uint256)
func (_Staker *StakerCallerSession) BASISPOINTSDIVISOR() (*big.Int, error) {
	return _Staker.Contract.BASISPOINTSDIVISOR(&_Staker.CallOpts)
}

// PRECISION is a free data retrieval call binding the contract method 0xaaf5eb68.
//
// Solidity: function PRECISION() view returns(uint256)
func (_Staker *StakerCaller) PRECISION(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "PRECISION")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// PRECISION is a free data retrieval call binding the contract method 0xaaf5eb68.
//
// Solidity: function PRECISION() view returns(uint256)
func (_Staker *StakerSession) PRECISION() (*big.Int, error) {
	return _Staker.Contract.PRECISION(&_Staker.CallOpts)
}

// PRECISION is a free data retrieval call binding the contract method 0xaaf5eb68.
//
// Solidity: function PRECISION() view returns(uint256)
func (_Staker *StakerCallerSession) PRECISION() (*big.Int, error) {
	return _Staker.Contract.PRECISION(&_Staker.CallOpts)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Staker *StakerCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Staker *StakerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Staker.Contract.Allowance(&_Staker.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_Staker *StakerCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _Staker.Contract.Allowance(&_Staker.CallOpts, owner, spender)
}

// ApyForDuration is a free data retrieval call binding the contract method 0xff62f34c.
//
// Solidity: function apyForDuration(uint256 ) view returns(uint256)
func (_Staker *StakerCaller) ApyForDuration(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "apyForDuration", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ApyForDuration is a free data retrieval call binding the contract method 0xff62f34c.
//
// Solidity: function apyForDuration(uint256 ) view returns(uint256)
func (_Staker *StakerSession) ApyForDuration(arg0 *big.Int) (*big.Int, error) {
	return _Staker.Contract.ApyForDuration(&_Staker.CallOpts, arg0)
}

// ApyForDuration is a free data retrieval call binding the contract method 0xff62f34c.
//
// Solidity: function apyForDuration(uint256 ) view returns(uint256)
func (_Staker *StakerCallerSession) ApyForDuration(arg0 *big.Int) (*big.Int, error) {
	return _Staker.Contract.ApyForDuration(&_Staker.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_Staker *StakerCaller) BalanceOf(opts *bind.CallOpts, _account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "balanceOf", _account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_Staker *StakerSession) BalanceOf(_account common.Address) (*big.Int, error) {
	return _Staker.Contract.BalanceOf(&_Staker.CallOpts, _account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _account) view returns(uint256)
func (_Staker *StakerCallerSession) BalanceOf(_account common.Address) (*big.Int, error) {
	return _Staker.Contract.BalanceOf(&_Staker.CallOpts, _account)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_Staker *StakerCaller) Balances(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "balances", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_Staker *StakerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _Staker.Contract.Balances(&_Staker.CallOpts, arg0)
}

// Balances is a free data retrieval call binding the contract method 0x27e235e3.
//
// Solidity: function balances(address ) view returns(uint256)
func (_Staker *StakerCallerSession) Balances(arg0 common.Address) (*big.Int, error) {
	return _Staker.Contract.Balances(&_Staker.CallOpts, arg0)
}

// ClaimableTokens is a free data retrieval call binding the contract method 0xe91220e5.
//
// Solidity: function claimableTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerCaller) ClaimableTokens(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "claimableTokens", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ClaimableTokens is a free data retrieval call binding the contract method 0xe91220e5.
//
// Solidity: function claimableTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerSession) ClaimableTokens(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.ClaimableTokens(&_Staker.CallOpts, arg0)
}

// ClaimableTokens is a free data retrieval call binding the contract method 0xe91220e5.
//
// Solidity: function claimableTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerCallerSession) ClaimableTokens(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.ClaimableTokens(&_Staker.CallOpts, arg0)
}

// CumulativeFeeRewardPerToken is a free data retrieval call binding the contract method 0xd5e44069.
//
// Solidity: function cumulativeFeeRewardPerToken() view returns(uint256)
func (_Staker *StakerCaller) CumulativeFeeRewardPerToken(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "cumulativeFeeRewardPerToken")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CumulativeFeeRewardPerToken is a free data retrieval call binding the contract method 0xd5e44069.
//
// Solidity: function cumulativeFeeRewardPerToken() view returns(uint256)
func (_Staker *StakerSession) CumulativeFeeRewardPerToken() (*big.Int, error) {
	return _Staker.Contract.CumulativeFeeRewardPerToken(&_Staker.CallOpts)
}

// CumulativeFeeRewardPerToken is a free data retrieval call binding the contract method 0xd5e44069.
//
// Solidity: function cumulativeFeeRewardPerToken() view returns(uint256)
func (_Staker *StakerCallerSession) CumulativeFeeRewardPerToken() (*big.Int, error) {
	return _Staker.Contract.CumulativeFeeRewardPerToken(&_Staker.CallOpts)
}

// CumulativeTokens is a free data retrieval call binding the contract method 0x68837e88.
//
// Solidity: function cumulativeTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerCaller) CumulativeTokens(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "cumulativeTokens", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CumulativeTokens is a free data retrieval call binding the contract method 0x68837e88.
//
// Solidity: function cumulativeTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerSession) CumulativeTokens(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.CumulativeTokens(&_Staker.CallOpts, arg0)
}

// CumulativeTokens is a free data retrieval call binding the contract method 0x68837e88.
//
// Solidity: function cumulativeTokens(bytes32 ) view returns(uint256)
func (_Staker *StakerCallerSession) CumulativeTokens(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.CumulativeTokens(&_Staker.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Staker *StakerCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Staker *StakerSession) Decimals() (uint8, error) {
	return _Staker.Contract.Decimals(&_Staker.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_Staker *StakerCallerSession) Decimals() (uint8, error) {
	return _Staker.Contract.Decimals(&_Staker.CallOpts)
}

// GetAccountForStakeId is a free data retrieval call binding the contract method 0x86a3a843.
//
// Solidity: function getAccountForStakeId(bytes32 stakeId) view returns(bytes32)
func (_Staker *StakerCaller) GetAccountForStakeId(opts *bind.CallOpts, stakeId [32]byte) ([32]byte, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getAccountForStakeId", stakeId)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetAccountForStakeId is a free data retrieval call binding the contract method 0x86a3a843.
//
// Solidity: function getAccountForStakeId(bytes32 stakeId) view returns(bytes32)
func (_Staker *StakerSession) GetAccountForStakeId(stakeId [32]byte) ([32]byte, error) {
	return _Staker.Contract.GetAccountForStakeId(&_Staker.CallOpts, stakeId)
}

// GetAccountForStakeId is a free data retrieval call binding the contract method 0x86a3a843.
//
// Solidity: function getAccountForStakeId(bytes32 stakeId) view returns(bytes32)
func (_Staker *StakerCallerSession) GetAccountForStakeId(stakeId [32]byte) ([32]byte, error) {
	return _Staker.Contract.GetAccountForStakeId(&_Staker.CallOpts, stakeId)
}

// GetAmountForStakeId is a free data retrieval call binding the contract method 0x6b0988e1.
//
// Solidity: function getAmountForStakeId(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerCaller) GetAmountForStakeId(opts *bind.CallOpts, stakeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getAmountForStakeId", stakeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetAmountForStakeId is a free data retrieval call binding the contract method 0x6b0988e1.
//
// Solidity: function getAmountForStakeId(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerSession) GetAmountForStakeId(stakeId [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetAmountForStakeId(&_Staker.CallOpts, stakeId)
}

// GetAmountForStakeId is a free data retrieval call binding the contract method 0x6b0988e1.
//
// Solidity: function getAmountForStakeId(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerCallerSession) GetAmountForStakeId(stakeId [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetAmountForStakeId(&_Staker.CallOpts, stakeId)
}

// GetStakeIdRewards is a free data retrieval call binding the contract method 0x616b8e65.
//
// Solidity: function getStakeIdRewards(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerCaller) GetStakeIdRewards(opts *bind.CallOpts, stakeId [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getStakeIdRewards", stakeId)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetStakeIdRewards is a free data retrieval call binding the contract method 0x616b8e65.
//
// Solidity: function getStakeIdRewards(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerSession) GetStakeIdRewards(stakeId [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetStakeIdRewards(&_Staker.CallOpts, stakeId)
}

// GetStakeIdRewards is a free data retrieval call binding the contract method 0x616b8e65.
//
// Solidity: function getStakeIdRewards(bytes32 stakeId) view returns(uint256)
func (_Staker *StakerCallerSession) GetStakeIdRewards(stakeId [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetStakeIdRewards(&_Staker.CallOpts, stakeId)
}

// GetUnclaimedUserRewards is a free data retrieval call binding the contract method 0x90359a8d.
//
// Solidity: function getUnclaimedUserRewards(bytes32 _account) view returns(uint256)
func (_Staker *StakerCaller) GetUnclaimedUserRewards(opts *bind.CallOpts, _account [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getUnclaimedUserRewards", _account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetUnclaimedUserRewards is a free data retrieval call binding the contract method 0x90359a8d.
//
// Solidity: function getUnclaimedUserRewards(bytes32 _account) view returns(uint256)
func (_Staker *StakerSession) GetUnclaimedUserRewards(_account [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetUnclaimedUserRewards(&_Staker.CallOpts, _account)
}

// GetUnclaimedUserRewards is a free data retrieval call binding the contract method 0x90359a8d.
//
// Solidity: function getUnclaimedUserRewards(bytes32 _account) view returns(uint256)
func (_Staker *StakerCallerSession) GetUnclaimedUserRewards(_account [32]byte) (*big.Int, error) {
	return _Staker.Contract.GetUnclaimedUserRewards(&_Staker.CallOpts, _account)
}

// GetUserIds is a free data retrieval call binding the contract method 0x9246ebef.
//
// Solidity: function getUserIds(bytes32 _user) view returns(bytes32[])
func (_Staker *StakerCaller) GetUserIds(opts *bind.CallOpts, _user [32]byte) ([][32]byte, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "getUserIds", _user)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetUserIds is a free data retrieval call binding the contract method 0x9246ebef.
//
// Solidity: function getUserIds(bytes32 _user) view returns(bytes32[])
func (_Staker *StakerSession) GetUserIds(_user [32]byte) ([][32]byte, error) {
	return _Staker.Contract.GetUserIds(&_Staker.CallOpts, _user)
}

// GetUserIds is a free data retrieval call binding the contract method 0x9246ebef.
//
// Solidity: function getUserIds(bytes32 _user) view returns(bytes32[])
func (_Staker *StakerCallerSession) GetUserIds(_user [32]byte) ([][32]byte, error) {
	return _Staker.Contract.GetUserIds(&_Staker.CallOpts, _user)
}

// InPrivateClaimingMode is a free data retrieval call binding the contract method 0xf76033d3.
//
// Solidity: function inPrivateClaimingMode() view returns(bool)
func (_Staker *StakerCaller) InPrivateClaimingMode(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "inPrivateClaimingMode")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// InPrivateClaimingMode is a free data retrieval call binding the contract method 0xf76033d3.
//
// Solidity: function inPrivateClaimingMode() view returns(bool)
func (_Staker *StakerSession) InPrivateClaimingMode() (bool, error) {
	return _Staker.Contract.InPrivateClaimingMode(&_Staker.CallOpts)
}

// InPrivateClaimingMode is a free data retrieval call binding the contract method 0xf76033d3.
//
// Solidity: function inPrivateClaimingMode() view returns(bool)
func (_Staker *StakerCallerSession) InPrivateClaimingMode() (bool, error) {
	return _Staker.Contract.InPrivateClaimingMode(&_Staker.CallOpts)
}

// InPrivateStakingMode is a free data retrieval call binding the contract method 0xc5fa2730.
//
// Solidity: function inPrivateStakingMode() view returns(bool)
func (_Staker *StakerCaller) InPrivateStakingMode(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "inPrivateStakingMode")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// InPrivateStakingMode is a free data retrieval call binding the contract method 0xc5fa2730.
//
// Solidity: function inPrivateStakingMode() view returns(bool)
func (_Staker *StakerSession) InPrivateStakingMode() (bool, error) {
	return _Staker.Contract.InPrivateStakingMode(&_Staker.CallOpts)
}

// InPrivateStakingMode is a free data retrieval call binding the contract method 0xc5fa2730.
//
// Solidity: function inPrivateStakingMode() view returns(bool)
func (_Staker *StakerCallerSession) InPrivateStakingMode() (bool, error) {
	return _Staker.Contract.InPrivateStakingMode(&_Staker.CallOpts)
}

// InPrivateTransferMode is a free data retrieval call binding the contract method 0xdfbaefb1.
//
// Solidity: function inPrivateTransferMode() view returns(bool)
func (_Staker *StakerCaller) InPrivateTransferMode(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "inPrivateTransferMode")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// InPrivateTransferMode is a free data retrieval call binding the contract method 0xdfbaefb1.
//
// Solidity: function inPrivateTransferMode() view returns(bool)
func (_Staker *StakerSession) InPrivateTransferMode() (bool, error) {
	return _Staker.Contract.InPrivateTransferMode(&_Staker.CallOpts)
}

// InPrivateTransferMode is a free data retrieval call binding the contract method 0xdfbaefb1.
//
// Solidity: function inPrivateTransferMode() view returns(bool)
func (_Staker *StakerCallerSession) InPrivateTransferMode() (bool, error) {
	return _Staker.Contract.InPrivateTransferMode(&_Staker.CallOpts)
}

// IsHandler is a free data retrieval call binding the contract method 0x46ea87af.
//
// Solidity: function isHandler(address ) view returns(bool)
func (_Staker *StakerCaller) IsHandler(opts *bind.CallOpts, arg0 common.Address) (bool, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "isHandler", arg0)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsHandler is a free data retrieval call binding the contract method 0x46ea87af.
//
// Solidity: function isHandler(address ) view returns(bool)
func (_Staker *StakerSession) IsHandler(arg0 common.Address) (bool, error) {
	return _Staker.Contract.IsHandler(&_Staker.CallOpts, arg0)
}

// IsHandler is a free data retrieval call binding the contract method 0x46ea87af.
//
// Solidity: function isHandler(address ) view returns(bool)
func (_Staker *StakerCallerSession) IsHandler(arg0 common.Address) (bool, error) {
	return _Staker.Contract.IsHandler(&_Staker.CallOpts, arg0)
}

// IsStakeActive is a free data retrieval call binding the contract method 0x94fa2b8c.
//
// Solidity: function isStakeActive(bytes32 stakeId) view returns(bool)
func (_Staker *StakerCaller) IsStakeActive(opts *bind.CallOpts, stakeId [32]byte) (bool, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "isStakeActive", stakeId)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsStakeActive is a free data retrieval call binding the contract method 0x94fa2b8c.
//
// Solidity: function isStakeActive(bytes32 stakeId) view returns(bool)
func (_Staker *StakerSession) IsStakeActive(stakeId [32]byte) (bool, error) {
	return _Staker.Contract.IsStakeActive(&_Staker.CallOpts, stakeId)
}

// IsStakeActive is a free data retrieval call binding the contract method 0x94fa2b8c.
//
// Solidity: function isStakeActive(bytes32 stakeId) view returns(bool)
func (_Staker *StakerCallerSession) IsStakeActive(stakeId [32]byte) (bool, error) {
	return _Staker.Contract.IsStakeActive(&_Staker.CallOpts, stakeId)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Staker *StakerCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Staker *StakerSession) Name() (string, error) {
	return _Staker.Contract.Name(&_Staker.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_Staker *StakerCallerSession) Name() (string, error) {
	return _Staker.Contract.Name(&_Staker.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Staker *StakerCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Staker *StakerSession) Owner() (common.Address, error) {
	return _Staker.Contract.Owner(&_Staker.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Staker *StakerCallerSession) Owner() (common.Address, error) {
	return _Staker.Contract.Owner(&_Staker.CallOpts)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xd5744a2f.
//
// Solidity: function stakedAmounts(bytes32 ) view returns(uint256)
func (_Staker *StakerCaller) StakedAmounts(opts *bind.CallOpts, arg0 [32]byte) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "stakedAmounts", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakedAmounts is a free data retrieval call binding the contract method 0xd5744a2f.
//
// Solidity: function stakedAmounts(bytes32 ) view returns(uint256)
func (_Staker *StakerSession) StakedAmounts(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.StakedAmounts(&_Staker.CallOpts, arg0)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xd5744a2f.
//
// Solidity: function stakedAmounts(bytes32 ) view returns(uint256)
func (_Staker *StakerCallerSession) StakedAmounts(arg0 [32]byte) (*big.Int, error) {
	return _Staker.Contract.StakedAmounts(&_Staker.CallOpts, arg0)
}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 account, uint256 amount, uint256 duration, uint256 apy, uint256 startTime)
func (_Staker *StakerCaller) Stakes(opts *bind.CallOpts, arg0 [32]byte) (struct {
	Account   [32]byte
	Amount    *big.Int
	Duration  *big.Int
	Apy       *big.Int
	StartTime *big.Int
}, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "stakes", arg0)

	outstruct := new(struct {
		Account   [32]byte
		Amount    *big.Int
		Duration  *big.Int
		Apy       *big.Int
		StartTime *big.Int
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Account = *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	outstruct.Amount = *abi.ConvertType(out[1], new(*big.Int)).(**big.Int)
	outstruct.Duration = *abi.ConvertType(out[2], new(*big.Int)).(**big.Int)
	outstruct.Apy = *abi.ConvertType(out[3], new(*big.Int)).(**big.Int)
	outstruct.StartTime = *abi.ConvertType(out[4], new(*big.Int)).(**big.Int)

	return *outstruct, err

}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 account, uint256 amount, uint256 duration, uint256 apy, uint256 startTime)
func (_Staker *StakerSession) Stakes(arg0 [32]byte) (struct {
	Account   [32]byte
	Amount    *big.Int
	Duration  *big.Int
	Apy       *big.Int
	StartTime *big.Int
}, error) {
	return _Staker.Contract.Stakes(&_Staker.CallOpts, arg0)
}

// Stakes is a free data retrieval call binding the contract method 0x8fee6407.
//
// Solidity: function stakes(bytes32 ) view returns(bytes32 account, uint256 amount, uint256 duration, uint256 apy, uint256 startTime)
func (_Staker *StakerCallerSession) Stakes(arg0 [32]byte) (struct {
	Account   [32]byte
	Amount    *big.Int
	Duration  *big.Int
	Apy       *big.Int
	StartTime *big.Int
}, error) {
	return _Staker.Contract.Stakes(&_Staker.CallOpts, arg0)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Staker *StakerCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Staker *StakerSession) Symbol() (string, error) {
	return _Staker.Contract.Symbol(&_Staker.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_Staker *StakerCallerSession) Symbol() (string, error) {
	return _Staker.Contract.Symbol(&_Staker.CallOpts)
}

// TotalDepositSupply is a free data retrieval call binding the contract method 0x661bbcc3.
//
// Solidity: function totalDepositSupply() view returns(uint256)
func (_Staker *StakerCaller) TotalDepositSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "totalDepositSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalDepositSupply is a free data retrieval call binding the contract method 0x661bbcc3.
//
// Solidity: function totalDepositSupply() view returns(uint256)
func (_Staker *StakerSession) TotalDepositSupply() (*big.Int, error) {
	return _Staker.Contract.TotalDepositSupply(&_Staker.CallOpts)
}

// TotalDepositSupply is a free data retrieval call binding the contract method 0x661bbcc3.
//
// Solidity: function totalDepositSupply() view returns(uint256)
func (_Staker *StakerCallerSession) TotalDepositSupply() (*big.Int, error) {
	return _Staker.Contract.TotalDepositSupply(&_Staker.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Staker *StakerCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Staker *StakerSession) TotalSupply() (*big.Int, error) {
	return _Staker.Contract.TotalSupply(&_Staker.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_Staker *StakerCallerSession) TotalSupply() (*big.Int, error) {
	return _Staker.Contract.TotalSupply(&_Staker.CallOpts)
}

// UserIds is a free data retrieval call binding the contract method 0x3ad78350.
//
// Solidity: function userIds(bytes32 , uint256 ) view returns(bytes32)
func (_Staker *StakerCaller) UserIds(opts *bind.CallOpts, arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Staker.contract.Call(opts, &out, "userIds", arg0, arg1)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// UserIds is a free data retrieval call binding the contract method 0x3ad78350.
//
// Solidity: function userIds(bytes32 , uint256 ) view returns(bytes32)
func (_Staker *StakerSession) UserIds(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _Staker.Contract.UserIds(&_Staker.CallOpts, arg0, arg1)
}

// UserIds is a free data retrieval call binding the contract method 0x3ad78350.
//
// Solidity: function userIds(bytes32 , uint256 ) view returns(bytes32)
func (_Staker *StakerCallerSession) UserIds(arg0 [32]byte, arg1 *big.Int) ([32]byte, error) {
	return _Staker.Contract.UserIds(&_Staker.CallOpts, arg0, arg1)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Staker *StakerTransactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Staker *StakerSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Approve(&_Staker.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_Staker *StakerTransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Approve(&_Staker.TransactOpts, spender, amount)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0x6facd1c0.
//
// Solidity: function claimTokens(bytes32 _account) returns(uint256)
func (_Staker *StakerTransactor) ClaimTokens(opts *bind.TransactOpts, _account [32]byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "claimTokens", _account)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0x6facd1c0.
//
// Solidity: function claimTokens(bytes32 _account) returns(uint256)
func (_Staker *StakerSession) ClaimTokens(_account [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.ClaimTokens(&_Staker.TransactOpts, _account)
}

// ClaimTokens is a paid mutator transaction binding the contract method 0x6facd1c0.
//
// Solidity: function claimTokens(bytes32 _account) returns(uint256)
func (_Staker *StakerTransactorSession) ClaimTokens(_account [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.ClaimTokens(&_Staker.TransactOpts, _account)
}

// ClaimTokensForAccount is a paid mutator transaction binding the contract method 0x7c984652.
//
// Solidity: function claimTokensForAccount(bytes32 _account, address _receiver) returns(uint256)
func (_Staker *StakerTransactor) ClaimTokensForAccount(opts *bind.TransactOpts, _account [32]byte, _receiver common.Address) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "claimTokensForAccount", _account, _receiver)
}

// ClaimTokensForAccount is a paid mutator transaction binding the contract method 0x7c984652.
//
// Solidity: function claimTokensForAccount(bytes32 _account, address _receiver) returns(uint256)
func (_Staker *StakerSession) ClaimTokensForAccount(_account [32]byte, _receiver common.Address) (*types.Transaction, error) {
	return _Staker.Contract.ClaimTokensForAccount(&_Staker.TransactOpts, _account, _receiver)
}

// ClaimTokensForAccount is a paid mutator transaction binding the contract method 0x7c984652.
//
// Solidity: function claimTokensForAccount(bytes32 _account, address _receiver) returns(uint256)
func (_Staker *StakerTransactorSession) ClaimTokensForAccount(_account [32]byte, _receiver common.Address) (*types.Transaction, error) {
	return _Staker.Contract.ClaimTokensForAccount(&_Staker.TransactOpts, _account, _receiver)
}

// Initialize is a paid mutator transaction binding the contract method 0x4cd88b76.
//
// Solidity: function initialize(string _name, string _symbol) returns()
func (_Staker *StakerTransactor) Initialize(opts *bind.TransactOpts, _name string, _symbol string) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "initialize", _name, _symbol)
}

// Initialize is a paid mutator transaction binding the contract method 0x4cd88b76.
//
// Solidity: function initialize(string _name, string _symbol) returns()
func (_Staker *StakerSession) Initialize(_name string, _symbol string) (*types.Transaction, error) {
	return _Staker.Contract.Initialize(&_Staker.TransactOpts, _name, _symbol)
}

// Initialize is a paid mutator transaction binding the contract method 0x4cd88b76.
//
// Solidity: function initialize(string _name, string _symbol) returns()
func (_Staker *StakerTransactorSession) Initialize(_name string, _symbol string) (*types.Transaction, error) {
	return _Staker.Contract.Initialize(&_Staker.TransactOpts, _name, _symbol)
}

// Receive is a paid mutator transaction binding the contract method 0xa3e76c0f.
//
// Solidity: function receive() payable returns()
func (_Staker *StakerTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "receive")
}

// Receive is a paid mutator transaction binding the contract method 0xa3e76c0f.
//
// Solidity: function receive() payable returns()
func (_Staker *StakerSession) Receive() (*types.Transaction, error) {
	return _Staker.Contract.Receive(&_Staker.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract method 0xa3e76c0f.
//
// Solidity: function receive() payable returns()
func (_Staker *StakerTransactorSession) Receive() (*types.Transaction, error) {
	return _Staker.Contract.Receive(&_Staker.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Staker *StakerTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Staker *StakerSession) RenounceOwnership() (*types.Transaction, error) {
	return _Staker.Contract.RenounceOwnership(&_Staker.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Staker *StakerTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Staker.Contract.RenounceOwnership(&_Staker.TransactOpts)
}

// Restake is a paid mutator transaction binding the contract method 0xd22fd7af.
//
// Solidity: function restake(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerTransactor) Restake(opts *bind.TransactOpts, _account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "restake", _account, _stakeId, _duration)
}

// Restake is a paid mutator transaction binding the contract method 0xd22fd7af.
//
// Solidity: function restake(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerSession) Restake(_account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Restake(&_Staker.TransactOpts, _account, _stakeId, _duration)
}

// Restake is a paid mutator transaction binding the contract method 0xd22fd7af.
//
// Solidity: function restake(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerTransactorSession) Restake(_account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Restake(&_Staker.TransactOpts, _account, _stakeId, _duration)
}

// RestakeForAccount is a paid mutator transaction binding the contract method 0x4cf28e08.
//
// Solidity: function restakeForAccount(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerTransactor) RestakeForAccount(opts *bind.TransactOpts, _account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "restakeForAccount", _account, _stakeId, _duration)
}

// RestakeForAccount is a paid mutator transaction binding the contract method 0x4cf28e08.
//
// Solidity: function restakeForAccount(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerSession) RestakeForAccount(_account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.RestakeForAccount(&_Staker.TransactOpts, _account, _stakeId, _duration)
}

// RestakeForAccount is a paid mutator transaction binding the contract method 0x4cf28e08.
//
// Solidity: function restakeForAccount(bytes32 _account, bytes32 _stakeId, uint256 _duration) returns()
func (_Staker *StakerTransactorSession) RestakeForAccount(_account [32]byte, _stakeId [32]byte, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.RestakeForAccount(&_Staker.TransactOpts, _account, _stakeId, _duration)
}

// SetAPRForDurationInDays is a paid mutator transaction binding the contract method 0x24fde20a.
//
// Solidity: function setAPRForDurationInDays(uint256 _duration, uint256 _apy) returns()
func (_Staker *StakerTransactor) SetAPRForDurationInDays(opts *bind.TransactOpts, _duration *big.Int, _apy *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "setAPRForDurationInDays", _duration, _apy)
}

// SetAPRForDurationInDays is a paid mutator transaction binding the contract method 0x24fde20a.
//
// Solidity: function setAPRForDurationInDays(uint256 _duration, uint256 _apy) returns()
func (_Staker *StakerSession) SetAPRForDurationInDays(_duration *big.Int, _apy *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.SetAPRForDurationInDays(&_Staker.TransactOpts, _duration, _apy)
}

// SetAPRForDurationInDays is a paid mutator transaction binding the contract method 0x24fde20a.
//
// Solidity: function setAPRForDurationInDays(uint256 _duration, uint256 _apy) returns()
func (_Staker *StakerTransactorSession) SetAPRForDurationInDays(_duration *big.Int, _apy *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.SetAPRForDurationInDays(&_Staker.TransactOpts, _duration, _apy)
}

// SetHandler is a paid mutator transaction binding the contract method 0x9cb7de4b.
//
// Solidity: function setHandler(address _handler, bool _isActive) returns()
func (_Staker *StakerTransactor) SetHandler(opts *bind.TransactOpts, _handler common.Address, _isActive bool) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "setHandler", _handler, _isActive)
}

// SetHandler is a paid mutator transaction binding the contract method 0x9cb7de4b.
//
// Solidity: function setHandler(address _handler, bool _isActive) returns()
func (_Staker *StakerSession) SetHandler(_handler common.Address, _isActive bool) (*types.Transaction, error) {
	return _Staker.Contract.SetHandler(&_Staker.TransactOpts, _handler, _isActive)
}

// SetHandler is a paid mutator transaction binding the contract method 0x9cb7de4b.
//
// Solidity: function setHandler(address _handler, bool _isActive) returns()
func (_Staker *StakerTransactorSession) SetHandler(_handler common.Address, _isActive bool) (*types.Transaction, error) {
	return _Staker.Contract.SetHandler(&_Staker.TransactOpts, _handler, _isActive)
}

// SetInPrivateClaimingMode is a paid mutator transaction binding the contract method 0x3cd7f700.
//
// Solidity: function setInPrivateClaimingMode(bool _inPrivateClaimingMode) returns()
func (_Staker *StakerTransactor) SetInPrivateClaimingMode(opts *bind.TransactOpts, _inPrivateClaimingMode bool) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "setInPrivateClaimingMode", _inPrivateClaimingMode)
}

// SetInPrivateClaimingMode is a paid mutator transaction binding the contract method 0x3cd7f700.
//
// Solidity: function setInPrivateClaimingMode(bool _inPrivateClaimingMode) returns()
func (_Staker *StakerSession) SetInPrivateClaimingMode(_inPrivateClaimingMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateClaimingMode(&_Staker.TransactOpts, _inPrivateClaimingMode)
}

// SetInPrivateClaimingMode is a paid mutator transaction binding the contract method 0x3cd7f700.
//
// Solidity: function setInPrivateClaimingMode(bool _inPrivateClaimingMode) returns()
func (_Staker *StakerTransactorSession) SetInPrivateClaimingMode(_inPrivateClaimingMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateClaimingMode(&_Staker.TransactOpts, _inPrivateClaimingMode)
}

// SetInPrivateStakingMode is a paid mutator transaction binding the contract method 0x1d30d5bc.
//
// Solidity: function setInPrivateStakingMode(bool _inPrivateStakingMode) returns()
func (_Staker *StakerTransactor) SetInPrivateStakingMode(opts *bind.TransactOpts, _inPrivateStakingMode bool) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "setInPrivateStakingMode", _inPrivateStakingMode)
}

// SetInPrivateStakingMode is a paid mutator transaction binding the contract method 0x1d30d5bc.
//
// Solidity: function setInPrivateStakingMode(bool _inPrivateStakingMode) returns()
func (_Staker *StakerSession) SetInPrivateStakingMode(_inPrivateStakingMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateStakingMode(&_Staker.TransactOpts, _inPrivateStakingMode)
}

// SetInPrivateStakingMode is a paid mutator transaction binding the contract method 0x1d30d5bc.
//
// Solidity: function setInPrivateStakingMode(bool _inPrivateStakingMode) returns()
func (_Staker *StakerTransactorSession) SetInPrivateStakingMode(_inPrivateStakingMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateStakingMode(&_Staker.TransactOpts, _inPrivateStakingMode)
}

// SetInPrivateTransferMode is a paid mutator transaction binding the contract method 0x5a47a1a7.
//
// Solidity: function setInPrivateTransferMode(bool _inPrivateTransferMode) returns()
func (_Staker *StakerTransactor) SetInPrivateTransferMode(opts *bind.TransactOpts, _inPrivateTransferMode bool) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "setInPrivateTransferMode", _inPrivateTransferMode)
}

// SetInPrivateTransferMode is a paid mutator transaction binding the contract method 0x5a47a1a7.
//
// Solidity: function setInPrivateTransferMode(bool _inPrivateTransferMode) returns()
func (_Staker *StakerSession) SetInPrivateTransferMode(_inPrivateTransferMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateTransferMode(&_Staker.TransactOpts, _inPrivateTransferMode)
}

// SetInPrivateTransferMode is a paid mutator transaction binding the contract method 0x5a47a1a7.
//
// Solidity: function setInPrivateTransferMode(bool _inPrivateTransferMode) returns()
func (_Staker *StakerTransactorSession) SetInPrivateTransferMode(_inPrivateTransferMode bool) (*types.Transaction, error) {
	return _Staker.Contract.SetInPrivateTransferMode(&_Staker.TransactOpts, _inPrivateTransferMode)
}

// Stake is a paid mutator transaction binding the contract method 0x2daedd52.
//
// Solidity: function stake(bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerTransactor) Stake(opts *bind.TransactOpts, _account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "stake", _account, _amount, _duration)
}

// Stake is a paid mutator transaction binding the contract method 0x2daedd52.
//
// Solidity: function stake(bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerSession) Stake(_account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Stake(&_Staker.TransactOpts, _account, _amount, _duration)
}

// Stake is a paid mutator transaction binding the contract method 0x2daedd52.
//
// Solidity: function stake(bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerTransactorSession) Stake(_account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Stake(&_Staker.TransactOpts, _account, _amount, _duration)
}

// StakeForAccount is a paid mutator transaction binding the contract method 0x96e363dd.
//
// Solidity: function stakeForAccount(address _fundingAccount, bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerTransactor) StakeForAccount(opts *bind.TransactOpts, _fundingAccount common.Address, _account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "stakeForAccount", _fundingAccount, _account, _amount, _duration)
}

// StakeForAccount is a paid mutator transaction binding the contract method 0x96e363dd.
//
// Solidity: function stakeForAccount(address _fundingAccount, bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerSession) StakeForAccount(_fundingAccount common.Address, _account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.StakeForAccount(&_Staker.TransactOpts, _fundingAccount, _account, _amount, _duration)
}

// StakeForAccount is a paid mutator transaction binding the contract method 0x96e363dd.
//
// Solidity: function stakeForAccount(address _fundingAccount, bytes32 _account, uint256 _amount, uint256 _duration) returns()
func (_Staker *StakerTransactorSession) StakeForAccount(_fundingAccount common.Address, _account [32]byte, _amount *big.Int, _duration *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.StakeForAccount(&_Staker.TransactOpts, _fundingAccount, _account, _amount, _duration)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_Staker *StakerTransactor) Transfer(opts *bind.TransactOpts, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "transfer", recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_Staker *StakerSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Transfer(&_Staker.TransactOpts, recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_Staker *StakerTransactorSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.Transfer(&_Staker.TransactOpts, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_Staker *StakerTransactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "transferFrom", sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_Staker *StakerSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.TransferFrom(&_Staker.TransactOpts, sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_Staker *StakerTransactorSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.TransferFrom(&_Staker.TransactOpts, sender, recipient, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Staker *StakerTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Staker *StakerSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Staker.Contract.TransferOwnership(&_Staker.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Staker *StakerTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Staker.Contract.TransferOwnership(&_Staker.TransactOpts, newOwner)
}

// Unstake is a paid mutator transaction binding the contract method 0x767e0e10.
//
// Solidity: function unstake(bytes32 account, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerTransactor) Unstake(opts *bind.TransactOpts, account [32]byte, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "unstake", account, _stakeId)
}

// Unstake is a paid mutator transaction binding the contract method 0x767e0e10.
//
// Solidity: function unstake(bytes32 account, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerSession) Unstake(account [32]byte, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.Unstake(&_Staker.TransactOpts, account, _stakeId)
}

// Unstake is a paid mutator transaction binding the contract method 0x767e0e10.
//
// Solidity: function unstake(bytes32 account, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerTransactorSession) Unstake(account [32]byte, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.Unstake(&_Staker.TransactOpts, account, _stakeId)
}

// UnstakeForAccount is a paid mutator transaction binding the contract method 0x20c704d0.
//
// Solidity: function unstakeForAccount(bytes32 _account, address _receiver, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerTransactor) UnstakeForAccount(opts *bind.TransactOpts, _account [32]byte, _receiver common.Address, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "unstakeForAccount", _account, _receiver, _stakeId)
}

// UnstakeForAccount is a paid mutator transaction binding the contract method 0x20c704d0.
//
// Solidity: function unstakeForAccount(bytes32 _account, address _receiver, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerSession) UnstakeForAccount(_account [32]byte, _receiver common.Address, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.UnstakeForAccount(&_Staker.TransactOpts, _account, _receiver, _stakeId)
}

// UnstakeForAccount is a paid mutator transaction binding the contract method 0x20c704d0.
//
// Solidity: function unstakeForAccount(bytes32 _account, address _receiver, bytes32 _stakeId) returns(uint256)
func (_Staker *StakerTransactorSession) UnstakeForAccount(_account [32]byte, _receiver common.Address, _stakeId [32]byte) (*types.Transaction, error) {
	return _Staker.Contract.UnstakeForAccount(&_Staker.TransactOpts, _account, _receiver, _stakeId)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x01e33667.
//
// Solidity: function withdrawToken(address _token, address _account, uint256 _amount) returns()
func (_Staker *StakerTransactor) WithdrawToken(opts *bind.TransactOpts, _token common.Address, _account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Staker.contract.Transact(opts, "withdrawToken", _token, _account, _amount)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x01e33667.
//
// Solidity: function withdrawToken(address _token, address _account, uint256 _amount) returns()
func (_Staker *StakerSession) WithdrawToken(_token common.Address, _account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.WithdrawToken(&_Staker.TransactOpts, _token, _account, _amount)
}

// WithdrawToken is a paid mutator transaction binding the contract method 0x01e33667.
//
// Solidity: function withdrawToken(address _token, address _account, uint256 _amount) returns()
func (_Staker *StakerTransactorSession) WithdrawToken(_token common.Address, _account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Staker.Contract.WithdrawToken(&_Staker.TransactOpts, _token, _account, _amount)
}

// StakerApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the Staker contract.
type StakerApprovalIterator struct {
	Event *StakerApproval // Event containing the contract specifics and raw log

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
func (it *StakerApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerApproval)
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
		it.Event = new(StakerApproval)
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
func (it *StakerApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerApproval represents a Approval event raised by the Staker contract.
type StakerApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Staker *StakerFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*StakerApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &StakerApprovalIterator{contract: _Staker.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Staker *StakerFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *StakerApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerApproval)
				if err := _Staker.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_Staker *StakerFilterer) ParseApproval(log types.Log) (*StakerApproval, error) {
	event := new(StakerApproval)
	if err := _Staker.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerClaimIterator is returned from FilterClaim and is used to iterate over the raw logs and unpacked data for Claim events raised by the Staker contract.
type StakerClaimIterator struct {
	Event *StakerClaim // Event containing the contract specifics and raw log

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
func (it *StakerClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerClaim)
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
		it.Event = new(StakerClaim)
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
func (it *StakerClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerClaim represents a Claim event raised by the Staker contract.
type StakerClaim struct {
	Receiver [32]byte
	Amount   *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterClaim is a free log retrieval operation binding the contract event 0xac86c2f8a32db75c8fd0ea87c8be8c73c16136f1f4ee544bc56031c6a12d9528.
//
// Solidity: event Claim(bytes32 receiver, uint256 amount)
func (_Staker *StakerFilterer) FilterClaim(opts *bind.FilterOpts) (*StakerClaimIterator, error) {

	logs, sub, err := _Staker.contract.FilterLogs(opts, "Claim")
	if err != nil {
		return nil, err
	}
	return &StakerClaimIterator{contract: _Staker.contract, event: "Claim", logs: logs, sub: sub}, nil
}

// WatchClaim is a free log subscription operation binding the contract event 0xac86c2f8a32db75c8fd0ea87c8be8c73c16136f1f4ee544bc56031c6a12d9528.
//
// Solidity: event Claim(bytes32 receiver, uint256 amount)
func (_Staker *StakerFilterer) WatchClaim(opts *bind.WatchOpts, sink chan<- *StakerClaim) (event.Subscription, error) {

	logs, sub, err := _Staker.contract.WatchLogs(opts, "Claim")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerClaim)
				if err := _Staker.contract.UnpackLog(event, "Claim", log); err != nil {
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

// ParseClaim is a log parse operation binding the contract event 0xac86c2f8a32db75c8fd0ea87c8be8c73c16136f1f4ee544bc56031c6a12d9528.
//
// Solidity: event Claim(bytes32 receiver, uint256 amount)
func (_Staker *StakerFilterer) ParseClaim(log types.Log) (*StakerClaim, error) {
	event := new(StakerClaim)
	if err := _Staker.contract.UnpackLog(event, "Claim", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the Staker contract.
type StakerInitializedIterator struct {
	Event *StakerInitialized // Event containing the contract specifics and raw log

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
func (it *StakerInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerInitialized)
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
		it.Event = new(StakerInitialized)
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
func (it *StakerInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerInitialized represents a Initialized event raised by the Staker contract.
type StakerInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Staker *StakerFilterer) FilterInitialized(opts *bind.FilterOpts) (*StakerInitializedIterator, error) {

	logs, sub, err := _Staker.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &StakerInitializedIterator{contract: _Staker.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_Staker *StakerFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *StakerInitialized) (event.Subscription, error) {

	logs, sub, err := _Staker.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerInitialized)
				if err := _Staker.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_Staker *StakerFilterer) ParseInitialized(log types.Log) (*StakerInitialized, error) {
	event := new(StakerInitialized)
	if err := _Staker.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Staker contract.
type StakerOwnershipTransferredIterator struct {
	Event *StakerOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *StakerOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerOwnershipTransferred)
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
		it.Event = new(StakerOwnershipTransferred)
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
func (it *StakerOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerOwnershipTransferred represents a OwnershipTransferred event raised by the Staker contract.
type StakerOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Staker *StakerFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*StakerOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &StakerOwnershipTransferredIterator{contract: _Staker.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Staker *StakerFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *StakerOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerOwnershipTransferred)
				if err := _Staker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Staker *StakerFilterer) ParseOwnershipTransferred(log types.Log) (*StakerOwnershipTransferred, error) {
	event := new(StakerOwnershipTransferred)
	if err := _Staker.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// StakerTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the Staker contract.
type StakerTransferIterator struct {
	Event *StakerTransfer // Event containing the contract specifics and raw log

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
func (it *StakerTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(StakerTransfer)
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
		it.Event = new(StakerTransfer)
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
func (it *StakerTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *StakerTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// StakerTransfer represents a Transfer event raised by the Staker contract.
type StakerTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Staker *StakerFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*StakerTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Staker.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &StakerTransferIterator{contract: _Staker.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Staker *StakerFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *StakerTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _Staker.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(StakerTransfer)
				if err := _Staker.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_Staker *StakerFilterer) ParseTransfer(log types.Log) (*StakerTransfer, error) {
	event := new(StakerTransfer)
	if err := _Staker.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
