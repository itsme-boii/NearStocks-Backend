// nstestnet sets up near-stocks on NEAR testnet (Development.md §13.2 G4).
//
//	go run ./nearchain/cmd/nstestnet create <account.testnet>...   new accounts via the testnet faucet
//	go run ./nearchain/cmd/nstestnet status <account.testnet>...   balances
//	go run ./nearchain/cmd/nstestnet deploy-core    deploy + init near-stocks.testnet
//	go run ./nearchain/cmd/nstestnet deploy-logx    deploy + init the LogX token (supply to the DAO)
//	go run ./nearchain/cmd/nstestnet setup          DAO configures products, accounts and limits
//	go run ./nearchain/cmd/nstestnet smoke          read back config and products
//	go run ./nearchain/cmd/nstestnet e2e            live flows: tick, session key, LogX deposit/stake/withdraw
//	go run ./nearchain/cmd/nstestnet flows <step>   user flows through the local stack (see flows.go)
//
// Keys are generated locally and written to local/near-credentials/testnet/<account>.json,
// relative to the repo root (override with NEAR_CREDENTIALS_DIR), in the near-cli format,
// mode 0600. Private keys are never printed.
package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/mr-tron/base58"
)

// rpc.testnet.near.org is deprecated and intermittently answers with a warning instead of a result
var rpcURL = envOr("NEAR_RPC_URL", "https://test.rpc.fastnear.com")

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

const faucetURL = "https://helper.nearprotocol.com/account"

type creds struct {
	AccountId  string `json:"account_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

func credsPath(account string) string {
	dir := os.Getenv("NEAR_CREDENTIALS_DIR")
	if dir == "" {
		dir = "local/near-credentials/testnet"
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

func create(account string) error {
	if _, err := os.Stat(credsPath(account)); err == nil {
		return fmt.Errorf("%s: credentials already exist, not overwriting", account)
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
	// save the key first: if the faucet creates the account but the response is lost, the key is kept
	if err := os.WriteFile(credsPath(account), raw, 0o600); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"newAccountId": account, "newAccountPublicKey": c.PublicKey})
	resp, err := http.Post(faucetURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: faucet: %w", account, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("%s: faucet %s: %.300s", account, resp.Status, out)
	}
	return nil
}

func status(ctx context.Context, rpc *nearchain.Client, account string) {
	a, err := rpc.ViewAccount(ctx, account)
	if err != nil {
		fmt.Printf("%-32s not found (%v)\n", account, err)
		return
	}
	fmt.Printf("%-32s balance %s yoctoNEAR, storage %d bytes\n", account, a.Amount, a.StorageUsage)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: nstestnet create|status|deploy-core|deploy-logx|setup|smoke ...")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	rpc := nearchain.NewClient(rpcURL)
	switch os.Args[1] {
	case "create":
		for _, a := range os.Args[2:] {
			if err := create(a); err != nil {
				log.Printf("FAILED %v", err)
				continue
			}
			fmt.Printf("created %s (key in %s)\n", a, credsPath(a))
		}
		time.Sleep(3 * time.Second)
		for _, a := range os.Args[2:] {
			status(ctx, rpc, a)
		}
	case "status":
		for _, a := range os.Args[2:] {
			status(ctx, rpc, a)
		}
	case "deploy-core":
		must(deployCore(ctx, rpc))
	case "upgrade-core":
		code, err := os.ReadFile(wasmPath("near_stocks_core"))
		must(err)
		// owner-only upgrade: the new code is the raw call input; the contract deploys it to itself and
		// re-reads its state through migrate()
		res, err := send(ctx, rpc, daoAccount, coreAccount, nearchain.FunctionCall{MethodName: "upgrade", Args: code, Gas: 300 * nearchain.TGas, Deposit: new(big.Int)})
		must(err)
		fmt.Printf("upgraded %s (%d bytes), tx %s\n", coreAccount, len(code), res.Transaction.Hash)
	case "deploy-logx":
		must(deployLogx(ctx, rpc))
	case "setup":
		must(setup(ctx, rpc))
	case "smoke":
		must(smoke(ctx, rpc))
	case "e2e":
		must(e2e(ctx, rpc))
	case "flows":
		flows(ctx, rpc, os.Args[2:])
	case "register-usdc":
		// the DAO pays USDC storage so these accounts can receive USDC (e.g. from the Circle faucet)
		storage := new(big.Int).Mul(big.NewInt(125), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil))
		for _, a := range os.Args[2:] {
			_, err := send(ctx, rpc, daoAccount, usdcTestnet, call("storage_deposit", map[string]any{"account_id": a, "registration_only": true}, 30, storage))
			must(err)
			fmt.Printf("%s can now receive USDC (balance %s units)\n", a, ftBalance(ctx, rpc, usdcTestnet, a))
		}
	default:
		log.Fatalf("unknown command %s", os.Args[1])
	}
}
