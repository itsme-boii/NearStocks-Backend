// Package nearbatch sends queued batch_tables rows to near-stocks.near::submit_transactions
// (Development.md §8.3). Rules:
//   - Order: rows go out in transaction_counter order, in batches sized by gas, log bytes and
//     count (budget.go), one batch in flight at a time.
//   - Crash safety: a batch's planned contract indexes are written before it is sent. After a crash,
//     the contract's n_submissions says whether that batch landed (n >= start+count), did not
//     (n == start), or something is badly wrong (anything else: stop and alert).
//   - Isolation: when the contract refuses a batch because of one transaction, the batch is bisected
//     until that transaction is alone; it is parked as failed, its subaccounts are paused (their later
//     rows wait for a human), and everything else still goes out in order.
//   - Systemic refusals (stale prices, pause, wrong index...) never bisect: nothing is parked, the
//     run stops and alerts, and the rows are retried next run.
package nearbatch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/nearchain"
)

type Store interface {
	NearPending(limit int, exclude []string) ([]db.BatchTable, error)
	NearInflight() ([]db.BatchTable, error)
	NearMarkInflight(ids []uint, startIdx uint64) error
	NearSetTxHash(ids []uint, hash string) error
	NearMarkLanded(ids []uint) error
	NearClearInflight(ids []uint) error
	NearMarkFailed(id uint, reason string) error
}

type Chain interface {
	NSubmissions(ctx context.Context) (uint64, error)
	Submit(ctx context.Context, idx uint64, txs, sigs, sigs2 [][]byte) nearchain.SubmitResult
}

// Pauser is the sequencer pause list (the same Redis hash the appchain batcher used).
type Pauser interface {
	Paused() []string
	Pause(subaccount, reason string)
}

type Batcher struct {
	Store  Store
	Chain  Chain
	Pauser Pauser
	Alert  func(msg string)
	Budget Budget
	// MaxRows bounds how many queued rows one run reads (default 1000).
	MaxRows int
}

var (
	ErrUnknownOutcome = errors.New("batch outcome unknown; will reconcile from n_submissions next run")
	ErrSystemic       = errors.New("batch refused for a systemic reason")
)

// systemic lists contract refusals that are about the batch or the contract, not one transaction.
var systemic = []string{
	"stale price", "no price for product", "paused", "migration not finished", "expected idx", "only sequencer",
	"same non-zero length", "Exceeded the prepaid gas", "GasLimitExceeded", "log message", "number of logs",
}

func isSystemic(failure string) bool {
	for _, s := range systemic {
		if strings.Contains(failure, s) {
			return true
		}
	}
	return false
}

// RunOnce recovers any in-flight batch, then sends everything queued. It returns how many rows
// landed.
func (b *Batcher) RunOnce(ctx context.Context) (int, error) {
	if err := b.recover(ctx); err != nil {
		return 0, err
	}
	limit := b.MaxRows
	if limit <= 0 {
		limit = 1000
	}
	rows, err := b.Store.NearPending(limit, b.Pauser.Paused())
	if err != nil {
		return 0, err
	}
	landed := 0
	for len(rows) > 0 {
		rows = b.dropPaused(rows)
		if len(rows) == 0 {
			break
		}
		n := b.Budget.Fit(rows)
		batch := rows[:n]
		rows = rows[n:]
		got, err := b.send(ctx, batch)
		landed += got
		if err != nil {
			return landed, err
		}
	}
	return landed, nil
}

// recover settles a batch that was in flight when the previous run stopped.
func (b *Batcher) recover(ctx context.Context) error {
	inflight, err := b.Store.NearInflight()
	if err != nil || len(inflight) == 0 {
		return err
	}
	start := uint64(inflight[0].NSubmissionIdx)
	count := uint64(len(inflight))
	n, err := b.Chain.NSubmissions(ctx)
	if err != nil {
		return err
	}
	ids := idsOf(inflight)
	switch {
	case n >= start+count:
		return b.Store.NearMarkLanded(ids)
	case n == start:
		return b.Store.NearClearInflight(ids)
	default:
		b.alert(fmt.Sprintf("NEAR BATCHER STOPPED: n_submissions=%d is inside in-flight batch [%d, %d)", n, start, start+count))
		return nearchain.ErrInconsistent
	}
}

// send submits rows, bisecting on a per-transaction refusal. Returns rows landed.
func (b *Batcher) send(ctx context.Context, rows []db.BatchTable) (int, error) {
	rows = b.dropPaused(rows)
	if len(rows) == 0 {
		return 0, nil
	}
	idx, err := b.Chain.NSubmissions(ctx)
	if err != nil {
		return 0, err
	}
	ids := idsOf(rows)
	if err := b.Store.NearMarkInflight(ids, idx); err != nil {
		return 0, err
	}
	txs, sigs, sigs2 := make([][]byte, len(rows)), make([][]byte, len(rows)), make([][]byte, len(rows))
	for i, r := range rows {
		txs[i], sigs[i], sigs2[i] = r.Transaction, r.Signature1, r.Signature2
	}
	res := b.Chain.Submit(ctx, idx, txs, sigs, sigs2)
	if res.TxHash != "" {
		_ = b.Store.NearSetTxHash(ids, res.TxHash)
	}
	switch res.Outcome {
	case nearchain.Landed:
		return len(rows), b.Store.NearMarkLanded(ids)
	case nearchain.Unknown:
		b.alert(fmt.Sprintf("NEAR batch outcome unknown (tx %s, %d rows from index %d): %s", res.TxHash, len(rows), idx, res.Failure))
		return 0, ErrUnknownOutcome
	}
	// Rejected or NotSent: nothing changed on-chain
	if err := b.Store.NearClearInflight(ids); err != nil {
		return 0, err
	}
	if res.Outcome == nearchain.NotSent {
		return 0, fmt.Errorf("batch not sent: %s", res.Failure)
	}
	if isSystemic(res.Failure) {
		b.alert(fmt.Sprintf("NEAR batch refused (systemic, not bisected; %d rows retried next run): %s", len(rows), res.Failure))
		return 0, ErrSystemic
	}
	if len(rows) == 1 {
		r := rows[0]
		if err := b.Store.NearMarkFailed(r.ID, res.Failure); err != nil {
			return 0, err
		}
		for _, s := range []string{r.SubAccountID1, r.SubAccountID2} {
			if s != "" {
				b.Pauser.Pause(s, res.Failure)
			}
		}
		b.alert(fmt.Sprintf("NEAR tx refused and parked: row %d (%s, counter %d), subaccounts paused %q %q%s: %s",
			r.ID, r.FunctionName, r.TransactionCounter, r.SubAccountID1, r.SubAccountID2, culpritNote(r, res.Failure), res.Failure))
		return 0, nil
	}
	mid := len(rows) / 2
	left, err := b.send(ctx, rows[:mid])
	if err != nil {
		return left, err
	}
	right, err := b.send(ctx, rows[mid:])
	return left + right, err
}

func (b *Batcher) dropPaused(rows []db.BatchTable) []db.BatchTable {
	paused := map[string]bool{}
	for _, p := range b.Pauser.Paused() {
		paused[strings.ToLower(p)] = true
	}
	if len(paused) == 0 {
		return rows
	}
	out := rows[:0:0]
	for _, r := range rows {
		if paused[strings.ToLower(r.SubAccountID1)] || paused[strings.ToLower(r.SubAccountID2)] {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (b *Batcher) alert(msg string) {
	if b.Alert != nil {
		b.Alert(msg)
	}
}

func idsOf(rows []db.BatchTable) []uint {
	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}

// culpritNote names the subaccount the contract blamed ("... for 0x<sub>"). Both parties stay
// paused: the backend already applied the refused fill to each side, so the counterparty's backend
// balance is off by it too until the fill is unwound.
func culpritNote(r db.BatchTable, failure string) string {
	fee := ""
	switch r.FunctionName {
	case "MatchOrders", "LiquidateSubaccount", "WithdrawCollateral", "WithdrawLogX":
		// the backend credited this row's fee to the fee account (D-7) when it queued it
		fee = "; the fee account's backend balance includes this row's fee: resync it too"
	}
	return culprit(r, failure) + fee
}

func culprit(r db.BatchTable, failure string) string {
	i := strings.LastIndex(failure, " for 0x")
	if i < 0 {
		return ""
	}
	hex := failure[i+len(" for "):]
	if len(hex) < 66 {
		return ""
	}
	culprit := strings.ToLower(hex[:66])
	for _, s := range []string{r.SubAccountID1, r.SubAccountID2} {
		if s != "" && !strings.EqualFold(s, culprit) {
			return fmt.Sprintf(" (refused because of %s; %s is paused because its backend state includes the refused fill: unwind it before unpausing)", culprit, s)
		}
	}
	return fmt.Sprintf(" (refused because of %s)", culprit)
}
