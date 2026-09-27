package contract

// NEAR branch of every EndpointContract builder (Development.md §8.1). With NEAR_SETTLEMENT=1 each
// builder queues a Borsh payload for near-stocks.near::submit_transactions (nearchain/payloads.go,
// pinned against the Rust contract by vectors/payloads.json) instead of appchain ABI calldata. The
// callers keep their signatures and the batch table keeps its layout: signature1/signature2 are the
// session-key signatures the contract checks (taker/maker for matches, the liquidator's order for
// liquidations, the user's request otherwise).

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/nearchain"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// ErrNotOnNear marks flows that do not exist on NEAR (Development.md §5.3): appchain deposits,
// EVM-receiver withdrawals, Kroma cleanup, SHIFT_BALANCE (D-3), campaign rewards (D-4), and the
// REGISTER transaction (users call register_session_key themselves, §6.3).
var ErrNotOnNear = errors.New("not supported with NEAR settlement")

// nearSub accepts both subaccount id forms used by callers: "broker_0xaddr_n" and 0x-hex bytes32.
func nearSub(id string) ([32]byte, error) {
	if strings.HasPrefix(id, "0x") && len(id) == 66 {
		var out [32]byte
		b, err := hex.DecodeString(id[2:])
		if err != nil {
			return out, fmt.Errorf("invalid subaccount %q: %w", id, err)
		}
		copy(out[:], b)
		return out, nil
	}
	return cutils.SubaccountIdToBytes32(id)
}

func nearKey(addr string) [20]byte { return common.HexToAddress(addr) }

func u128(v int64) *big.Int { return big.NewInt(v) }

func nearSig(signatureHex string) ([]byte, error) {
	sig, err := hexutil.Decode(signatureHex)
	if err != nil {
		return nil, fmt.Errorf("cannot decode signature: %w", err)
	}
	if len(sig) != 65 {
		return nil, fmt.Errorf("signature must be 65 bytes, got %d", len(sig))
	}
	return sig, nil
}

func hexOf(b [32]byte) string { return "0x" + common.Bytes2Hex(b[:]) }

// sortedPrices turns a product -> price map into the contract's (pid, price) list, in pid order
// so the payload is deterministic.
func sortedPrices(maps ...map[uint32]*big.Int) []nearchain.PidValue {
	var out []nearchain.PidValue
	for _, m := range maps {
		for pid, p := range m {
			if p != nil && p.Sign() > 0 {
				out = append(out, nearchain.PidValue{ProductId: pid, Value: new(big.Int).Set(p)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ProductId < out[j].ProductId })
	return out
}

func pairedPrices(pids []uint32, prices []*big.Int) ([]nearchain.PidValue, error) {
	if len(pids) != len(prices) {
		return nil, fmt.Errorf("%d product ids but %d prices", len(pids), len(prices))
	}
	m := map[uint32]*big.Int{}
	for i, pid := range pids {
		m[pid] = prices[i]
	}
	return sortedPrices(m), nil
}

// QueuedTx is one row a NEAR builder writes to batch_tables.
type QueuedTx struct {
	Sub1, Sub2   string
	Payload      []byte
	Sig1, Sig2   []byte
	FunctionName string
	TxnCounter   uint
}

// nearQueueFn writes the row; tests swap it with SetNearQueueForTest to capture payloads.
var nearQueueFn = func(q QueuedTx) error {
	return AddTransactionToBatchDb(q.Sub1, q.Sub2, q.Payload, q.Sig1, q.Sig2, q.FunctionName, q.TxnCounter)
}

// SetNearQueueForTest replaces the batch-table writer and returns a function that restores it.
func SetNearQueueForTest(f func(QueuedTx) error) (restore func()) {
	old := nearQueueFn
	nearQueueFn = f
	return func() { nearQueueFn = old }
}

// nearRecordDigests remembers order digests for the prune job; tests swap it out.
var nearRecordDigests = func(digests []string, expirationsMs []uint64) error {
	return db.NearOrderDigestDB{}.Record(digests, expirationsMs)
}

// SetNearDigestRecorderForTest replaces the digest recorder and returns a restore function.
func SetNearDigestRecorderForTest(f func([]string, []uint64) error) (restore func()) {
	old := nearRecordDigests
	nearRecordDigests = f
	return func() { nearRecordDigests = old }
}

// recordOrders stores the near-stocks digests of the orders in a queued transaction. Pruning is a
// storage optimisation, so a failure here is logged and never fails the trade.
func recordOrders(orders ...nearchain.Order) {
	var digests []string
	var exps []uint64
	for _, o := range orders {
		d, err := contractUtils.NearOrderDigest(contractUtils.NearStocksAccount(), contractUtils.NearStocksChainId(), o.Subaccount, o.PriceX18, o.Amount, o.Expiration, o.IsReduce, common.BytesToAddress(o.SessionKey[:]).Hex(), o.ProductId)
		if err != nil {
			xlog.Errorf("NEAR order digest: %v", err)
			continue
		}
		digests = append(digests, "0x"+hex.EncodeToString(d))
		exps = append(exps, o.Expiration)
	}
	if err := nearRecordDigests(digests, exps); err != nil {
		xlog.Errorf("recording NEAR order digests for pruning: %v", err)
	}
}

func (epc *EndpointContract) nearQueue(sub1, sub2 string, payload, sig1, sig2 []byte, name string, counter uint) error {
	if sig1 == nil {
		sig1 = []byte{} // batch_tables.signature1 is NOT NULL; empty = sequencer-only transaction
	}
	return nearQueueFn(QueuedTx{Sub1: sub1, Sub2: sub2, Payload: payload, Sig1: sig1, Sig2: sig2, FunctionName: name, TxnCounter: counter})
}

// ---------------------------------------------------------------- trading (5, 1, 19, 20, 21, 0)

func (epc *EndpointContract) nearMatchOrders(r MatchOrderRequest, counter uint) error {
	taker, err := nearSub(r.TakerSubaccountId)
	if err != nil {
		return err
	}
	maker, err := nearSub(r.MakerSubaccountId)
	if err != nil {
		return err
	}
	// same sign rules as the appchain path: amounts are negative on the selling side and the
	// matched amount carries the maker's sign
	takerAmount, makerAmount, matched := new(big.Int).Set(r.TakerMaxAmountX18), new(big.Int).Set(r.MakerMaxAmountX18), new(big.Int).Set(r.MatchedAmountx18)
	if r.TakerSide == ctypes.ORDER_SIDE_SELL {
		takerAmount.Neg(takerAmount)
	} else {
		makerAmount.Neg(makerAmount)
		matched.Neg(matched)
	}
	p := nearchain.MatchOrders{
		ProductId: r.ProductId,
		Taker: nearchain.Order{Subaccount: taker, PriceX18: r.TakerPriceX18, Amount: takerAmount, Expiration: r.TakerExpiryTs,
			IsReduce: r.TakerIsReduce, SessionKey: nearKey(r.TakerSessionKey), ProductId: r.ProductId},
		Maker: nearchain.Order{Subaccount: maker, PriceX18: r.MakerPriceX18, Amount: makerAmount, Expiration: r.MakerExpiryTs,
			IsReduce: r.MakerIsReduce, SessionKey: nearKey(r.MakerSessionKey), ProductId: r.ProductId},
		MatchedAmount: matched,
	}
	if err := epc.nearQueue(hexOf(taker), hexOf(maker), p.Encode(), common.FromHex(r.TakerSignature), common.FromHex(r.MakerSignature), "MatchOrders", counter); err != nil {
		return err
	}
	recordOrders(p.Taker, p.Maker)
	return nil
}

func (epc *EndpointContract) nearLiquidate(r LiquidationRequest, counter uint) error {
	liquidator, err := nearSub(r.LiquidatorSubaccountId)
	if err != nil {
		return err
	}
	liquidatee, err := nearSub(r.LiquidateeSubaccountId)
	if err != nil {
		return err
	}
	liquidatorAmount := new(big.Int).Set(r.LiquidatorAmount)
	if r.LiquidatorSide == ctypes.ORDER_SIDE_SELL {
		liquidatorAmount.Neg(liquidatorAmount)
	}
	// the contract's amount is the matched size with the liquidatee's sign
	// (FinaliseLiquidationRequest.AmountX18, order.service.go matchAmountWithTakerSign)
	amount := new(big.Int).Abs(r.MatchedAmountx18)
	if r.LiquidateeSide == ctypes.ORDER_SIDE_SELL {
		amount.Neg(amount)
	}
	p := nearchain.Liquidate{
		ProductId: r.ProductId,
		Prices:    sortedPrices(r.PerpOraclePricesX18, r.SpotOraclePricesX18),
		Liquidator: nearchain.Order{Subaccount: liquidator, PriceX18: r.LiquidatorPriceX18, Amount: liquidatorAmount, Expiration: r.LiquidatorExpiryTs,
			IsReduce: r.LiquidatorIsReduce, SessionKey: nearKey(r.LiquidatorSessionKey), ProductId: r.ProductId},
		Liquidatee: liquidatee,
		Amount:     amount,
	}
	if err := epc.nearQueue(hexOf(liquidatee), hexOf(liquidator), p.Encode(), common.FromHex(r.LiquidatorSignature), nil, "LiquidateSubaccount", counter); err != nil {
		return err
	}
	recordOrders(p.Liquidator)
	return nil
}

func (epc *EndpointContract) nearSettleUserPnl(r contractUtils.SettleUserPnlRequest, counter uint) error {
	var subs [][32]byte
	for _, id := range r.SubaccountIds {
		b, err := cutils.SubAccountIdStrToBytes32(id)
		if err != nil {
			return err
		}
		subs = append(subs, b)
	}
	prices, err := pairedPrices(r.ProductIds, r.OraclePrices)
	if err != nil {
		return err
	}
	return epc.nearQueue("", "", nearchain.SettleUserPnl{Subaccounts: subs, SpotPrices: prices}.Encode(), nil, nil, "SettleUserPnl", counter)
}

func (epc *EndpointContract) nearSocialise(r SocialiseSubaccountRequest, counter uint) error {
	sub, err := nearSub(r.SubaccountId)
	if err != nil {
		return err
	}
	prices, err := pairedPrices(r.ProductIds, r.OraclePricesX18)
	if err != nil {
		return err
	}
	return epc.nearQueue(hexOf(sub), "", nearchain.Socialise{Subaccount: sub, SpotPrices: prices}.Encode(), nil, nil, "SocialiseSubaccount", counter)
}

func (epc *EndpointContract) nearSetNonce(subHex string, delta uint64, counter uint) error {
	sub, err := nearSub(subHex)
	if err != nil {
		return err
	}
	return epc.nearQueue(hexOf(sub), "", nearchain.SetNonce{Subaccount: sub, Delta: delta}.Encode(), nil, nil, "SetNonce", counter)
}

// FundingTick queues a funding accrual. On NEAR the rates are keyed by product id (H-2); on the
// appchain it is the old positional PerpTick.
func (epc *EndpointContract) FundingTick(productIds []uint, rates []*big.Int, fundingTimestamp int64, counter uint) error {
	if !contractUtils.NearSettlement() {
		epc.PerpTick(rates, fundingTimestamp, counter)
		return nil
	}
	if len(productIds) != len(rates) {
		return fmt.Errorf("%d markets but %d rates", len(productIds), len(rates))
	}
	var rs []nearchain.PidValue
	for i, pid := range productIds {
		rs = append(rs, nearchain.PidValue{ProductId: uint32(pid), Value: rates[i]})
	}
	p := nearchain.PerpTick{Time: uint64(fundingTimestamp), Rates: rs}
	return epc.nearQueue("", "", p.Encode(), nil, nil, "PerpTick", counter)
}

// PriceTick queues a price-only PERPTICK (NEAR only): the contract checks every trade's margin
// against its stored prices, which go stale after price_max_age_sec (§5.7).
func (epc *EndpointContract) PriceTick(prices map[uint32]*big.Int, timestamp int64, counter uint) error {
	if !contractUtils.NearSettlement() {
		return ErrNotOnNear
	}
	p := nearchain.PerpTick{Time: uint64(timestamp), Prices: sortedPrices(prices)}
	return epc.nearQueue("", "", p.Encode(), nil, nil, "PriceTick", counter)
}

// ---------------------------------------------------------------- withdrawals (3, 13)

// NearWithdraw queues WITHDRAW_COLLATERAL (product 4 and other collateral) or WITHDRAW_LOGX
// (product 0): the user's session key signed contractUtils.NearWithdraw with the exact receiver.
func (epc *EndpointContract) NearWithdraw(w contractUtils.NearWithdraw, signatureHex string, counter uint) error {
	sub, err := nearSub(w.SubaccountId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	amount, ok := new(big.Int).SetString(w.Amount, 10)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid amount %q", w.Amount)
	}
	p := nearchain.NearWithdraw{Subaccount: sub, SessionKey: nearKey(w.SessionKey), ProductId: w.ProductId, Amount: amount, Nonce: u128(w.Nonce), Receiver: w.Receiver}
	payload, name := p.Encode(), "WithdrawCollateral"
	if w.ProductId == 0 {
		payload, name = p.EncodeLogX(), "WithdrawLogX"
	}
	return epc.nearQueue(hexOf(sub), "", payload, sig, nil, name, counter)
}

// ---------------------------------------------------------------- products (24-27)

func (epc *EndpointContract) nearPlaceOption(b contractUtils.UserOptionBet, amount, entryPrice *big.Int, orderId uint, payout, fee uint32, signatureHex string, counter uint) error {
	sub, err := nearSub(b.SubAccountId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	p := nearchain.PlaceOption{
		Bet:           nearchain.OptionBet{Subaccount: sub, ProductId: b.ProductId, Amount: amount, Interval: b.Interval, Nonce: u128(b.Nonce), SessionKey: nearKey(b.SessionKey)},
		OrderId:       uint64(orderId),
		EntryPriceX18: entryPrice,
		PayoutPct:     payout,
		FeePct:        fee,
	}
	return epc.nearQueue(hexOf(sub), "", p.Encode(), sig, nil, "PlaceOptionBet", counter)
}

func (epc *EndpointContract) nearCloseOption(orderId uint, exitPrice *big.Int, counter uint, subHex string) error {
	p := nearchain.CloseOption{OrderId: uint64(orderId), ExitPriceX18: exitPrice}
	return epc.nearQueue(subHex, "", p.Encode(), nil, nil, "CloseOptionBet", counter)
}

func (epc *EndpointContract) nearPoolTrade(o contractUtils.PlacePreMarketOrderRequest, quoteDelta, fees *big.Int, signatureHex string, counter uint, synthetic bool) error {
	sub, err := nearSub(o.SubAccountId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	amount, ok := new(big.Int).SetString(o.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount %q", o.Amount)
	}
	p := nearchain.PoolTrade{
		Order:      nearchain.PoolOrder{Subaccount: sub, ProductId: o.ProductId, Amount: amount, IsBuy: o.IsBuy, Nonce: u128(o.Nonce), SessionKey: nearKey(o.SessionKey)},
		QuoteDelta: quoteDelta,
		Fees:       fees,
	}
	if synthetic {
		return epc.nearQueue(hexOf(sub), "", p.EncodeSyntheticSpot(), sig, nil, "SyntheticSpotOrderRequest", counter)
	}
	return epc.nearQueue(hexOf(sub), "", p.EncodePreMarket(), sig, nil, "PreMarketOrderRequest", counter)
}

// ---------------------------------------------------------------- LogX (14, 15, 16, 18, 22)

func (epc *EndpointContract) nearStake(stake bool, subId, stakerContract, sessionKey string, productId uint32, nonce int64, amount *big.Int, signatureHex string, counter uint) error {
	sub, err := nearSub(subId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	p := nearchain.StakeRequest{Subaccount: sub, ProductId: productId, Amount: amount, StakerContract: nearKey(stakerContract), SessionKey: nearKey(sessionKey), Nonce: u128(nonce)}
	if stake {
		return epc.nearQueue(hexOf(sub), "", p.EncodeStake(), sig, nil, "StakeLogx", counter)
	}
	return epc.nearQueue(hexOf(sub), "", p.EncodeUnstake(), sig, nil, "UnstakeLogx", counter)
}

func (epc *EndpointContract) nearClaimRewards(c contractUtils.ClaimRewards, claimablex18 *big.Int, signatureHex string, counter uint) error {
	sub, err := nearSub(c.SubAccountId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	p := nearchain.ClaimRewards{Subaccount: sub, SessionKey: nearKey(c.SessionKey), StakerContract: nearKey(c.StakerContract), ProductId: c.ProductId, Nonce: u128(c.Nonce), AmountX18: claimablex18}
	// the rewards pool pays the claim: listing it makes the reconciler check it and a refusal pause it
	return epc.nearQueue(hexOf(sub), strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID), p.Encode(), sig, nil, "ClaimRewards", counter)
}

func (epc *EndpointContract) nearClaimLogX(d contractUtils.DepositRequest, tokenAmount *big.Int, signatureHex string, counter uint) error {
	sub, err := nearSub(d.SubAccountId)
	if err != nil {
		return err
	}
	sig, err := nearSig(signatureHex)
	if err != nil {
		return err
	}
	p := nearchain.ClaimLogX{Subaccount: sub, TokenAmount: tokenAmount, SessionKey: nearKey(d.SessionKey), Nonce: u128(d.Nonce)}
	return epc.nearQueue(hexOf(sub), strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID), p.Encode(), sig, nil, "ClaimLogX", counter)
}

func (epc *EndpointContract) nearRewardRateTick(rate *big.Int, counter uint) error {
	return epc.nearQueue("", "", nearchain.RewardRateTick{CumulativeRateX18: rate}.Encode(), nil, nil, "RewardRateTick", counter)
}
