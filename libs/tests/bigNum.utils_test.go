package tests

import (
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"math/big"
	"testing"
)

func TestQuantumToX18(t *testing.T) {
	quantum := uint64(1)
	// POSITIVE EXPO CASE:
	quantumConversionExpo := int32(10)
	expected := big.NewInt(100_000_000)
	res := cutils.QuantumToX18(quantum, quantumConversionExpo)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	// NEGATIVE EXPO CASE:
	quantum = uint64(2)
	quantumConversionExpo = int32(-4)
	// 2 * 1e18 / 1e-4 = 2 * 1e22 = 20000000000000000000000
	expected, _ = new(big.Int).SetString("20000000000000000000000", 10)
	res = cutils.QuantumToX18(quantum, quantumConversionExpo)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}
}

func TestQuantumPrecisionCheck(t *testing.T) {
	// Positive quantumConversionExpo case
	// Shift the decimal point to the right by quantumConversionExpo
	// Check if the decimal part has only 0s
	// Eg: 123.45 with quantumConversionExpo 2 will be 12345
	floatValueStr := "123.45"
	quantumConversionExpo := int32(2)
	err := cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 1: Expected error but got nil")
	}

	// Negative quantumConversionExpo case
	// Shift the decimal point to the left by -(quantumConversionExpo)
	// Check if the decimal part has only 0s
	// Eg: 123.45 with quantumConversionExpo -2 will be 1.2345
	floatValueStr = "1234500"
	quantumConversionExpo = int32(-2)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 2: Expected error but got nil")
	}

	// Positive edge case
	floatValueStr = "67798.941486"
	quantumConversionExpo = int32(6)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 3: Expected nil but got %v", err)
	}

	// Negative edge case
	floatValueStr = "67798000000"
	quantumConversionExpo = int32(-6)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 4: Expected nil but got %v", err)
	}

	// Failure case: positive
	floatValueStr = "67798.941481"
	quantumConversionExpo = int32(2)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err == nil {
		t.Errorf("Test 5: Expected error but got nil")
	}

	// Failure case: negative
	floatValueStr = "67798.0"
	quantumConversionExpo = int32(-1)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err == nil {
		t.Errorf("Test 6: Expected error but got nil")
	}

	// Failure case: negative
	floatValueStr = "67798.0"
	quantumConversionExpo = int32(-10)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err == nil {
		t.Errorf("Test 7: Expected error but got nil")
	}

	// Success case: negative
	floatValueStr = "0"
	quantumConversionExpo = int32(-1)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 8: Expected error but got nil")
	}

	// Success case: positive
	floatValueStr = "0"
	quantumConversionExpo = int32(1)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 9: Expected error but got nil")
	}

	// Success case: positive
	floatValueStr = "3517.646608"
	quantumConversionExpo = int32(6)
	err = cutils.QuantumPrecisionCheck(floatValueStr, quantumConversionExpo)
	if err != nil {
		t.Errorf("Test 10: Expected error but got nil")
	}
}

func TestX18ToQuantum(t *testing.T) {
	// POSITIVE EXPO CASE:
	x18 := big.NewInt(100_000_000)
	quantumConversionExpo := int32(10)
	// 100000000 * 1e10 / 1e18 = 1
	expected := uint64(1)
	res := cutils.X18ToQuantum(x18, quantumConversionExpo)
	if res != expected {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	// NEGATIVE EXPO CASE:
	// 2*1e22
	x18, _ = new(big.Int).SetString("20000000000000000000000", 10)
	quantumConversionExpo = int32(-4)
	// 2*1e22 * 1e-4 / 1e18 = 2
	expected = uint64(2)
	res = cutils.X18ToQuantum(x18, quantumConversionExpo)
	if res != expected {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}
}

func TestQuantumToFloatStr(t *testing.T) {
	quantum := uint64(1)
	quantumConversionExpo := int32(10)
	// 1 / 1e10 = 0.0000000001
	expected := "0.0000000001"
	res := cutils.QuantumToFloatStr(quantum, quantumConversionExpo)
	if res != expected {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	quantum = uint64(2)
	quantumConversionExpo = int32(-4)
	// 2 / 1e-4  = 20000
	expected = "20000"
	res = cutils.QuantumToFloatStr(quantum, quantumConversionExpo)
	if res != expected {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}
}

func TestFloatStrToX18(t *testing.T) {
	floatValue := "0.0000000001"
	// 0.0000000001 * 1e18 = 1e8
	expected := big.NewInt(100_000_000)
	res := cutils.FloatStrToX18(floatValue)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	floatValue = "0.127371543212"
	// 0.127371543212 * 1e18 = 127371543212e6
	expected, _ = new(big.Int).SetString("127371543212000000", 10)
	res = cutils.FloatStrToX18(floatValue)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}

	// Without decimal points
	floatValue = "1123"
	// 1123 * 1e18 = 1123e18
	expected, _ = new(big.Int).SetString("1123000000000000000000", 10)
	res = cutils.FloatStrToX18(floatValue)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 3: Expected %v but got %v", expected, res)
	}

	floatValue = ""
	// 0 * 1e18 = 0
	expected = big.NewInt(0)
	res = cutils.FloatStrToX18(floatValue)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 4: Expected %v but got %v", expected, res)
	}
}

func TestX18ToFloatStr(t *testing.T) {
	x18 := big.NewInt(100_000_000)
	// 1e8 / 1e18 = 0.0000000001 (1e-10)
	expected := "0.0000000001"
	res := cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	x18, _ = new(big.Int).SetString("200", 10)
	// 2e2 / 1e18 = 0.0000000000000002 (2e-16)
	expected = "0.0000000000000002"
	res = cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}

	x18, _ = new(big.Int).SetString("2000000000000000000000", 10)
	// 1e21 / 1e18 = 1000
	expected = "2000"
	res = cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 3: Expected %v but got %v", expected, res)
	}

	x18, _ = new(big.Int).SetString("-2000100000000000000000", 10)
	// -1e21 / 1e18 = -1000
	expected = "-2000.1"
	res = cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 4: Expected %v but got %v", expected, res)
	}

	x18, _ = new(big.Int).SetString("-2000000000000000", 10)
	// -1e17 / 1e18 = -0.1
	expected = "-0.002"
	res = cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 5: Expected %v but got %v", expected, res)
	}

	x18, _ = new(big.Int).SetString("00000", 10)
	// 0 / 1e18 = 0
	expected = "0"
	res = cutils.X18ToFloatStr(x18)
	if res != expected {
		t.Errorf("Test 6: Expected %v but got %v", expected, res)
	}
}

func TestFloatStrZeroCheck(t *testing.T) {
	// Failure case
	floatValue := "0.0000000001"
	res := cutils.FloatStrZeroCheck(floatValue)
	if res {
		t.Errorf("Test 1: Expected true but got false")
	}

	// Success case
	floatValue = "0.0000000000"
	res = cutils.FloatStrZeroCheck(floatValue)
	if !res {
		t.Errorf("Test 2: Expected false but got true")
	}

	// Success case
	floatValue = "0"
	res = cutils.FloatStrZeroCheck(floatValue)
	if !res {
		t.Errorf("Test 3: Expected false but got true")
	}

	// Failure case
	floatValue = "10"
	res = cutils.FloatStrZeroCheck(floatValue)
	if res {
		t.Errorf("Test 4: Expected true but got false")
	}
}

func TestMaxBigInt(t *testing.T) {
	a := big.NewInt(100)
	b := big.NewInt(200)
	expected := b
	res := cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 1: Expected %v but got %v", expected, res)
	}

	a = big.NewInt(300)
	b = big.NewInt(200)
	expected = a
	res = cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 2: Expected %v but got %v", expected, res)
	}

	a = big.NewInt(0)
	b = big.NewInt(0)
	expected = a
	res = cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 3: Expected %v but got %v", expected, res)
	}

	a = big.NewInt(-100)
	b = big.NewInt(100)
	expected = b
	res = cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 4: Expected %v but got %v", expected, res)
	}

	a = big.NewInt(-200)
	b = big.NewInt(-100)
	expected = b
	res = cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 5: Expected %v but got %v", expected, res)
	}

	a = big.NewInt(-200)
	b = big.NewInt(0)
	expected = b
	res = cutils.MaxBigInt(a, b)
	if res.Cmp(expected) != 0 {
		t.Errorf("Test 6: Expected %v but got %v", expected, res)
	}
}


