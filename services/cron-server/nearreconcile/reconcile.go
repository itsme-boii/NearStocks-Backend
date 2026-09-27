// Package nearreconcile compares near-stocks.near with the backend (Development.md §8.6). The
// parity harness shows the two ledgers compute identical numbers; this cron proves they actually
// agree in production. Every recently settled subaccount whose queue is empty is read from the
// contract (get_subaccount) and compared with the balance-server, field by field, with no tolerance.
// A difference pauses the subaccount's sequencing, alerts, and is recorded.
package nearreconcile

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
)

// Snapshot is one subaccount as both sides report it: spot and perp values keyed by product id.
type Snapshot struct {
	Spots map[uint32]*big.Int
	Perps map[uint32][3]*big.Int // amount, vQuote, lastCumFunding
	// Pools are pre-market (kind 1) and synthetic spot (kind 2) token balances, keyed {kind, product}.
	Pools map[[2]uint32]*big.Int
}

// FromChain parses the contract's get_subaccount view.
func FromChain(raw []byte) (Snapshot, error) {
	var v struct {
		Spots [][2]json.RawMessage `json:"spots"`
		Perps [][4]json.RawMessage `json:"perps"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return Snapshot{}, err
	}
	s := Snapshot{Spots: map[uint32]*big.Int{}, Perps: map[uint32][3]*big.Int{}}
	pid := func(r json.RawMessage) uint32 { var n uint32; _ = json.Unmarshal(r, &n); return n }
	num := func(r json.RawMessage) *big.Int {
		var str string
		_ = json.Unmarshal(r, &str)
		b, _ := new(big.Int).SetString(str, 10)
		if b == nil {
			return new(big.Int)
		}
		return b
	}
	for _, e := range v.Spots {
		s.Spots[pid(e[0])] = num(e[1])
	}
	for _, e := range v.Perps {
		s.Perps[pid(e[0])] = [3]*big.Int{num(e[1]), num(e[2]), num(e[3])}
	}
	return s, nil
}

// FromBackend converts balance-server balances.
func FromBackend(b subaccountTypes.SubaccountBalances) Snapshot {
	s := Snapshot{Spots: map[uint32]*big.Int{}, Perps: map[uint32][3]*big.Int{}}
	for pid, sp := range b.SpotBalances {
		s.Spots[pid] = sp.Balancex18
	}
	for pid, p := range b.PerpBalances {
		s.Perps[pid] = [3]*big.Int{p.Amountx18, p.VQuoteBalancex18, p.LastCumFundingRatex18}
	}
	return s
}

func val(m map[uint32]*big.Int, k uint32) *big.Int {
	if v := m[k]; v != nil {
		return v
	}
	return new(big.Int)
}

// Diff lists every field that differs (a missing entry counts as 0). Empty means equal.
func Diff(chain, backend Snapshot) []string {
	var out []string
	keys := map[uint32]bool{}
	for k := range chain.Spots {
		keys[k] = true
	}
	for k := range backend.Spots {
		keys[k] = true
	}
	for k := range keys {
		if c, b := val(chain.Spots, k), val(backend.Spots, k); c.Cmp(b) != 0 {
			out = append(out, fmt.Sprintf("spot %d: chain %s backend %s", k, c, b))
		}
	}
	pkeys := map[uint32]bool{}
	for k := range chain.Perps {
		pkeys[k] = true
	}
	for k := range backend.Perps {
		pkeys[k] = true
	}
	poolKeys := map[[2]uint32]bool{}
	for k := range chain.Pools {
		poolKeys[k] = true
	}
	for k := range backend.Pools {
		poolKeys[k] = true
	}
	for k := range poolKeys {
		c, b := chain.Pools[k], backend.Pools[k]
		if c == nil {
			c = new(big.Int)
		}
		if b == nil {
			b = new(big.Int)
		}
		if c.Cmp(b) != 0 {
			out = append(out, fmt.Sprintf("pool %d/%d: chain %s backend %s", k[0], k[1], c, b))
		}
	}
	names := [3]string{"amount", "vQuote", "lastCumFunding"}
	for k := range pkeys {
		c, b := chain.Perps[k], backend.Perps[k]
		for i := 0; i < 3; i++ {
			cv, bv := new(big.Int), new(big.Int)
			if c[i] != nil {
				cv = c[i]
			}
			if b[i] != nil {
				bv = b[i]
			}
			// an untouched position (0, 0, x) and a missing one are the same thing
			if i == 2 && val(map[uint32]*big.Int{0: c[0]}, 0).Sign() == 0 && val(map[uint32]*big.Int{0: b[0]}, 0).Sign() == 0 {
				continue
			}
			if cv.Cmp(bv) != 0 {
				out = append(out, fmt.Sprintf("perp %d %s: chain %s backend %s", k, names[i], cv, bv))
			}
		}
	}
	sort.Strings(out)
	return out
}

// Sources are the three things the reconciler reads and the three it may write.
type Sources struct {
	RecentlyLanded func(since time.Time) ([]string, error)
	HasQueued      func(subs []string) (map[string]bool, error)
	Chain          func(ctx context.Context, sub string) (Snapshot, error)
	Backend        func(subs []string) ([]subaccountTypes.SubaccountBalances, error)
	// Pools (optional) returns a subaccount's pool balances on both sides; nil skips them.
	Pools  func(ctx context.Context, sub string) (chain, backend map[[2]uint32]*big.Int, err error)
	Pause  func(sub, reason string)
	Record func(sub, detail string) error
	Alert  func(string)
}

type Result struct {
	Checked, Skipped int
	Mismatched       []string
}

// Run reconciles subaccounts that settled since `since`.
func Run(ctx context.Context, src Sources, since time.Time) (Result, error) {
	var res Result
	subs, err := src.RecentlyLanded(since)
	if err != nil || len(subs) == 0 {
		return res, err
	}
	queued, err := src.HasQueued(subs)
	if err != nil {
		return res, err
	}
	var ready []string
	for _, s := range subs {
		if queued[s] {
			res.Skipped++
			continue
		}
		ready = append(ready, s)
	}
	if len(ready) == 0 {
		return res, nil
	}
	backend, err := src.Backend(ready)
	if err != nil {
		return res, err
	}
	for i, sub := range ready {
		chain, err := src.Chain(ctx, sub)
		if err != nil {
			return res, fmt.Errorf("get_subaccount %s: %w", sub, err)
		}
		b := FromBackend(backend[i])
		if src.Pools != nil {
			if chain.Pools, b.Pools, err = src.Pools(ctx, sub); err != nil {
				return res, fmt.Errorf("pool balances %s: %w", sub, err)
			}
		}
		res.Checked++
		d := Diff(chain, b)
		if len(d) == 0 {
			continue
		}
		detail := strings.Join(d, "; ")
		res.Mismatched = append(res.Mismatched, sub)
		src.Pause(sub, "reconciler: "+detail)
		if err := src.Record(sub, detail); err != nil {
			return res, err
		}
		src.Alert(fmt.Sprintf("NEAR RECONCILER: %s differs from the chain and is paused: %s", sub, detail))
	}
	return res, nil
}
