package contractUtils

import (
	"os"
	"strconv"
)

// NEAR settlement mode (Development.md §8). With NEAR_SETTLEMENT=1:
//   - session-key signatures are verified under the near-stocks EIP-712 domain, with the
//     configured chain id (397 mainnet / 398 testnet), in every Verify* function;
//   - EndpointContract builds Borsh payloads for near-stocks.near::submit_transactions
//     (contract/endpoint.near.go) instead of ABI calldata for the old appchain.
// Without it, behaviour is exactly as before, so the switch can be flipped at cutover (§11).

func NearSettlement() bool { return os.Getenv("NEAR_SETTLEMENT") == "1" }

func NearTestnet() bool { return os.Getenv("NEAR_NETWORK") == "testnet" }

// NearStocksAccount is the core contract account (NEAR_STOCKS_ACCOUNT, else the network default).
func NearStocksAccount() string {
	if a := os.Getenv("NEAR_STOCKS_ACCOUNT"); a != "" {
		return a
	}
	if NearTestnet() {
		return NEAR_STOCKS_TESTNET_ACCOUNT
	}
	return NEAR_STOCKS_MAINNET_ACCOUNT
}

// NearStocksChainId is the EIP-712 chain id for the network (NEAR_STOCKS_CHAIN_ID overrides).
func NearStocksChainId() int64 {
	if v, err := strconv.ParseInt(os.Getenv("NEAR_STOCKS_CHAIN_ID"), 10, 64); err == nil && v > 0 {
		return v
	}
	if NearTestnet() {
		return NEAR_STOCKS_TESTNET_CHAIN_ID
	}
	return NEAR_STOCKS_MAINNET_CHAIN_ID
}

// SessionKeyChainId is the chainId inside signed orders: 1 on the appchain (SESSION_KEY_CHAIN_ID),
// the near-stocks chain id with NEAR settlement (the contract hashes orders with it).
func SessionKeyChainId() int64 {
	if NearSettlement() {
		return NearStocksChainId()
	}
	return SESSION_KEY_CHAIN_ID
}

// EndpointChainId is the chainId user requests (options, pre-market, synthetic, staking, claims)
// must sign: near-stocks' chain id with NEAR settlement, the LogX appchain's otherwise.
func EndpointChainId() int64 {
	if NearSettlement() {
		return NearStocksChainId()
	}
	return int64(LOGX_CHAIN_ID)
}
