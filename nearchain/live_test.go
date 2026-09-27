package nearchain

import (
	"context"
	"os"
	"testing"
)

// Read-only checks against a real mainnet RPC. Run with NEAR_LIVE=1.
func TestLiveMainnetReadOnly(t *testing.T) {
	if os.Getenv("NEAR_LIVE") != "1" {
		t.Skip("set NEAR_LIVE=1 to run against mainnet")
	}
	c := NewClient("https://free.rpc.fastnear.com")
	ctx := context.Background()

	if _, err := c.FinalBlockHash(ctx); err != nil {
		t.Fatalf("block: %v", err)
	}
	bal, err := c.FtBalanceOf(ctx, USDCMainnet, "v2.ref-finance.near")
	if err != nil || bal.Sign() <= 0 {
		t.Fatalf("ft_balance_of: %v %v", bal, err)
	}
	reg, err := c.StorageRegistered(ctx, USDCMainnet, "v2.ref-finance.near")
	if err != nil || !reg {
		t.Fatalf("storage_balance_of: %v %v", reg, err)
	}
	min, err := c.StorageMinimum(ctx, USDCMainnet)
	if err != nil || min.Sign() <= 0 {
		t.Fatalf("storage_balance_bounds: %v %v", min, err)
	}
	t.Logf("USDC storage minimum: %s yoctoNEAR", min)

	tx, err := c.TxStatus(ctx, "4zvGftYD56iVPGgSMK9ccq3Aka41SWnd5FbAkoJzNQFz", "silentxml7180.near")
	if err != nil {
		t.Fatalf("tx: %v", err)
	}
	transfers := ftTransfers(tx, USDCMainnet)
	if len(transfers) == 0 || transfers[0].To != "v2.ref-finance.near" {
		t.Fatalf("expected USDC ft_transfer event, got %+v", transfers)
	}
	// It was an ft_transfer_call into a DEX, so it must NOT verify as a direct deposit.
	if _, err := VerifyDirectDeposit(tx, "silentxml7180.near", "v2.ref-finance.near", USDCMainnet); err == nil {
		t.Fatal("ft_transfer_call must be rejected")
	}
	if _, err := c.ViewAccessKey(ctx, "near", "ed25519:11111111111111111111111111111111"); err != ErrAccessKeyNotFound {
		t.Fatalf("missing key: %v", err)
	}
}
