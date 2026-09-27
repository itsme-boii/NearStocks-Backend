package nearbatch

import (
	"context"
	"errors"
	"sort"
	"strings"
	"testing"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/nearchain"
)

// ---- fakes

type memStore struct{ rows map[uint]*db.BatchTable }

func newStore(rows ...db.BatchTable) *memStore {
	s := &memStore{rows: map[uint]*db.BatchTable{}}
	for i := range rows {
		r := rows[i]
		s.rows[r.ID] = &r
	}
	return s
}

func (s *memStore) sorted(f func(*db.BatchTable) bool, key func(*db.BatchTable) uint) []db.BatchTable {
	var out []db.BatchTable
	for _, r := range s.rows {
		if f(r) {
			out = append(out, *r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return key(&out[i]) < key(&out[j]) })
	return out
}

func (s *memStore) NearPending(limit int, exclude []string) ([]db.BatchTable, error) {
	ex := map[string]bool{}
	for _, e := range exclude {
		ex[e] = true
	}
	out := s.sorted(func(r *db.BatchTable) bool {
		return !r.DeletedAt.Valid && r.NearState == "" && !ex[r.SubAccountID1] && !ex[r.SubAccountID2]
	}, func(r *db.BatchTable) uint { return r.TransactionCounter })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func (s *memStore) NearInflight() ([]db.BatchTable, error) {
	return s.sorted(func(r *db.BatchTable) bool { return r.NearState == db.NearStateInflight },
		func(r *db.BatchTable) uint { return r.NSubmissionIdx }), nil
}
func (s *memStore) NearMarkInflight(ids []uint, start uint64) error {
	for i, id := range ids {
		s.rows[id].NearState, s.rows[id].NSubmissionIdx = db.NearStateInflight, uint(start)+uint(i)
	}
	return nil
}
func (s *memStore) NearSetTxHash(ids []uint, h string) error {
	for _, id := range ids {
		s.rows[id].NearTxHash = h
	}
	return nil
}
func (s *memStore) NearMarkLanded(ids []uint) error {
	for _, id := range ids {
		s.rows[id].NearState = ""
		s.rows[id].DeletedAt.Valid = true
	}
	return nil
}
func (s *memStore) NearClearInflight(ids []uint) error {
	for _, id := range ids {
		s.rows[id].NearState, s.rows[id].NSubmissionIdx = "", 0
	}
	return nil
}
func (s *memStore) NearMarkFailed(id uint, reason string) error {
	s.rows[id].NearState, s.rows[id].NearFailure = db.NearStateFailed, reason
	return nil
}

// fakeChain accepts a batch unless it contains a "bad" payload; it records every landed payload.
type fakeChain struct {
	n             uint64
	bad           map[string]string // payload -> panic message
	landed        [][]byte
	calls         int
	outcomes      []nearchain.Outcome // forced outcomes for the next calls
	landOnUnknown bool
}

func (c *fakeChain) NSubmissions(context.Context) (uint64, error) { return c.n, nil }
func (c *fakeChain) Submit(_ context.Context, idx uint64, txs, _, _ [][]byte) nearchain.SubmitResult {
	c.calls++
	if idx != c.n {
		return nearchain.SubmitResult{Outcome: nearchain.Rejected, Failure: "expected idx"}
	}
	if len(c.outcomes) > 0 {
		o := c.outcomes[0]
		c.outcomes = c.outcomes[1:]
		if o == nearchain.Unknown && c.landOnUnknown {
			c.n += uint64(len(txs))
			c.landed = append(c.landed, txs...)
		}
		return nearchain.SubmitResult{Outcome: o, Failure: "forced", TxHash: "h"}
	}
	for _, t := range txs {
		if msg, ok := c.bad[string(t)]; ok {
			return nearchain.SubmitResult{Outcome: nearchain.Rejected, Failure: msg, TxHash: "h"}
		}
	}
	c.n += uint64(len(txs))
	c.landed = append(c.landed, txs...)
	return nearchain.SubmitResult{Outcome: nearchain.Landed, TxHash: "h"}
}

type memPauser struct{ paused []string }

func (p *memPauser) Paused() []string         { return p.paused }
func (p *memPauser) Pause(s string, _ string) { p.paused = append(p.paused, s) }

func row(id uint, sub string, payload string) db.BatchTable {
	return db.BatchTable{BaseTable: db.BaseTable{ID: id}, TransactionCounter: id, SubAccountID1: sub,
		Transaction: append([]byte{nearchain.TxMatchOrders}, payload...), Signature1: []byte{1}, FunctionName: "MatchOrders"}
}

func payloads(txs [][]byte) []string {
	var out []string
	for _, t := range txs {
		out = append(out, string(t[1:]))
	}
	return out
}

func newBatcher(s *memStore, c *fakeChain, p *memPauser, alerts *[]string) *Batcher {
	return &Batcher{Store: s, Chain: c, Pauser: p, Alert: func(m string) { *alerts = append(*alerts, m) }}
}

// ---- tests

func TestSendsEverythingInCounterOrder(t *testing.T) {
	s := newStore(row(3, "c", "3"), row(1, "a", "1"), row(2, "b", "2"))
	c := &fakeChain{}
	var alerts []string
	n, err := newBatcher(s, c, &memPauser{}, &alerts).RunOnce(context.Background())
	if err != nil || n != 3 || c.n != 3 {
		t.Fatalf("n=%d err=%v chain=%d", n, err, c.n)
	}
	if got := strings.Join(payloads(c.landed), ","); got != "1,2,3" {
		t.Fatalf("order %s", got)
	}
	if left, _ := s.NearPending(10, nil); len(left) != 0 {
		t.Fatalf("%d rows left", len(left))
	}
}

func TestOneBadTransactionIsIsolatedAndItsSubaccountPaused(t *testing.T) {
	var rows []db.BatchTable
	for i := uint(1); i <= 8; i++ {
		sub := "u" + string(rune('0'+i))
		if i == 7 {
			sub = "u5" // same subaccount as the bad row: must wait
		}
		rows = append(rows, row(i, sub, string(rune('0'+i))))
	}
	s := newStore(rows...)
	c := &fakeChain{bad: map[string]string{string(append([]byte{nearchain.TxMatchOrders}, '5')): "unhealthy trade for u5"}}
	p := &memPauser{}
	var alerts []string
	n, err := newBatcher(s, c, p, &alerts).RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 || strings.Join(payloads(c.landed), ",") != "1,2,3,4,6,8" {
		t.Fatalf("landed %d: %v", n, payloads(c.landed))
	}
	if s.rows[5].NearState != db.NearStateFailed || !strings.Contains(s.rows[5].NearFailure, "unhealthy") {
		t.Fatalf("row 5: %+v", s.rows[5])
	}
	if s.rows[7].NearState != "" || s.rows[7].DeletedAt.Valid {
		t.Fatal("row 7 (paused subaccount) must stay queued")
	}
	if len(p.paused) != 1 || p.paused[0] != "u5" || len(alerts) != 1 {
		t.Fatalf("paused %v alerts %v", p.paused, alerts)
	}
	if c.calls > 8 {
		t.Fatalf("bisecting 8 rows took %d submissions", c.calls)
	}
}

func TestSystemicRefusalDoesNotBisectOrPark(t *testing.T) {
	s := newStore(row(1, "a", "1"), row(2, "b", "2"), row(3, "c", "3"))
	bad := map[string]string{}
	for _, r := range s.rows {
		bad[string(r.Transaction)] = "stale price for product 3"
	}
	c := &fakeChain{bad: bad}
	p := &memPauser{}
	var alerts []string
	_, err := newBatcher(s, c, p, &alerts).RunOnce(context.Background())
	if !errors.Is(err, ErrSystemic) || c.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, c.calls)
	}
	pending, _ := s.NearPending(10, nil)
	if len(pending) != 3 || len(p.paused) != 0 {
		t.Fatalf("pending %d paused %v", len(pending), p.paused)
	}
}

func TestUnknownOutcomeIsReconciledFromTheContract(t *testing.T) {
	for _, landed := range []bool{true, false} {
		s := newStore(row(1, "a", "1"), row(2, "b", "2"))
		c := &fakeChain{outcomes: []nearchain.Outcome{nearchain.Unknown}, landOnUnknown: landed}
		var alerts []string
		b := newBatcher(s, c, &memPauser{}, &alerts)
		if _, err := b.RunOnce(context.Background()); !errors.Is(err, ErrUnknownOutcome) {
			t.Fatalf("err %v", err)
		}
		if inflight, _ := s.NearInflight(); len(inflight) != 2 {
			t.Fatal("rows stay in flight until the contract says what happened")
		}
		// next run: recovery reads n_submissions
		n, err := b.RunOnce(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if got := len(c.landed); got != 2 {
			t.Fatalf("landed=%v: %d payloads on chain, want exactly 2 (no double send)", landed, got)
		}
		if landed && n != 0 || !landed && n != 2 {
			t.Fatalf("landed=%v: second run sent %d", landed, n)
		}
		if left, _ := s.NearPending(10, nil); len(left) != 0 {
			t.Fatal("nothing left")
		}
	}
}

func TestInconsistentIndexStopsTheBatcher(t *testing.T) {
	s := newStore(row(1, "a", "1"), row(2, "b", "2"), row(3, "c", "3"))
	_ = s.NearMarkInflight([]uint{1, 2, 3}, 10)
	c := &fakeChain{n: 11}
	var alerts []string
	if _, err := newBatcher(s, c, &memPauser{}, &alerts).RunOnce(context.Background()); !errors.Is(err, nearchain.ErrInconsistent) {
		t.Fatalf("err %v", err)
	}
	if len(alerts) != 1 || c.calls != 0 {
		t.Fatal("must alert and send nothing")
	}
}

func TestBudgetSplitsBatches(t *testing.T) {
	var rows []db.BatchTable
	for i := uint(1); i <= 9; i++ {
		r := row(i, "u", "x")
		r.Transaction = []byte{nearchain.TxWithdrawCollateral, byte(i)}
		rows = append(rows, r)
	}
	// 3 base + 30 per withdrawal: six fit under 200 Tgas
	if got := (Budget{}).Fit(rows); got != 6 {
		t.Fatalf("fit %d withdrawals", got)
	}
	var matches []db.BatchTable
	for i := uint(1); i <= 200; i++ {
		matches = append(matches, row(i, "u", "m"))
	}
	if got := (Budget{}).Fit(matches); got != 80 {
		t.Fatalf("fit %d matches, want the 80 cap", got)
	}
	if got := (Budget{MaxLogBytes: 1_000}).Fit(matches); got != 5 {
		t.Fatalf("log budget: fit %d", got)
	}
	s := newStore(rows...)
	c := &fakeChain{}
	var alerts []string
	if n, err := newBatcher(s, c, &memPauser{}, &alerts).RunOnce(context.Background()); err != nil || n != 9 || c.calls != 2 {
		t.Fatalf("n=%d err=%v calls=%d", n, err, c.calls)
	}
}

func TestBalanceAlerts(t *testing.T) {
	ok := &nearchain.AccountView{Amount: near(50).String(), StorageUsage: 100_000} // 1 NEAR of state
	if a := BalanceAlerts(ok, ok, near(5), near(20)); len(a) != 0 {
		t.Fatalf("no alert expected: %v", a)
	}
	low := &nearchain.AccountView{Amount: near(1).String()}
	tight := &nearchain.AccountView{Amount: near(25).String(), StorageUsage: 1_000_000} // 10 NEAR locked, 15 free
	a := BalanceAlerts(low, tight, near(5), near(20))
	if len(a) != 2 || !strings.Contains(a[0], "sequencer") || !strings.Contains(a[1], "storage headroom") {
		t.Fatalf("%v", a)
	}
}

func TestCulpritNoteNamesTheRefusedParty(t *testing.T) {
	user := "0x000000000002984c2655307e4c2219b5829cdf7eb1f87f88d69f000000000001"
	amm := "0x0000000000010000000000000000000000000000000000000001000000000001"
	r := db.BatchTable{SubAccountID1: user, SubAccountID2: amm, FunctionName: "MatchOrders"}
	note := culpritNote(r, `{"ActionError":{"kind":{"FunctionCallError":{"ExecutionError":"Smart contract panicked: session key not registered or expired for `+user+`"}}}}`)
	if !strings.Contains(note, "refused because of "+user) || !strings.Contains(note, amm+" is paused because its backend state") ||
		!strings.Contains(note, "fee account") {
		t.Fatalf("%q", note)
	}
	if culprit(r, "Smart contract panicked: stale price") != "" {
		t.Fatal("no culprit named, no note")
	}
}
