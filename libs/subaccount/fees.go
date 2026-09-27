package subaccount

import (
	"context"
	"fmt"
	"math/big"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/xredis"

	"github.com/redis/go-redis/v9"
)

// D-7 fee account (behavior-spec §4). Every match and liquidation credits it, so a plain
// read-modify-write would either serialise all matching on one lock or lose updates. Instead:
//   - AccrueFee appends the fee to a Redis list inside the caller's own MULTI pipeline, so the
//     fee is recorded atomically with the trade that produced it;
//   - FoldFeeAccruals moves queued fees into the fee subaccount's quote balance under that
//     subaccount's balance lock, writing the new balance and trimming the queue in one MULTI.
// Only the head of the list is trimmed, so fees appended while a fold runs are kept for the next.
// Readers that need the exact figure (the reconciler) call FoldFeeAccruals first.

func FeeAccrualKey() string { return "near-stocks:fee-accrual" }

// AccrueFee queues a positive fee on pipe (a TxPipeline).
func AccrueFee(pipe redis.Pipeliner, feex18 *big.Int) {
	if feex18 != nil && feex18.Sign() > 0 {
		pipe.RPush(context.Background(), FeeAccrualKey(), feex18.String())
	}
}

// FoldFeeAccruals credits every queued fee to TRADING_FEES_SUBACCOUNT_ID and returns the total.
func FoldFeeAccruals(rc *redis.Client, sb SubaccountBalance) (*big.Int, error) {
	ctx := context.Background()
	return xredis.WithRedisLock(xredis.GetBalanceRedisLockKey(contractUtils.TRADING_FEES_SUBACCOUNT_ID), func() (*big.Int, error) {
		vals, err := rc.LRange(ctx, FeeAccrualKey(), 0, -1).Result()
		if err != nil {
			return nil, err
		}
		total := big.NewInt(0)
		if len(vals) == 0 {
			return total, nil
		}
		for _, v := range vals {
			f, ok := new(big.Int).SetString(v, 10)
			if !ok {
				return nil, fmt.Errorf("bad fee accrual entry %q", v)
			}
			total.Add(total, f)
		}
		bal := sb.MustGetSubaccountBalancesFromIds([]string{contractUtils.TRADING_FEES_SUBACCOUNT_ID})[0]
		bal.UpdateSpotBalance(contractUtils.QUOTE_TOKEN_PRODUCT_ID, total)
		_, err = rc.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			if err := ReplaceBalanceInRedis(pipe, &bal); err != nil {
				return err
			}
			pipe.LTrim(ctx, FeeAccrualKey(), int64(len(vals)), -1)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return total, nil
	})
}
