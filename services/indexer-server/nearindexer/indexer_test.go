package nearindexer

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"os"
	"testing"
	"time"
)

type vec struct {
	Contract      string              `json:"contract"`
	Subaccount    string              `json:"subaccount"`
	FeeSubaccount string              `json:"feeSubaccount"`
	Events        map[string][]string `json:"events"`
}

func loadVec(t *testing.T) vec {
	raw, err := os.ReadFile("../../../vectors/events.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vec
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// block wraps logs as one receipt executed by executor with the given status.
func block(h uint64, id, executor string, success bool, logs []string) *Block {
	var b Block
	b.Block.Header.Height = h
	b.Shards = make([]struct {
		ReceiptExecutionOutcomes []struct {
			TxHash           string `json:"tx_hash"`
			ExecutionOutcome struct {
				Id      string `json:"id"`
				Outcome struct {
					ExecutorId string                     `json:"executor_id"`
					Logs       []string                   `json:"logs"`
					Status     map[string]json.RawMessage `json:"status"`
				} `json:"outcome"`
			} `json:"execution_outcome"`
		} `json:"receipt_execution_outcomes"`
	}, 1)
	b.Shards[0].ReceiptExecutionOutcomes = make([]struct {
		TxHash           string `json:"tx_hash"`
		ExecutionOutcome struct {
			Id      string `json:"id"`
			Outcome struct {
				ExecutorId string                     `json:"executor_id"`
				Logs       []string                   `json:"logs"`
				Status     map[string]json.RawMessage `json:"status"`
			} `json:"outcome"`
		} `json:"execution_outcome"`
	}, 1)
	r := &b.Shards[0].ReceiptExecutionOutcomes[0]
	r.TxHash = "tx-" + id
	r.ExecutionOutcome.Id = id
	r.ExecutionOutcome.Outcome.ExecutorId = executor
	r.ExecutionOutcome.Outcome.Logs = logs
	if success {
		r.ExecutionOutcome.Outcome.Status = map[string]json.RawMessage{"SuccessValue": json.RawMessage(`""`)}
	} else {
		r.ExecutionOutcome.Outcome.Status = map[string]json.RawMessage{"Failure": json.RawMessage(`{}`)}
	}
	return &b
}

type fakeSink struct {
	claimed  map[string]bool
	balances map[string]*big.Int
	accounts []string
	history  []string
	revoked  []string
}

func newSink() *fakeSink {
	return &fakeSink{claimed: map[string]bool{}, balances: map[string]*big.Int{}}
}
func (s *fakeSink) Claim(e Event) (bool, error) {
	if s.claimed[e.Key] {
		return false, nil
	}
	s.claimed[e.Key] = true
	return true, nil
}
func (s *fakeSink) Done(Event) error           { return nil }
func (s *fakeSink) Failed(Event, string) error { return nil }
func (s *fakeSink) CreditBalance(sub string, pid uint32, a *big.Int) error {
	k := sub + "/" + big.NewInt(int64(pid)).String()
	if s.balances[k] == nil {
		s.balances[k] = new(big.Int)
	}
	s.balances[k].Add(s.balances[k], a)
	return nil
}
func (s *fakeSink) EnsureNearAccount(acct, _ string) error {
	s.accounts = append(s.accounts, acct)
	return nil
}
func (s *fakeSink) RevokeSessionKey(key string) error {
	s.revoked = append(s.revoked, key)
	return nil
}
func (s *fakeSink) RecordTransfer(key, _, _ string, _ uint32, _ *big.Int, dep bool) error {
	s.history = append(s.history, key)
	return nil
}

type memSource struct{ blocks map[uint64]*Block }

func (m memSource) FinalHeight(context.Context) (uint64, error) {
	var max uint64
	for h := range m.blocks {
		if h > max {
			max = h
		}
	}
	return max, nil
}
func (m memSource) Block(_ context.Context, h uint64) (*Block, error) { return m.blocks[h], nil }

type memCheckpoint struct {
	h  uint64
	ok bool
}

func (c *memCheckpoint) Last() (uint64, bool, error) { return c.h, c.ok, nil }
func (c *memCheckpoint) Save(h uint64) error         { c.h, c.ok = h, true; return nil }

func bal(s *fakeSink, sub string, pid string) string {
	if b := s.balances[sub+"/"+pid]; b != nil {
		return b.String()
	}
	return "0"
}

func TestIndexerMirrorsRealContractEvents(t *testing.T) {
	v := loadVec(t)
	e := v.Events
	src := memSource{blocks: map[uint64]*Block{
		10: block(10, "r1", v.Contract, true, e["deposit"]),
		11: nil, // skipped height
		12: block(12, "r2", v.Contract, true, append(append([]string{}, e["withdraw_pending"]...), e["batch"]...)),
		13: block(13, "r3", v.Contract, true, e["withdraw_done"]),
		14: block(14, "r4", v.Contract, true, e["withdraw_failed"]),
		15: block(15, "r5", v.Contract, true, e["fee_sweep_pending"]),
		16: block(16, "r6", v.Contract, false, e["deposit"]),     // failed receipt: ignored
		17: block(17, "r7", "impostor.near", true, e["deposit"]), // another contract: ignored
	}}
	sink := newSink()
	var alerts []string
	cp := &memCheckpoint{}
	x := &Indexer{Source: src, Checkpoint: cp, Contract: v.Contract, StartHeight: 10,
		Handler: &Handler{Sink: sink, FeeSubaccount: v.FeeSubaccount, Alert: func(m string) { alerts = append(alerts, m) }}}
	n, err := x.Step(context.Background())
	if err != nil || n != 8 || cp.h != 17 {
		t.Fatalf("n=%d err=%v checkpoint=%d", n, err, cp.h)
	}
	// deposit 250 USDC; withdraw_failed re-credits 10 (the backend debited it when queueing)
	if got := bal(sink, v.Subaccount, "4"); got != "260000000000000000000" {
		t.Fatalf("user USDC %s", got)
	}
	// fee account: +0.5 from withdraw_done, -0.2 from the DAO sweep
	if got := bal(sink, v.FeeSubaccount, "4"); got != "300000000000000000" {
		t.Fatalf("fee account %s", got)
	}
	if len(sink.accounts) != 1 || sink.accounts[0] != "alice.near" || len(sink.history) != 2 {
		t.Fatalf("accounts %v history %v", sink.accounts, sink.history)
	}
	// replaying the same blocks changes nothing (claims are exactly-once)
	cp.ok, cp.h = false, 0
	if _, err := x.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := bal(sink, v.Subaccount, "4"); got != "260000000000000000000" {
		t.Fatalf("replay changed the balance: %s", got)
	}
}

func TestIndexerAlertsOnOperationalEvents(t *testing.T) {
	v := loadVec(t)
	var alerts []string
	h := &Handler{Sink: newSink(), FeeSubaccount: v.FeeSubaccount, Alert: func(m string) { alerts = append(alerts, m) }}
	evs, err := Extract(block(1, "r", v.Contract, true, v.Events["circuit_breaker"]), v.Contract)
	if err != nil || len(evs) != 1 {
		t.Fatalf("%v %v", evs, err)
	}
	if err := h.Apply(evs[0]); err != nil || len(alerts) != 1 {
		t.Fatalf("alerts %v err %v", alerts, err)
	}
}

func TestSystemDepositCreditsWithoutLinkingTheDAO(t *testing.T) {
	sink := newSink()
	h := &Handler{Sink: sink}
	amm := "0x0000000000010000000000000000000000000000000000000001000000000001"
	e := Event{Name: "deposit", Key: "r1:0", Data: json.RawMessage(`{"subaccount":"` + amm + `","account_id":"near-stocks-dao.testnet",
		"product_id":4,"amount_x18":"5000000000000000000","source":"system"}`)}
	if err := h.Apply(e); err != nil {
		t.Fatal(err)
	}
	if bal(sink, amm, "4") != "5000000000000000000" || len(sink.accounts) != 0 || len(sink.history) != 1 {
		t.Fatalf("balance %s accounts %v history %v", bal(sink, amm, "4"), sink.accounts, sink.history)
	}
}

func TestRevokedSessionKeyIsExpiredInTheBackend(t *testing.T) {
	sink := newSink()
	h := &Handler{Sink: sink}
	e := Event{Name: "session_key_revoked", Key: "r2:0", Data: json.RawMessage(`{"subaccount":"0xaa","session_key":"0x20cd74c6b1ee2be10217b53bd45d0283d91af784"}`)}
	if err := h.Apply(e); err != nil {
		t.Fatal(err)
	}
	if err := h.Apply(e); err != nil { // replay: claimed once
		t.Fatal(err)
	}
	if len(sink.revoked) != 1 || sink.revoked[0] != "0x20cd74c6b1ee2be10217b53bd45d0283d91af784" {
		t.Fatalf("revoked %v", sink.revoked)
	}
}

// slowSource answers later heights first (like concurrent HTTP fetches) and fails one height.
type slowSource struct {
	memSource
	fail uint64
}

func (s slowSource) Block(ctx context.Context, h uint64) (*Block, error) {
	time.Sleep(time.Duration(30-h%30) * time.Millisecond) // lower heights finish last
	if h == s.fail {
		return nil, errors.New("429 Too Many Requests")
	}
	return s.memSource.Block(ctx, h)
}

func TestParallelFetchAppliesInOrderAndStopsAtAFailure(t *testing.T) {
	v := loadVec(t)
	blocks := map[uint64]*Block{}
	for h := uint64(1); h <= 20; h++ {
		blocks[h] = nil
	}
	blocks[3] = block(3, "d1", v.Contract, true, v.Events["deposit"])
	blocks[9] = block(9, "w1", v.Contract, true, v.Events["withdraw_failed"])
	sink := newSink()
	cp := &memCheckpoint{}
	x := &Indexer{Source: slowSource{memSource{blocks}, 12}, Checkpoint: cp, Contract: v.Contract, StartHeight: 1, Parallel: 8,
		Handler: &Handler{Sink: sink, FeeSubaccount: v.FeeSubaccount}}
	n, err := x.Step(context.Background())
	if err == nil || n != 11 || cp.h != 11 {
		t.Fatalf("must stop before the failed block 12: n=%d checkpoint=%d err=%v", n, cp.h, err)
	}
	if got := bal(sink, v.Subaccount, "4"); got != "260000000000000000000" {
		t.Fatalf("both events before the failure applied once: %s", got)
	}
	// the next Step resumes at 12 once it is served
	x.Source = memSource{blocks}
	if n, err := x.Step(context.Background()); err != nil || n != 9 || cp.h != 20 {
		t.Fatalf("resume: n=%d checkpoint=%d err=%v", n, cp.h, err)
	}
}
