package tests

import (
	"context"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/xredis"
	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSubAccountPaused(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		subaccountId := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"
		// Check if subaccount is paused
		paused := cutils.IsSubAccountPaused(subaccountId)
		assert.False(t, paused, "Subaccount should not be paused")

		err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetSubacountPauseSequencerKey(), xredis.GetSubaccountLvlSequencerField(subaccountId), 1).Err()
		assert.NoError(t, err, "Error in setting subaccount paused")

		// Check if subaccount is paused
		paused = cutils.IsSubAccountPaused(subaccountId)
		assert.True(t, paused, "Subaccount should be paused")
	})
}

func TestIsWhitelistedSubaccount(t *testing.T) {
	testutils.WithSetupMockRedis(t, func() {
		subaccountId := "0x0000000000017f9082df4d0fbfe668407b9b2066dd3ac06c9d80000000000002"
		// Check if subaccount is whitelisted
		whitelisted := cutils.IsWhitelistedSubaccount(subaccountId)
		assert.False(t, whitelisted, "Subaccount should not be whitelisted")

		// Whitelist subaccount
		err := xredis.GetRedisClient().HSet(context.Background(), xredis.GetWhitelistHashKey(), xredis.GetWhitelistSubaccountField(subaccountId), 1).Err()
		assert.NoError(t, err, "Error in setting subaccount whitelisted")

		// Check if subaccount is whitelisted
		whitelisted = cutils.IsWhitelistedSubaccount(subaccountId)
		assert.True(t, whitelisted, "Subaccount should be whitelisted")
	})
}
