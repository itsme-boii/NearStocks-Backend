package tests

import (
	"sync"
	"sync/atomic"
	"testing"

	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/testutils"

	"github.com/stretchr/testify/assert"
)

func TestNearAccountGetOrCreate(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	n := &db.NearAccountDB{}

	a, err := n.GetOrCreate("alice.near", 1, "1_0xfB734EC1D441bFAee35EdDb6DDdB0774b8F0ec67_0", "0xfB734EC1D441bFAee35EdDb6DDdB0774b8F0ec67")
	assert.NoError(t, err)
	b, err := n.GetOrCreate("alice.near", 1, "ignored-on-second-call", "x")
	assert.NoError(t, err)
	assert.Equal(t, a.ID, b.ID)
	assert.Equal(t, "1_0xfB734EC1D441bFAee35EdDb6DDdB0774b8F0ec67_0", b.SubaccountId)
	assert.Equal(t, "alice.near", n.GetBySubaccountId(a.SubaccountId).AccountId)
	assert.Nil(t, n.GetByAccountId("alice.near", 2))
}

// The credit guard must let exactly one of many concurrent callers through.
func TestIntentsTransferClaimCreditExactlyOnce(t *testing.T) {
	testutils.SetupDBEnv(t)
	db.Init()
	it := &db.IntentsTransferDB{}

	row := &db.IntentsTransferTable{Key: "1click:dep1", Direction: "deposit", Kind: "1click", SubaccountId: "s", Status: "PENDING_DEPOSIT"}
	assert.NoError(t, it.Create(row))
	assert.ErrorIs(t, it.Create(&db.IntentsTransferTable{Key: "1click:dep1"}), db.ErrIntentsTransferExists)

	var wins atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := it.ClaimCredit("1click:dep1", "5000000", "hash")
			assert.NoError(t, err)
			if ok {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(1), wins.Load())

	got := it.GetByKey("1click:dep1")
	assert.NotNil(t, got.CreditedAt)
	assert.Equal(t, "SUCCESS_CREDITED", got.Status)
	assert.Equal(t, "5000000", got.AmountUSDC)

	// Status updates can't touch a credited row, and failure marking keeps credited_at.
	assert.NoError(t, it.UpdateStatus("1click:dep1", "PROCESSING", ""))
	assert.Equal(t, "SUCCESS_CREDITED", it.GetByKey("1click:dep1").Status)
	assert.NoError(t, it.MarkCreditFailed("1click:dep1", "balance server down"))
	assert.NotNil(t, it.GetByKey("1click:dep1").CreditedAt)
	ok, _ := it.ClaimCredit("1click:dep1", "5000000", "hash")
	assert.False(t, ok, "a failed credit must never be re-claimable automatically")
}
