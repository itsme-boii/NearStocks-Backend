package nearindexer

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Sink is everything the handler changes off-chain (production: sink.go; tests: a fake).
type Sink interface {
	// Claim is true exactly once per event key.
	Claim(e Event) (bool, error)
	Done(e Event) error
	Failed(e Event, reason string) error
	// CreditBalance adds x18 (may be negative) to a subaccount's token balance in the balance-server.
	CreditBalance(subaccountHex string, productId uint32, amountX18 *big.Int) error
	// EnsureNearAccount creates the account -> subaccount mapping for a first-time depositor.
	EnsureNearAccount(accountId, subaccountHex string) error
	// RecordTransfer stores the deposit/withdrawal history row the UI lists.
	RecordTransfer(key, txHash, subaccountHex string, productId uint32, amountX18 *big.Int, isDeposit bool) error
	// RevokeSessionKey expires a session key the user revoked on-chain, so the API stops accepting
	// requests signed with it (the contract would refuse them and pause the subaccount).
	RevokeSessionKey(sessionKey string) error
}

type Handler struct {
	Sink Sink
	// FeeSubaccount is TRADING_FEES_SUBACCOUNT_ID (D-7).
	FeeSubaccount string
	Alert         func(string)
}

func num(v json.RawMessage) (*big.Int, error) {
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		var n json.Number
		if err2 := json.Unmarshal(v, &n); err2 != nil {
			return nil, fmt.Errorf("not a number: %s", v)
		}
		s = n.String()
	}
	b, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("not a number: %q", s)
	}
	return b, nil
}

type fields map[string]json.RawMessage

func (f fields) str(k string) string {
	var s string
	_ = json.Unmarshal(f[k], &s)
	return s
}

func (f fields) pid(k string) uint32 {
	var n uint32
	_ = json.Unmarshal(f[k], &n)
	return n
}

// Apply handles one event exactly once. It returns an error only for infrastructure failures
// (the claim could not be written), which stop the indexer at this block so it is retried.
func (h *Handler) Apply(e Event) error {
	switch e.Name {
	case "deposit", "withdraw_failed", "withdraw_done", "fee_sweep_pending", "session_key_revoked":
	// held and unclaimed (S-6) deposits are in custody but in nobody's balance yet: alert; the
	// later `deposit` (release_held_deposit / assign_unclaimed) is what credits
	case "circuit_breaker", "paused", "price_set", "migration_finished", "deposit_held", "deposit_unclaimed":
		h.alert(fmt.Sprintf("near-stocks %s: %s", e.Name, e.Data))
		return nil
	default:
		return nil // batch, withdraw_pending, session_key_created, reward_rate: nothing to mirror
	}
	claimed, err := h.Sink.Claim(e)
	if err != nil || !claimed {
		return err
	}
	if err := h.apply(e); err != nil {
		h.alert(fmt.Sprintf("NEAR INDEXER: %s %s not mirrored (manual fix needed): %v", e.Name, e.Key, err))
		return h.Sink.Failed(e, err.Error())
	}
	return h.Sink.Done(e)
}

func (h *Handler) apply(e Event) error {
	var f fields
	if err := json.Unmarshal(e.Data, &f); err != nil {
		return err
	}
	sub := strings.ToLower(f.str("subaccount"))
	pid := f.pid("product_id")
	switch e.Name {
	case "deposit":
		// the chain credited first; mirror it (Development.md §7.1)
		amount, err := num(f["amount_x18"])
		if err != nil {
			return err
		}
		// a system deposit (the DAO funding the AMM or insurance) names the DAO as account_id:
		// it must not link the DAO's NEAR account to that subaccount
		if f.str("source") != "system" {
			if err := h.Sink.EnsureNearAccount(f.str("account_id"), sub); err != nil {
				return err
			}
		}
		if err := h.Sink.CreditBalance(sub, pid, amount); err != nil {
			return err
		}
		return h.Sink.RecordTransfer(e.Key, e.TxHash, sub, pid, amount, true)
	case "withdraw_failed":
		// the contract's callback re-credited the whole debit; the backend debited it when it
		// queued the withdrawal, so it re-credits too
		amount, err := num(f["amount"])
		if err != nil {
			return err
		}
		return h.Sink.CreditBalance(sub, pid, amount)
	case "withdraw_done":
		// the fee plus sub-unit dust moved to the fee account on-chain (D-7). A fee sweep
		// (subaccount = the fee account itself) carries fee 0.
		fee, err := num(f["fee"])
		if err != nil {
			return err
		}
		if fee.Sign() > 0 {
			if err := h.Sink.CreditBalance(strings.ToLower(h.FeeSubaccount), pid, fee); err != nil {
				return err
			}
		}
		amount, err := num(f["amount"])
		if err != nil {
			return err
		}
		return h.Sink.RecordTransfer(e.Key, e.TxHash, sub, pid, amount, false)
	case "session_key_revoked":
		return h.Sink.RevokeSessionKey(f.str("session_key"))
	case "fee_sweep_pending":
		// the DAO moved fees out: the fee account's ledger balance goes down now, and comes back
		// through withdraw_failed if the transfer fails
		amount, err := num(f["amount"])
		if err != nil {
			return err
		}
		return h.Sink.CreditBalance(strings.ToLower(h.FeeSubaccount), pid, new(big.Int).Neg(amount))
	}
	return nil
}

func (h *Handler) alert(msg string) {
	if h.Alert != nil {
		h.Alert(msg)
	}
}
