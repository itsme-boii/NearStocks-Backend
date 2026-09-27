package contractUtils

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// near-stocks EIP-712 domain (Development.md §5.5). Session-key signatures stay secp256k1/EIP-712;
// only the domain values change. Chain IDs are NEAR's EIP-155 registry entries
// (ethereum-lists/chains eip155-397 "NEAR Protocol", eip155-398 "NEAR Protocol Testnet").
const (
	NEAR_STOCKS_DOMAIN_NAME    = "near-stocks"
	NEAR_STOCKS_DOMAIN_VERSION = "1"

	NEAR_STOCKS_MAINNET_ACCOUNT  = "near-stocks.near"
	NEAR_STOCKS_TESTNET_ACCOUNT  = "near-stocks.testnet"
	NEAR_STOCKS_MAINNET_CHAIN_ID = 397
	NEAR_STOCKS_TESTNET_CHAIN_ID = 398
)

// NearStocksVerifyingContract is keccak256(contractAccountId)[12..32]: a stable 20-byte stand-in
// for the EVM verifyingContract, derived from the NEAR account the contract is deployed to.
func NearStocksVerifyingContract(contractAccountId string) common.Address {
	return common.BytesToAddress(crypto.Keccak256([]byte(contractAccountId))[12:])
}

func NearStocksDomain(contractAccountId string, chainId int64) apitypes.TypedDataDomain {
	return apitypes.TypedDataDomain{
		Name:              NEAR_STOCKS_DOMAIN_NAME,
		Version:           NEAR_STOCKS_DOMAIN_VERSION,
		ChainId:           math.NewHexOrDecimal256(chainId),
		VerifyingContract: NearStocksVerifyingContract(contractAccountId).Hex(),
	}
}
