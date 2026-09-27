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

// IClearinghouseWithdrawCollateral is an auto generated low-level Go binding around an user-defined struct.
type IClearinghouseWithdrawCollateral struct {
	SubAccountId       [32]byte
	SessionKey         common.Address
	ProductId          uint32
	Amount             *big.Int
	Nonce              *big.Int
	DestinationChainId *big.Int
	Receiver           common.Address
	ChainId            *big.Int
	SpotProductIds     []uint32
	SpotPricesX18      []*big.Int
}

// IClearinghouseWithdrawLogX is an auto generated low-level Go binding around an user-defined struct.
type IClearinghouseWithdrawLogX struct {
	SubAccountId       [32]byte
	TokenAmount        *big.Int
	SessionKey         common.Address
	DestinationChainId *big.Int
	BridgeOutContract  common.Address
	Nonce              *big.Int
	Receiver           common.Address
	ChainId            *big.Int
}

// IEndpointDepositInsurance is an auto generated low-level Go binding around an user-defined struct.
type IEndpointDepositInsurance struct {
	Amount *big.Int
}

// IEndpointLiquidateSubaccount is an auto generated low-level Go binding around an user-defined struct.
type IEndpointLiquidateSubaccount struct {
	Sender          [32]byte
	Liquidatee      [32]byte
	ProductId       uint32
	IsEncodedSpread bool
	Amount          *big.Int
	Nonce           uint64
}

// IEndpointSettleUserPnl is an auto generated low-level Go binding around an user-defined struct.
type IEndpointSettleUserPnl struct {
	SubAccountId   [32]byte
	SpotProductIds []uint32
	SpotPricesX18  []*big.Int
	SessionKey     common.Address
	Nonce          *big.Int
	ChainId        *big.Int
}

// IEndpointTransferQuote is an auto generated low-level Go binding around an user-defined struct.
type IEndpointTransferQuote struct {
	Sender    [32]byte
	Recipient [32]byte
	Amount    *big.Int
	Nonce     uint64
}

// ClearingHouseMetaData contains all meta data concerning the ClearingHouse contract.
var ClearingHouseMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"receive\",\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"addEngine\",\"inputs\":[{\"name\":\"engine\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"offchainExchange\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"engineType\",\"type\":\"uint8\",\"internalType\":\"enumIProductEngine.EngineType\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"claimRewards\",\"inputs\":[{\"name\":\"stakerContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"subAccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"deductBridgingFees\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"tokenAmount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"depositInsurance\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.DepositInsurance\",\"components\":[{\"name\":\"amount\",\"type\":\"uint128\",\"internalType\":\"uint128\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"getClearinghouseLiq\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEndpoint\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEngineByProduct\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getEngineByType\",\"inputs\":[{\"name\":\"engineType\",\"type\":\"uint8\",\"internalType\":\"enumIProductEngine.EngineType\"}],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getHealth\",\"inputs\":[{\"name\":\"subaccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"healthType\",\"type\":\"uint8\",\"internalType\":\"enumIProductEngine.HealthType\"}],\"outputs\":[{\"name\":\"health\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getInsurance\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"int128\",\"internalType\":\"int128\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getQuote\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getSpreads\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"getVersion\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint64\",\"internalType\":\"uint64\"}],\"stateMutability\":\"pure\"},{\"type\":\"function\",\"name\":\"initialize\",\"inputs\":[{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_quote\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_clearinghouseLiq\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"_spreads\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"liquidateSubaccount\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.LiquidateSubaccount\",\"components\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"liquidatee\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"isEncodedSpread\",\"type\":\"bool\",\"internalType\":\"bool\"},{\"name\":\"amount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"owner\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"address\",\"internalType\":\"address\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"registerProduct\",\"inputs\":[{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"renounceOwnership\",\"inputs\":[],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"setEndpoint\",\"inputs\":[{\"name\":\"_endpoint\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"settleUserPnl\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.SettleUserPnl\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"spotProductIds\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"},{\"name\":\"spotPricesX18\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"stakeLogXForAccount\",\"inputs\":[{\"name\":\"stakerContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"subAccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"tokenAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"duration\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferOwnership\",\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferQuote\",\"inputs\":[{\"name\":\"txn\",\"type\":\"tuple\",\"internalType\":\"structIEndpoint.TransferQuote\",\"components\":[{\"name\":\"sender\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"recipient\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nonce\",\"type\":\"uint64\",\"internalType\":\"uint64\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"unStakeLogXForAccount\",\"inputs\":[{\"name\":\"stakerContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"subAccount\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"stakeId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"}],\"outputs\":[{\"name\":\"unStakeAmount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"upgradeClearinghouseLiq\",\"inputs\":[{\"name\":\"_clearinghouseLiq\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawCollateral\",\"inputs\":[{\"name\":\"withdrawCollateralRequest\",\"type\":\"tuple\",\"internalType\":\"structIClearinghouse.WithdrawCollateral\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"productId\",\"type\":\"uint32\",\"internalType\":\"uint32\"},{\"name\":\"amount\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"nonce\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"destinationChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"spotProductIds\",\"type\":\"uint32[]\",\"internalType\":\"uint32[]\"},{\"name\":\"spotPricesX18\",\"type\":\"int128[]\",\"internalType\":\"int128[]\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawLogX\",\"inputs\":[{\"name\":\"withdrawRequest\",\"type\":\"tuple\",\"internalType\":\"structIClearinghouse.WithdrawLogX\",\"components\":[{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"internalType\":\"bytes32\"},{\"name\":\"tokenAmount\",\"type\":\"int128\",\"internalType\":\"int128\"},{\"name\":\"sessionKey\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"destinationChainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"bridgeOutContract\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"nonce\",\"type\":\"uint128\",\"internalType\":\"uint128\"},{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"chainId\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"ClearinghouseInitialized\",\"inputs\":[{\"name\":\"endpoint\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"},{\"name\":\"quote\",\"type\":\"address\",\"indexed\":false,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Initialized\",\"inputs\":[{\"name\":\"version\",\"type\":\"uint64\",\"indexed\":false,\"internalType\":\"uint64\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Liquidation\",\"inputs\":[{\"name\":\"liquidatorSubaccount\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"liquidateeSubaccount\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":false,\"internalType\":\"uint32\"},{\"name\":\"isEncodedSpread\",\"type\":\"bool\",\"indexed\":false,\"internalType\":\"bool\"},{\"name\":\"amount\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"amountQuote\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"OwnershipTransferred\",\"inputs\":[{\"name\":\"previousOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"newOwner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"UserPnLSettled\",\"inputs\":[{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"totalPnl\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"WithdrawToken\",\"inputs\":[{\"name\":\"userAddress\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"subAccountId\",\"type\":\"bytes32\",\"indexed\":true,\"internalType\":\"bytes32\"},{\"name\":\"amount\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"fees\",\"type\":\"int128\",\"indexed\":false,\"internalType\":\"int128\"},{\"name\":\"destinationChainId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"productId\",\"type\":\"uint32\",\"indexed\":true,\"internalType\":\"uint32\"},{\"name\":\"sourceChainId\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"InvalidInitialization\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"NotInitializing\",\"inputs\":[]},{\"type\":\"error\",\"name\":\"OwnableInvalidOwner\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"OwnableUnauthorizedAccount\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
}

// ClearingHouseABI is the input ABI used to generate the binding from.
// Deprecated: Use ClearingHouseMetaData.ABI instead.
var ClearingHouseABI = ClearingHouseMetaData.ABI

// ClearingHouse is an auto generated Go binding around an Ethereum contract.
type ClearingHouse struct {
	ClearingHouseCaller     // Read-only binding to the contract
	ClearingHouseTransactor // Write-only binding to the contract
	ClearingHouseFilterer   // Log filterer for contract events
}

// ClearingHouseCaller is an auto generated read-only Go binding around an Ethereum contract.
type ClearingHouseCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearingHouseTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ClearingHouseTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearingHouseFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ClearingHouseFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ClearingHouseSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ClearingHouseSession struct {
	Contract     *ClearingHouse    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ClearingHouseCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ClearingHouseCallerSession struct {
	Contract *ClearingHouseCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ClearingHouseTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ClearingHouseTransactorSession struct {
	Contract     *ClearingHouseTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ClearingHouseRaw is an auto generated low-level Go binding around an Ethereum contract.
type ClearingHouseRaw struct {
	Contract *ClearingHouse // Generic contract binding to access the raw methods on
}

// ClearingHouseCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ClearingHouseCallerRaw struct {
	Contract *ClearingHouseCaller // Generic read-only contract binding to access the raw methods on
}

// ClearingHouseTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ClearingHouseTransactorRaw struct {
	Contract *ClearingHouseTransactor // Generic write-only contract binding to access the raw methods on
}

// NewClearingHouse creates a new instance of ClearingHouse, bound to a specific deployed contract.
func NewClearingHouse(address common.Address, backend bind.ContractBackend) (*ClearingHouse, error) {
	contract, err := bindClearingHouse(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ClearingHouse{ClearingHouseCaller: ClearingHouseCaller{contract: contract}, ClearingHouseTransactor: ClearingHouseTransactor{contract: contract}, ClearingHouseFilterer: ClearingHouseFilterer{contract: contract}}, nil
}

// NewClearingHouseCaller creates a new read-only instance of ClearingHouse, bound to a specific deployed contract.
func NewClearingHouseCaller(address common.Address, caller bind.ContractCaller) (*ClearingHouseCaller, error) {
	contract, err := bindClearingHouse(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseCaller{contract: contract}, nil
}

// NewClearingHouseTransactor creates a new write-only instance of ClearingHouse, bound to a specific deployed contract.
func NewClearingHouseTransactor(address common.Address, transactor bind.ContractTransactor) (*ClearingHouseTransactor, error) {
	contract, err := bindClearingHouse(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseTransactor{contract: contract}, nil
}

// NewClearingHouseFilterer creates a new log filterer instance of ClearingHouse, bound to a specific deployed contract.
func NewClearingHouseFilterer(address common.Address, filterer bind.ContractFilterer) (*ClearingHouseFilterer, error) {
	contract, err := bindClearingHouse(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseFilterer{contract: contract}, nil
}

// bindClearingHouse binds a generic wrapper to an already deployed contract.
func bindClearingHouse(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ClearingHouseMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearingHouse *ClearingHouseRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearingHouse.Contract.ClearingHouseCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearingHouse *ClearingHouseRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearingHouse.Contract.ClearingHouseTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearingHouse *ClearingHouseRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearingHouse.Contract.ClearingHouseTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ClearingHouse *ClearingHouseCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ClearingHouse.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ClearingHouse *ClearingHouseTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearingHouse.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ClearingHouse *ClearingHouseTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ClearingHouse.Contract.contract.Transact(opts, method, params...)
}

// GetClearinghouseLiq is a free data retrieval call binding the contract method 0x9b0861c1.
//
// Solidity: function getClearinghouseLiq() view returns(address)
func (_ClearingHouse *ClearingHouseCaller) GetClearinghouseLiq(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getClearinghouseLiq")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetClearinghouseLiq is a free data retrieval call binding the contract method 0x9b0861c1.
//
// Solidity: function getClearinghouseLiq() view returns(address)
func (_ClearingHouse *ClearingHouseSession) GetClearinghouseLiq() (common.Address, error) {
	return _ClearingHouse.Contract.GetClearinghouseLiq(&_ClearingHouse.CallOpts)
}

// GetClearinghouseLiq is a free data retrieval call binding the contract method 0x9b0861c1.
//
// Solidity: function getClearinghouseLiq() view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) GetClearinghouseLiq() (common.Address, error) {
	return _ClearingHouse.Contract.GetClearinghouseLiq(&_ClearingHouse.CallOpts)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_ClearingHouse *ClearingHouseCaller) GetEndpoint(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getEndpoint")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_ClearingHouse *ClearingHouseSession) GetEndpoint() (common.Address, error) {
	return _ClearingHouse.Contract.GetEndpoint(&_ClearingHouse.CallOpts)
}

// GetEndpoint is a free data retrieval call binding the contract method 0xaed8e967.
//
// Solidity: function getEndpoint() view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) GetEndpoint() (common.Address, error) {
	return _ClearingHouse.Contract.GetEndpoint(&_ClearingHouse.CallOpts)
}

// GetEngineByProduct is a free data retrieval call binding the contract method 0xdeb14ec3.
//
// Solidity: function getEngineByProduct(uint32 productId) view returns(address)
func (_ClearingHouse *ClearingHouseCaller) GetEngineByProduct(opts *bind.CallOpts, productId uint32) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getEngineByProduct", productId)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEngineByProduct is a free data retrieval call binding the contract method 0xdeb14ec3.
//
// Solidity: function getEngineByProduct(uint32 productId) view returns(address)
func (_ClearingHouse *ClearingHouseSession) GetEngineByProduct(productId uint32) (common.Address, error) {
	return _ClearingHouse.Contract.GetEngineByProduct(&_ClearingHouse.CallOpts, productId)
}

// GetEngineByProduct is a free data retrieval call binding the contract method 0xdeb14ec3.
//
// Solidity: function getEngineByProduct(uint32 productId) view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) GetEngineByProduct(productId uint32) (common.Address, error) {
	return _ClearingHouse.Contract.GetEngineByProduct(&_ClearingHouse.CallOpts, productId)
}

// GetEngineByType is a free data retrieval call binding the contract method 0x5d2e9ad1.
//
// Solidity: function getEngineByType(uint8 engineType) view returns(address)
func (_ClearingHouse *ClearingHouseCaller) GetEngineByType(opts *bind.CallOpts, engineType uint8) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getEngineByType", engineType)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetEngineByType is a free data retrieval call binding the contract method 0x5d2e9ad1.
//
// Solidity: function getEngineByType(uint8 engineType) view returns(address)
func (_ClearingHouse *ClearingHouseSession) GetEngineByType(engineType uint8) (common.Address, error) {
	return _ClearingHouse.Contract.GetEngineByType(&_ClearingHouse.CallOpts, engineType)
}

// GetEngineByType is a free data retrieval call binding the contract method 0x5d2e9ad1.
//
// Solidity: function getEngineByType(uint8 engineType) view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) GetEngineByType(engineType uint8) (common.Address, error) {
	return _ClearingHouse.Contract.GetEngineByType(&_ClearingHouse.CallOpts, engineType)
}

// GetHealth is a free data retrieval call binding the contract method 0x88b6496f.
//
// Solidity: function getHealth(bytes32 subaccount, uint8 healthType) view returns(int128 health)
func (_ClearingHouse *ClearingHouseCaller) GetHealth(opts *bind.CallOpts, subaccount [32]byte, healthType uint8) (*big.Int, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getHealth", subaccount, healthType)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetHealth is a free data retrieval call binding the contract method 0x88b6496f.
//
// Solidity: function getHealth(bytes32 subaccount, uint8 healthType) view returns(int128 health)
func (_ClearingHouse *ClearingHouseSession) GetHealth(subaccount [32]byte, healthType uint8) (*big.Int, error) {
	return _ClearingHouse.Contract.GetHealth(&_ClearingHouse.CallOpts, subaccount, healthType)
}

// GetHealth is a free data retrieval call binding the contract method 0x88b6496f.
//
// Solidity: function getHealth(bytes32 subaccount, uint8 healthType) view returns(int128 health)
func (_ClearingHouse *ClearingHouseCallerSession) GetHealth(subaccount [32]byte, healthType uint8) (*big.Int, error) {
	return _ClearingHouse.Contract.GetHealth(&_ClearingHouse.CallOpts, subaccount, healthType)
}

// GetInsurance is a free data retrieval call binding the contract method 0x267a8da0.
//
// Solidity: function getInsurance() view returns(int128)
func (_ClearingHouse *ClearingHouseCaller) GetInsurance(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getInsurance")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetInsurance is a free data retrieval call binding the contract method 0x267a8da0.
//
// Solidity: function getInsurance() view returns(int128)
func (_ClearingHouse *ClearingHouseSession) GetInsurance() (*big.Int, error) {
	return _ClearingHouse.Contract.GetInsurance(&_ClearingHouse.CallOpts)
}

// GetInsurance is a free data retrieval call binding the contract method 0x267a8da0.
//
// Solidity: function getInsurance() view returns(int128)
func (_ClearingHouse *ClearingHouseCallerSession) GetInsurance() (*big.Int, error) {
	return _ClearingHouse.Contract.GetInsurance(&_ClearingHouse.CallOpts)
}

// GetQuote is a free data retrieval call binding the contract method 0x171755b1.
//
// Solidity: function getQuote() view returns(address)
func (_ClearingHouse *ClearingHouseCaller) GetQuote(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getQuote")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetQuote is a free data retrieval call binding the contract method 0x171755b1.
//
// Solidity: function getQuote() view returns(address)
func (_ClearingHouse *ClearingHouseSession) GetQuote() (common.Address, error) {
	return _ClearingHouse.Contract.GetQuote(&_ClearingHouse.CallOpts)
}

// GetQuote is a free data retrieval call binding the contract method 0x171755b1.
//
// Solidity: function getQuote() view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) GetQuote() (common.Address, error) {
	return _ClearingHouse.Contract.GetQuote(&_ClearingHouse.CallOpts)
}

// GetSpreads is a free data retrieval call binding the contract method 0xf16dec06.
//
// Solidity: function getSpreads() view returns(uint256)
func (_ClearingHouse *ClearingHouseCaller) GetSpreads(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getSpreads")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetSpreads is a free data retrieval call binding the contract method 0xf16dec06.
//
// Solidity: function getSpreads() view returns(uint256)
func (_ClearingHouse *ClearingHouseSession) GetSpreads() (*big.Int, error) {
	return _ClearingHouse.Contract.GetSpreads(&_ClearingHouse.CallOpts)
}

// GetSpreads is a free data retrieval call binding the contract method 0xf16dec06.
//
// Solidity: function getSpreads() view returns(uint256)
func (_ClearingHouse *ClearingHouseCallerSession) GetSpreads() (*big.Int, error) {
	return _ClearingHouse.Contract.GetSpreads(&_ClearingHouse.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_ClearingHouse *ClearingHouseCaller) GetVersion(opts *bind.CallOpts) (uint64, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "getVersion")

	if err != nil {
		return *new(uint64), err
	}

	out0 := *abi.ConvertType(out[0], new(uint64)).(*uint64)

	return out0, err

}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_ClearingHouse *ClearingHouseSession) GetVersion() (uint64, error) {
	return _ClearingHouse.Contract.GetVersion(&_ClearingHouse.CallOpts)
}

// GetVersion is a free data retrieval call binding the contract method 0x0d8e6e2c.
//
// Solidity: function getVersion() pure returns(uint64)
func (_ClearingHouse *ClearingHouseCallerSession) GetVersion() (uint64, error) {
	return _ClearingHouse.Contract.GetVersion(&_ClearingHouse.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ClearingHouse *ClearingHouseCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ClearingHouse.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ClearingHouse *ClearingHouseSession) Owner() (common.Address, error) {
	return _ClearingHouse.Contract.Owner(&_ClearingHouse.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_ClearingHouse *ClearingHouseCallerSession) Owner() (common.Address, error) {
	return _ClearingHouse.Contract.Owner(&_ClearingHouse.CallOpts)
}

// AddEngine is a paid mutator transaction binding the contract method 0x56e49ef3.
//
// Solidity: function addEngine(address engine, address offchainExchange, uint8 engineType) returns()
func (_ClearingHouse *ClearingHouseTransactor) AddEngine(opts *bind.TransactOpts, engine common.Address, offchainExchange common.Address, engineType uint8) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "addEngine", engine, offchainExchange, engineType)
}

// AddEngine is a paid mutator transaction binding the contract method 0x56e49ef3.
//
// Solidity: function addEngine(address engine, address offchainExchange, uint8 engineType) returns()
func (_ClearingHouse *ClearingHouseSession) AddEngine(engine common.Address, offchainExchange common.Address, engineType uint8) (*types.Transaction, error) {
	return _ClearingHouse.Contract.AddEngine(&_ClearingHouse.TransactOpts, engine, offchainExchange, engineType)
}

// AddEngine is a paid mutator transaction binding the contract method 0x56e49ef3.
//
// Solidity: function addEngine(address engine, address offchainExchange, uint8 engineType) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) AddEngine(engine common.Address, offchainExchange common.Address, engineType uint8) (*types.Transaction, error) {
	return _ClearingHouse.Contract.AddEngine(&_ClearingHouse.TransactOpts, engine, offchainExchange, engineType)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xf8e97d72.
//
// Solidity: function claimRewards(address stakerContract, bytes32 subAccount) returns(uint256 amount)
func (_ClearingHouse *ClearingHouseTransactor) ClaimRewards(opts *bind.TransactOpts, stakerContract common.Address, subAccount [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "claimRewards", stakerContract, subAccount)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xf8e97d72.
//
// Solidity: function claimRewards(address stakerContract, bytes32 subAccount) returns(uint256 amount)
func (_ClearingHouse *ClearingHouseSession) ClaimRewards(stakerContract common.Address, subAccount [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.ClaimRewards(&_ClearingHouse.TransactOpts, stakerContract, subAccount)
}

// ClaimRewards is a paid mutator transaction binding the contract method 0xf8e97d72.
//
// Solidity: function claimRewards(address stakerContract, bytes32 subAccount) returns(uint256 amount)
func (_ClearingHouse *ClearingHouseTransactorSession) ClaimRewards(stakerContract common.Address, subAccount [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.ClaimRewards(&_ClearingHouse.TransactOpts, stakerContract, subAccount)
}

// DeductBridgingFees is a paid mutator transaction binding the contract method 0x41202ab9.
//
// Solidity: function deductBridgingFees(uint32 productId, int128 tokenAmount, bytes32 subAccountId) returns(int128, int128)
func (_ClearingHouse *ClearingHouseTransactor) DeductBridgingFees(opts *bind.TransactOpts, productId uint32, tokenAmount *big.Int, subAccountId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "deductBridgingFees", productId, tokenAmount, subAccountId)
}

// DeductBridgingFees is a paid mutator transaction binding the contract method 0x41202ab9.
//
// Solidity: function deductBridgingFees(uint32 productId, int128 tokenAmount, bytes32 subAccountId) returns(int128, int128)
func (_ClearingHouse *ClearingHouseSession) DeductBridgingFees(productId uint32, tokenAmount *big.Int, subAccountId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.DeductBridgingFees(&_ClearingHouse.TransactOpts, productId, tokenAmount, subAccountId)
}

// DeductBridgingFees is a paid mutator transaction binding the contract method 0x41202ab9.
//
// Solidity: function deductBridgingFees(uint32 productId, int128 tokenAmount, bytes32 subAccountId) returns(int128, int128)
func (_ClearingHouse *ClearingHouseTransactorSession) DeductBridgingFees(productId uint32, tokenAmount *big.Int, subAccountId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.DeductBridgingFees(&_ClearingHouse.TransactOpts, productId, tokenAmount, subAccountId)
}

// DepositInsurance is a paid mutator transaction binding the contract method 0x3a91c58b.
//
// Solidity: function depositInsurance((uint128) txn) returns()
func (_ClearingHouse *ClearingHouseTransactor) DepositInsurance(opts *bind.TransactOpts, txn IEndpointDepositInsurance) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "depositInsurance", txn)
}

// DepositInsurance is a paid mutator transaction binding the contract method 0x3a91c58b.
//
// Solidity: function depositInsurance((uint128) txn) returns()
func (_ClearingHouse *ClearingHouseSession) DepositInsurance(txn IEndpointDepositInsurance) (*types.Transaction, error) {
	return _ClearingHouse.Contract.DepositInsurance(&_ClearingHouse.TransactOpts, txn)
}

// DepositInsurance is a paid mutator transaction binding the contract method 0x3a91c58b.
//
// Solidity: function depositInsurance((uint128) txn) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) DepositInsurance(txn IEndpointDepositInsurance) (*types.Transaction, error) {
	return _ClearingHouse.Contract.DepositInsurance(&_ClearingHouse.TransactOpts, txn)
}

// Initialize is a paid mutator transaction binding the contract method 0xcf756fdf.
//
// Solidity: function initialize(address _endpoint, address _quote, address _clearinghouseLiq, uint256 _spreads) returns()
func (_ClearingHouse *ClearingHouseTransactor) Initialize(opts *bind.TransactOpts, _endpoint common.Address, _quote common.Address, _clearinghouseLiq common.Address, _spreads *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "initialize", _endpoint, _quote, _clearinghouseLiq, _spreads)
}

// Initialize is a paid mutator transaction binding the contract method 0xcf756fdf.
//
// Solidity: function initialize(address _endpoint, address _quote, address _clearinghouseLiq, uint256 _spreads) returns()
func (_ClearingHouse *ClearingHouseSession) Initialize(_endpoint common.Address, _quote common.Address, _clearinghouseLiq common.Address, _spreads *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.Contract.Initialize(&_ClearingHouse.TransactOpts, _endpoint, _quote, _clearinghouseLiq, _spreads)
}

// Initialize is a paid mutator transaction binding the contract method 0xcf756fdf.
//
// Solidity: function initialize(address _endpoint, address _quote, address _clearinghouseLiq, uint256 _spreads) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) Initialize(_endpoint common.Address, _quote common.Address, _clearinghouseLiq common.Address, _spreads *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.Contract.Initialize(&_ClearingHouse.TransactOpts, _endpoint, _quote, _clearinghouseLiq, _spreads)
}

// LiquidateSubaccount is a paid mutator transaction binding the contract method 0x52efadf1.
//
// Solidity: function liquidateSubaccount((bytes32,bytes32,uint32,bool,int128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseTransactor) LiquidateSubaccount(opts *bind.TransactOpts, txn IEndpointLiquidateSubaccount) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "liquidateSubaccount", txn)
}

// LiquidateSubaccount is a paid mutator transaction binding the contract method 0x52efadf1.
//
// Solidity: function liquidateSubaccount((bytes32,bytes32,uint32,bool,int128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseSession) LiquidateSubaccount(txn IEndpointLiquidateSubaccount) (*types.Transaction, error) {
	return _ClearingHouse.Contract.LiquidateSubaccount(&_ClearingHouse.TransactOpts, txn)
}

// LiquidateSubaccount is a paid mutator transaction binding the contract method 0x52efadf1.
//
// Solidity: function liquidateSubaccount((bytes32,bytes32,uint32,bool,int128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) LiquidateSubaccount(txn IEndpointLiquidateSubaccount) (*types.Transaction, error) {
	return _ClearingHouse.Contract.LiquidateSubaccount(&_ClearingHouse.TransactOpts, txn)
}

// RegisterProduct is a paid mutator transaction binding the contract method 0x8762d422.
//
// Solidity: function registerProduct(uint32 productId) returns()
func (_ClearingHouse *ClearingHouseTransactor) RegisterProduct(opts *bind.TransactOpts, productId uint32) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "registerProduct", productId)
}

// RegisterProduct is a paid mutator transaction binding the contract method 0x8762d422.
//
// Solidity: function registerProduct(uint32 productId) returns()
func (_ClearingHouse *ClearingHouseSession) RegisterProduct(productId uint32) (*types.Transaction, error) {
	return _ClearingHouse.Contract.RegisterProduct(&_ClearingHouse.TransactOpts, productId)
}

// RegisterProduct is a paid mutator transaction binding the contract method 0x8762d422.
//
// Solidity: function registerProduct(uint32 productId) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) RegisterProduct(productId uint32) (*types.Transaction, error) {
	return _ClearingHouse.Contract.RegisterProduct(&_ClearingHouse.TransactOpts, productId)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ClearingHouse *ClearingHouseTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ClearingHouse *ClearingHouseSession) RenounceOwnership() (*types.Transaction, error) {
	return _ClearingHouse.Contract.RenounceOwnership(&_ClearingHouse.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_ClearingHouse *ClearingHouseTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _ClearingHouse.Contract.RenounceOwnership(&_ClearingHouse.TransactOpts)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_ClearingHouse *ClearingHouseTransactor) SetEndpoint(opts *bind.TransactOpts, _endpoint common.Address) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "setEndpoint", _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_ClearingHouse *ClearingHouseSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.SetEndpoint(&_ClearingHouse.TransactOpts, _endpoint)
}

// SetEndpoint is a paid mutator transaction binding the contract method 0xdbbb4155.
//
// Solidity: function setEndpoint(address _endpoint) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) SetEndpoint(_endpoint common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.SetEndpoint(&_ClearingHouse.TransactOpts, _endpoint)
}

// SettleUserPnl is a paid mutator transaction binding the contract method 0x736bec72.
//
// Solidity: function settleUserPnl((bytes32,uint32[],int128[],address,uint128,uint256) txn) returns()
func (_ClearingHouse *ClearingHouseTransactor) SettleUserPnl(opts *bind.TransactOpts, txn IEndpointSettleUserPnl) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "settleUserPnl", txn)
}

// SettleUserPnl is a paid mutator transaction binding the contract method 0x736bec72.
//
// Solidity: function settleUserPnl((bytes32,uint32[],int128[],address,uint128,uint256) txn) returns()
func (_ClearingHouse *ClearingHouseSession) SettleUserPnl(txn IEndpointSettleUserPnl) (*types.Transaction, error) {
	return _ClearingHouse.Contract.SettleUserPnl(&_ClearingHouse.TransactOpts, txn)
}

// SettleUserPnl is a paid mutator transaction binding the contract method 0x736bec72.
//
// Solidity: function settleUserPnl((bytes32,uint32[],int128[],address,uint128,uint256) txn) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) SettleUserPnl(txn IEndpointSettleUserPnl) (*types.Transaction, error) {
	return _ClearingHouse.Contract.SettleUserPnl(&_ClearingHouse.TransactOpts, txn)
}

// StakeLogXForAccount is a paid mutator transaction binding the contract method 0x4bc331d2.
//
// Solidity: function stakeLogXForAccount(address stakerContract, bytes32 subAccount, uint256 tokenAmount, uint256 duration) returns()
func (_ClearingHouse *ClearingHouseTransactor) StakeLogXForAccount(opts *bind.TransactOpts, stakerContract common.Address, subAccount [32]byte, tokenAmount *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "stakeLogXForAccount", stakerContract, subAccount, tokenAmount, duration)
}

// StakeLogXForAccount is a paid mutator transaction binding the contract method 0x4bc331d2.
//
// Solidity: function stakeLogXForAccount(address stakerContract, bytes32 subAccount, uint256 tokenAmount, uint256 duration) returns()
func (_ClearingHouse *ClearingHouseSession) StakeLogXForAccount(stakerContract common.Address, subAccount [32]byte, tokenAmount *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.Contract.StakeLogXForAccount(&_ClearingHouse.TransactOpts, stakerContract, subAccount, tokenAmount, duration)
}

// StakeLogXForAccount is a paid mutator transaction binding the contract method 0x4bc331d2.
//
// Solidity: function stakeLogXForAccount(address stakerContract, bytes32 subAccount, uint256 tokenAmount, uint256 duration) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) StakeLogXForAccount(stakerContract common.Address, subAccount [32]byte, tokenAmount *big.Int, duration *big.Int) (*types.Transaction, error) {
	return _ClearingHouse.Contract.StakeLogXForAccount(&_ClearingHouse.TransactOpts, stakerContract, subAccount, tokenAmount, duration)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ClearingHouse *ClearingHouseTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ClearingHouse *ClearingHouseSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.TransferOwnership(&_ClearingHouse.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.TransferOwnership(&_ClearingHouse.TransactOpts, newOwner)
}

// TransferQuote is a paid mutator transaction binding the contract method 0x1d97d22f.
//
// Solidity: function transferQuote((bytes32,bytes32,uint128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseTransactor) TransferQuote(opts *bind.TransactOpts, txn IEndpointTransferQuote) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "transferQuote", txn)
}

// TransferQuote is a paid mutator transaction binding the contract method 0x1d97d22f.
//
// Solidity: function transferQuote((bytes32,bytes32,uint128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseSession) TransferQuote(txn IEndpointTransferQuote) (*types.Transaction, error) {
	return _ClearingHouse.Contract.TransferQuote(&_ClearingHouse.TransactOpts, txn)
}

// TransferQuote is a paid mutator transaction binding the contract method 0x1d97d22f.
//
// Solidity: function transferQuote((bytes32,bytes32,uint128,uint64) txn) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) TransferQuote(txn IEndpointTransferQuote) (*types.Transaction, error) {
	return _ClearingHouse.Contract.TransferQuote(&_ClearingHouse.TransactOpts, txn)
}

// UnStakeLogXForAccount is a paid mutator transaction binding the contract method 0x1888e352.
//
// Solidity: function unStakeLogXForAccount(address stakerContract, bytes32 subAccount, bytes32 stakeId) returns(uint256 unStakeAmount)
func (_ClearingHouse *ClearingHouseTransactor) UnStakeLogXForAccount(opts *bind.TransactOpts, stakerContract common.Address, subAccount [32]byte, stakeId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "unStakeLogXForAccount", stakerContract, subAccount, stakeId)
}

// UnStakeLogXForAccount is a paid mutator transaction binding the contract method 0x1888e352.
//
// Solidity: function unStakeLogXForAccount(address stakerContract, bytes32 subAccount, bytes32 stakeId) returns(uint256 unStakeAmount)
func (_ClearingHouse *ClearingHouseSession) UnStakeLogXForAccount(stakerContract common.Address, subAccount [32]byte, stakeId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.UnStakeLogXForAccount(&_ClearingHouse.TransactOpts, stakerContract, subAccount, stakeId)
}

// UnStakeLogXForAccount is a paid mutator transaction binding the contract method 0x1888e352.
//
// Solidity: function unStakeLogXForAccount(address stakerContract, bytes32 subAccount, bytes32 stakeId) returns(uint256 unStakeAmount)
func (_ClearingHouse *ClearingHouseTransactorSession) UnStakeLogXForAccount(stakerContract common.Address, subAccount [32]byte, stakeId [32]byte) (*types.Transaction, error) {
	return _ClearingHouse.Contract.UnStakeLogXForAccount(&_ClearingHouse.TransactOpts, stakerContract, subAccount, stakeId)
}

// UpgradeClearinghouseLiq is a paid mutator transaction binding the contract method 0x3c54c2de.
//
// Solidity: function upgradeClearinghouseLiq(address _clearinghouseLiq) returns()
func (_ClearingHouse *ClearingHouseTransactor) UpgradeClearinghouseLiq(opts *bind.TransactOpts, _clearinghouseLiq common.Address) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "upgradeClearinghouseLiq", _clearinghouseLiq)
}

// UpgradeClearinghouseLiq is a paid mutator transaction binding the contract method 0x3c54c2de.
//
// Solidity: function upgradeClearinghouseLiq(address _clearinghouseLiq) returns()
func (_ClearingHouse *ClearingHouseSession) UpgradeClearinghouseLiq(_clearinghouseLiq common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.UpgradeClearinghouseLiq(&_ClearingHouse.TransactOpts, _clearinghouseLiq)
}

// UpgradeClearinghouseLiq is a paid mutator transaction binding the contract method 0x3c54c2de.
//
// Solidity: function upgradeClearinghouseLiq(address _clearinghouseLiq) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) UpgradeClearinghouseLiq(_clearinghouseLiq common.Address) (*types.Transaction, error) {
	return _ClearingHouse.Contract.UpgradeClearinghouseLiq(&_ClearingHouse.TransactOpts, _clearinghouseLiq)
}

// WithdrawCollateral is a paid mutator transaction binding the contract method 0xbf714791.
//
// Solidity: function withdrawCollateral((bytes32,address,uint32,uint128,uint128,uint256,address,uint256,uint32[],int128[]) withdrawCollateralRequest) returns()
func (_ClearingHouse *ClearingHouseTransactor) WithdrawCollateral(opts *bind.TransactOpts, withdrawCollateralRequest IClearinghouseWithdrawCollateral) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "withdrawCollateral", withdrawCollateralRequest)
}

// WithdrawCollateral is a paid mutator transaction binding the contract method 0xbf714791.
//
// Solidity: function withdrawCollateral((bytes32,address,uint32,uint128,uint128,uint256,address,uint256,uint32[],int128[]) withdrawCollateralRequest) returns()
func (_ClearingHouse *ClearingHouseSession) WithdrawCollateral(withdrawCollateralRequest IClearinghouseWithdrawCollateral) (*types.Transaction, error) {
	return _ClearingHouse.Contract.WithdrawCollateral(&_ClearingHouse.TransactOpts, withdrawCollateralRequest)
}

// WithdrawCollateral is a paid mutator transaction binding the contract method 0xbf714791.
//
// Solidity: function withdrawCollateral((bytes32,address,uint32,uint128,uint128,uint256,address,uint256,uint32[],int128[]) withdrawCollateralRequest) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) WithdrawCollateral(withdrawCollateralRequest IClearinghouseWithdrawCollateral) (*types.Transaction, error) {
	return _ClearingHouse.Contract.WithdrawCollateral(&_ClearingHouse.TransactOpts, withdrawCollateralRequest)
}

// WithdrawLogX is a paid mutator transaction binding the contract method 0xd88b6f92.
//
// Solidity: function withdrawLogX((bytes32,int128,address,uint256,address,uint128,address,uint256) withdrawRequest) returns()
func (_ClearingHouse *ClearingHouseTransactor) WithdrawLogX(opts *bind.TransactOpts, withdrawRequest IClearinghouseWithdrawLogX) (*types.Transaction, error) {
	return _ClearingHouse.contract.Transact(opts, "withdrawLogX", withdrawRequest)
}

// WithdrawLogX is a paid mutator transaction binding the contract method 0xd88b6f92.
//
// Solidity: function withdrawLogX((bytes32,int128,address,uint256,address,uint128,address,uint256) withdrawRequest) returns()
func (_ClearingHouse *ClearingHouseSession) WithdrawLogX(withdrawRequest IClearinghouseWithdrawLogX) (*types.Transaction, error) {
	return _ClearingHouse.Contract.WithdrawLogX(&_ClearingHouse.TransactOpts, withdrawRequest)
}

// WithdrawLogX is a paid mutator transaction binding the contract method 0xd88b6f92.
//
// Solidity: function withdrawLogX((bytes32,int128,address,uint256,address,uint128,address,uint256) withdrawRequest) returns()
func (_ClearingHouse *ClearingHouseTransactorSession) WithdrawLogX(withdrawRequest IClearinghouseWithdrawLogX) (*types.Transaction, error) {
	return _ClearingHouse.Contract.WithdrawLogX(&_ClearingHouse.TransactOpts, withdrawRequest)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_ClearingHouse *ClearingHouseTransactor) Receive(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ClearingHouse.contract.RawTransact(opts, nil) // calldata is disallowed for receive function
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_ClearingHouse *ClearingHouseSession) Receive() (*types.Transaction, error) {
	return _ClearingHouse.Contract.Receive(&_ClearingHouse.TransactOpts)
}

// Receive is a paid mutator transaction binding the contract receive function.
//
// Solidity: receive() payable returns()
func (_ClearingHouse *ClearingHouseTransactorSession) Receive() (*types.Transaction, error) {
	return _ClearingHouse.Contract.Receive(&_ClearingHouse.TransactOpts)
}

// ClearingHouseClearinghouseInitializedIterator is returned from FilterClearinghouseInitialized and is used to iterate over the raw logs and unpacked data for ClearinghouseInitialized events raised by the ClearingHouse contract.
type ClearingHouseClearinghouseInitializedIterator struct {
	Event *ClearingHouseClearinghouseInitialized // Event containing the contract specifics and raw log

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
func (it *ClearingHouseClearinghouseInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseClearinghouseInitialized)
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
		it.Event = new(ClearingHouseClearinghouseInitialized)
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
func (it *ClearingHouseClearinghouseInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseClearinghouseInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseClearinghouseInitialized represents a ClearinghouseInitialized event raised by the ClearingHouse contract.
type ClearingHouseClearinghouseInitialized struct {
	Endpoint common.Address
	Quote    common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterClearinghouseInitialized is a free log retrieval operation binding the contract event 0x85cbc94663dc3e10fe6f4fb22712d52d5939321301933ac1b1132d47023698bd.
//
// Solidity: event ClearinghouseInitialized(address endpoint, address quote)
func (_ClearingHouse *ClearingHouseFilterer) FilterClearinghouseInitialized(opts *bind.FilterOpts) (*ClearingHouseClearinghouseInitializedIterator, error) {

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "ClearinghouseInitialized")
	if err != nil {
		return nil, err
	}
	return &ClearingHouseClearinghouseInitializedIterator{contract: _ClearingHouse.contract, event: "ClearinghouseInitialized", logs: logs, sub: sub}, nil
}

// WatchClearinghouseInitialized is a free log subscription operation binding the contract event 0x85cbc94663dc3e10fe6f4fb22712d52d5939321301933ac1b1132d47023698bd.
//
// Solidity: event ClearinghouseInitialized(address endpoint, address quote)
func (_ClearingHouse *ClearingHouseFilterer) WatchClearinghouseInitialized(opts *bind.WatchOpts, sink chan<- *ClearingHouseClearinghouseInitialized) (event.Subscription, error) {

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "ClearinghouseInitialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseClearinghouseInitialized)
				if err := _ClearingHouse.contract.UnpackLog(event, "ClearinghouseInitialized", log); err != nil {
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

// ParseClearinghouseInitialized is a log parse operation binding the contract event 0x85cbc94663dc3e10fe6f4fb22712d52d5939321301933ac1b1132d47023698bd.
//
// Solidity: event ClearinghouseInitialized(address endpoint, address quote)
func (_ClearingHouse *ClearingHouseFilterer) ParseClearinghouseInitialized(log types.Log) (*ClearingHouseClearinghouseInitialized, error) {
	event := new(ClearingHouseClearinghouseInitialized)
	if err := _ClearingHouse.contract.UnpackLog(event, "ClearinghouseInitialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearingHouseInitializedIterator is returned from FilterInitialized and is used to iterate over the raw logs and unpacked data for Initialized events raised by the ClearingHouse contract.
type ClearingHouseInitializedIterator struct {
	Event *ClearingHouseInitialized // Event containing the contract specifics and raw log

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
func (it *ClearingHouseInitializedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseInitialized)
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
		it.Event = new(ClearingHouseInitialized)
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
func (it *ClearingHouseInitializedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseInitializedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseInitialized represents a Initialized event raised by the ClearingHouse contract.
type ClearingHouseInitialized struct {
	Version uint64
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterInitialized is a free log retrieval operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_ClearingHouse *ClearingHouseFilterer) FilterInitialized(opts *bind.FilterOpts) (*ClearingHouseInitializedIterator, error) {

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return &ClearingHouseInitializedIterator{contract: _ClearingHouse.contract, event: "Initialized", logs: logs, sub: sub}, nil
}

// WatchInitialized is a free log subscription operation binding the contract event 0xc7f505b2f371ae2175ee4913f4499e1f2633a7b5936321eed1cdaeb6115181d2.
//
// Solidity: event Initialized(uint64 version)
func (_ClearingHouse *ClearingHouseFilterer) WatchInitialized(opts *bind.WatchOpts, sink chan<- *ClearingHouseInitialized) (event.Subscription, error) {

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "Initialized")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseInitialized)
				if err := _ClearingHouse.contract.UnpackLog(event, "Initialized", log); err != nil {
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
func (_ClearingHouse *ClearingHouseFilterer) ParseInitialized(log types.Log) (*ClearingHouseInitialized, error) {
	event := new(ClearingHouseInitialized)
	if err := _ClearingHouse.contract.UnpackLog(event, "Initialized", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearingHouseLiquidationIterator is returned from FilterLiquidation and is used to iterate over the raw logs and unpacked data for Liquidation events raised by the ClearingHouse contract.
type ClearingHouseLiquidationIterator struct {
	Event *ClearingHouseLiquidation // Event containing the contract specifics and raw log

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
func (it *ClearingHouseLiquidationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseLiquidation)
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
		it.Event = new(ClearingHouseLiquidation)
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
func (it *ClearingHouseLiquidationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseLiquidationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseLiquidation represents a Liquidation event raised by the ClearingHouse contract.
type ClearingHouseLiquidation struct {
	LiquidatorSubaccount [32]byte
	LiquidateeSubaccount [32]byte
	ProductId            uint32
	IsEncodedSpread      bool
	Amount               *big.Int
	AmountQuote          *big.Int
	Raw                  types.Log // Blockchain specific contextual infos
}

// FilterLiquidation is a free log retrieval operation binding the contract event 0x494f937f5cc892f798248aa831acfb4ad7c4bf35edd8498c5fb431ce1e38b035.
//
// Solidity: event Liquidation(bytes32 indexed liquidatorSubaccount, bytes32 indexed liquidateeSubaccount, uint32 productId, bool isEncodedSpread, int128 amount, int128 amountQuote)
func (_ClearingHouse *ClearingHouseFilterer) FilterLiquidation(opts *bind.FilterOpts, liquidatorSubaccount [][32]byte, liquidateeSubaccount [][32]byte) (*ClearingHouseLiquidationIterator, error) {

	var liquidatorSubaccountRule []interface{}
	for _, liquidatorSubaccountItem := range liquidatorSubaccount {
		liquidatorSubaccountRule = append(liquidatorSubaccountRule, liquidatorSubaccountItem)
	}
	var liquidateeSubaccountRule []interface{}
	for _, liquidateeSubaccountItem := range liquidateeSubaccount {
		liquidateeSubaccountRule = append(liquidateeSubaccountRule, liquidateeSubaccountItem)
	}

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "Liquidation", liquidatorSubaccountRule, liquidateeSubaccountRule)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseLiquidationIterator{contract: _ClearingHouse.contract, event: "Liquidation", logs: logs, sub: sub}, nil
}

// WatchLiquidation is a free log subscription operation binding the contract event 0x494f937f5cc892f798248aa831acfb4ad7c4bf35edd8498c5fb431ce1e38b035.
//
// Solidity: event Liquidation(bytes32 indexed liquidatorSubaccount, bytes32 indexed liquidateeSubaccount, uint32 productId, bool isEncodedSpread, int128 amount, int128 amountQuote)
func (_ClearingHouse *ClearingHouseFilterer) WatchLiquidation(opts *bind.WatchOpts, sink chan<- *ClearingHouseLiquidation, liquidatorSubaccount [][32]byte, liquidateeSubaccount [][32]byte) (event.Subscription, error) {

	var liquidatorSubaccountRule []interface{}
	for _, liquidatorSubaccountItem := range liquidatorSubaccount {
		liquidatorSubaccountRule = append(liquidatorSubaccountRule, liquidatorSubaccountItem)
	}
	var liquidateeSubaccountRule []interface{}
	for _, liquidateeSubaccountItem := range liquidateeSubaccount {
		liquidateeSubaccountRule = append(liquidateeSubaccountRule, liquidateeSubaccountItem)
	}

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "Liquidation", liquidatorSubaccountRule, liquidateeSubaccountRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseLiquidation)
				if err := _ClearingHouse.contract.UnpackLog(event, "Liquidation", log); err != nil {
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

// ParseLiquidation is a log parse operation binding the contract event 0x494f937f5cc892f798248aa831acfb4ad7c4bf35edd8498c5fb431ce1e38b035.
//
// Solidity: event Liquidation(bytes32 indexed liquidatorSubaccount, bytes32 indexed liquidateeSubaccount, uint32 productId, bool isEncodedSpread, int128 amount, int128 amountQuote)
func (_ClearingHouse *ClearingHouseFilterer) ParseLiquidation(log types.Log) (*ClearingHouseLiquidation, error) {
	event := new(ClearingHouseLiquidation)
	if err := _ClearingHouse.contract.UnpackLog(event, "Liquidation", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearingHouseOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the ClearingHouse contract.
type ClearingHouseOwnershipTransferredIterator struct {
	Event *ClearingHouseOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *ClearingHouseOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseOwnershipTransferred)
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
		it.Event = new(ClearingHouseOwnershipTransferred)
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
func (it *ClearingHouseOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseOwnershipTransferred represents a OwnershipTransferred event raised by the ClearingHouse contract.
type ClearingHouseOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ClearingHouse *ClearingHouseFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ClearingHouseOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseOwnershipTransferredIterator{contract: _ClearingHouse.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_ClearingHouse *ClearingHouseFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ClearingHouseOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseOwnershipTransferred)
				if err := _ClearingHouse.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_ClearingHouse *ClearingHouseFilterer) ParseOwnershipTransferred(log types.Log) (*ClearingHouseOwnershipTransferred, error) {
	event := new(ClearingHouseOwnershipTransferred)
	if err := _ClearingHouse.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearingHouseUserPnLSettledIterator is returned from FilterUserPnLSettled and is used to iterate over the raw logs and unpacked data for UserPnLSettled events raised by the ClearingHouse contract.
type ClearingHouseUserPnLSettledIterator struct {
	Event *ClearingHouseUserPnLSettled // Event containing the contract specifics and raw log

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
func (it *ClearingHouseUserPnLSettledIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseUserPnLSettled)
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
		it.Event = new(ClearingHouseUserPnLSettled)
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
func (it *ClearingHouseUserPnLSettledIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseUserPnLSettledIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseUserPnLSettled represents a UserPnLSettled event raised by the ClearingHouse contract.
type ClearingHouseUserPnLSettled struct {
	UserAddress  common.Address
	SubAccountId [32]byte
	ProductId    uint32
	TotalPnl     *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterUserPnLSettled is a free log retrieval operation binding the contract event 0x7f469318408aed085007758831e5048a871159824ba9f3d08c7aa65532587c5a.
//
// Solidity: event UserPnLSettled(address indexed userAddress, bytes32 indexed subAccountId, uint32 indexed productId, int128 totalPnl)
func (_ClearingHouse *ClearingHouseFilterer) FilterUserPnLSettled(opts *bind.FilterOpts, userAddress []common.Address, subAccountId [][32]byte, productId []uint32) (*ClearingHouseUserPnLSettledIterator, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "UserPnLSettled", userAddressRule, subAccountIdRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseUserPnLSettledIterator{contract: _ClearingHouse.contract, event: "UserPnLSettled", logs: logs, sub: sub}, nil
}

// WatchUserPnLSettled is a free log subscription operation binding the contract event 0x7f469318408aed085007758831e5048a871159824ba9f3d08c7aa65532587c5a.
//
// Solidity: event UserPnLSettled(address indexed userAddress, bytes32 indexed subAccountId, uint32 indexed productId, int128 totalPnl)
func (_ClearingHouse *ClearingHouseFilterer) WatchUserPnLSettled(opts *bind.WatchOpts, sink chan<- *ClearingHouseUserPnLSettled, userAddress []common.Address, subAccountId [][32]byte, productId []uint32) (event.Subscription, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}
	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "UserPnLSettled", userAddressRule, subAccountIdRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseUserPnLSettled)
				if err := _ClearingHouse.contract.UnpackLog(event, "UserPnLSettled", log); err != nil {
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

// ParseUserPnLSettled is a log parse operation binding the contract event 0x7f469318408aed085007758831e5048a871159824ba9f3d08c7aa65532587c5a.
//
// Solidity: event UserPnLSettled(address indexed userAddress, bytes32 indexed subAccountId, uint32 indexed productId, int128 totalPnl)
func (_ClearingHouse *ClearingHouseFilterer) ParseUserPnLSettled(log types.Log) (*ClearingHouseUserPnLSettled, error) {
	event := new(ClearingHouseUserPnLSettled)
	if err := _ClearingHouse.contract.UnpackLog(event, "UserPnLSettled", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ClearingHouseWithdrawTokenIterator is returned from FilterWithdrawToken and is used to iterate over the raw logs and unpacked data for WithdrawToken events raised by the ClearingHouse contract.
type ClearingHouseWithdrawTokenIterator struct {
	Event *ClearingHouseWithdrawToken // Event containing the contract specifics and raw log

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
func (it *ClearingHouseWithdrawTokenIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ClearingHouseWithdrawToken)
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
		it.Event = new(ClearingHouseWithdrawToken)
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
func (it *ClearingHouseWithdrawTokenIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ClearingHouseWithdrawTokenIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ClearingHouseWithdrawToken represents a WithdrawToken event raised by the ClearingHouse contract.
type ClearingHouseWithdrawToken struct {
	UserAddress        common.Address
	SubAccountId       [32]byte
	Amount             *big.Int
	Fees               *big.Int
	DestinationChainId *big.Int
	ProductId          uint32
	SourceChainId      *big.Int
	Raw                types.Log // Blockchain specific contextual infos
}

// FilterWithdrawToken is a free log retrieval operation binding the contract event 0x5eabbfded04cb017d78d4e9713cc35da0dd0af2f1ef2a1923d9b9df829cc1a66.
//
// Solidity: event WithdrawToken(address indexed userAddress, bytes32 indexed subAccountId, int128 amount, int128 fees, uint256 destinationChainId, uint32 indexed productId, uint256 sourceChainId)
func (_ClearingHouse *ClearingHouseFilterer) FilterWithdrawToken(opts *bind.FilterOpts, userAddress []common.Address, subAccountId [][32]byte, productId []uint32) (*ClearingHouseWithdrawTokenIterator, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _ClearingHouse.contract.FilterLogs(opts, "WithdrawToken", userAddressRule, subAccountIdRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return &ClearingHouseWithdrawTokenIterator{contract: _ClearingHouse.contract, event: "WithdrawToken", logs: logs, sub: sub}, nil
}

// WatchWithdrawToken is a free log subscription operation binding the contract event 0x5eabbfded04cb017d78d4e9713cc35da0dd0af2f1ef2a1923d9b9df829cc1a66.
//
// Solidity: event WithdrawToken(address indexed userAddress, bytes32 indexed subAccountId, int128 amount, int128 fees, uint256 destinationChainId, uint32 indexed productId, uint256 sourceChainId)
func (_ClearingHouse *ClearingHouseFilterer) WatchWithdrawToken(opts *bind.WatchOpts, sink chan<- *ClearingHouseWithdrawToken, userAddress []common.Address, subAccountId [][32]byte, productId []uint32) (event.Subscription, error) {

	var userAddressRule []interface{}
	for _, userAddressItem := range userAddress {
		userAddressRule = append(userAddressRule, userAddressItem)
	}
	var subAccountIdRule []interface{}
	for _, subAccountIdItem := range subAccountId {
		subAccountIdRule = append(subAccountIdRule, subAccountIdItem)
	}

	var productIdRule []interface{}
	for _, productIdItem := range productId {
		productIdRule = append(productIdRule, productIdItem)
	}

	logs, sub, err := _ClearingHouse.contract.WatchLogs(opts, "WithdrawToken", userAddressRule, subAccountIdRule, productIdRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ClearingHouseWithdrawToken)
				if err := _ClearingHouse.contract.UnpackLog(event, "WithdrawToken", log); err != nil {
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

// ParseWithdrawToken is a log parse operation binding the contract event 0x5eabbfded04cb017d78d4e9713cc35da0dd0af2f1ef2a1923d9b9df829cc1a66.
//
// Solidity: event WithdrawToken(address indexed userAddress, bytes32 indexed subAccountId, int128 amount, int128 fees, uint256 destinationChainId, uint32 indexed productId, uint256 sourceChainId)
func (_ClearingHouse *ClearingHouseFilterer) ParseWithdrawToken(log types.Log) (*ClearingHouseWithdrawToken, error) {
	event := new(ClearingHouseWithdrawToken)
	if err := _ClearingHouse.contract.UnpackLog(event, "WithdrawToken", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
