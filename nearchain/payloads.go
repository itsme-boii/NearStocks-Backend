package nearchain

// Borsh payloads for near-stocks.near::submit_transactions (Development.md §5.4). Every type here
// mirrors a struct in near-stocks-contracts/core/src/{tx,eip712}.rs field for field; the encodings
// are pinned by vectors/payloads.json, which Rust decodes and re-encodes byte for byte.
//
// A batch transaction is `u8 type ‖ borsh(payload)`; the call arguments are
// borsh((u64 idx, Vec<Vec<u8>> txs, Vec<Vec<u8>> sigs, Vec<Vec<u8>> sigs2)).

import (
	"encoding/binary"
	"fmt"
	"math/big"
)

// Transaction type bytes (contract/types/transaction.type.go and core/src/tx.rs).
const (
	TxPerpTick            byte = 0
	TxLiquidateSubaccount byte = 1
	TxWithdrawCollateral  byte = 3
	TxMatchOrders         byte = 5
	TxWithdrawLogX        byte = 13
	TxClaimRewards        byte = 14
	TxStakeLogX           byte = 15
	TxUnstakeLogX         byte = 16
	TxClaimLogX           byte = 18
	TxSettleUserPnl       byte = 19
	TxSocialiseSubaccount byte = 20
	TxSetNonce            byte = 21
	TxRewardRateTick      byte = 22
	TxPlaceOptionsBet     byte = 24
	TxCloseOptionsBet     byte = 25
	TxPreMarketOrder      byte = 26
	TxSyntheticSpotOrder  byte = 27
)

// W is a Borsh writer. Integer helpers panic on out-of-range values: they are programming errors,
// and a silently wrapped amount would be worse than a crash.
type W struct{ b []byte }

func (w *W) Bytes() []byte      { return w.b }
func (w *W) U8(v uint8) *W      { w.b = append(w.b, v); return w }
func (w *W) U32(v uint32) *W    { w.b = binary.LittleEndian.AppendUint32(w.b, v); return w }
func (w *W) U64(v uint64) *W    { w.b = binary.LittleEndian.AppendUint64(w.b, v); return w }
func (w *W) Fixed(v []byte) *W  { w.b = append(w.b, v...); return w }
func (w *W) String(s string) *W { w.U32(uint32(len(s))); w.b = append(w.b, s...); return w }
func (w *W) VecU8(v []byte) *W  { w.U32(uint32(len(v))); w.b = append(w.b, v...); return w }
func (w *W) Bool(v bool) *W {
	if v {
		return w.U8(1)
	}
	return w.U8(0)
}

// U128 writes an unsigned 128-bit little-endian integer.
func (w *W) U128(v *big.Int) *W { w.b = putU128(w.b, v); return w }

// I128 writes a signed 128-bit two's-complement little-endian integer.
func (w *W) I128(v *big.Int) *W {
	if v == nil {
		v = new(big.Int)
	}
	if v.BitLen() > 127 && !(v.Sign() < 0 && new(big.Int).Neg(v).Cmp(new(big.Int).Lsh(big.NewInt(1), 127)) == 0) {
		panic(fmt.Sprintf("i128 out of range: %s", v))
	}
	u := new(big.Int).Set(v)
	if u.Sign() < 0 {
		u.Add(u, new(big.Int).Lsh(big.NewInt(1), 128))
	}
	return w.U128(u)
}

// PidValue is a (product id, x18 value) pair: prices and funding rates.
type PidValue struct {
	ProductId uint32
	Value     *big.Int
}

func (w *W) pidValues(v []PidValue) *W {
	w.U32(uint32(len(v)))
	for _, p := range v {
		w.U32(p.ProductId).I128(p.Value)
	}
	return w
}

func envelope(ty byte, body *W) []byte { return append([]byte{ty}, body.Bytes()...) }

// ---------------------------------------------------------------------------- core (Phase 3)

type Order struct {
	Subaccount [32]byte
	PriceX18   *big.Int
	Amount     *big.Int
	Expiration uint64
	IsReduce   bool
	SessionKey [20]byte
	ProductId  uint32
}

func (o Order) write(w *W) *W {
	return w.Fixed(o.Subaccount[:]).I128(o.PriceX18).I128(o.Amount).U64(o.Expiration).Bool(o.IsReduce).Fixed(o.SessionKey[:]).U32(o.ProductId)
}

type PerpTick struct {
	Time   uint64
	Rates  []PidValue
	Prices []PidValue
}

func (p PerpTick) Encode() []byte {
	return envelope(TxPerpTick, new(W).U64(p.Time).pidValues(p.Rates).pidValues(p.Prices))
}

type MatchOrders struct {
	ProductId     uint32
	Taker, Maker  Order
	MatchedAmount *big.Int // maker's sign
}

func (m MatchOrders) Encode() []byte {
	w := new(W).U32(m.ProductId)
	m.Taker.write(w)
	m.Maker.write(w)
	return envelope(TxMatchOrders, w.I128(m.MatchedAmount))
}

type Liquidate struct {
	ProductId  uint32
	Prices     []PidValue
	Liquidator Order
	Liquidatee [32]byte
	Amount     *big.Int // liquidatee's sign
}

func (l Liquidate) Encode() []byte {
	w := new(W).U32(l.ProductId).pidValues(l.Prices)
	l.Liquidator.write(w)
	return envelope(TxLiquidateSubaccount, w.Fixed(l.Liquidatee[:]).I128(l.Amount))
}

// NearWithdraw is WITHDRAW_COLLATERAL (3), and WITHDRAW_LOGX (13) with ProductId 0.
type NearWithdraw struct {
	Subaccount [32]byte
	SessionKey [20]byte
	ProductId  uint32
	Amount     *big.Int
	Nonce      *big.Int
	Receiver   string
}

func (n NearWithdraw) body() *W {
	return new(W).Fixed(n.Subaccount[:]).Fixed(n.SessionKey[:]).U32(n.ProductId).U128(n.Amount).U128(n.Nonce).String(n.Receiver)
}
func (n NearWithdraw) Encode() []byte     { return envelope(TxWithdrawCollateral, n.body()) }
func (n NearWithdraw) EncodeLogX() []byte { return envelope(TxWithdrawLogX, n.body()) }

type SettleUserPnl struct {
	Subaccounts [][32]byte
	SpotPrices  []PidValue
}

func (s SettleUserPnl) Encode() []byte {
	w := new(W).U32(uint32(len(s.Subaccounts)))
	for _, sub := range s.Subaccounts {
		w.Fixed(sub[:])
	}
	return envelope(TxSettleUserPnl, w.pidValues(s.SpotPrices))
}

type Socialise struct {
	Subaccount [32]byte
	SpotPrices []PidValue
}

func (s Socialise) Encode() []byte {
	return envelope(TxSocialiseSubaccount, new(W).Fixed(s.Subaccount[:]).pidValues(s.SpotPrices))
}

type SetNonce struct {
	Subaccount [32]byte
	Delta      uint64
}

func (s SetNonce) Encode() []byte {
	return envelope(TxSetNonce, new(W).Fixed(s.Subaccount[:]).U64(s.Delta))
}

// ---------------------------------------------------------------------------- products (Phase 4)

type OptionBet struct {
	Subaccount [32]byte
	ProductId  uint32
	Amount     *big.Int // sign = direction
	Interval   uint32
	Nonce      *big.Int
	SessionKey [20]byte
}

type PlaceOption struct {
	Bet           OptionBet
	OrderId       uint64
	EntryPriceX18 *big.Int
	PayoutPct     uint32
	FeePct        uint32
}

func (p PlaceOption) Encode() []byte {
	b := p.Bet
	w := new(W).Fixed(b.Subaccount[:]).U32(b.ProductId).I128(b.Amount).U32(b.Interval).U128(b.Nonce).Fixed(b.SessionKey[:])
	return envelope(TxPlaceOptionsBet, w.U64(p.OrderId).I128(p.EntryPriceX18).U32(p.PayoutPct).U32(p.FeePct))
}

type CloseOption struct {
	OrderId      uint64
	ExitPriceX18 *big.Int
}

func (c CloseOption) Encode() []byte {
	return envelope(TxCloseOptionsBet, new(W).U64(c.OrderId).I128(c.ExitPriceX18))
}

// PoolOrder is the user-signed pre-market / synthetic-spot order.
type PoolOrder struct {
	Subaccount [32]byte
	ProductId  uint32
	Amount     *big.Int
	IsBuy      bool
	Nonce      *big.Int
	SessionKey [20]byte
}

type PoolTrade struct {
	Order      PoolOrder
	QuoteDelta *big.Int
	Fees       *big.Int
}

func (p PoolTrade) body() *W {
	o := p.Order
	return new(W).Fixed(o.Subaccount[:]).U32(o.ProductId).I128(o.Amount).Bool(o.IsBuy).U128(o.Nonce).Fixed(o.SessionKey[:]).I128(p.QuoteDelta).I128(p.Fees)
}
func (p PoolTrade) EncodePreMarket() []byte     { return envelope(TxPreMarketOrder, p.body()) }
func (p PoolTrade) EncodeSyntheticSpot() []byte { return envelope(TxSyntheticSpotOrder, p.body()) }

type StakeRequest struct {
	Subaccount     [32]byte
	ProductId      uint32
	Amount         *big.Int
	StakerContract [20]byte
	SessionKey     [20]byte
	Nonce          *big.Int
}

func (s StakeRequest) body() *W {
	return new(W).Fixed(s.Subaccount[:]).U32(s.ProductId).I128(s.Amount).Fixed(s.StakerContract[:]).Fixed(s.SessionKey[:]).U128(s.Nonce)
}
func (s StakeRequest) EncodeStake() []byte   { return envelope(TxStakeLogX, s.body()) }
func (s StakeRequest) EncodeUnstake() []byte { return envelope(TxUnstakeLogX, s.body()) }

type ClaimRewards struct {
	Subaccount     [32]byte
	SessionKey     [20]byte
	StakerContract [20]byte
	ProductId      uint32
	Nonce          *big.Int
	AmountX18      *big.Int // backend-computed claimable (D-6)
}

func (c ClaimRewards) Encode() []byte {
	w := new(W).Fixed(c.Subaccount[:]).Fixed(c.SessionKey[:]).Fixed(c.StakerContract[:]).U32(c.ProductId).U128(c.Nonce)
	return envelope(TxClaimRewards, w.I128(c.AmountX18))
}

type ClaimLogX struct {
	Subaccount  [32]byte
	TokenAmount *big.Int
	SessionKey  [20]byte
	Nonce       *big.Int
}

func (c ClaimLogX) Encode() []byte {
	return envelope(TxClaimLogX, new(W).Fixed(c.Subaccount[:]).I128(c.TokenAmount).Fixed(c.SessionKey[:]).U128(c.Nonce))
}

type RewardRateTick struct{ CumulativeRateX18 *big.Int }

func (r RewardRateTick) Encode() []byte {
	return envelope(TxRewardRateTick, new(W).I128(r.CumulativeRateX18))
}

// ---------------------------------------------------------------------------- call arguments

// SubmitArgs encodes the Borsh arguments of submit_transactions.
func SubmitArgs(idx uint64, txs, sigs, sigs2 [][]byte) []byte {
	if len(txs) != len(sigs) || len(txs) != len(sigs2) {
		panic("txs, sigs and sigs2 must have the same length")
	}
	w := new(W).U64(idx)
	for _, list := range [][][]byte{txs, sigs, sigs2} {
		w.U32(uint32(len(list)))
		for _, b := range list {
			w.VecU8(b)
		}
	}
	return w.Bytes()
}
