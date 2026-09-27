package tests

import (
	"github/eugenix-io/logx-inf-backend/services/amm/utils"
	"testing"
)

func TestFormatNumberWithPrecision(t *testing.T) {
	// Positive precision case
	number := 123.456
	precision := 2
	expected := "123.46"
	res := utils.FormatNumberWithPrecision(number, precision)
	if res != expected {
		t.Errorf("Test 1: Expected %s but got %s", expected, res)
	}

	// Negative precision case
	number = 123.456
	precision = -2
	expected = "100"
	res = utils.FormatNumberWithPrecision(number, precision)
	if res != expected {
		t.Errorf("Test 2: Expected %s but got %s", expected, res)
	}

	// Zero precision case
	number = 123.456
	precision = 0
	expected = "123"
	res = utils.FormatNumberWithPrecision(number, precision)
	if res != expected {
		t.Errorf("Test 3: Expected %s but got %s", expected, res)
	}
}