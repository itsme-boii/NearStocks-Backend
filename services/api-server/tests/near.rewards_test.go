package tests

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github/eugenix-io/logx-inf-backend/contract/contractUtils"
	"github/eugenix-io/logx-inf-backend/libs/subaccount"
	"github/eugenix-io/logx-inf-backend/libs/subaccountTypes"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// With NEAR settlement, reward and airdrop claims are paid from the DAO-funded LogX rewards pool,
// exactly like the contract: a claim the pool cannot cover is refused and changes nothing.
func TestNearClaimsArePaidFromTheRewardsPool(t *testing.T) {
	bal := &fakeBalance{bal: map[string]*big.Int{}}
	bs := bal.server()
	defer bs.Close()
	os.Setenv("BALANCE_SERVER_URL", bs.URL)
	defer os.Unsetenv("BALANCE_SERVER_URL")
	xclient.InitBalanceClient()
	e18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	x := func(n int64) *big.Int { return new(big.Int).Mul(big.NewInt(n), e18) }
	pool := strings.ToLower(contractUtils.LOGX_REWARDS_SUBACCOUNT_ID)

	testutils.WithSetupMockRedis(t, func() {
		// off NEAR: claims are credited as before
		refund, err := services.ReserveLogxRewards(x(1_000_000))
		require.NoError(t, err)
		refund()
		assert.Equal(t, "0", bal.get(pool).String(), "no pool involved without NEAR settlement")

		os.Setenv("NEAR_SETTLEMENT", "1")
		defer os.Unsetenv("NEAR_SETTLEMENT")
		setPool := func(v *big.Int) {
			pipe := xredis.GetRedisClient().TxPipeline()
			b := subaccountTypes.SubaccountBalances{SubaccountId: pool, SpotBalances: map[uint32]subaccountTypes.SpotBalance{
				contractUtils.LOGX: {ProductId: contractUtils.LOGX, Balancex18: v, Lockedx18: big.NewInt(0)}}}
			require.NoError(t, subaccount.ReplaceBalanceInRedis(pipe, &b))
			_, err := pipe.Exec(t.Context())
			require.NoError(t, err)
		}
		setPool(x(10))

		_, err = services.ReserveLogxRewards(x(11))
		assert.ErrorContains(t, err, "exhausted", "more than the pool holds is refused")
		assert.Equal(t, "0", bal.get(pool).String(), "a refused claim debits nothing")

		refund, err = services.ReserveLogxRewards(x(4))
		require.NoError(t, err)
		assert.Equal(t, new(big.Int).Neg(x(4)).String(), bal.get(pool).String(), "the pool is debited by the claim")
		refund()
		assert.Equal(t, "0", bal.get(pool).String(), "a claim that fails later is refunded")
	})
}
