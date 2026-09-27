package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/db"
	"github/eugenix-io/logx-inf-backend/testutils"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAllIdsFromStartGID(t *testing.T) {
	// testutils.SetupDBEnv()
	testutils.SetupDBEnv(t)

	db.Init()

	mockSubaccountDb := &db.SubaccountDB{}

	// Add subaccount_tables entries
	mockSubaccountDb.MockCreate("0xabc", 1, 1)
	mockSubaccountDb.MockCreate("0xdef", 1, 20_000)
	mockSubaccountDb.MockCreate("0xghi", 1, 1_000_000)

	ids, lastGID := mockSubaccountDb.GetAllIdsFromStartGID(0, 10_000)
	assert.Equal(t, 1, len(*ids))
	assert.Equal(t, uint(20_000-1), lastGID)

	ids, lastGID = mockSubaccountDb.GetAllIdsFromStartGID(lastGID+1, 10_000)
	assert.Equal(t, 1, len(*ids))
	assert.Equal(t, uint(1_000_000-1), lastGID)

	ids, lastGID = mockSubaccountDb.GetAllIdsFromStartGID(lastGID+1, 10_000)
	assert.Equal(t, 1, len(*ids))
	assert.Equal(t, uint(1_000_000), lastGID)
}
