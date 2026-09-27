package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringMapToOrderTable(t *testing.T) {
	// Testing with old data

	data := map[string]string{
		"amount_x18":       "2000000000000000000000",
		"signature":        "0xcba04cd50f13344915eb18fb9db6f89e881f6be080484d809f05dab587df062a6661b5e80082346799e42dd128edc24a5936cd111d6f0e2ec28a15c075746cd31c",
		"broker_id":        "1",
		"session_key":      "0xaF3d1D297DFad00d88aDB28BF25A2252Be00f593",
		"party":            "SOLVER",
		"expiry_ts":        "1726122253547",
		"status":           "PARTIAL",
		"type":             "LIMIT",
		"side":             "BUY",
		"market_id":        "21",
		"subaccount_id":    "1_0xd37eD507cA37Faa17079bFf5e46DcedE951577dB_1",
		"total_filled_x18": "0",
		"is_reduce":        "1",
		"price_x18":        "4126500000000000000",
		"timestamp":        "1726122073586",
	}

	var target db.OrderTable
	target.ID = 1
	db.StringMapToOrderTable(data, &target)

	expected := db.OrderTable{
		BaseTable: db.BaseTable{
			ID: 1,
		},
		Amountx18:        ctypes.BigInt{Val: cutils.StrToBigInt("2000000000000000000000")},
		BrokerId:         1,
		SubaccountId:     "1_0xd37eD507cA37Faa17079bFf5e46DcedE951577dB_1",
		SessionKey:       "0xaF3d1D297DFad00d88aDB28BF25A2252Be00f593",
		MarketId:         21,
		Side:             "BUY",
		Party:            "SOLVER",
		Type:             "LIMIT",
		TotalFilledx18:   ctypes.BigInt{Val: big.NewInt(0)},
		ExpiryTs:         1726122253547,
		Status:           "PARTIAL",
		Timestamp:        1726122073586,
		IsReduce:         true,
		Signature:        "0xcba04cd50f13344915eb18fb9db6f89e881f6be080484d809f05dab587df062a6661b5e80082346799e42dd128edc24a5936cd111d6f0e2ec28a15c075746cd31c",
		Pricex18:         ctypes.BigInt{Val: big.NewInt(4126500000000000000)},
		TriggerPricex18:  ctypes.BigInt{Val: big.NewInt(0)},
		TriggerCondition: "",
	}

	// Testing with new data
	assert.Equalf(t, target, expected, "Expected x, got something else")

	// Incorrect data
	data["trigger_price_x18"] = ""
	db.StringMapToOrderTable(data, &target)
	// Testing with new data
	assert.Equalf(t, target, expected, "Expected x, got something else")

	// New data
	data["trigger_price_x18"] = "1000000000000000000000"
	data["trigger_condition"] = "TP"

	db.StringMapToOrderTable(data, &target)

	expected.TriggerPricex18 = ctypes.BigInt{Val: cutils.StrToBigInt("1000000000000000000000")}
	expected.TriggerCondition = "TP"

	assert.Equalf(t, target, expected, "Expected x, got something else")
}
