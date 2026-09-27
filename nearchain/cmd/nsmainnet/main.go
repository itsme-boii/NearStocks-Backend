// nsmainnet: mainnet deployment tooling for near-stocks (Development.md §16.10's mainnet gap).
// Unlike nstestnet, there is no faucet on mainnet — every account either already exists (funded
// out of band) or is a fresh keypair that still needs a funded account to pay for its creation.
//
//	go run ./nearchain/cmd/nsmainnet generate <account.near>...
//	    Generates an ed25519 keypair for each account LOCALLY ONLY. Nothing touches the chain: no
//	    account is created, no NEAR is spent. Keys are saved to local/near-credentials/mainnet/
//	    (near-cli format, mode 0600) and never printed. Safe to run before you have any funded
//	    account at all — it's just cryptography.
//
//	go run ./nearchain/cmd/nsmainnet status <account.near>...
//	    Balances and storage usage, straight from mainnet RPC.
//
//	go run ./nearchain/cmd/nsmainnet fund <funder.near> <account.near> <amountNear>
//	    Plain NEAR transfer from an already-funded account to another EXISTING account. Real,
//	    irreversible mainnet spend — needs `<funder>`'s private key in local/near-credentials/mainnet/.
//
//	go run ./nearchain/cmd/nsmainnet create-account <funder.near> <amountNear>
//	    Creates near-stocks.near for real, funded by `<funder>`, via the "near" registrar's own
//	    create_account method (see deploy.go's createAccount doc comment for why). Requires
//	    near-stocks.near's keypair to already exist locally (run `generate` first).
//
//	go run ./nearchain/cmd/nsmainnet deploy-core
//	    Deploys the wasm and calls new(...). Requires near-stocks.near to already be created (its
//	    own key signs this) and near_stocks_core.wasm to already be built.
//
//	go run ./nearchain/cmd/nsmainnet setup
//	    Owner (drytea2911.near) configures USDC as collateral, lists ETH/BTC/SOL at live Hyperliquid
//	    prices, and finishes migration. Requires drytea2911.near's private key.
//
//	go run ./nearchain/cmd/nsmainnet list-stocks
//	    Owner lists TSLA, NVDA, AAPL, GOOGL — the 4 stocks with a confirmed, tested, live price
//	    source today (100exhange-oracle's Hyperliquid-backed feed) — on-chain via upsert_perp, plus
//	    backend MarketTable rows + OI caps. Requires ORACLE_SERVER_URL pointed at a real running
//	    100exhange-oracle (NOT nsoracle — that's Yahoo Finance and test-only) and DATABASE_URL/
//	    REDIS_URL pointed at the real mainnet backend. See nearchain/cmd/nsmainnet/stocks.go for how
//	    to add more once they have a confirmed price source.
//
//	go run ./nearchain/cmd/nsmainnet smoke
//	    Reads back get_config/get_products.
//
// All of the above except `generate` and `status` spend real NEAR and cannot be undone — dry-run
// nothing, confirm the accounts/amounts printed before running each one.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/mr-tron/base58"
)

var rpcURL = envOr("NEAR_RPC_URL", "https://rpc.mainnet.fastnear.com")

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type creds struct {
	AccountId  string `json:"account_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func credsPath(account string) string {
	dir := os.Getenv("NEAR_MAINNET_CREDENTIALS_DIR")
	if dir == "" {
		dir = "local/near-credentials/mainnet"
	}
	return filepath.Join(dir, account+".json")
}

func loadCreds(account string) (*creds, error) {
	raw, err := os.ReadFile(credsPath(account))
	if err != nil {
		return nil, err
	}
	var c creds
	return &c, json.Unmarshal(raw, &c)
}

func status(ctx context.Context, rpc *nearchain.Client, account string) {
	a, err := rpc.ViewAccount(ctx, account)
	if err != nil {
		fmt.Printf("%-24s not found (%v)\n", account, err)
		return
	}
	fmt.Printf("%-24s balance %s yoctoNEAR, storage %d bytes\n", account, a.Amount, a.StorageUsage)
}

// nearYocto is 1 NEAR in yoctoNEAR (10^24) — NOT e18 (that's the ledger's x18 fixed-point scale,
// a completely different unit used for USDC/perp amounts inside the contract).
var nearYocto = new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)

func nearToYocto(s string) (*big.Int, error) {
	f, ok := new(big.Float).SetPrec(200).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid NEAR amount %q", s)
	}
	v, _ := new(big.Float).SetPrec(200).Mul(f, new(big.Float).SetInt(nearYocto)).Int(nil)
	return v, nil
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// generate creates one ed25519 keypair and saves it locally. It never touches the network and
// never prints the private key — only the account id and public key, which are safe to share.
func generate(account string) error {
	if _, err := os.Stat(credsPath(account)); err == nil {
		return fmt.Errorf("%s: credentials already exist at %s, not overwriting", account, credsPath(account))
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	c := creds{AccountId: account, PublicKey: nearchain.PublicKeyString(pub), PrivateKey: "ed25519:" + base58.Encode(priv)}
	if err := os.MkdirAll(filepath.Dir(credsPath(account)), 0o700); err != nil {
		return err
	}
	raw, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(credsPath(account), raw, 0o600); err != nil {
		return err
	}
	fmt.Printf("generated %-28s public key %s\n  saved to %s (private key not printed)\n", account, c.PublicKey, credsPath(account))
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: nsmainnet generate|status|fund|create-account|deploy-core|setup|list-stocks|smoke ...")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rpc := nearchain.NewClient(rpcURL)
	switch os.Args[1] {
	case "generate":
		if len(os.Args) < 3 {
			log.Fatal("usage: nsmainnet generate <account.near>...")
		}
		for _, a := range os.Args[2:] {
			must(generate(a))
		}
	case "status":
		if len(os.Args) < 3 {
			log.Fatal("usage: nsmainnet status <account.near>...")
		}
		for _, a := range os.Args[2:] {
			status(ctx, rpc, a)
		}
	case "fund":
		if len(os.Args) != 5 {
			log.Fatal("usage: nsmainnet fund <funder.near> <account.near> <amountNear>")
		}
		amount, err := nearToYocto(os.Args[4])
		must(err)
		must(fund(ctx, rpc, os.Args[2], os.Args[3], amount))
	case "create-account":
		if len(os.Args) != 4 {
			log.Fatal("usage: nsmainnet create-account <funder.near> <amountNear>")
		}
		amount, err := nearToYocto(os.Args[3])
		must(err)
		must(createAccount(ctx, rpc, os.Args[2], coreAccount, amount))
	case "deploy-core":
		must(deployCore(ctx, rpc))
	case "setup":
		must(setup(ctx, rpc))
	case "list-stocks":
		must(listStocks(ctx, rpc))
	case "smoke":
		must(smoke(ctx, rpc))
	default:
		log.Fatal("usage: nsmainnet generate|status|fund|create-account|deploy-core|setup|list-stocks|smoke ...")
	}
}
