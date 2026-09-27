// nsmainnet: mainnet deployment tooling for near-stocks (Development.md §16.10's mainnet gap).
// Unlike nstestnet, there is no faucet on mainnet — account creation is a two-step process:
//
//	go run ./nearchain/cmd/nsmainnet generate <account.near>...
//	    Generates an ed25519 keypair for each account LOCALLY ONLY. Nothing touches the chain: no
//	    account is created, no NEAR is spent. Keys are saved to local/near-credentials/mainnet/
//	    (near-cli format, mode 0600) and never printed. Safe to run before you have any funded
//	    account at all — it's just cryptography.
//
// Creating the accounts for real (registering them on-chain) needs a funded parent account to pay
// for each one, which this tool does not yet do — that command gets added once a funded mainnet
// account is available to fund from.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/mr-tron/base58"
)

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
	if len(os.Args) < 3 || os.Args[1] != "generate" {
		fmt.Fprintln(os.Stderr, "usage: nsmainnet generate <account.near>...")
		os.Exit(1)
	}
	for _, a := range os.Args[2:] {
		must(generate(a))
	}
}
