package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github/eugenix-io/logx-inf-backend/nearchain"
)

// Testnet layout (created with `create`). Owner = DAO; the contract account itself only deploys.
const (
	coreAccount     = "near-stocks.testnet"
	daoAccount      = "near-stocks-dao.testnet"
	guardianAccount = "near-stocks-guardian.testnet"
	seqAccount      = "near-stocks-seq.testnet"
	logxAccount     = "near-stocks-logx.testnet"
	usdcTestnet     = "3e2210e1184b45b64c8a434c0a7e7b23cc04ea7eb7a6c3c32520d03d4afcb8af" // Circle native USDC
	chainIdTestnet  = 398
	brokerId        = 2
	// system subaccounts: contractUtils AMM / INSURANCE / TRADING_FEES / OPTIONS / PRE_MARKETS / SYN_SPOTS
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

// perps on testnet: product id -> Hyperliquid symbol (ids as in constants.utils.go)
var testnetPerps = []struct {
	id     uint32
	symbol string
}{{1, "ETH"}, {3, "BTC"}, {5, "SOL"}}

var e18 = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
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

func call(method string, args any, tgas uint64, deposit *big.Int) nearchain.FunctionCall {
	raw, _ := json.Marshal(args)
	if deposit == nil {
		deposit = new(big.Int)
	}
	return nearchain.FunctionCall{MethodName: method, Args: raw, Gas: tgas * nearchain.TGas, Deposit: deposit}
}

func deployCore(ctx context.Context, rpc *nearchain.Client) error {
	code, err := os.ReadFile(wasmPath("near_stocks_core"))
	if err != nil {
		return fmt.Errorf("build it first (cargo near build in near-stocks-contracts/core): %w", err)
	}
	init := call("new", map[string]any{
		"owner": daoAccount, "guardian": guardianAccount, "sequencer": seqAccount, "chain_id": chainIdTestnet, "broker_id": brokerId,
		"amm_subaccount": ammSub, "insurance_subaccount": insSub, "fee_subaccount": feeSub,
	}, 100, nil)
	res, err := send(ctx, rpc, coreAccount, coreAccount, nearchain.DeployContract{Code: code}, init)
	if err != nil {
		return err
	}
	fmt.Printf("deployed %s (%d bytes), tx %s\n", coreAccount, len(code), res.Transaction.Hash)
	return nil
}

func deployLogx(ctx context.Context, rpc *nearchain.Client) error {
	code, err := os.ReadFile(wasmPath("logx_token"))
	if err != nil {
		return fmt.Errorf("build it first (cargo near build in near-stocks-contracts/logx-token): %w", err)
	}
	supply := new(big.Int).Mul(big.NewInt(100_000_000), e18)
	init := call("new", map[string]any{
		"owner_id": daoAccount, "total_supply": supply.String(),
		"metadata": map[string]any{"spec": "ft-1.0.0", "name": "LogX (testnet)", "symbol": "LOGX", "decimals": 18},
	}, 100, nil)
	res, err := send(ctx, rpc, logxAccount, logxAccount, nearchain.DeployContract{Code: code}, init)
	if err != nil {
		return err
	}
	fmt.Printf("deployed %s, 100,000,000 LOGX minted to %s, tx %s\n", logxAccount, daoAccount, res.Transaction.Hash)
	return nil
}

// hyperliquidPrices returns x18 mid prices for the given symbols.
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

func setup(ctx context.Context, rpc *nearchain.Client) error {
	var symbols []string
	for _, p := range testnetPerps {
		symbols = append(symbols, p.symbol)
	}
	prices, err := hyperliquidPrices(ctx, symbols)
	if err != nil {
		return err
	}
	x := func(num, den int64) string {
		return new(big.Int).Div(new(big.Int).Mul(big.NewInt(num), e18), big.NewInt(den)).String()
	}
	actions := []nearchain.Action{
		call("upsert_spot", map[string]any{"product_id": 4, "config": map[string]any{
			"token": usdcTestnet, "decimals": 6, "weighted": true, "withdraw_fee_x18": x(1, 2), "price_x18": e18.String(), "max_deviation_bps": 200}}, 20, nil),
		call("upsert_spot", map[string]any{"product_id": 0, "config": map[string]any{
			"token": logxAccount, "decimals": 18, "weighted": false, "withdraw_fee_x18": x(25, 1), "price_x18": "0", "max_deviation_bps": 0}}, 20, nil),
		call("upsert_spot", map[string]any{"product_id": 2, "config": map[string]any{
			"token": nil, "decimals": 18, "weighted": false, "withdraw_fee_x18": "0", "price_x18": "0", "max_deviation_bps": 0}}, 20, nil),
	}
	for _, p := range testnetPerps {
		actions = append(actions, call("upsert_perp", map[string]any{"product_id": p.id, "config": map[string]any{
			"imf_x18": x(1, 10), "mmf_x18": x(1, 20), "liq_frac_x18": x(15, 1000), "price_x18": prices[p.symbol].String(),
			"max_deviation_bps": 1000, "amm_max_position_x18": "0"}}, 20, nil))
	}
	actions = append(actions,
		call("set_product_order", map[string]any{"spot_order": []int{4}, "collateral_order": []int{4}}, 20, nil),
		call("set_product_accounts", map[string]any{"options_x": optionsX, "options_fees": optionsFees, "pre_market_x": preX,
			"pre_market_fees": preFees, "synthetic_x": synX, "synthetic_fees": synFees}, 20, nil),
		call("set_claim_limits", map[string]any{"limits": map[string]any{"max_logx_claim_x18": x(100_000, 1), "max_reward_claim_x18": x(10_000, 1)}}, 20, nil),
		call("finish_migration", map[string]any{"n_submissions": 0}, 20, nil),
	)
	if _, err := send(ctx, rpc, daoAccount, coreAccount, actions...); err != nil {
		return err
	}
	fmt.Printf("configured %s: USDC, LogX, stLogX, perps %v at Hyperliquid prices, product accounts, claim limits, migration finished\n", coreAccount, symbols)
	// the contract must hold token storage to receive USDC and LogX
	storage := new(big.Int).Mul(big.NewInt(125), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)) // 0.00125 NEAR
	for _, token := range []string{usdcTestnet, logxAccount} {
		if _, err := send(ctx, rpc, daoAccount, token, call("storage_deposit", map[string]any{"account_id": coreAccount, "registration_only": true}, 30, storage)); err != nil {
			return fmt.Errorf("storage_deposit on %s: %w", token, err)
		}
	}
	fmt.Printf("registered %s with USDC and LogX storage\n", coreAccount)
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
