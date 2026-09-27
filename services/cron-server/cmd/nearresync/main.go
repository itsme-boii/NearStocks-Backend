// nearresync is the ops tool that repairs a paused subaccount's backend ledger from near-stocks.
// The contract is the settlement truth: when a transaction was refused and parked (the backend had
// already applied it) or the reconciler found a difference, the subaccount is paused. Once nothing
// of it is pending or in flight, its spot and perp balances are copied from the contract.
//
//	nearresync [-unpause] 0x<subaccount hex>...      (with the service env loaded)
//
// It refuses a subaccount that is not paused or still has unsettled rows, takes the balance-server's
// lock for the write, keeps open-order locks, resets the backend nonce to the chain's (a parked
// signed transaction consumed one the chain never took), records the correction and alerts.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/cron-server/nearbatch"
	"github/eugenix-io/logx-inf-backend/services/cron-server/nearreconcile"
	"github/eugenix-io/logx-inf-backend/xclient"
)

type nop struct{}

func main() {
	unpause := flag.Bool("unpause", false, "remove the subaccount from the sequencer pause list afterwards")
	flag.Parse()
	if flag.NArg() == 0 {
		log.Fatal("usage: nearresync [-unpause] 0x<subaccount hex>...")
	}
	contractUtils.Init()
	db.Init()
	xclient.InitDiscordClient()
	xclient.InitBalanceClient()
	if !contractUtils.NearSettlement() {
		log.Fatal("NEAR_SETTLEMENT is off: nothing to resync from")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rpc := nearchain.NewClient(os.Getenv("NEAR_RPC_URL"))
	failed := false
	for _, sub := range flag.Args() {
		if err := resync(ctx, rpc, strings.ToLower(sub), *unpause); err != nil {
			fmt.Printf("%s: NOT resynced: %v\n", sub, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

func resync(ctx context.Context, rpc *nearchain.Client, sub string, unpause bool) error {
	r := xredis.GetRedisClient()
	pauseKey, field := xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(sub)
	paused, err := r.HExists(ctx, pauseKey, field).Result()
	if err != nil {
		return err
	}
	if !paused {
		return fmt.Errorf("not paused: pause it first so nothing new is sequenced while its ledger is rewritten")
	}
	if busy, err := (&db.BatchDB{}).NearHasUnsettled(sub); err != nil || busy {
		return fmt.Errorf("has pending or in-flight rows (the chain is not final for it yet): %v", err)
	}
	raw, err := rpc.CallView(ctx, contractUtils.NearStocksAccount(), "get_subaccount", map[string]any{"subaccount": sub})
	if err != nil {
		return err
	}
	chain, err := nearreconcile.FromChain(raw)
	if err != nil {
		return err
	}
	var diff []string
	_, err = xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(sub), func() (*nop, error) {
		before := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{sub})[0]
		diff = nearreconcile.Diff(chain, nearreconcile.FromBackend(before))
		if len(diff) == 0 {
			return nil, nil
		}
		fixed := nearreconcile.FromChainOnto(before, chain)
		pipe := r.TxPipeline()
		if err := subaccount.ReplaceBalanceInRedis(pipe, &fixed); err != nil {
			return nil, err
		}
		_, err := pipe.Exec(ctx)
		return nil, err
	})
	if err != nil {
		return err
	}
	after := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{sub})[0]
	if d := nearreconcile.Diff(chain, nearreconcile.FromBackend(after)); len(d) != 0 {
		return fmt.Errorf("still differs after the write: %v", d)
	}
	// the nonce: a parked signed transaction consumed a backend nonce the chain never took, so every
	// later request of the subaccount would be refused ("bad nonce") until they agree again
	nraw, err := rpc.CallView(ctx, contractUtils.NearStocksAccount(), "get_nonce", map[string]any{"subaccount": sub})
	if err != nil {
		return err
	}
	chainNonce := strings.TrimSpace(string(nraw))
	_, err = xredis.WithRedisLock(xredis.GetNonceLockKey(sub), func() (*nop, error) {
		cur, err := r.Get(ctx, xredis.GetNonceKey(sub)).Result()
		if err != nil && err.Error() != "redis: nil" {
			return nil, err
		}
		if cur != chainNonce {
			diff = append(diff, fmt.Sprintf("nonce: chain %s backend %s", chainNonce, cur))
			return nil, r.Set(ctx, xredis.GetNonceKey(sub), chainNonce, 0).Err()
		}
		return nil, nil
	})
	if err != nil {
		return err
	}
	// pre-market / synthetic token balances: the balance-server applies deltas
	pdiff, err := resyncPools(ctx, rpc, sub)
	if err != nil {
		return err
	}
	diff = append(diff, pdiff...)
	if len(diff) > 0 {
		detail := "resynced from chain: " + strings.Join(diff, "; ")
		if err := (&db.BatchDB{}).NearRecordDiff(sub, detail); err != nil {
			return err
		}
		if c := xclient.GetGlobalDiscordClient(); c != nil {
			c.SendWebhookMessage(fmt.Sprintf("NEAR RESYNC: %s %s", sub, detail))
		}
	}
	fmt.Printf("%s: backend == chain (%d fields corrected)\n", sub, len(diff))
	if unpause {
		if err := r.HDel(ctx, pauseKey, field).Err(); err != nil {
			return err
		}
		fmt.Printf("%s: unpaused\n", sub)
	}
	return nil
}

func resyncPools(ctx context.Context, rpc *nearchain.Client, sub string) ([]string, error) {
	seq := &nearchain.Sequencer{RPC: rpc, Contract: contractUtils.NearStocksAccount()}
	chain, backend, err := nearbatch.PoolBalances(ctx, seq, sub)
	if err != nil {
		return nil, err
	}
	var out []string
	for k, c := range chain {
		b := backend[k]
		if b == nil {
			b = new(big.Int)
		}
		delta := new(big.Int).Sub(c, b)
		if delta.Sign() == 0 {
			continue
		}
		var ok bool
		if k[0] == 1 {
			ok, err = xclient.GlobalBalanceClient.UpdatePreMarketBalance(sub, k[1], delta.String())
		} else {
			ok, err = xclient.GlobalBalanceClient.UpdateSyntheticSpotBalance(sub, k[1], delta.String())
		}
		if err != nil || !ok {
			return out, fmt.Errorf("pool %d/%d: %v", k[0], k[1], err)
		}
		out = append(out, fmt.Sprintf("pool %d/%d: chain %s backend %s", k[0], k[1], c, b))
	}
	return out, nil
}
