package main

// Phase 4 vectors:
//   payloads.json  every submit_transactions payload, Borsh-encoded by nearchain/payloads.go
//   requests.json  every session-key request type kept on NEAR, hashed by go-ethereum under the
//                  near-stocks domain and signed
// Rust (core/tests/vectors.rs) decodes each payload, compares every field and re-encodes it;
// Rust and TypeScript recompute each request digest and recover the signer.

import (
	"encoding/hex"
	"log"
	"math/big"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

func bi(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		log.Fatalf("bad int %q", s)
	}
	return v
}

func b32(sub string) [32]byte {
	b, err := cutils.SubaccountIdToBytes32(sub)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func b20(addr string) [20]byte { return common.HexToAddress(addr) }

func pv(pairs ...string) []nearchain.PidValue {
	var out []nearchain.PidValue
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, nearchain.PidValue{ProductId: uint32(bi(pairs[i]).Uint64()), Value: bi(pairs[i+1])})
	}
	return out
}

func pvJSON(v []nearchain.PidValue) [][2]string {
	out := [][2]string{}
	for _, p := range v {
		out = append(out, [2]string{big.NewInt(int64(p.ProductId)).String(), p.Value.String()})
	}
	return out
}

func orderJSON(o nearchain.Order) map[string]any {
	return map[string]any{
		"subaccount": hx(o.Subaccount[:]), "priceX18": o.PriceX18.String(), "amount": o.Amount.String(),
		"expiration": o.Expiration, "isReduce": o.IsReduce, "sessionKey": hx(o.SessionKey[:]), "productId": o.ProductId,
	}
}

func genPayloads(dir string) {
	addrA, _ := cutils.NearAccountToAddr20("alice.near")
	addrB, _ := cutils.NearAccountToAddr20("bob.near")
	subA, subB := b32(cutils.CreateSubaccountId(2, addrA, 0)), b32(cutils.CreateSubaccountId(2, addrB, 0))
	key := b20("0x4599B3307aFd037958b1f583CC5F44F62dc3E970")
	staker := b20("0x00000000000000000000000000000000000000aa")
	taker := nearchain.Order{Subaccount: subA, PriceX18: bi("0"), Amount: bi("-1500000000000000000"), Expiration: 1790000000000, IsReduce: true, SessionKey: key, ProductId: 3}
	maker := nearchain.Order{Subaccount: subB, PriceX18: bi("65000123456789012345678"), Amount: bi("2000000000000000000"), Expiration: 1790000000001, SessionKey: key, ProductId: 3}
	minI128 := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 127))

	type c = map[string]any
	var cases []c
	add := func(name string, enc []byte, fields c) {
		cases = append(cases, c{"name": name, "type": int(enc[0]), "hex": "0x" + hex.EncodeToString(enc), "fields": fields})
	}

	tick := nearchain.PerpTick{Time: 1760000123, Rates: pv("1", "-871494086134", "3", "391173349443"), Prices: pv("1", "2500123000000000000000", "4", "999900000000000000")}
	add("perp_tick", tick.Encode(), c{"time": tick.Time, "rates": pvJSON(tick.Rates), "prices": pvJSON(tick.Prices)})
	add("perp_tick_empty", nearchain.PerpTick{Time: 7}.Encode(), c{"time": 7, "rates": [][2]string{}, "prices": [][2]string{}})

	m := nearchain.MatchOrders{ProductId: 3, Taker: taker, Maker: maker, MatchedAmount: bi("1500000000000000000")}
	add("match_orders", m.Encode(), c{"productId": 3, "taker": orderJSON(taker), "maker": orderJSON(maker), "matchedAmount": m.MatchedAmount.String()})

	l := nearchain.Liquidate{ProductId: 3, Prices: pv("3", "60000000000000000000000", "4", "1000000000000000000"), Liquidator: maker, Liquidatee: subA, Amount: minI128}
	add("liquidate", l.Encode(), c{"productId": 3, "prices": pvJSON(l.Prices), "liquidator": orderJSON(maker), "liquidatee": hx(subA[:]), "amount": minI128.String()})

	w := nearchain.NearWithdraw{Subaccount: subA, SessionKey: key, ProductId: 4, Amount: bi("340282366920938463463374607431768211455"), Nonce: bi("12"), Receiver: "0f5c6d7e8a9b0c1d2e3f405162738495a6b7c8d9e0f1a2b3c4d5e6f708192a3b"}
	wf := c{"subaccount": hx(subA[:]), "sessionKey": hx(key[:]), "productId": 4, "amount": w.Amount.String(), "nonce": "12", "receiver": w.Receiver}
	add("withdraw_collateral", w.Encode(), wf)
	wl := w
	wl.ProductId, wl.Amount, wl.Receiver = 0, bi("100000000000000000000"), "alice.near"
	add("withdraw_logx", wl.EncodeLogX(), c{"subaccount": hx(subA[:]), "sessionKey": hx(key[:]), "productId": 0, "amount": wl.Amount.String(), "nonce": "12", "receiver": "alice.near"})

	s := nearchain.SettleUserPnl{Subaccounts: [][32]byte{subA, subB}, SpotPrices: pv("4", "1000000000000000000", "74", "999999999999999999")}
	add("settle_user_pnl", s.Encode(), c{"subaccounts": []string{hx(subA[:]), hx(subB[:])}, "spotPrices": pvJSON(s.SpotPrices)})
	so := nearchain.Socialise{Subaccount: subB, SpotPrices: pv("4", "1000000000000000000")}
	add("socialise", so.Encode(), c{"subaccount": hx(subB[:]), "spotPrices": pvJSON(so.SpotPrices)})
	add("set_nonce", nearchain.SetNonce{Subaccount: subA, Delta: 5}.Encode(), c{"subaccount": hx(subA[:]), "delta": 5})

	bet := nearchain.OptionBet{Subaccount: subA, ProductId: 3, Amount: bi("-100000000000000000000"), Interval: 5, Nonce: bi("9"), SessionKey: key}
	po := nearchain.PlaceOption{Bet: bet, OrderId: 424242, EntryPriceX18: bi("65000000000000000000000"), PayoutPct: 180, FeePct: 5}
	add("place_option", po.Encode(), c{"subaccount": hx(subA[:]), "productId": 3, "amount": bet.Amount.String(), "interval": 5, "nonce": "9", "sessionKey": hx(key[:]),
		"orderId": 424242, "entryPriceX18": po.EntryPriceX18.String(), "payoutPct": 180, "feePct": 5})
	co := nearchain.CloseOption{OrderId: 424242, ExitPriceX18: bi("64999999999999999999999")}
	add("close_option", co.Encode(), c{"orderId": 424242, "exitPriceX18": co.ExitPriceX18.String()})

	pool := nearchain.PoolTrade{Order: nearchain.PoolOrder{Subaccount: subA, ProductId: 1001, Amount: bi("100000000000000000000"), IsBuy: true, Nonce: bi("10"), SessionKey: key}, QuoteDelta: bi("50000000000000000000"), Fees: bi("1000000000000000000")}
	pf := c{"subaccount": hx(subA[:]), "productId": 1001, "amount": pool.Order.Amount.String(), "isBuy": true, "nonce": "10", "sessionKey": hx(key[:]), "quoteDelta": pool.QuoteDelta.String(), "fees": pool.Fees.String()}
	add("pre_market_order", pool.EncodePreMarket(), pf)
	add("synthetic_spot_order", pool.EncodeSyntheticSpot(), pf)

	st := nearchain.StakeRequest{Subaccount: subA, ProductId: 0, Amount: bi("400000000000000000000"), StakerContract: staker, SessionKey: key, Nonce: bi("11")}
	sf := c{"subaccount": hx(subA[:]), "productId": 0, "amount": st.Amount.String(), "stakerContract": hx(staker[:]), "sessionKey": hx(key[:]), "nonce": "11"}
	add("stake_logx", st.EncodeStake(), sf)
	add("unstake_logx", st.EncodeUnstake(), sf)

	cr := nearchain.ClaimRewards{Subaccount: subA, SessionKey: key, StakerContract: staker, ProductId: 0, Nonce: bi("13"), AmountX18: bi("42000000000000000000")}
	add("claim_rewards", cr.Encode(), c{"subaccount": hx(subA[:]), "sessionKey": hx(key[:]), "stakerContract": hx(staker[:]), "productId": 0, "nonce": "13", "amountX18": cr.AmountX18.String()})
	cl := nearchain.ClaimLogX{Subaccount: subA, TokenAmount: bi("1000000000000000000000"), SessionKey: key, Nonce: bi("14")}
	add("claim_logx", cl.Encode(), c{"subaccount": hx(subA[:]), "tokenAmount": cl.TokenAmount.String(), "sessionKey": hx(key[:]), "nonce": "14"})
	add("reward_rate_tick", nearchain.RewardRateTick{CumulativeRateX18: bi("5000000000000000000")}.Encode(), c{"cumulativeRateX18": "5000000000000000000"})

	args := nearchain.SubmitArgs(77, [][]byte{tick.Encode(), m.Encode()}, [][]byte{{}, {1, 2, 3}}, [][]byte{{}, {4}})
	write(dir, "payloads.json", map[string]any{
		"spec":   "u8 type ‖ borsh(payload); field order = near-stocks-contracts/core/src/tx.rs; generated by nearchain/payloads.go",
		"cases":  cases,
		"submit": map[string]any{"idx": 77, "txs": []string{hx(tick.Encode()), hx(m.Encode())}, "sigs": []string{"0x", "0x010203"}, "sigs2": []string{"0x", "0x04"}, "hex": hx(args)},
	})
}

func genRequests(dir string) {
	sk, _ := crypto.ToECDSA(crypto.Keccak256([]byte("near-stocks vector session key 1")))
	key := crypto.PubkeyToAddress(sk.PublicKey).Hex()
	addr, _ := cutils.NearAccountToAddr20("alice.near")
	sub := b32(cutils.CreateSubaccountId(2, addr, 0))
	acct, chain := contractUtils.NEAR_STOCKS_MAINNET_ACCOUNT, int64(contractUtils.NEAR_STOCKS_MAINNET_CHAIN_ID)
	staker := "0x00000000000000000000000000000000000000aa"
	d := func(v int64) *math.HexOrDecimal256 { return math.NewHexOrDecimal256(v) }

	type req struct {
		primary string
		fields  []apitypes.Type
		msg     apitypes.TypedDataMessage
		json    map[string]any
	}
	reqs := []req{
		{"UserOptionBet", contractUtils.NearUserOptionBetType,
			apitypes.TypedDataMessage{"subAccountId": sub, "productId": d(3), "amount": "-100000000000000000000", "interval": d(5), "nonce": d(9), "sessionKey": key, "chainId": d(chain)},
			map[string]any{"productId": 3, "amount": "-100000000000000000000", "interval": 5, "nonce": "9"}},
		{"PlacePreMarketOrderRequest", contractUtils.NearPoolOrderType,
			apitypes.TypedDataMessage{"subAccountId": sub, "productId": d(1001), "amount": "100000000000000000000", "isBuy": true, "nonce": d(10), "sessionKey": key, "chainId": d(chain)},
			map[string]any{"productId": 1001, "amount": "100000000000000000000", "isBuy": true, "nonce": "10"}},
		{"PlaceSyntheticSpotOrderRequest", contractUtils.NearPoolOrderType,
			apitypes.TypedDataMessage{"subAccountId": sub, "productId": d(2001), "amount": "7000000000000000000", "isBuy": false, "nonce": d(11), "sessionKey": key, "chainId": d(chain)},
			map[string]any{"productId": 2001, "amount": "7000000000000000000", "isBuy": false, "nonce": "11"}},
		{"StakeLogXRequest", contractUtils.NearStakeType,
			apitypes.TypedDataMessage{"subAccountId": sub, "productId": d(0), "tokenAmount": "400000000000000000000", "stakerContract": staker, "sessionKey": key, "nonce": d(12), "chainId": d(chain)},
			map[string]any{"productId": 0, "amount": "400000000000000000000", "stakerContract": staker, "nonce": "12"}},
		{"UnstakeLogXRequest", contractUtils.NearUnstakeType,
			apitypes.TypedDataMessage{"subAccountId": sub, "productId": d(0), "amount": "100000000000000000000", "stakerContract": staker, "sessionKey": key, "nonce": d(13), "chainId": d(chain)},
			map[string]any{"productId": 0, "amount": "100000000000000000000", "stakerContract": staker, "nonce": "13"}},
		{"ClaimRewards", contractUtils.NearClaimRewardsType,
			apitypes.TypedDataMessage{"subAccountId": sub, "sessionKey": key, "stakerContract": staker, "productId": d(0), "nonce": d(14), "chainId": d(chain)},
			map[string]any{"productId": 0, "stakerContract": staker, "nonce": "14"}},
		{"ClaimLogX", contractUtils.NearClaimLogXType,
			apitypes.TypedDataMessage{"subAccountId": sub, "tokenAmount": "1000000000000000000000", "sessionKey": key, "nonce": d(15), "chainId": d(chain)},
			map[string]any{"amount": "1000000000000000000000", "nonce": "15"}},
	}
	var out []map[string]any
	for _, r := range reqs {
		h, err := contractUtils.NearTypedHash(acct, chain, r.primary, r.fields, r.msg)
		if err != nil {
			log.Fatalf("%s: %v", r.primary, err)
		}
		sig, err := crypto.Sign(h, sk)
		if err != nil {
			log.Fatal(err)
		}
		sig[64] += 27
		if err := contractUtils.VerifyNearTyped(acct, chain, r.primary, r.fields, r.msg, hx(sig), key); err != nil {
			log.Fatalf("%s verify: %v", r.primary, err)
		}
		types := []map[string]string{}
		for _, f := range r.fields {
			types = append(types, map[string]string{"name": f.Name, "type": f.Type})
		}
		r.json["primaryType"], r.json["types"], r.json["digest"], r.json["signature"] = r.primary, types, hx(h), hx(sig)
		out = append(out, r.json)
	}
	write(dir, "requests.json", map[string]any{
		"spec":            "session-key requests kept on NEAR, near-stocks domain; hashed by go-ethereum apitypes",
		"contractAccount": acct, "chainId": chain, "sessionKey": key, "subAccountId": hx(sub[:]),
		"requests": out,
	})
}
