package nearchain

// The sequencer's side of near-stocks.near (Development.md §5.4, §8.3): read n_submissions, and
// send submit_transactions batches signed by the sequencer key (local or AWS KMS, kms.go).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

// MaxGas is the most a single NEAR function call can attach.
const MaxGas = 300 * TGas

type Sequencer struct {
	RPC      *Client
	Signer   Signer
	SignerId string // the sequencer account
	Contract string // near-stocks.near

	mu        sync.Mutex
	nextNonce uint64
}

// NSubmissions is the contract's next expected batch index.
func (s *Sequencer) NSubmissions(ctx context.Context) (uint64, error) {
	raw, err := s.RPC.CallView(ctx, s.Contract, "n_submissions", map[string]any{})
	if err != nil {
		return 0, err
	}
	var n uint64
	if err := json.Unmarshal(raw, &n); err != nil {
		return 0, fmt.Errorf("n_submissions: %w", err)
	}
	return n, nil
}

// Outcome classifies a submission, the way SendTreasuryPayout does for payouts.
type Outcome int

const (
	// Landed: the batch executed; the contract advanced n_submissions.
	Landed Outcome = iota
	// Rejected: the batch receipt failed (a transaction panicked), so nothing changed on-chain.
	Rejected
	// NotSent: nothing was broadcast, or the node refused the transaction before execution.
	NotSent
	// Unknown: it may or may not have landed (timeout, RPC error after broadcast). The caller
	// must not resubmit until n_submissions says which.
	Unknown
)

type SubmitResult struct {
	Outcome Outcome
	TxHash  string
	// Failure is the contract's panic message when Outcome is Rejected.
	Failure string
	Result  *TxResult
}

// Submit sends one submit_transactions call and waits for finality.
func (s *Sequencer) Submit(ctx context.Context, idx uint64, txs, sigs, sigs2 [][]byte) SubmitResult {
	return s.Call(ctx, "submit_transactions", SubmitArgs(idx, txs, sigs, sigs2), MaxGas)
}

// PruneFilled removes replay-protection entries of expired orders (sequencer-only view of storage).
func (s *Sequencer) PruneFilled(ctx context.Context, digests []string) SubmitResult {
	args, _ := json.Marshal(map[string]any{"digests": digests})
	return s.Call(ctx, "prune_filled", args, 100*TGas)
}

// Call sends any function call from the sequencer account and classifies the result.
func (s *Sequencer) Call(ctx context.Context, method string, args []byte, gas uint64) SubmitResult {
	s.mu.Lock()
	defer s.mu.Unlock()
	nonce, err := s.nonce(ctx)
	if err != nil {
		return SubmitResult{Outcome: NotSent, Failure: err.Error()}
	}
	block, err := s.RPC.FinalBlockHash(ctx)
	if err != nil {
		return SubmitResult{Outcome: NotSent, Failure: err.Error()}
	}
	tx := &Transaction{
		SignerId: s.SignerId, PublicKey: s.Signer.PublicKey(), Nonce: nonce, ReceiverId: s.Contract, BlockHash: block,
		Actions: []Action{FunctionCall{MethodName: method, Args: args, Gas: gas, Deposit: new(big.Int)}},
	}
	signed, hash, err := SignTransaction(tx, s.Signer)
	if err != nil {
		return SubmitResult{Outcome: NotSent, Failure: err.Error()}
	}
	res, err := s.RPC.SendTx(ctx, signed)
	hashStr := HashString(hash)
	switch {
	case err == nil && res.Succeeded():
		s.nextNonce = nonce + 1
		return SubmitResult{Outcome: Landed, TxHash: hashStr, Result: res}
	case err == nil:
		s.nextNonce = nonce + 1 // the transaction itself was included; only its receipt failed
		return SubmitResult{Outcome: Rejected, TxHash: hashStr, Failure: FailureMessage(res), Result: res}
	case strings.Contains(err.Error(), "INVALID_TRANSACTION"), strings.Contains(err.Error(), "InvalidNonce"):
		s.nextNonce = 0 // refetch from the chain next time
		return SubmitResult{Outcome: NotSent, TxHash: hashStr, Failure: err.Error()}
	default:
		s.nextNonce = 0
		return SubmitResult{Outcome: Unknown, TxHash: hashStr, Failure: err.Error()}
	}
}

func (s *Sequencer) nonce(ctx context.Context) (uint64, error) {
	if s.nextNonce != 0 {
		return s.nextNonce, nil
	}
	key, err := s.RPC.ViewAccessKey(ctx, s.SignerId, PublicKeyString(s.Signer.PublicKey()))
	if err != nil {
		return 0, fmt.Errorf("sequencer access key: %w", err)
	}
	return key.Nonce + 1, nil
}

// FailureMessage extracts the panic text from a failed transaction ("" when it succeeded).
func FailureMessage(res *TxResult) string {
	if res == nil {
		return ""
	}
	for _, o := range append([]ExecutionOutcome{res.TransactionOutcome}, res.ReceiptsOutcome...) {
		if f, ok := o.Outcome.Status["Failure"]; ok {
			return string(f)
		}
	}
	if f, ok := res.Status["Failure"]; ok {
		return string(f)
	}
	return ""
}

// ErrInconsistent means the contract's n_submissions is neither before nor after an in-flight
// batch. Batches are atomic, so this needs a human.
var ErrInconsistent = errors.New("n_submissions is inside an in-flight batch")
