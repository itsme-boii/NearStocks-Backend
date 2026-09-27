// Package nearindexer mirrors near-stocks.near into the backend (Development.md §8.4). The chain
// credits deposits first and the indexer mirrors them into the balance-server; withdrawal outcomes
// and fee sweeps are mirrored the same way. It follows finalized blocks from neardata, reads the
// NEP-297 events of successful receipts executed by our contract, and applies each event exactly
// once (db.NearChainEventDB claims).
package nearindexer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Block is the subset of a neardata block the indexer reads.
type Block struct {
	Block struct {
		Header struct {
			Height uint64 `json:"height"`
			Hash   string `json:"hash"`
		} `json:"header"`
	} `json:"block"`
	Shards []struct {
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
	} `json:"shards"`
}

// Event is one near-stocks NEP-297 event.
type Event struct {
	Key       string // receipt id + log index: unique forever
	Block     uint64
	ReceiptId string
	TxHash    string
	Name      string
	Data      json.RawMessage // the first (only) element of "data"
}

// Extract returns our contract's events from successful receipts, in execution order. A failed
// receipt still carries its logs, but its state changes were reverted, so its events are skipped.
func Extract(b *Block, contract string) ([]Event, error) {
	var out []Event
	for _, sh := range b.Shards {
		for _, r := range sh.ReceiptExecutionOutcomes {
			o := r.ExecutionOutcome.Outcome
			if o.ExecutorId != contract {
				continue
			}
			_, v := o.Status["SuccessValue"]
			_, rid := o.Status["SuccessReceiptId"]
			if !v && !rid {
				continue
			}
			for i, l := range o.Logs {
				body, ok := strings.CutPrefix(l, "EVENT_JSON:")
				if !ok {
					continue
				}
				var e struct {
					Standard string            `json:"standard"`
					Event    string            `json:"event"`
					Data     []json.RawMessage `json:"data"`
				}
				if err := json.Unmarshal([]byte(body), &e); err != nil {
					return nil, fmt.Errorf("block %d receipt %s log %d: %w", b.Block.Header.Height, r.ExecutionOutcome.Id, i, err)
				}
				if e.Standard != "near-stocks" || len(e.Data) == 0 {
					continue
				}
				out = append(out, Event{
					Key:       fmt.Sprintf("near:%s:%d", r.ExecutionOutcome.Id, i),
					Block:     b.Block.Header.Height,
					ReceiptId: r.ExecutionOutcome.Id,
					TxHash:    r.TxHash,
					Name:      e.Event,
					Data:      e.Data[0],
				})
			}
		}
	}
	return out, nil
}
