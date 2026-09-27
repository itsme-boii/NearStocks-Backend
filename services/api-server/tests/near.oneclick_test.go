package tests

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/nearchain"
	"github/eugenix-io/logx-inf-backend/services/api-server/services"
	"github/eugenix-io/logx-inf-backend/testutils"
	"github/eugenix-io/logx-inf-backend/xclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOneClickDepositSync(t *testing.T) {
	testutils.SetMainnetEnv()
	defer testutils.ResetEnv()
	testutils.SetupDBEnv(t)
	db.Init()

	var mu sync.Mutex
	statusByAddr := map[string]string{"dep-ok": nearchain.StatusProcessing, "dep-evil": nearchain.StatusSuccess}
	oc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"), "API key header must be sent")
		addr := r.URL.Query().Get("depositAddress")
		mu.Lock()
		st := statusByAddr[addr]
		mu.Unlock()
		recipient := testTreasury
		if addr == "dep-evil" {
			recipient = "attacker.near" // a quote that doesn't pay our treasury must never credit
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"correlationId": "c", "status": st, "updatedAt": "now",
			"quoteResponse": map[string]any{"quoteRequest": map[string]any{
				"recipient": recipient, "recipientType": "DESTINATION_CHAIN", "destinationAsset": "nep141:" + nearchain.USDCMainnet,
			}, "quote": map[string]any{"depositAddress": addr}},
			"swapDetails": map[string]any{"intentHashes": []string{}, "nearTxHashes": []string{}, "originChainTxHashes": []any{},
				"destinationChainTxHashes": []map[string]string{{"hash": "neartxhash", "explorerUrl": "x"}}, "amountOut": "7250000"},
		})
	}))
	defer oc.Close()
	bal := &fakeBalance{bal: map[string]*big.Int{}}
	bs := bal.server()
	defer bs.Close()
	for k, v := range map[string]string{"NEAR_NETWORK": "mainnet", "NEAR_TREASURY_ACCOUNT": testTreasury, "NEAR_RPC_URL": "http://unused",
		"ONECLICK_BASE_URL": oc.URL, "ONECLICK_API_KEY": "test-key", "BALANCE_SERVER_URL": bs.URL} {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}
	services.SetNearConfigForTest(nil)
	defer services.SetNearConfigForTest(nil)
	xclient.InitBalanceClient()

	addr20, _ := cutils.NearAccountToAddr20(testUser)
	subId := cutils.CreateSubaccountId(2, addr20, 1)
	subHex, _ := cutils.SubaccountIdToHex(subId)
	it := &db.IntentsTransferDB{}
	for _, a := range []string{"dep-ok", "dep-evil"} {
		require.NoError(t, it.Create(&db.IntentsTransferTable{Key: "1click:" + a + ":", Direction: "deposit", Kind: "1click", SubaccountId: subId, DepositAddress: a, Status: nearchain.StatusPendingDeposit}))
	}

	testutils.WithSetupMockRedis(t, func() {
		ctx := context.Background()
		st, err := services.SyncOneClickDeposit(ctx, it.GetByKey("1click:dep-ok:"))
		require.NoError(t, err)
		assert.Equal(t, nearchain.StatusProcessing, st)
		assert.Equal(t, "0", bal.get(subHex).String(), "nothing credited while processing")

		mu.Lock()
		statusByAddr["dep-ok"] = nearchain.StatusSuccess
		mu.Unlock()
		for i := 0; i < 3; i++ { // repeated polls credit once
			st, err = services.SyncOneClickDeposit(ctx, it.GetByKey("1click:dep-ok:"))
			require.NoError(t, err)
			assert.Equal(t, "SUCCESS_CREDITED", st)
		}
		assert.Equal(t, nearchain.ScaleUSDCToX18(big.NewInt(7_250_000)).String(), bal.get(subHex).String())
		row := it.GetByKey("1click:dep-ok:")
		assert.Equal(t, "7250000", row.AmountUSDC)
		assert.Equal(t, "neartxhash", row.NearTxHash)

		st, err = services.SyncOneClickDeposit(ctx, it.GetByKey("1click:dep-evil:"))
		assert.Error(t, err)
		assert.Equal(t, "MISMATCH", st)
		assert.Nil(t, it.GetByKey("1click:dep-evil:").CreditedAt)
		assert.Equal(t, nearchain.ScaleUSDCToX18(big.NewInt(7_250_000)).String(), bal.get(subHex).String(), "mismatched quote not credited")
	})
}
