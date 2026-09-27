package tests

import (
	"encoding/json"
	"testing"

	"github/eugenix-io/logx-inf-backend/services/api-server/services"

	"github.com/stretchr/testify/assert"
)

// With NEAR settlement 1Click delivers to the contract with an ft_on_transfer msg naming the
// user's trading subaccount (§7.2); in Phase 2 it delivers to the plain treasury with no msg.
func TestDepositRecipientSwitch(t *testing.T) {
	c := &services.NearConfig{Treasury: "treasury.near", ContractAccount: "near-stocks.near"}
	assert.Equal(t, "treasury.near", c.DepositRecipient())
	assert.Equal(t, "", services.DepositRecipientMsg("alice.near"))

	t.Setenv("NEAR_SETTLEMENT", "1")
	assert.Equal(t, "near-stocks.near", c.DepositRecipient())
	var msg map[string]any
	assert.NoError(t, json.Unmarshal([]byte(services.DepositRecipientMsg("alice.near")), &msg))
	assert.Equal(t, map[string]any{"account_id": "alice.near", "subaccount_number": float64(1), "source": "1click"}, msg)
}
