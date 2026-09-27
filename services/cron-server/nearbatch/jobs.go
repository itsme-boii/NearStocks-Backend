package nearbatch

// Production wiring for NEAR settlement (NEAR_SETTLEMENT=1): the batch sender, the price tick,
// the reconciler and the replay-protection prune job. All three use the sequencer key (local or AWS KMS).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract"
	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/appstate"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/marketutils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/transaction"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/cron-server/nearreconcile"
	"github/eugenix-io/logx-inf-backend/xclient"
)

// redisPauser is the sequencer pause list the appchain batcher used (SUBACCOUNT_PAUSE_SEQUENCER).
type redisPauser struct{}

func (redisPauser) Paused() []string {
	keys, err := xredis.GetRedisClient().HKeys(context.Background(), xredis.GetSubacountPauseSequencerKey()).Result()
	if err != nil {
		xlog.Errorf("NEAR batcher: reading pause list: %v", err)
		return nil
	}
	return keys
}

func (redisPauser) Pause(sub, reason string) {
	err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(strings.ToLower(sub)), 1).Err()
	if err != nil {
		xlog.Errorf("NEAR batcher: pausing %s (%s): %v", sub, reason, err)
	}
}

func discord(msg string) {
	xlog.Errorf("%s", msg)
	if c := xclient.GetGlobalDiscordClient(); c != nil {
		c.SendWebhookMessage(msg)
	}
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return time.Duration(v) * time.Millisecond
	}
	return def
}

// Start runs the NEAR jobs until ctx is cancelled. It returns an error only if the sequencer
// cannot be configured.
func Start(ctx context.Context) error {
	seq, err := nearchain.SequencerFromEnv(ctx, contractUtils.NearStocksAccount())
	if err != nil {
		return err
	}
	b := &Batcher{Store: &db.BatchDB{}, Chain: seq, Pauser: redisPauser{}, Alert: discord}
	go loop(ctx, "batch", envDuration("NEAR_BATCH_INTERVAL_MS", 2*time.Second), func() error {
		n, err := b.RunOnce(ctx)
		if n > 0 {
			xlog.Infof("NEAR batcher: %d transactions landed", n)
		}
		if errors.Is(err, nearchain.ErrInconsistent) {
			return err // stops the loop: a human must look (already alerted)
		}
		if err != nil {
			xlog.Errorf("NEAR batcher: %v", err)
		}
		return nil
	})
	go loop(ctx, "price tick", envDuration("NEAR_PRICE_TICK_MS", 30*time.Second), func() error {
		if err := priceTick(ctx, seq); err != nil {
			xlog.Errorf("NEAR price tick: %v", err)
		}
		return nil
	})
	every := envDuration("NEAR_RECONCILE_INTERVAL_MS", 5*time.Minute)
	since := time.Now().Add(-every)
	go loop(ctx, "reconcile", every, func() error {
		started := time.Now()
		res, err := nearreconcile.Run(ctx, reconcileSources(seq), since.Add(-time.Minute))
		if err != nil {
			xlog.Errorf("NEAR reconcile: %v", err)
			return nil // retried from the same `since` next time
		}
		since = started
		xlog.Infof("NEAR reconcile: %d checked, %d skipped (queued), %d mismatched", res.Checked, res.Skipped, len(res.Mismatched))
		return nil
	})
	go loop(ctx, "balances", envDuration("NEAR_BALANCE_CHECK_MS", 10*time.Minute), func() error {
		if err := checkBalances(ctx, seq); err != nil {
			xlog.Errorf("NEAR balance check: %v", err)
		}
		return nil
	})
	go loop(ctx, "prune", envDuration("NEAR_PRUNE_INTERVAL_MS", 10*time.Minute), func() error {
		if err := prune(ctx, seq); err != nil {
			xlog.Errorf("NEAR prune: %v", err)
		}
		return nil
	})
	return nil
}

func loop(ctx context.Context, name string, every time.Duration, f func() error) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := f(); err != nil {
				discord(fmt.Sprintf("NEAR %s job stopped: %v", name, err))
				return
			}
		}
	}
}

func reconcileSources(seq *nearchain.Sequencer) nearreconcile.Sources {
	batches := &db.BatchDB{}
	return nearreconcile.Sources{
		RecentlyLanded: batches.NearRecentlyLanded,
		HasQueued:      batches.NearHasQueued,
		Chain: func(ctx context.Context, sub string) (nearreconcile.Snapshot, error) {
			raw, err := seq.RPC.CallView(ctx, seq.Contract, "get_subaccount", map[string]any{"subaccount": strings.ToLower(sub)})
			if err != nil {
				return nearreconcile.Snapshot{}, err
			}
			return nearreconcile.FromChain(raw)
		},
		Backend: func(subs []string) ([]subaccountTypes.SubaccountBalances, error) {
			return subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds(subs), nil
		},
		Pools: func(ctx context.Context, sub string) (map[[2]uint32]*big.Int, map[[2]uint32]*big.Int, error) {
			return PoolBalances(ctx, seq, strings.ToLower(sub))
		},
		Pause:  redisPauser{}.Pause,
		Record: batches.NearRecordDiff,
		Alert:  discord,
	}
}

// PoolBalances reads a subaccount's pre-market (kind 1) and synthetic spot (kind 2) balances for
// every pool product listed on-chain, from the contract and from the balance-server.
func PoolBalances(ctx context.Context, seq *nearchain.Sequencer, sub string) (map[[2]uint32]*big.Int, map[[2]uint32]*big.Int, error) {
	raw, err := seq.RPC.CallView(ctx, seq.Contract, "get_side_products", map[string]any{})
	if err != nil {
		return nil, nil, err
	}
	var listed []struct {
		Kind      uint32 `json:"kind"`
		ProductId uint32 `json:"product_id"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return nil, nil, err
	}
	chain, backend := map[[2]uint32]*big.Int{}, map[[2]uint32]*big.Int{}
	var pre, syn map[uint32]string
	for _, p := range listed {
		if p.Kind != 1 && p.Kind != 2 {
			continue // options have no token balance
		}
		raw, err := seq.RPC.CallView(ctx, seq.Contract, "get_pool_balance", map[string]any{"kind": p.Kind, "subaccount": sub, "product_id": p.ProductId})
		if err != nil {
			return nil, nil, err
		}
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return nil, nil, err
		}
		k := [2]uint32{p.Kind, p.ProductId}
		if chain[k], _ = new(big.Int).SetString(v, 10); chain[k] == nil {
			return nil, nil, fmt.Errorf("get_pool_balance: %q", v)
		}
		if p.Kind == 1 && pre == nil {
			if pre, err = xclient.GlobalBalanceClient.GetPreMarketBalances(sub); err != nil {
				return nil, nil, err
			}
		}
		if p.Kind == 2 && syn == nil {
			if syn, err = xclient.GlobalBalanceClient.GetSyntheticSpotBalances(sub); err != nil {
				return nil, nil, err
			}
		}
		src := pre
		if p.Kind == 2 {
			src = syn
		}
		if b, ok := new(big.Int).SetString(src[p.ProductId], 10); ok {
			backend[k] = b
		}
	}
	return chain, backend, nil
}

// onChainProducts lists the product ids configured in the contract (unknown ids make a tick panic).
func onChainProducts(ctx context.Context, seq *nearchain.Sequencer) (map[uint32]bool, error) {
	raw, err := seq.RPC.CallView(ctx, seq.Contract, "get_products", map[string]any{})
	if err != nil {
		return nil, err
	}
	var v struct {
		Spots []struct {
			ProductId uint32 `json:"product_id"`
		} `json:"spots"`
		Perps []struct {
			ProductId uint32 `json:"product_id"`
		} `json:"perps"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	out := map[uint32]bool{}
	for _, p := range append(v.Spots, v.Perps...) {
		out[p.ProductId] = true
	}
	return out, nil
}

// priceTick queues a price-only PERPTICK with the oracle price of every on-chain product.
func priceTick(ctx context.Context, seq *nearchain.Sequencer) error {
	products, err := onChainProducts(ctx, seq)
	if err != nil {
		return err
	}
	oracle, err := appstate.NewAppState().GetAllOraclePrices(time.Second)
	if err != nil {
		return err
	}
	prices := map[uint32]*big.Int{}
	var missing []string
	for pid := range products {
		sym, ok := marketutils.GetBaseSymbolForProduct(pid)
		if !ok {
			continue
		}
		p, ok := oracle[sym]
		if !ok || p.Pricex18 == nil || p.Pricex18.Sign() <= 0 {
			missing = append(missing, sym)
			continue
		}
		prices[pid] = p.Pricex18
	}
	if len(missing) > 0 {
		discord(fmt.Sprintf("NEAR price tick: no oracle price for %v (their on-chain price will go stale)", missing))
	}
	if len(prices) == 0 {
		return nil
	}
	counter, err := transaction.IncrementCounter(1)
	if err != nil {
		return err
	}
	return contract.GlobalContracts.EndpointContract.PriceTick(prices, time.Now().Unix(), counter)
}

// prune removes replay-protection entries of orders that expired more than an hour ago.
func prune(ctx context.Context, seq *nearchain.Sequencer) error {
	cutoff := uint64(time.Now().Add(-time.Hour).UnixMilli())
	digests, err := (db.NearOrderDigestDB{}).ExpiredUnpruned(cutoff, 200)
	if err != nil || len(digests) == 0 {
		return err
	}
	res := seq.PruneFilled(ctx, digests)
	if res.Outcome != nearchain.Landed {
		return fmt.Errorf("prune_filled: %s", res.Failure)
	}
	return (db.NearOrderDigestDB{}).MarkPruned(digests)
}

// yoctoNEAR per byte of state (Development.md N4: 1 NEAR per 100 KB).
var storageBytePrice = new(big.Int).Exp(big.NewInt(10), big.NewInt(19), nil)

func near(n int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(n), new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil))
}

// BalanceAlerts returns warnings for the sequencer's gas balance and the contract's storage
// headroom (the NEAR replacement of the appchain relayer balance check).
func BalanceAlerts(sequencer, contract *nearchain.AccountView, minSequencer, minHeadroom *big.Int) []string {
	var out []string
	if s, ok := new(big.Int).SetString(sequencer.Amount, 10); ok && s.Cmp(minSequencer) < 0 {
		out = append(out, fmt.Sprintf("sequencer balance %s yoctoNEAR is below %s: batches will stop when it runs out of gas", s, minSequencer))
	}
	amount, ok := new(big.Int).SetString(contract.Amount, 10)
	if ok {
		locked := new(big.Int).Mul(big.NewInt(int64(contract.StorageUsage)), storageBytePrice)
		if free := new(big.Int).Sub(amount, locked); free.Cmp(minHeadroom) < 0 {
			out = append(out, fmt.Sprintf("contract storage headroom %s yoctoNEAR is below %s (state %d bytes): new subaccounts and orders will fail", free, minHeadroom, contract.StorageUsage))
		}
	}
	return out
}

func checkBalances(ctx context.Context, seq *nearchain.Sequencer) error {
	s, err := seq.RPC.ViewAccount(ctx, seq.SignerId)
	if err != nil {
		return err
	}
	c, err := seq.RPC.ViewAccount(ctx, seq.Contract)
	if err != nil {
		return err
	}
	for _, a := range BalanceAlerts(s, c, near(5), near(20)) {
		discord("NEAR BALANCE: " + a)
	}
	return nil
}
