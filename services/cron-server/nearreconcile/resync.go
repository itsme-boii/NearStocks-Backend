package nearreconcile

import (
	"math/big"

	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
)

// FromChainOnto returns the backend balances with every spot and perp value replaced by the
// chain's (the contract is the settlement truth). Off-chain state the contract does not hold —
// locked amounts of open orders — is kept. Used by nearresync to repair a paused subaccount whose
// queue is empty, e.g. after a refused fill the backend had already applied.
func FromChainOnto(b subaccountTypes.SubaccountBalances, chain Snapshot) subaccountTypes.SubaccountBalances {
	out := subaccountTypes.SubaccountBalances{
		SubaccountId: b.SubaccountId,
		SpotBalances: map[uint32]subaccountTypes.SpotBalance{},
		PerpBalances: map[uint32]subaccountTypes.PerpBalance{},
	}
	for pid, v := range chain.Spots {
		locked := new(big.Int)
		if old, ok := b.SpotBalances[pid]; ok && old.Lockedx18 != nil {
			locked = new(big.Int).Set(old.Lockedx18)
		}
		out.SpotBalances[pid] = subaccountTypes.SpotBalance{ProductId: pid, Balancex18: new(big.Int).Set(v), Lockedx18: locked}
	}
	for pid, old := range b.SpotBalances { // a spot the chain no longer has, but with an open-order lock
		if _, ok := out.SpotBalances[pid]; !ok && old.Lockedx18 != nil && old.Lockedx18.Sign() != 0 {
			out.SpotBalances[pid] = subaccountTypes.SpotBalance{ProductId: pid, Balancex18: new(big.Int), Lockedx18: new(big.Int).Set(old.Lockedx18)}
		}
	}
	for pid, p := range chain.Perps {
		out.PerpBalances[pid] = subaccountTypes.PerpBalance{ProductId: pid, Amountx18: new(big.Int).Set(p[0]), VQuoteBalancex18: new(big.Int).Set(p[1]), LastCumFundingRatex18: new(big.Int).Set(p[2])}
	}
	return out
}
