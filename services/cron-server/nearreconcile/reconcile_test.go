package nearreconcile

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
)

func bi(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }

// The chain side is parsed from the contract's real get_subaccount JSON shape.
const chainJSON = `{"subaccount":"0xaa","owner":"alice.near","nonce":3,
 "spots":[[4,"996100000000000000000"],[6,"0"]],
 "perps":[[3,"100000000000000000","-6500000000000000000000","0"],[1,"0","0","77"]]}`

func backend(quote string) subaccountTypes.SubaccountBalances {
	return subaccountTypes.SubaccountBalances{
		SubaccountId: "0xaa",
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{4: {ProductId: 4, Balancex18: bi(quote), Lockedx18: big.NewInt(0)}},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{3: {ProductId: 3, Amountx18: bi("100000000000000000"), VQuoteBalancex18: bi("-6500000000000000000000"), LastCumFundingRatex18: big.NewInt(0)}},
	}
}

func TestDiffTreatsMissingAsZeroAndFindsRealDifferences(t *testing.T) {
	chain, err := FromChain([]byte(chainJSON))
	if err != nil {
		t.Fatal(err)
	}
	if d := Diff(chain, FromBackend(backend("996100000000000000000"))); len(d) != 0 {
		t.Fatalf("equal ledgers reported %v", d)
	}
	d := Diff(chain, FromBackend(backend("996100000000000000001")))
	if len(d) != 1 || !strings.Contains(d[0], "spot 4") {
		t.Fatalf("one wei must be reported: %v", d)
	}
	b := backend("996100000000000000000")
	b.PerpBalances[3] = subaccountTypes.PerpBalance{ProductId: 3, Amountx18: bi("100000000000000000"), VQuoteBalancex18: bi("-6499000000000000000000"), LastCumFundingRatex18: big.NewInt(0)}
	if d := Diff(chain, FromBackend(b)); len(d) != 1 || !strings.Contains(d[0], "vQuote") {
		t.Fatalf("vQuote difference: %v", d)
	}
}

func TestRunSkipsQueuedPausesAndRecordsMismatches(t *testing.T) {
	var paused, recorded, alerts []string
	src := Sources{
		RecentlyLanded: func(time.Time) ([]string, error) { return []string{"0xaa", "0xbb", "0xcc"}, nil },
		HasQueued:      func([]string) (map[string]bool, error) { return map[string]bool{"0xcc": true}, nil },
		Chain: func(_ context.Context, sub string) (Snapshot, error) {
			return FromChain([]byte(chainJSON))
		},
		Backend: func(subs []string) ([]subaccountTypes.SubaccountBalances, error) {
			// 0xaa agrees, 0xbb has one extra USDC in the backend
			return []subaccountTypes.SubaccountBalances{backend("996100000000000000000"), backend("997100000000000000000")}, nil
		},
		Pause:  func(s, _ string) { paused = append(paused, s) },
		Record: func(s, _ string) error { recorded = append(recorded, s); return nil },
		Alert:  func(m string) { alerts = append(alerts, m) },
	}
	res, err := Run(context.Background(), src, time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if res.Checked != 2 || res.Skipped != 1 || len(res.Mismatched) != 1 || res.Mismatched[0] != "0xbb" {
		t.Fatalf("%+v", res)
	}
	if len(paused) != 1 || paused[0] != "0xbb" || len(recorded) != 1 || len(alerts) != 1 {
		t.Fatalf("paused %v recorded %v alerts %v", paused, recorded, alerts)
	}
}

func TestPoolBalancesAreReconciled(t *testing.T) {
	chain, _ := FromChain([]byte(chainJSON))
	chain.Pools = map[[2]uint32]*big.Int{{1, 900}: bi("50000000000000000000")}
	b := FromBackend(backend("996100000000000000000"))
	b.Pools = map[[2]uint32]*big.Int{{1, 900}: bi("50000000000000000000")}
	if d := Diff(chain, b); len(d) != 0 {
		t.Fatalf("equal pools reported %v", d)
	}
	b.Pools[[2]uint32{2, 901}] = bi("1")
	if d := Diff(chain, b); len(d) != 1 || !strings.Contains(d[0], "pool 2/901") {
		t.Fatalf("a synthetic balance only the backend has must be reported: %v", d)
	}
	var paused []string
	src := Sources{
		RecentlyLanded: func(time.Time) ([]string, error) { return []string{"0xaa"}, nil },
		HasQueued:      func([]string) (map[string]bool, error) { return map[string]bool{}, nil },
		Chain:          func(context.Context, string) (Snapshot, error) { return FromChain([]byte(chainJSON)) },
		Backend: func([]string) ([]subaccountTypes.SubaccountBalances, error) {
			return []subaccountTypes.SubaccountBalances{backend("996100000000000000000")}, nil
		},
		Pools: func(context.Context, string) (map[[2]uint32]*big.Int, map[[2]uint32]*big.Int, error) {
			return map[[2]uint32]*big.Int{{1, 900}: bi("5")}, map[[2]uint32]*big.Int{{1, 900}: bi("6")}, nil
		},
		Pause:  func(s, _ string) { paused = append(paused, s) },
		Record: func(string, string) error { return nil },
		Alert:  func(string) {},
	}
	res, err := Run(context.Background(), src, time.Now().Add(-time.Hour))
	if err != nil || len(res.Mismatched) != 1 || len(paused) != 1 {
		t.Fatalf("a pool mismatch must pause: %+v %v %v", res, paused, err)
	}
}

func TestFromChainOntoMakesTheLedgersEqualAndKeepsLocks(t *testing.T) {
	chain, _ := FromChain([]byte(chainJSON))
	b := backend("997100000000000000000") // one USDC too many (a refused fill the backend applied)
	b.SpotBalances[4] = subaccountTypes.SpotBalance{ProductId: 4, Balancex18: bi("997100000000000000000"), Lockedx18: bi("5")}
	b.PerpBalances[1] = subaccountTypes.PerpBalance{ProductId: 1, Amountx18: bi("10000000000000000"), VQuoteBalancex18: bi("-27000000000000000000"), LastCumFundingRatex18: big.NewInt(0)}
	if len(Diff(chain, FromBackend(b))) == 0 {
		t.Fatal("setup: ledgers should differ")
	}
	fixed := FromChainOnto(b, chain)
	if d := Diff(chain, FromBackend(fixed)); len(d) != 0 {
		t.Fatalf("after resync: %v", d)
	}
	if fixed.SpotBalances[4].Lockedx18.String() != "5" {
		t.Fatal("open-order locks are off-chain state and must be kept")
	}
}
