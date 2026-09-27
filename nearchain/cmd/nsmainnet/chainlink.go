package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// chainlinkPrice reads a Chainlink AggregatorV3Interface feed's latestRoundData() directly off
// Ethereum mainnet and rescales the answer to x18. Used for SPY: Ondo made Chainlink its official
// oracle provider, and SPYon/QQQon/TSLAon feeds are live on Ethereum mainnet — free to read (only
// cost is your own Ethereum RPC call, no Ondo or Chainlink fee). No ABI binding/SDK needed, just the
// two well-known 4-byte selectors (decimals(), latestRoundData()) any AggregatorV3Interface exposes.
//
// Rejects a stale answer (>1h old) rather than trust it blindly — a stuck feed silently returning
// yesterday's price would be worse than an explicit failure right before a real-money listing.
func chainlinkPrice(ctx context.Context, feedAddr, rpcURL string) (*big.Int, error) {
	if !common.IsHexAddress(feedAddr) {
		return nil, fmt.Errorf("not a valid address: %q", feedAddr)
	}
	if rpcURL == "" {
		rpcURL = "https://ethereum-rpc.publicnode.com" // free public RPC, no key needed
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", rpcURL, err)
	}
	defer client.Close()
	addr := common.HexToAddress(feedAddr)

	decRaw, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: common.FromHex("0x313ce567")}, nil) // decimals()
	if err != nil {
		return nil, fmt.Errorf("decimals(): %w", err)
	}
	if len(decRaw) < 32 {
		return nil, fmt.Errorf("decimals(): short response (%d bytes)", len(decRaw))
	}
	decimals := int64(decRaw[31]) // uint8 result, right-aligned in a 32-byte word

	roundRaw, err := client.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: common.FromHex("0xfeaf968c")}, nil) // latestRoundData()
	if err != nil {
		return nil, fmt.Errorf("latestRoundData(): %w", err)
	}
	if len(roundRaw) < 32*5 {
		return nil, fmt.Errorf("latestRoundData(): short response (%d bytes)", len(roundRaw))
	}
	answer := new(big.Int).SetBytes(roundRaw[32:64])     // 2nd word: int256 answer
	updatedAt := new(big.Int).SetBytes(roundRaw[96:128]) // 4th word: uint256 updatedAt (unix seconds)

	age := time.Now().Unix() - updatedAt.Int64()
	if age > 3600 {
		return nil, fmt.Errorf("feed %s is stale: last updated %ds ago", feedAddr, age)
	}

	shift := 18 - decimals
	if shift >= 0 {
		return new(big.Int).Mul(answer, new(big.Int).Exp(big.NewInt(10), big.NewInt(shift), nil)), nil
	}
	return new(big.Int).Quo(answer, new(big.Int).Exp(big.NewInt(10), big.NewInt(-shift), nil)), nil
}

// resolveChainlinkFeed reads the feed address from the given env var — refuses to run rather than
// guess or default to any address, since a wrong feed address for real money is far worse than
// failing loudly.
func resolveChainlinkFeed(envVar string) (string, error) {
	addr := os.Getenv(envVar)
	if addr == "" {
		return "", fmt.Errorf("%s is not set — get the exact proxy address from https://data.chain.link (Ethereum mainnet) or https://docs.chain.link/data-feeds/tokenized-equity-feeds/ondo first, do not guess", envVar)
	}
	return addr, nil
}
