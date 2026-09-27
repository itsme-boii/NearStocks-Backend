package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

const userAccount = "near-stocks-user1.testnet"

func sequencer() (*nearchain.Sequencer, error) {
	s, err := signer(seqAccount)
	if err != nil {
		return nil, err
	}
	return &nearchain.Sequencer{RPC: nearchain.NewClient(rpcURL), Signer: s, SignerId: seqAccount, Contract: coreAccount}, nil
}

func submit(ctx context.Context, seq *nearchain.Sequencer, label string, payload, sig []byte) error {
	idx, err := seq.NSubmissions(ctx)
	if err != nil {
		return err
	}
	if sig == nil {
		sig = []byte{}
	}
	res := seq.Submit(ctx, idx, [][]byte{payload}, [][]byte{sig}, [][]byte{{}})
	if res.Outcome != nearchain.Landed {
		return fmt.Errorf("%s: outcome %d: %s", label, res.Outcome, res.Failure)
	}
	fmt.Printf("  %-28s landed at index %d, tx %s\n", label, idx, res.TxHash)
	return nil
}

func ftBalance(ctx context.Context, rpc *nearchain.Client, token, account string) string {
	raw, err := rpc.CallView(ctx, token, "ft_balance_of", map[string]any{"account_id": account})
	if err != nil {
		return "?"
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func units(v string) string {
	b, ok := new(big.Int).SetString(v, 10)
	if !ok {
		return v
	}
	q, r := new(big.Int).QuoRem(b, e18, new(big.Int))
	if r.Sign() == 0 {
		return q.String()
	}
	return b.String() + " wei"
}

func ledger(ctx context.Context, rpc *nearchain.Client, sub string) (map[string]string, error) {
	raw, err := rpc.CallView(ctx, coreAccount, "get_subaccount", map[string]any{"subaccount": sub})
	if err != nil {
		return nil, err
	}
	var v struct {
		Spots [][2]json.RawMessage `json:"spots"`
		Nonce uint64               `json:"nonce"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	out := map[string]string{"nonce": fmt.Sprint(v.Nonce)}
	for _, e := range v.Spots {
		var pid uint32
		var bal string
		_ = json.Unmarshal(e[0], &pid)
		_ = json.Unmarshal(e[1], &bal)
		out[map[uint32]string{0: "LogX", 2: "stLogX", 4: "USDC"}[pid]] = units(bal)
	}
	return out, nil
}

// e2e runs real flows on testnet through the production Go encoders, signing and sequencer.
func e2e(ctx context.Context, rpc *nearchain.Client) error {
	seq, err := sequencer()
	if err != nil {
		return err
	}
	fmt.Println("1. price tick (PERPTICK) from the Go sequencer")
	var syms []string
	for _, p := range testnetPerps {
		syms = append(syms, p.symbol)
	}
	prices, err := hyperliquidPrices(ctx, syms)
	if err != nil {
		return err
	}
	tick := nearchain.PerpTick{Time: uint64(time.Now().Unix() - 5), Prices: []nearchain.PidValue{{ProductId: 4, Value: e18}}}
	for _, p := range testnetPerps {
		tick.Prices = append(tick.Prices, nearchain.PidValue{ProductId: p.id, Value: prices[p.symbol]})
	}
	if err := submit(ctx, seq, "PERPTICK", tick.Encode(), nil); err != nil {
		return err
	}

	fmt.Println("2. test user account and session key")
	if _, err := os.Stat(credsPath(userAccount)); err != nil {
		if err := create(userAccount); err != nil {
			return err
		}
		time.Sleep(3 * time.Second)
	}
	sessionKey, _ := crypto.GenerateKey() // this run's browser session key (not stored)
	sessionAddr := crypto.PubkeyToAddress(sessionKey.PublicKey).Hex()
	expiry := time.Now().Add(24 * time.Hour).UnixMilli()
	deposit := new(big.Int).Mul(big.NewInt(5), new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil)) // 0.05 NEAR
	if _, err := send(ctx, rpc, userAccount, coreAccount, call("register_session_key",
		map[string]any{"subaccount_number": 1, "session_key": strings.ToLower(sessionAddr), "expiry_ms": expiry}, 30, deposit)); err != nil {
		return err
	}
	addr20, _ := cutils.NearAccountToAddr20(userAccount)
	subId := cutils.CreateSubaccountId(brokerId, addr20, 1)
	subB, _ := cutils.SubaccountIdToBytes32(subId)
	subHex := "0x" + common.Bytes2Hex(subB[:])
	fmt.Printf("  registered session key %s for %s (subaccount %s)\n", sessionAddr, userAccount, subId)

	fmt.Println("3. LogX: the DAO funds the user, the user deposits into near-stocks with ft_transfer_call")
	storage := new(big.Int).Mul(big.NewInt(125), new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil))
	one := big.NewInt(1)
	if _, err := send(ctx, rpc, daoAccount, logxAccount,
		call("storage_deposit", map[string]any{"account_id": userAccount, "registration_only": true}, 30, storage),
		call("ft_transfer", map[string]any{"receiver_id": userAccount, "amount": new(big.Int).Mul(big.NewInt(1000), e18).String()}, 30, one)); err != nil {
		return err
	}
	if _, err := send(ctx, rpc, userAccount, logxAccount, call("ft_transfer_call", map[string]any{
		"receiver_id": coreAccount, "amount": new(big.Int).Mul(big.NewInt(500), e18).String(), "msg": ""}, 100, one)); err != nil {
		return err
	}
	l, err := ledger(ctx, rpc, subHex)
	if err != nil {
		return err
	}
	fmt.Printf("  wallet LogX %s, contract ledger %v\n", units(ftBalance(ctx, rpc, logxAccount, userAccount)), l)

	sign := func(primary string, fields []apitypes.Type, msg apitypes.TypedDataMessage) ([]byte, error) {
		h, err := contractUtils.NearTypedHash(coreAccount, chainIdTestnet, primary, fields, msg)
		if err != nil {
			return nil, err
		}
		sig, err := crypto.Sign(h, sessionKey)
		if err != nil {
			return nil, err
		}
		sig[64] += 27
		return sig, nil
	}
	nonce := func() int64 {
		l, _ := ledger(ctx, rpc, subHex)
		var n int64
		fmt.Sscan(l["nonce"], &n)
		return n
	}

	fmt.Println("4. stake 100 LogX (STAKE_LOGX, session-key signed, sequencer submitted)")
	n := nonce()
	staker := "0x0000000000000000000000000000000000000000"
	sig, err := sign("StakeLogXRequest", contractUtils.NearStakeType, apitypes.TypedDataMessage{
		"subAccountId": subB[:], "productId": math.NewHexOrDecimal256(0), "tokenAmount": new(big.Int).Mul(big.NewInt(100), e18).String(),
		"stakerContract": staker, "sessionKey": sessionAddr, "nonce": math.NewHexOrDecimal256(n), "chainId": math.NewHexOrDecimal256(chainIdTestnet)})
	if err != nil {
		return err
	}
	st := nearchain.StakeRequest{Subaccount: subB, ProductId: 0, Amount: new(big.Int).Mul(big.NewInt(100), e18), StakerContract: common.HexToAddress(staker), SessionKey: common.HexToAddress(sessionAddr), Nonce: big.NewInt(n)}
	if err := submit(ctx, seq, "STAKE_LOGX", st.EncodeStake(), sig); err != nil {
		return err
	}
	l, _ = ledger(ctx, rpc, subHex)
	fmt.Printf("  contract ledger %v\n", l)

	fmt.Println("5. withdraw 100 LogX to the user's wallet (WITHDRAW_LOGX, 25 LogX fee)")
	n = nonce()
	amount := new(big.Int).Mul(big.NewInt(100), e18)
	w := contractUtils.NearWithdraw{SubaccountId: subId, SessionKey: sessionAddr, ProductId: 0, Amount: amount.String(), Nonce: n, Receiver: userAccount}
	h, err := contractUtils.NearWithdrawHash(coreAccount, chainIdTestnet, w)
	if err != nil {
		return err
	}
	wsig, _ := crypto.Sign(h, sessionKey)
	wsig[64] += 27
	wp := nearchain.NearWithdraw{Subaccount: subB, SessionKey: common.HexToAddress(sessionAddr), ProductId: 0, Amount: amount, Nonce: big.NewInt(n), Receiver: userAccount}
	if err := submit(ctx, seq, "WITHDRAW_LOGX", wp.EncodeLogX(), wsig); err != nil {
		return err
	}
	time.Sleep(3 * time.Second) // the ft_transfer and its callback run in later blocks
	l, _ = ledger(ctx, rpc, subHex)
	fmt.Printf("  wallet LogX %s, contract ledger %v\n", units(ftBalance(ctx, rpc, logxAccount, userAccount)), l)
	fees, _ := ledger(ctx, rpc, feeSub)
	fmt.Printf("  fee account %v\n", fees)
	fmt.Println("done: every step used the production Go encoders and signing against the live contract")
	return nil
}
