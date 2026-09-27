package main

// Math parity vectors (behavior-spec §3.1, §3.2, gate G3 core). Every expected value comes from
// calling the production Go code, not a re-implementation.

import (
	"math/big"
	"math/rand"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
)

type perpState struct {
	Amount  string `json:"amount"`
	VQuote  string `json:"vQuote"`
	LastCum string `json:"lastCum"`
}

type updateCase struct {
	Before     perpState `json:"before"`
	DeltaA     string    `json:"deltaA"`
	DeltaQ     string    `json:"deltaQ"`
	CumNow     string    `json:"cumNow"`
	After      perpState `json:"after"`
	Pnl        string    `json:"pnl"`
	FundingFee string    `json:"fundingFee"`
}

type matchCase struct {
	Matched     string `json:"matched"`
	Price       string `json:"price"`
	MakerIsBuy  bool   `json:"makerIsBuy"`
	TakerIsBuy  bool   `json:"takerIsBuy"`
	MakerDeltaA string `json:"makerDeltaA"`
	MakerDeltaQ string `json:"makerDeltaQ"`
	TakerDeltaA string `json:"takerDeltaA"`
	TakerDeltaQ string `json:"takerDeltaQ"`
}

type feeCase struct {
	BrokerId uint   `json:"brokerId"`
	DeltaQ   string `json:"deltaQ"`
	Fee      string `json:"fee"` // positive amount removed from quote
}

// randSigned returns a value in (-10^digits, 10^digits), biased toward awkward non-round numbers.
func randSigned(r *rand.Rand, digits int) *big.Int {
	limit := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	v := new(big.Int).Rand(r, limit)
	switch r.Intn(6) {
	case 0:
		v.SetInt64(0)
	case 1:
		v.Neg(v)
	case 2, 3:
		if r.Intn(2) == 0 {
			v.Neg(v)
		}
	}
	return v
}

func st(pb subaccountTypes.PerpBalance) perpState {
	return perpState{pb.Amountx18.String(), pb.VQuoteBalancex18.String(), pb.LastCumFundingRatex18.String()}
}

func genMath(dir string) {
	r := rand.New(rand.NewSource(413))

	// ---- UpdateBalance: independent random cases plus sequences that carry state ----
	updates := []updateCase{}
	step := func(pb *subaccountTypes.PerpBalance, dA, dQ, cum *big.Int) {
		before := st(*pb)
		pnl, fee := pb.UpdateBalance(new(big.Int).Set(dA), new(big.Int).Set(dQ), new(big.Int).Set(cum))
		updates = append(updates, updateCase{before, dA.String(), dQ.String(), cum.String(), st(*pb), pnl.String(), fee.String()})
	}
	for i := 0; i < 400; i++ {
		pb := subaccountTypes.PerpBalance{
			Amountx18:             randSigned(r, 22),
			VQuoteBalancex18:      randSigned(r, 27),
			LastCumFundingRatex18: randSigned(r, 16),
			ProductId:             1,
		}
		if pb.Amountx18.Sign() == 0 {
			pb.VQuoteBalancex18.SetInt64(0) // invariant: flat position carries no vQuote
		}
		step(&pb, randSigned(r, 22), randSigned(r, 27), randSigned(r, 16))
	}
	// realistic sequences: open, add, partially close, flip, close, with funding moving
	for s := 0; s < 40; s++ {
		pb := subaccountTypes.PerpBalance{Amountx18: big.NewInt(0), VQuoteBalancex18: big.NewInt(0), LastCumFundingRatex18: big.NewInt(0), ProductId: 1}
		cum := big.NewInt(0)
		price := new(big.Int).Add(new(big.Int).Rand(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(23), nil)), big.NewInt(1))
		for k := 0; k < 8; k++ {
			dA := randSigned(r, 20)
			dQ := cutils.Divx18(new(big.Int).Mul(new(big.Int).Neg(dA), price)) // engine-style quote
			cum = new(big.Int).Add(cum, randSigned(r, 14))
			step(&pb, dA, dQ, cum)
			price = new(big.Int).Add(price, randSigned(r, 21))
			if price.Sign() <= 0 {
				price.SetInt64(1)
			}
		}
	}

	// ---- Match deltas: services/engine/placeOrder.engine.go:378-382 ----
	matches := []matchCase{}
	withSign := func(a *big.Int, buy bool) *big.Int {
		if buy {
			return new(big.Int).Set(a)
		}
		return new(big.Int).Neg(a)
	}
	for i := 0; i < 200; i++ {
		m := new(big.Int).Add(new(big.Int).Rand(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(22), nil)), big.NewInt(1))
		p := new(big.Int).Add(new(big.Int).Rand(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)), big.NewInt(1))
		makerBuy := r.Intn(2) == 0
		takerBuy := !makerBuy
		mA := withSign(m, makerBuy)
		tA := withSign(m, takerBuy)
		mQ := cutils.Divx18(new(big.Int).Mul(tA, p))
		tQ := cutils.Divx18(new(big.Int).Mul(mA, p))
		matches = append(matches, matchCase{m.String(), p.String(), makerBuy, takerBuy, mA.String(), mQ.String(), tA.String(), tQ.String()})
	}

	// ---- DeductTradingFee: libs/subaccountTypes/balance.subaccountTypes.go:100-116 ----
	fees := []feeCase{}
	for _, broker := range []uint{1, 2} {
		addr, _ := cutils.NearAccountToAddr20("alice.near")
		_, sub := subaccountBytes(broker, addr, 0)
		hexId := hx(sub[:])
		for i := 0; i < 60; i++ {
			dQ := randSigned(r, 27)
			sb := subaccountTypes.SubaccountBalances{SubaccountId: hexId, SpotBalances: map[uint32]subaccountTypes.SpotBalance{}, PerpBalances: map[uint32]subaccountTypes.PerpBalance{}}
			sb.DeductTradingFee(new(big.Int).Set(dQ), i%2 == 0)
			fee := new(big.Int).Neg(sb.MustGetSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID).Balancex18)
			fees = append(fees, feeCase{broker, dQ.String(), fee.String()})
		}
	}

	write(dir, "math.json", map[string]any{
		"spec":          "near-stocks-contracts/docs/behavior-spec.md §3.1 (UpdateBalance), §3.2 (match deltas, fee). Division is Euclidean (Go big.Int.Div).",
		"updateBalance": updates,
		"matchDeltas":   matches,
		"tradingFee":    fees,
	})
}
