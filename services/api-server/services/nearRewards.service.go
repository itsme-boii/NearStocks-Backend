package services

import (
	"fmt"
	"math/big"
	"strings"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/xclient"
)

// ReserveLogxRewards takes amountX18 LogX out of the rewards pool before a reward or airdrop claim is
// credited, matching the contract: with NEAR settlement claims are paid from the DAO-funded pool
// (LOGX_REWARDS_SUBACCOUNT_ID), so claimed LogX is always backed by tokens the contract holds. It
// fails, changing nothing, when the pool cannot cover the claim. The returned refund puts the amount
// back and must be called if the claim is not completed. Without NEAR settlement it is a no-op.
func ReserveLogxRewards(amountX18 *big.Int) (refund func(), err error) {
	noop := func() {}
	if !contractUtils.NearSettlement() {
		return noop, nil
	}
	pool := strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID)
	_, err = xredis.WithRedisLock("near-logx-rewards-pool", func() (*struct{}, error) {
		b := subaccount.NewSubaccountBalanceImpl().MustGetSubaccountBalancesFromIds([]string{pool})[0]
		have := new(big.Int)
		if sp, ok := b.SpotBalances[contractUtils.LOGX]; ok && sp.Balancex18 != nil {
			have = sp.Balancex18
		}
		if have.Cmp(amountX18) < 0 {
			if c := xclient.GetGlobalDiscordClient(); c != nil {
				c.SendWebhookMessage(fmt.Sprintf("NEAR: LogX rewards pool has %s, a claim needs %s: the DAO must top it up (ft_transfer_call msg {\"system\":\"rewards\"})", have, amountX18))
			}
			return nil, fmt.Errorf("LogX rewards pool exhausted")
		}
		ok, err := xclient.GlobalBalanceClient.UpdateTokenBalance(pool, contractUtils.LOGX, new(big.Int).Neg(amountX18).String())
		if err != nil || !ok {
			return nil, fmt.Errorf("debiting the rewards pool: %v", err)
		}
		return nil, nil
	})
	if err != nil {
		return noop, err
	}
	return func() {
		if ok, err := xclient.GlobalBalanceClient.UpdateTokenBalance(pool, contractUtils.LOGX, amountX18.String()); err != nil || !ok {
			xlog.Errorf("NEAR: refunding %s to the LogX rewards pool failed: %v (the reconciler will flag it)", amountX18, err)
		}
	}, nil
}
