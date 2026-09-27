package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github/eugenix-io/logx-inf-backend/nearchain"
)

// Mainnet layout. Owner and treasury are the SAME account — a NEAR implicit account (a Nightly
// wallet, not a named .near account) holding a real 6.78 NEAR balance as of 2026-09-27, replacing
// the earlier drytea2911.near/saduni1186.near placeholders. There is no separate DAO account yet,
// so `owner` in the contract is a plain keypair, not a multisig. Stand up a real multisig and
// transfer ownership before this holds meaningful funds (see [[pre-mainnet-contract-review]] — this
// is flagged there as the top pre-mainnet gap).
//
// VERIFY INDEPENDENTLY before spending real NEAR: usdcMainnet (nearchain.USDCMainnet) is Circle's
// native USDC contract id on NEAR mainnet as already used elsewhere in this codebase
// (nearFunding.service.go's default, nearchain's live tests) — not something newly asserted here,
// but still a real-money mainnet address worth your own confirmation.
const (
	coreAccount  = "near-stocks.near"
	ownerAccount = "8f4015265b9b67afca2ebbfd81c82e10b2a75e68c28916943f724b2068e30d15" // owner AND treasury (implicit account, Nightly wallet, 6.78 NEAR)
	// guardian and sequencer are the SAME implicit account (user's choice — note this means a
	// compromised sequencer key is also a compromised guardian key, reducing guardian's value as an
	// independent circuit breaker). Does NOT exist on-chain yet as of 2026-09-27 (0 access keys,
	// UNKNOWN_ACCOUNT) — a NEAR implicit account only comes into existence on its first received
	// transfer, so `fund` must run against it before it can sign anything as sequencer.
	guardianAccount = "9bce340bdc623a5fca6d12ca6ab3f49a42e2a1eac97d648339dd0209bff26946"
	seqAccount      = "9bce340bdc623a5fca6d12ca6ab3f49a42e2a1eac97d648339dd0209bff26946"
	chainIdMainnet  = 397
	brokerId        = 2 // same product-line id as testnet (services/api-server default NEAR_BROKER_ID=2) — not network-specific
	// system subaccounts: network-independent (broker_id=1-prefixed by convention inside the hex
	// itself, unrelated to the contract's own numeric broker_id above) — copied verbatim from
	// nstestnet/deploy.go, confirmed reusable for any network.
	ammSub      = "0x0000000000010000000000000000000000000000000000000001000000000001"
	insSub      = "0x0000000000010000000000000000000000000000000000000005000000000001"
	feeSub      = "0x000000000001000000000000000000000000000000000000000f000000000001"
	optionsX    = "0x0000000000010000000000000000000000000000000000000008000000000001"
	optionsFees = "0x0000000000010000000000000000000000000000000000000009000000000001"
	preX        = "0x000000000001000000000000000000000000000000000000000b000000000001"
	preFees     = "0x000000000001000000000000000000000000000000000000000c000000000001"
	synX        = "0x000000000001000000000000000000000000000000000000000d000000000001"
	synFees     = "0x000000000001000000000000000000000000000000000000000e000000000001"
)

var e18 = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

// mainnetPerps: the only markets free, live, and already proven (same feed testnet uses) at
// initial mainnet setup. Real US stocks/ETFs need a licensed production price feed first (no free
// one is wired into this backend today — see deploy.go's setup doc comment) and can be added later
// via a plain owner `upsert_perp` call, no redeploy required.
var mainnetPerps = []struct {
	id     uint32
	symbol string
}{{1, "ETH"}, {3, "BTC"}, {5, "SOL"}}

// hyperliquidPrices returns x18 mid prices for the given symbols, identical to nstestnet's helper.
func hyperliquidPrices(ctx context.Context, symbols []string) (map[string]*big.Int, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.hyperliquid.xyz/info", strings.NewReader(`{"type":"allMids"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var mids map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&mids); err != nil {
		return nil, err
	}
	out := map[string]*big.Int{}
	for _, s := range symbols {
		f, ok := new(big.Float).SetString(mids[s])
		if !ok {
			return nil, fmt.Errorf("no Hyperliquid price for %s", s)
		}
		v, _ := new(big.Float).Mul(f, new(big.Float).SetInt(e18)).Int(nil)
		out[s] = v
	}
	return out, nil
}

func wasmPath(name string) string {
	wd, _ := os.Getwd()
	return filepath.Join(wd, "near-stocks-contracts", "target", "near", name, name+".wasm")
}

func signer(account string) (*nearchain.LocalSigner, error) {
	c, err := loadCreds(account)
	if err != nil {
		return nil, fmt.Errorf("credentials for %s: %w", account, err)
	}
	return nearchain.ParseNearPrivateKey(c.PrivateKey)
}

// send signs and sends actions from `from` to `to`, and fails unless the transaction succeeds.
// Identical in shape to nstestnet's send/call — duplicated rather than shared because these are
// two independent `main` packages (Development.md's existing convention for the nstestnet/nsmainnet split).
func send(ctx context.Context, rpc *nearchain.Client, from, to string, actions ...nearchain.Action) (*nearchain.TxResult, error) {
	s, err := signer(from)
	if err != nil {
		return nil, err
	}
	key, err := rpc.ViewAccessKey(ctx, from, nearchain.PublicKeyString(s.PublicKey()))
	if err != nil {
		return nil, err
	}
	block, err := rpc.FinalBlockHash(ctx)
	if err != nil {
		return nil, err
	}
	tx := &nearchain.Transaction{SignerId: from, PublicKey: s.PublicKey(), Nonce: key.Nonce + 1, ReceiverId: to, BlockHash: block, Actions: actions}
	signed, hash, err := nearchain.SignTransaction(tx, s)
	if err != nil {
		return nil, err
	}
	res, err := rpc.SendTx(ctx, signed)
	if err != nil {
		return nil, fmt.Errorf("%s -> %s (%s): %w", from, to, nearchain.HashString(hash), err)
	}
	if !res.Succeeded() {
		return res, fmt.Errorf("%s -> %s (%s) failed: %s", from, to, nearchain.HashString(hash), nearchain.FailureMessage(res))
	}
	return res, nil
}

func callFn(method string, args any, tgas uint64, deposit *big.Int) nearchain.FunctionCall {
	raw, _ := json.Marshal(args)
	if deposit == nil {
		deposit = new(big.Int)
	}
	return nearchain.FunctionCall{MethodName: method, Args: raw, Gas: tgas * nearchain.TGas, Deposit: deposit}
}

// fund transfers plain NEAR from a funded account to another account. Both must already exist —
// this is a Transfer action, not account creation.
func fund(ctx context.Context, rpc *nearchain.Client, from, to string, amount *big.Int) error {
	res, err := send(ctx, rpc, from, to, nearchain.Transfer{Deposit: amount})
	if err != nil {
		return err
	}
	fmt.Printf("transferred %s yoctoNEAR %s -> %s, tx %s\n", amount, from, to, res.Transaction.Hash)
	return nil
}

// createAccount registers a brand-new second-level ".near" name. Unlike a subaccount (which its
// own parent can create directly with a CreateAccount action), "near-stocks.near" is a direct
// child of the "near" TLD registrar account, so creation must go through that account's own
// `create_account` method — it internally does CreateAccount + Transfer(deposit) + AddKey(pubkey,
// FullAccess) as one receipt. `deposit` becomes the new account's starting balance (must cover the
// contract code's storage cost once deployed, plus its own access key storage and gas headroom).
//
// VERIFY INDEPENDENTLY: this assumes mainnet's top-level registrar is still the "near" account
// itself, matching how near-cli/near-api-js have always created second-level .near names.
func createAccount(ctx context.Context, rpc *nearchain.Client, funder, newAccount string, deposit *big.Int) error {
	c, err := loadCreds(newAccount)
	if err != nil {
		return fmt.Errorf("no local credentials for %s yet — run `generate %s` first: %w", newAccount, newAccount, err)
	}
	res, err := send(ctx, rpc, funder, "near", callFn("create_account", map[string]any{
		"new_account_id": newAccount, "new_public_key": c.PublicKey,
	}, 30, deposit))
	if err != nil {
		return err
	}
	fmt.Printf("created %s funded with %s yoctoNEAR, tx %s\n", newAccount, deposit, res.Transaction.Hash)
	return nil
}

func deployCore(ctx context.Context, rpc *nearchain.Client) error {
	code, err := os.ReadFile(wasmPath("near_stocks_core"))
	if err != nil {
		return fmt.Errorf("build it first (cargo near build non-reproducible-wasm --no-embed-abi in near-stocks-contracts/core): %w", err)
	}
	init := callFn("new", map[string]any{
		"owner": ownerAccount, "guardian": guardianAccount, "sequencer": seqAccount, "chain_id": chainIdMainnet, "broker_id": brokerId,
		"amm_subaccount": ammSub, "insurance_subaccount": insSub, "fee_subaccount": feeSub,
	}, 100, nil)
	res, err := send(ctx, rpc, coreAccount, coreAccount, nearchain.DeployContract{Code: code}, init)
	if err != nil {
		return err
	}
	fmt.Printf("deployed %s (%d bytes), tx %s\n", coreAccount, len(code), res.Transaction.Hash)
	return nil
}

// setup configures USDC as the sole collateral/spot asset, lists ETH/BTC/SOL perps at live
// Hyperliquid prices (free, already proven on testnet — the only feed ready today), and finishes
// migration. Real US stocks/ETFs stay deferred: no free, licensed production price feed is wired
// into this backend yet (Tiingo/Finnhub/Polygon/Ondo all still need to be evaluated and integrated
// — see the conversation this was written from). `upsert_perp` is a plain owner call the moment one
// is chosen, no redeploy needed. Also does NOT call `set_claim_limits` — that method no longer
// exists on this contract (LogX/rewards removed).
func setup(ctx context.Context, rpc *nearchain.Client) error {
	x := func(num, den int64) string {
		return new(big.Int).Div(new(big.Int).Mul(big.NewInt(num), e18), big.NewInt(den)).String()
	}

	// idempotent: check current state first so re-running after a partial failure never re-sends
	// finish_migration (a repeat call panics and aborts the whole containing transaction, including
	// any other actions batched alongside it — that's exactly what happened the first time this ran).
	cfgRaw, err := rpc.CallView(ctx, coreAccount, "get_config", map[string]any{})
	if err != nil {
		return fmt.Errorf("get_config: %w", err)
	}
	var cfg struct {
		Migrated bool `json:"migrated"`
	}
	if err := json.Unmarshal(cfgRaw, &cfg); err != nil {
		return fmt.Errorf("get_config: %w", err)
	}

	var symbols []string
	for _, p := range mainnetPerps {
		symbols = append(symbols, p.symbol)
	}
	prices, err := hyperliquidPrices(ctx, symbols)
	if err != nil {
		return fmt.Errorf("fetching Hyperliquid prices: %w", err)
	}
	actions := []nearchain.Action{
		callFn("upsert_spot", map[string]any{"product_id": 4, "config": map[string]any{
			"token": nearchain.USDCMainnet, "decimals": 6, "weighted": true, "withdraw_fee_x18": x(1, 2), "price_x18": e18.String(), "max_deviation_bps": 200}}, 20, nil),
	}
	for _, p := range mainnetPerps {
		actions = append(actions, callFn("upsert_perp", map[string]any{"product_id": p.id, "config": map[string]any{
			"imf_x18": x(1, 10), "mmf_x18": x(1, 20), "liq_frac_x18": x(15, 1000), "price_x18": prices[p.symbol].String(),
			"max_deviation_bps": 1000, "amm_max_position_x18": "0"}}, 20, nil))
	}
	actions = append(actions,
		callFn("set_product_order", map[string]any{"spot_order": []int{4}, "collateral_order": []int{4}}, 20, nil),
		callFn("set_product_accounts", map[string]any{"options_x": optionsX, "options_fees": optionsFees, "pre_market_x": preX,
			"pre_market_fees": preFees, "synthetic_x": synX, "synthetic_fees": synFees}, 20, nil),
	)
	if !cfg.Migrated {
		actions = append(actions, callFn("finish_migration", map[string]any{"n_submissions": 0}, 20, nil))
	} else {
		fmt.Println("migration already finished, skipping finish_migration (re-sending product config only)")
	}
	if _, err := send(ctx, rpc, ownerAccount, coreAccount, actions...); err != nil {
		return err
	}
	fmt.Printf("configured %s: USDC collateral, perps %v at Hyperliquid prices, product accounts\n", coreAccount, symbols)

	// the contract must hold USDC storage to be able to receive it — also idempotent, skip if
	// already registered (a second storage_deposit is harmless but wastes the deposit as a top-up;
	// checking is free).
	storageRaw, err := rpc.CallView(ctx, nearchain.USDCMainnet, "storage_balance_of", map[string]any{"account_id": coreAccount})
	if err != nil {
		return fmt.Errorf("storage_balance_of: %w", err)
	}
	if string(storageRaw) == "null" {
		storage := new(big.Int).Mul(big.NewInt(125), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)) // 0.00125 NEAR
		if _, err := send(ctx, rpc, ownerAccount, nearchain.USDCMainnet, callFn("storage_deposit", map[string]any{"account_id": coreAccount, "registration_only": true}, 30, storage)); err != nil {
			return fmt.Errorf("storage_deposit on USDC: %w", err)
		}
		fmt.Printf("registered %s with USDC storage\n", coreAccount)
	} else {
		fmt.Printf("%s already registered with USDC storage\n", coreAccount)
	}
	return nil
}

func smoke(ctx context.Context, rpc *nearchain.Client) error {
	for _, m := range []string{"get_config", "get_products"} {
		raw, err := rpc.CallView(ctx, coreAccount, m, map[string]any{})
		if err != nil {
			return err
		}
		fmt.Printf("%s: %s\n\n", m, raw)
	}
	return nil
}
