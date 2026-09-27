package tests

// Parity scenarios for the NEAR settlement contract (behavior-spec §3, gate G3 core).
// Random trading sessions are run through the production Go ledger code, and every step and the
// final state are written to vectors/parity.json. near-stocks-contracts/core/tests/parity.rs
// replays the same steps through submit_transactions and must end with identical balances.
//
// Real Go code on each path:
//   match      BalanceService.UpdateLocalBalanceForOrderMatch (position, fee, 0.96*IM health rule)
//   OI         PerpUtils.UpdateLongShortPositions
//   liquidate  LiquidationService.FinaliseLiquidation on miniredis (D-2 insurance credit, D-7 fee
//              accrual and fold, R-5 liquidator health and the AMM position cap, OI)
//   settle     services.SettleLiqPnlUsingSpots, as SettlePnLForSubaccounts calls it
//   socialise  LiquidationService.SettleUsingInsurance on miniredis
//   withdraw   subaccount.GetWithdrawableBalance
// Harness-only (small, pinned elsewhere): funding accrual cum += rate*dt (funding.go
// storeFundingRate) and engine match deltas (vectors/math.json). Match fees returned by
// UpdateLocalBalanceForOrderMatchWithFee are credited to TRADING_FEES_SUBACCOUNT_ID, which is what
// AtomicUpdateBalanceForOrderMatch's accrual queue does (tested in fee_account_test.go).
//
// Regenerate: PARITY_OUT=../../vectors/parity.json go test ./tests/ -run TestGenerateParity

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/perputils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/balance-server/services"
	"github/eugenix-io/logx-inf-backend/services/balance-server/types"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/redis/go-redis/v9"
)

type pState struct {
	Sub   string      `json:"sub"`
	Spots [][2]string `json:"spots"` // [pid, balance]
	Perps [][4]string `json:"perps"` // [pid, amount, vQuote, lastCum]
}

type pStep struct {
	Kind string `json:"kind"`
	// tick
	Time   uint64      `json:"time,omitempty"`
	Rates  [][2]string `json:"rates,omitempty"`
	Prices [][2]string `json:"prices,omitempty"`
	// match / liquidate
	Pid        uint32 `json:"pid,omitempty"`
	Taker      string `json:"taker,omitempty"`
	Maker      string `json:"maker,omitempty"`
	Matched    string `json:"matched,omitempty"` // maker's sign
	Price      string `json:"price,omitempty"`   // maker / liquidator price
	TakerLimit string `json:"takerLimit,omitempty"`
	MakerRed   bool   `json:"makerReduce,omitempty"`
	TakerRed   bool   `json:"takerReduce,omitempty"`
	Liquidator string `json:"liquidator,omitempty"`
	Liquidatee string `json:"liquidatee,omitempty"`
	Amount     string `json:"amount,omitempty"` // liquidatee's sign; withdraw amount
	// settle / socialise / withdraw
	Subs []string `json:"subs,omitempty"`
	Sub  string   `json:"sub,omitempty"`
	// expected outcome: "" = accepted, otherwise a substring of the contract's panic
	Reject string `json:"reject,omitempty"`
}

type pScenario struct {
	Seed     int64       `json:"seed"`
	Perps    [][5]string `json:"perps"` // [pid, imf, mmf, liqFrac, ammMaxPosition]
	Spots    []uint32    `json:"spots"`
	Users    []string    `json:"users"` // NEAR account ids (broker 2, subaccount 0)
	Amm      string      `json:"amm"`
	Ins      string      `json:"insurance"`
	FeeSub   string      `json:"feeSub"`
	Initial  []pState    `json:"initial"`
	StartSec uint64      `json:"startSec"`
	Steps    []pStep     `json:"steps"`
	Final    []pState    `json:"final"`
	OI       [][3]string `json:"openInterest"` // [pid, long, short]
	Cum      [][2]string `json:"cumFunding"`
}

var e18 = cutils.GetBigx18()

func x18(f float64) *big.Int {
	return cutils.FloatStrToX18(strconv.FormatFloat(f, 'f', 9, 64))
}

func sub32(broker uint64, addr20 string, n uint64) string {
	b := make([]byte, 32)
	for i := 0; i < 6; i++ {
		b[5-i] = byte(broker >> (8 * i))
		b[31-i] = byte(n >> (8 * i))
	}
	a, _ := hex.DecodeString(addr20[2:])
	copy(b[6:26], a)
	return "0x" + hex.EncodeToString(b)
}

// stubAppState serves the scenario's own prices, markets and funding to FinaliseLiquidation.
type stubAppState struct{ l *ledger }

func (s stubAppState) GetAppState(_ ...appstate.AppStateCacheConfig) (map[string]ctypes.OraclePrice, map[uint]subaccountTypes.PerpetualMarket, map[string]*big.Int, error) {
	return s.l.prices, s.l.markets, s.l.cum, nil
}
func (s stubAppState) GetAllPerpMarkets(_ ...time.Duration) map[uint]subaccountTypes.PerpetualMarket {
	return s.l.markets
}
func (s stubAppState) GetAllFundingRates(_ ...time.Duration) map[string]*big.Int { return s.l.cum }
func (s stubAppState) GetAllOraclePrices(_ ...time.Duration) (map[string]ctypes.OraclePrice, error) {
	return s.l.prices, nil
}

type ledger struct {
	t        *testing.T
	r        *rand.Rand
	bs       *services.BalanceService
	ls       *services.LiquidationServiceImpl
	pu       *perputils.PerpUtils
	accounts map[string]*subaccountTypes.SubaccountBalances
	markets  map[uint]subaccountTypes.PerpetualMarket
	prices   map[string]ctypes.OraclePrice // by symbol
	cum      map[string]*big.Int           // by symbol
	lastTime map[uint32]uint64
	oiLong   map[uint32]*big.Int
	oiShort  map[uint32]*big.Int
	perps    []uint32
	now      uint64
	stress   bool // bigger price moves and thinner collateral: more liquidations and bad debt
}

func (l *ledger) sym(pid uint32) string {
	s, ok := marketutils.GetBaseSymbolForProduct(pid)
	if !ok {
		l.t.Fatalf("no symbol for %d", pid)
	}
	return s
}

func (l *ledger) acct(id string) *subaccountTypes.SubaccountBalances {
	a, ok := l.accounts[id]
	if !ok {
		a = &subaccountTypes.SubaccountBalances{SubaccountId: id, SpotBalances: map[uint32]subaccountTypes.SpotBalance{}, PerpBalances: map[uint32]subaccountTypes.PerpBalance{}}
		l.accounts[id] = a
	}
	return a
}

func clone(a *subaccountTypes.SubaccountBalances) *subaccountTypes.SubaccountBalances {
	c := &subaccountTypes.SubaccountBalances{SubaccountId: a.SubaccountId, SpotBalances: map[uint32]subaccountTypes.SpotBalance{}, PerpBalances: map[uint32]subaccountTypes.PerpBalance{}}
	for k, v := range a.SpotBalances {
		c.SpotBalances[k] = subaccountTypes.SpotBalance{ProductId: v.ProductId, Balancex18: new(big.Int).Set(v.Balancex18), Lockedx18: new(big.Int).Set(v.Lockedx18)}
	}
	for k, v := range a.PerpBalances {
		c.PerpBalances[k] = subaccountTypes.PerpBalance{ProductId: v.ProductId, Amountx18: new(big.Int).Set(v.Amountx18), VQuoteBalancex18: new(big.Int).Set(v.VQuoteBalancex18), LastCumFundingRatex18: new(big.Int).Set(v.LastCumFundingRatex18)}
	}
	return c
}

func quoteOf(a *subaccountTypes.SubaccountBalances) *big.Int {
	return new(big.Int).Set(a.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18)
}

func (l *ledger) spotPrices() map[uint32]*big.Int {
	m := map[uint32]*big.Int{}
	for _, pid := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		m[pid] = new(big.Int).Set(l.prices[l.sym(pid)].Pricex18)
	}
	return m
}

func (l *ledger) perpPrice(pid uint32) *big.Int {
	return new(big.Int).Set(l.prices[l.sym(pid)].Pricex18)
}

// safety and health exactly as balance.service.go (locked = 0 on chain and here)
func (l *ledger) safety(a *subaccountTypes.SubaccountBalances) *big.Int {
	eq := subaccount.GetTotalEquityx36(*a, l.prices, l.cum)
	_, im := subaccount.GetRequiredMarginValues(*a, l.markets, l.prices)
	return cutils.Divx18(new(big.Int).Sub(eq, cutils.DivxCust(new(big.Int).Mul(im, big.NewInt(96)), 2)))
}

func (l *ledger) belowMM(a *subaccountTypes.SubaccountBalances) bool {
	eq := subaccount.GetTotalEquityx36(*a, l.prices, l.cum)
	mm, _ := subaccount.GetRequiredMarginValues(*a, l.markets, l.prices)
	return mm.Sign() != 0 && eq.Cmp(mm) < 0
}

func (l *ledger) tick() pStep {
	dt := uint64(30 + l.r.Intn(240))
	l.now += dt
	st := pStep{Kind: "tick", Time: l.now}
	for _, pid := range l.perps {
		rate := int64(l.r.Intn(2_000_000_000_001)) - 1_000_000_000_000
		sym := l.sym(pid)
		if l.lastTime[pid] != 0 {
			delta := int64(l.now - l.lastTime[pid])
			l.cum[sym] = new(big.Int).Add(l.cum[sym], new(big.Int).Mul(big.NewInt(rate), big.NewInt(delta)))
		}
		l.lastTime[pid] = l.now
		st.Rates = append(st.Rates, [2]string{fmt.Sprint(pid), fmt.Sprint(rate)})
		// random walk within +-4% (+-9% in stress scenarios), non-round
		p := l.perpPrice(pid)
		band := 4000
		if l.stress {
			band = 9000
		}
		move := new(big.Int).Mul(p, big.NewInt(int64(l.r.Intn(2*band+1)-band)))
		p = new(big.Int).Add(p, new(big.Int).Div(move, big.NewInt(100_000)))
		p.Add(p, big.NewInt(int64(l.r.Intn(1_000_000_007))))
		l.prices[sym] = ctypes.OraclePrice{Pricex18: p, Symbol: sym}
		st.Prices = append(st.Prices, [2]string{fmt.Sprint(pid), p.String()})
	}
	// USDC wobbles by up to 0.05%
	usdc := new(big.Int).Add(e18, big.NewInt(int64(l.r.Intn(1_000_000_000_000_001))-500_000_000_000_000))
	l.prices["USDC"] = ctypes.OraclePrice{Pricex18: usdc, Symbol: "USDC"}
	for _, pid := range contractUtils.ALL_SPOTS_ON_CONTRACT {
		st.Prices = append(st.Prices, [2]string{fmt.Sprint(pid), usdc.String()})
	}
	return st
}

func (l *ledger) oi(pid uint32, old, delta *big.Int) {
	l.oiLong[pid], l.oiShort[pid] = l.pu.UpdateLongShortPositions(l.oiLong[pid], l.oiShort[pid], old, delta)
}

// applySide runs the real balance-server update on a copy and reports health and the fee.
func (l *ledger) applySide(id string, pid uint32, dA, dQ *big.Int, isTaker bool) (*subaccountTypes.SubaccountBalances, bool, *big.Int) {
	after, healthy, _, _, fee := l.bs.UpdateLocalBalanceForOrderMatchWithFee(*clone(l.acct(id)), types.BalancePerpPayload{Amountx18: dA, VQuoteBalancex18: dQ}, pid, isTaker, l.prices, l.markets, l.cum)
	return after, healthy, fee
}

// rejectReason names the contract panic for an unhealthy side: the AMM only fails on its cap.
func rejectReason(id string) string {
	if id == contractUtils.AMM_SUBACCOUNT_ID {
		return "AMM position cap"
	}
	return "unhealthy trade"
}

func (l *ledger) match(parties []string) pStep {
	pid := l.perps[l.r.Intn(len(l.perps))]
	i := l.r.Intn(len(parties))
	j := (i + 1 + l.r.Intn(len(parties)-1)) % len(parties)
	taker, maker := parties[i], parties[j]
	takerBuys := l.r.Intn(2) == 0
	oracle := l.perpPrice(pid)
	price := new(big.Int).Add(oracle, new(big.Int).Div(new(big.Int).Mul(oracle, big.NewInt(int64(l.r.Intn(201))-100)), big.NewInt(10_000)))
	price.Add(price, big.NewInt(int64(l.r.Intn(999_983))))
	// notional between ~50 and ~6000 USD, 1e-9 granularity plus odd wei
	notional := 50 + l.r.Float64()*5950
	amt := new(big.Int).Div(new(big.Int).Mul(x18(notional), e18), oracle)
	amt.Add(amt, big.NewInt(int64(l.r.Intn(1_000_003))))
	if amt.Sign() == 0 {
		amt.SetInt64(1)
	}
	// engine: matchedAmount carries the maker's sign; deltas use the other side (placeOrder.engine.go:381-382)
	takerA := new(big.Int).Set(amt)
	if !takerBuys {
		takerA.Neg(takerA)
	}
	makerA := new(big.Int).Neg(takerA)
	makerQ := cutils.Divx18(new(big.Int).Mul(takerA, price))
	takerQ := cutils.Divx18(new(big.Int).Mul(makerA, price))

	st := pStep{Kind: "match", Pid: pid, Taker: taker, Maker: maker, Matched: makerA.String(), Price: price.String()}
	if l.r.Intn(3) == 0 {
		st.TakerLimit = "0" // market order
	} else if takerBuys {
		st.TakerLimit = new(big.Int).Add(price, big.NewInt(int64(l.r.Intn(1000)))).String()
	} else {
		st.TakerLimit = new(big.Int).Sub(price, big.NewInt(int64(l.r.Intn(1000)))).String()
	}
	reduces := func(id string, d *big.Int) bool {
		cur := l.acct(id).MustGetPerpBalance(pid).Amountx18
		return cur.Sign() != 0 && cur.Sign() != d.Sign() && new(big.Int).Abs(d).Cmp(new(big.Int).Abs(cur)) <= 0
	}
	st.MakerRed = reduces(maker, makerA) && l.r.Intn(2) == 0
	st.TakerRed = reduces(taker, takerA) && l.r.Intn(2) == 0

	oldM := new(big.Int).Set(l.acct(maker).MustGetPerpBalance(pid).Amountx18)
	oldT := new(big.Int).Set(l.acct(taker).MustGetPerpBalance(pid).Amountx18)
	newM, okM, feeM := l.applySide(maker, pid, makerA, makerQ, false)
	newT, okT, feeT := l.applySide(taker, pid, takerA, takerQ, true)
	// the contract applies the maker first, so the maker's failure is the one it reports
	if !okM {
		st.Reject = rejectReason(maker)
		return st
	}
	if !okT {
		st.Reject = rejectReason(taker)
		return st
	}
	l.accounts[maker], l.accounts[taker] = newM, newT
	l.acct(contractUtils.TRADING_FEES_SUBACCOUNT_ID).UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Add(feeM, feeT))
	if maker != contractUtils.AMM_SUBACCOUNT_ID {
		l.oi(pid, oldM, makerA)
	}
	if taker != contractUtils.AMM_SUBACCOUNT_ID {
		l.oi(pid, oldT, takerA)
	}
	return st
}

func (l *ledger) seedRedis(ids ...string) {
	ctx := context.Background()
	rc := l.ls.GetRedisClient()
	_, _ = rc.FlushAll(ctx).Result()
	_, err := rc.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, id := range ids {
			if err := subaccount.ReplaceBalanceInRedis(pipe, l.acct(id)); err != nil {
				return err
			}
		}
		for _, pid := range l.perps {
			ps := strconv.FormatUint(uint64(pid), 10)
			pipe.Set(ctx, xredis.GetCumulativeFundingRateKey(l.sym(pid)), xredis.CumulativeFundingRateData{CumulativeFundingRate: l.cum[l.sym(pid)].String(), CumulativeFundingTimestamp: "0"}, 0)
			pipe.Set(ctx, xredis.GetTotalLongPositionKey(ps), l.oiLong[pid].String(), 0)
			pipe.Set(ctx, xredis.GetTotalShortPositionKey(ps), l.oiShort[pid].String(), 0)
		}
		return nil
	})
	if err != nil {
		l.t.Fatal(err)
	}
}

func (l *ledger) readRedis(ids ...string) {
	ctx := context.Background()
	for _, b := range l.ls.MustGetSubaccountBalancesFromIds(ids) {
		bb := b
		l.accounts[b.SubaccountId] = &bb
	}
	for _, pid := range l.perps {
		ps := strconv.FormatUint(uint64(pid), 10)
		lv, _ := l.ls.GetRedisClient().Get(ctx, xredis.GetTotalLongPositionKey(ps)).Result()
		sv, _ := l.ls.GetRedisClient().Get(ctx, xredis.GetTotalShortPositionKey(ps)).Result()
		l.oiLong[pid], _ = new(big.Int).SetString(lv, 10)
		l.oiShort[pid], _ = new(big.Int).SetString(sv, 10)
	}
}

func (l *ledger) liquidate(users []string) (pStep, bool) {
	for _, victim := range users {
		a := l.acct(victim)
		if !l.belowMM(a) {
			continue
		}
		for _, pid := range l.perps {
			pos := a.MustGetPerpBalance(pid).Amountx18
			if pos.Sign() == 0 {
				continue
			}
			amount := new(big.Int).Neg(pos) // liquidatee's sign: closes the position
			if l.r.Intn(3) == 0 {
				amount = new(big.Int).Quo(amount, big.NewInt(2))
				if amount.Sign() == 0 {
					continue
				}
			}
			oracle := l.perpPrice(pid)
			price := new(big.Int).Add(oracle, new(big.Int).Div(new(big.Int).Mul(oracle, big.NewInt(int64(l.r.Intn(101))-50)), big.NewInt(10_000)))
			// Liquidator: insurance, the AMM, or a random other user. FinaliseLiquidation itself
			// now refuses an unhealthy liquidator or an AMM over its cap (R-5), as the contract does.
			liquidator := contractUtils.INSURANCE_SUBACCOUNT_ID
			switch k := l.r.Intn(3); {
			case k == 1:
				liquidator = contractUtils.AMM_SUBACCOUNT_ID
			case k == 2:
				if u := users[l.r.Intn(len(users))]; u != victim {
					liquidator = u
				}
			}
			perpPrices := map[uint32]*big.Int{}
			for _, p := range l.perps {
				perpPrices[p] = l.perpPrice(p)
			}
			req := &types.FinaliseLiquidationRequest{
				TxnCounter: new(uint), LiquidatorSubaccountId: liquidator, LiquidateeSubaccountId: victim, ProductId: pid,
				AmountX18: amount, MatchPriceX18: price, SpotOraclePricesX18: l.spotPrices(), PerpOraclePricesX18: perpPrices,
			}
			ids := []string{victim, liquidator, contractUtils.TRADING_FEES_SUBACCOUNT_ID}
			if liquidator != contractUtils.INSURANCE_SUBACCOUNT_ID {
				ids = append(ids, contractUtils.INSURANCE_SUBACCOUNT_ID)
			}
			l.seedRedis(ids...)
			st := pStep{Kind: "liquidate", Pid: pid, Liquidator: liquidator, Liquidatee: victim, Amount: amount.String(), Price: price.String()}
			if _, err := l.ls.FinaliseLiquidation(req); err != nil {
				switch {
				case errors.Is(err, services.ErrLiquidatorUnhealthy):
					st.Reject = "liquidator would be unhealthy"
				case errors.Is(err, services.ErrAmmPositionCap):
					st.Reject = "AMM position cap"
				default:
					l.t.Fatal(err)
				}
				return st, true
			}
			l.readRedis(ids...)
			return st, true
		}
	}
	return pStep{}, false
}

func (l *ledger) settle(users []string) (pStep, bool) {
	var subs []string
	for _, u := range users {
		if quoteOf(l.acct(u)).Sign() < 0 {
			subs = append(subs, u)
		}
	}
	if len(subs) == 0 {
		return pStep{}, false
	}
	subs = append(subs, users[l.r.Intn(len(users))]) // a non-negative one is skipped
	prices := l.spotPrices()
	for _, u := range subs {
		a := l.acct(u)
		q := quoteOf(a)
		if q.Sign() >= 0 {
			continue
		}
		a.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(q))
		services.SettleLiqPnlUsingSpots(prices, cutils.Mulx18(q), a)
	}
	return pStep{Kind: "settle", Subs: subs}, true
}

func (l *ledger) socialise(users []string) (pStep, bool) {
	for _, u := range users {
		a := l.acct(u)
		open := false
		for _, pb := range a.PerpBalances {
			open = open || pb.Amountx18.Sign() != 0
		}
		if open || quoteOf(a).Sign() >= 0 {
			continue
		}
		l.seedRedis(u, contractUtils.INSURANCE_SUBACCOUNT_ID)
		err := l.ls.SettleUsingInsurance(&types.SettleWithInsuranceRequest{SubaccountId: u, SpotOraclePricesX18: l.spotPrices()})
		if err != nil {
			return pStep{Kind: "socialise", Sub: u, Reject: "insurance is out of funds"}, true
		}
		l.readRedis(u, contractUtils.INSURANCE_SUBACCOUNT_ID)
		return pStep{Kind: "socialise", Sub: u}, true
	}
	return pStep{}, false
}

// withdraw: exactly the Go withdrawable amount must pass, one wei more must fail.
func (l *ledger) withdraw(users []string) (pStep, bool) {
	u := users[l.r.Intn(len(users))]
	a := l.acct(u)
	w, err := subaccount.GetWithdrawableBalance(*clone(a), l.prices, l.markets, l.cum)
	if err != nil {
		l.t.Fatal(err)
	}
	amt, ok := new(big.Int).SetString(w[contractUtils.QUOTE_TOKEN_PRODUCT_ID], 10)
	if !ok || amt.Cmp(x18(1)) < 0 {
		return pStep{}, false
	}
	if l.r.Intn(2) == 0 {
		return pStep{Kind: "withdraw", Sub: u, Amount: new(big.Int).Add(amt, big.NewInt(1)).String(), Reject: "exceeds withdrawable"}, true
	}
	if l.r.Intn(2) == 0 {
		amt = new(big.Int).Quo(amt, big.NewInt(3))
	}
	a.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, new(big.Int).Neg(amt))
	return pStep{Kind: "withdraw", Sub: u, Amount: amt.String()}, true
}

func (l *ledger) dump(ids []string) []pState {
	var out []pState
	for _, id := range ids {
		a := l.acct(id)
		st := pState{Sub: id, Spots: [][2]string{}, Perps: [][4]string{}}
		var sp []int
		for pid := range a.SpotBalances {
			sp = append(sp, int(pid))
		}
		sort.Ints(sp)
		for _, pid := range sp {
			st.Spots = append(st.Spots, [2]string{fmt.Sprint(pid), a.SpotBalances[uint32(pid)].Balancex18.String()})
		}
		var pp []int
		for pid := range a.PerpBalances {
			pp = append(pp, int(pid))
		}
		sort.Ints(pp)
		for _, pid := range pp {
			b := a.PerpBalances[uint32(pid)]
			st.Perps = append(st.Perps, [4]string{fmt.Sprint(pid), b.Amountx18.String(), b.VQuoteBalancex18.String(), b.LastCumFundingRatex18.String()})
		}
		out = append(out, st)
	}
	return out
}

func runScenario(t *testing.T, seed int64, steps int, stress bool) pScenario {
	r := rand.New(rand.NewSource(seed))
	l := &ledger{
		t: t, r: r, bs: &services.BalanceService{}, pu: &perputils.PerpUtils{},
		accounts: map[string]*subaccountTypes.SubaccountBalances{}, markets: map[uint]subaccountTypes.PerpetualMarket{},
		prices: map[string]ctypes.OraclePrice{}, cum: map[string]*big.Int{}, lastTime: map[uint32]uint64{},
		oiLong: map[uint32]*big.Int{}, oiShort: map[uint32]*big.Int{},
		perps: []uint32{1, 3, contractUtils.GOAT_MARKET}, now: 1_760_000_000, stress: stress,
	}
	l.ls = &services.LiquidationServiceImpl{RedisClient: xredis.GetRedisClient(), SubaccountBal: subaccount.NewSubaccountBalanceImpl(), AppState: stubAppState{l}}
	sc := pScenario{Seed: seed, Spots: contractUtils.ALL_SPOTS_ON_CONTRACT, Amm: contractUtils.AMM_SUBACCOUNT_ID, Ins: contractUtils.INSURANCE_SUBACCOUNT_ID, FeeSub: contractUtils.TRADING_FEES_SUBACCOUNT_ID, StartSec: l.now}
	start := map[uint32]float64{1: 2500.123, 3: 65000.77, contractUtils.GOAT_MARKET: 0.4321}
	margins := map[uint32][2]float64{1: {0.1, 0.05}, 3: {0.1, 0.05}, contractUtils.GOAT_MARKET: {0.2, 0.1}}
	// AMM caps: tight enough that some trades hit them; GOAT has none
	ammCaps := map[uint32]float64{1: 6, 3: 0.25, contractUtils.GOAT_MARKET: 0}
	for _, pid := range l.perps {
		sym := l.sym(pid)
		imf, mmf := x18(margins[pid][0]), x18(margins[pid][1])
		ammCap := x18(ammCaps[pid])
		l.markets[uint(pid)] = subaccountTypes.PerpetualMarket{ProductId: uint(pid), BaseAsset: sym, InitialMarginFractionx18: imf, MaintenanceMarginFractionx18: mmf, AmmMaxPositionx18: ammCap}
		l.prices[sym] = ctypes.OraclePrice{Pricex18: x18(start[pid]), Symbol: sym}
		l.cum[sym] = big.NewInt(0)
		l.oiLong[pid], l.oiShort[pid] = big.NewInt(0), big.NewInt(0)
		sc.Perps = append(sc.Perps, [5]string{fmt.Sprint(pid), imf.String(), mmf.String(), marketutils.GetLiquidationFractionx18(pid).String(), ammCap.String()})
	}
	l.prices["USDC"] = ctypes.OraclePrice{Pricex18: new(big.Int).Set(e18), Symbol: "USDC"}

	var users []string
	for i := 0; i < 6; i++ {
		name := fmt.Sprintf("u%d-%d.near", seed, i)
		addr, err := cutils.NearAccountToAddr20(name)
		if err != nil {
			t.Fatal(err)
		}
		id := sub32(2, addr, 0)
		sc.Users = append(sc.Users, name)
		users = append(users, id)
		a := l.acct(id)
		if stress {
			a.UpdateSpotBalance(4, x18(80+r.Float64()*400))
		} else {
			a.UpdateSpotBalance(4, x18(200+r.Float64()*3000))
		}
		if r.Intn(2) == 0 {
			a.UpdateSpotBalance(74, x18(r.Float64()*800))
		}
	}
	l.acct(contractUtils.INSURANCE_SUBACCOUNT_ID).UpdateSpotBalance(4, x18(2_000+r.Float64()*20_000))
	l.acct(contractUtils.AMM_SUBACCOUNT_ID)
	l.acct(contractUtils.TRADING_FEES_SUBACCOUNT_ID)
	all := append(append([]string{}, users...), contractUtils.AMM_SUBACCOUNT_ID, contractUtils.INSURANCE_SUBACCOUNT_ID, contractUtils.TRADING_FEES_SUBACCOUNT_ID)
	sc.Initial = l.dump(all)
	parties := append(append([]string{}, users...), contractUtils.AMM_SUBACCOUNT_ID)

	sc.Steps = append(sc.Steps, l.tick())
	for len(sc.Steps) < steps {
		switch k := r.Intn(20); {
		case k < 3:
			sc.Steps = append(sc.Steps, l.tick())
		case k < 14:
			sc.Steps = append(sc.Steps, l.match(parties))
		case k < 16:
			if st, ok := l.liquidate(users); ok {
				sc.Steps = append(sc.Steps, st)
			}
		case k < 17:
			if st, ok := l.settle(users); ok {
				sc.Steps = append(sc.Steps, st)
			}
		case k < 18:
			if st, ok := l.socialise(users); ok {
				sc.Steps = append(sc.Steps, st)
			}
		default:
			if st, ok := l.withdraw(users); ok {
				sc.Steps = append(sc.Steps, st)
			}
		}
	}
	sc.Final = l.dump(all)
	for _, pid := range l.perps {
		sc.OI = append(sc.OI, [3]string{fmt.Sprint(pid), l.oiLong[pid].String(), l.oiShort[pid].String()})
		sc.Cum = append(sc.Cum, [2]string{fmt.Sprint(pid), l.cum[l.sym(pid)].String()})
	}
	return sc
}

func TestGenerateParity(t *testing.T) {
	out := os.Getenv("PARITY_OUT")
	if out == "" {
		t.Skip("set PARITY_OUT to write vectors/parity.json")
	}
	testutils.SetMainnetEnv()
	testutils.SetupBalanceMainnetEnv()
	defer testutils.ResetEnv()
	var scenarios []pScenario
	counts := map[string]int{}
	testutils.WithSetupMockRedis(t, func() {
		for seed := int64(1); seed <= 24; seed++ {
			sc := runScenario(t, seed, 160, seed > 16)
			for _, s := range sc.Steps {
				counts[s.Kind+"/"+s.Reject]++
			}
			scenarios = append(scenarios, sc)
		}
	})
	t.Logf("step counts: %v", counts)
	b, err := json.MarshalIndent(map[string]any{
		"spec":      "behavior-spec §3; generated by services/balance-server/tests/parity_gen_test.go from production Go code",
		"scenarios": scenarios,
	}, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
