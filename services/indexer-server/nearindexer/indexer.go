package nearindexer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github/eugenix-io/logx-inf-backend/nearchain"
)

// Source serves finalized blocks. Block returns (nil, nil) for a height with no block (skipped).
type Source interface {
	FinalHeight(ctx context.Context) (uint64, error)
	Block(ctx context.Context, height uint64) (*Block, error)
}

// Checkpoint stores the last fully processed height.
type Checkpoint interface {
	Last() (uint64, bool, error)
	Save(height uint64) error
}

type Indexer struct {
	Source     Source
	Checkpoint Checkpoint
	Handler    *Handler
	Contract   string
	// StartHeight is used when there is no checkpoint yet (the contract's deployment block).
	StartHeight uint64
	// MaxBlocks bounds one Step (default 100).
	MaxBlocks int
	// Parallel is how many blocks are fetched at once (default 8). Fetching is latency-bound: one
	// at a time (~0.6 s each) barely outpaces the chain, so catching up after downtime took hours.
	// Blocks are still applied and checkpointed strictly in order.
	Parallel int
}

// Step processes finalized blocks after the checkpoint. A block is checkpointed only after all of
// its events were applied, so a crash replays at most that block (and claims make replays no-ops).
func (x *Indexer) Step(ctx context.Context) (int, error) {
	last, ok, err := x.Checkpoint.Last()
	if err != nil {
		return 0, err
	}
	next := x.StartHeight
	if ok {
		next = last + 1
	}
	final, err := x.Source.FinalHeight(ctx)
	if err != nil {
		return 0, err
	}
	max := x.MaxBlocks
	if max <= 0 {
		max = 100
	}
	par := x.Parallel
	if par <= 0 {
		par = 8
	}
	type fetched struct {
		b   *Block
		err error
	}
	done := 0
	var window []chan fetched
	src := x.Source // prefetches may outlive an early return: they must not read x
	fetch := func(h uint64) chan fetched {
		ch := make(chan fetched, 1) // buffered: an abandoned prefetch never blocks
		go func() {
			b, err := src.Block(ctx, h)
			ch <- fetched{b, err}
		}()
		return ch
	}
	end := next + uint64(max) - 1
	if end > final {
		end = final
	}
	nextFetch := next
	for ; nextFetch <= end && len(window) < par; nextFetch++ {
		window = append(window, fetch(nextFetch))
	}
	for h := next; h <= end; h++ {
		r := <-window[0]
		window = window[1:]
		if nextFetch <= end {
			window = append(window, fetch(nextFetch))
			nextFetch++
		}
		b, err := r.b, r.err
		if err != nil {
			return done, fmt.Errorf("block %d: %w", h, err)
		}
		if b != nil {
			events, err := Extract(b, x.Contract)
			if err != nil {
				return done, err
			}
			for _, e := range events {
				if err := x.Handler.Apply(e); err != nil {
					return done, fmt.Errorf("block %d event %s: %w", h, e.Key, err)
				}
			}
		}
		if err := x.Checkpoint.Save(h); err != nil {
			return done, err
		}
		done++
	}
	return done, nil
}

// Neardata reads blocks from https://{mainnet,testnet}.neardata.xyz (N16).
type Neardata struct {
	BaseURL string
	// APIKey (optional, FastNEAR subscription) lifts the anonymous rate limit, which otherwise
	// throttles catch-up after downtime to a few blocks per second.
	APIKey string
	HTTP   *http.Client
}

func (n *Neardata) get(ctx context.Context, path string) ([]byte, error) {
	c := n.HTTP
	if c == nil {
		c = nearchain.NewHTTPClient(30 * time.Second)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, n.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	if n.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+n.APIKey)
	}
	resp, err := c.Do(req) // neardata redirects to its storage: the default client follows it
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("neardata %s: %s", path, resp.Status)
	}
	return body, nil
}

func (n *Neardata) FinalHeight(ctx context.Context) (uint64, error) {
	body, err := n.get(ctx, "/v0/last_block/final")
	if err != nil {
		return 0, err
	}
	var b Block
	if err := json.Unmarshal(body, &b); err != nil {
		return 0, err
	}
	if b.Block.Header.Height == 0 {
		return 0, errors.New("neardata: no final height")
	}
	return b.Block.Header.Height, nil
}

func (n *Neardata) Block(ctx context.Context, height uint64) (*Block, error) {
	body, err := n.get(ctx, "/v0/block/"+strconv.FormatUint(height, 10))
	if err != nil {
		return nil, err
	}
	if string(body) == "null" {
		return nil, nil // no block at this height
	}
	var b Block
	if err := json.Unmarshal(body, &b); err != nil {
		return nil, err
	}
	return &b, nil
}
