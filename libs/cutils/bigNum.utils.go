package cutils

import (
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github/eugenix-io/logx-inf-backend/libs/ctypes"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
)

// TODO: Write tests for this
func QuantumToX18(quantum uint64, quantumConversionExpo int32) *big.Int {
	quantumBig := big.NewInt(int64(quantum))
	x18Big := big.NewInt(1e18)
	// Quantum Big * 1e18 / unitQuantum Big
	res := new(big.Int).Mul(quantumBig, x18Big)
	if quantumConversionExpo > 0 {
		res = res.Div(res, GetBigxCust(uint32(quantumConversionExpo)))
	} else {
		res = res.Mul(res, GetBigxCust(uint32(-quantumConversionExpo)))
	}
	return res
}

// TODO: Write tests for this
// Quantum = (x18 * quantumConversionExpo) / 1e18
func X18ToQuantum(x18 *big.Int, quantumConversionExpo int32) uint64 {
	res := new(big.Int).Set(x18)
	if quantumConversionExpo > 0 {
		res = new(big.Int).Mul(res, GetBigxCust(uint32(quantumConversionExpo)))
	} else {
		res = new(big.Int).Div(res, GetBigxCust(uint32(-quantumConversionExpo)))
	}

	res = new(big.Int).Div(res, GetBigx18())
	return res.Uint64()
}

// TODO: Write tests for this
// NOTE: This can be optimized by directly converting quantum to float
func QuantumToFloatStr(quantum uint64, quantumConversionExpo int32) string {
	quantumBig := QuantumToX18(quantum, quantumConversionExpo)
	return X18ToFloatStr(quantumBig)
}

// TODO: Write tests for this
// Precision will be in 1, 2, 3, -1, -2 etc
func QuantumPrecisionCheck(floatValueStr string, quantumConversionExpo int32) error {
	// Positive quantumConversionExpo case
	// Shift the decimal point to the right by quantumConversionExpo
	// Check if the decimal part has only 0s
	// Eg: 123.45 with quantumConversionExpo 2 will be 12345
	if quantumConversionExpo > 0 {
		decimalIndex := strings.Index(floatValueStr, ".")
		// This means no decimal points
		if decimalIndex == -1 {
			return nil
		}
		floatStrNoZeros := strings.TrimRight(floatValueStr, "0")

		newDecimalIndex := decimalIndex + int(quantumConversionExpo)

		// This means no decimal points after shifting
		if newDecimalIndex >= len(floatStrNoZeros)-1 {
			return nil
		}
		return fmt.Errorf("invalid float: %s for quantumConversionExpo: %d", floatValueStr, quantumConversionExpo)
	}

	// Negative quantumConversionExpo case
	// Shift the decimal point to the left by -(quantumConversionExpo)
	// Check if the decimal part has only 0s
	// Eg: 123.45 with quantumConversionExpo -2 will be 1.2345
	if quantumConversionExpo < 0 {
		decimalIndex := strings.Index(floatValueStr, ".")
		// This means no decimal points
		if decimalIndex == -1 {
			decimalIndex = len(floatValueStr)
			floatValueStr += ".0"
		}
		newDecimalIndex := Max(0, decimalIndex+int(quantumConversionExpo))
		floatDecAfterShift := floatValueStr[newDecimalIndex:decimalIndex] + floatValueStr[decimalIndex+1:]
		floatDecStrNoZeros := strings.TrimRight(floatDecAfterShift, "0")
		if len(floatDecStrNoZeros) > 0 {
			return fmt.Errorf("invalid float: %s for quantumConversionExpo: %d", floatValueStr, quantumConversionExpo)
		}
	}

	return nil
}

const ZEROS18 = "000000000000000000"

// NOTE: If the string is not a valid big.Int, it will return 0
func StrToBigInt(intValueStr string) *big.Int {
	res, success := new(big.Int).SetString(intValueStr, 10)
	if !success {
		xlog.Errorf("failed to convert int to big.Int: %v.... returning 0", intValueStr)
		return big.NewInt(0)
	}
	return res
}

// When "" is passed or wrong input is provided, it returns 0
func FloatStrToX18(floatValueStr string) *big.Int {
	floatValueStr += ZEROS18
	decimalIndex := strings.Index(floatValueStr, ".")
	if decimalIndex != -1 {
		floatValueStr = floatValueStr[:decimalIndex] + floatValueStr[decimalIndex+1:decimalIndex+1+18]
	}
	res, success := new(big.Int).SetString(floatValueStr, 10)
	if !success {
		xlog.Errorf("failed to convert float to x18: %v", floatValueStr)
		return big.NewInt(0)
	}
	return res
}

// This return BigInt custom object
func FloatStrToBigIntX18(floatValueStr string) ctypes.BigInt {
	return ctypes.NewBigInt(FloatStrToX18(floatValueStr))
}

// Convert big.Int slice to custom BigInt slice
func X18SliceToBigIntSlice(x18s []*big.Int) []ctypes.BigInt {
	bigInts := make([]ctypes.BigInt, len(x18s))
	for i, x18 := range x18s {
		bigInts[i] = ctypes.NewBigInt(x18)
	}
	return bigInts
}

func X18ToFloatStr(x18 *big.Int) string {
	var x18Str string
	if x18.Cmp(big.NewInt(0)) < 0 {
		// Do not take the negative sign
		x18Str = ZEROS18 + x18.String()[1:]
	} else {
		x18Str = ZEROS18 + x18.String()
	}
	intPart := x18Str[:len(x18Str)-18]
	// Remove leading zeros from int
	intPart = strings.TrimLeft(intPart, "0")
	if len(intPart) == 0 {
		intPart = "0"
	}

	decimalPart := x18Str[len(x18Str)-18:]
	// Remove trailing zeros from decimal
	decimalPart = strings.TrimRight(decimalPart, "0")
	if x18.Sign() < 0 {
		intPart = "-" + intPart
	}
	if len(decimalPart) == 0 {
		return intPart
	}

	return fmt.Sprintf("%s.%s", intPart, decimalPart)
}

// This divides the big int by 1e18
func Divx18(x18 *big.Int) *big.Int {
	return new(big.Int).Div(x18, GetBigx18())
}

// Multiplies the big int by 1e18
func Mulx18(x18 *big.Int) *big.Int {
	return new(big.Int).Mul(x18, GetBigx18())
}

func MulxCust(x *big.Int, expo int64) *big.Int {
	if expo < 0 {
		return new(big.Int).Div(x, new(big.Int).Exp(big.NewInt(10), big.NewInt(-expo), nil))
	} else {
		return new(big.Int).Mul(x, new(big.Int).Exp(big.NewInt(10), big.NewInt(expo), nil))
	}
}

func DivxCust(x *big.Int, expo int64) *big.Int {
	if expo < 0 {
		return new(big.Int).Mul(x, new(big.Int).Exp(big.NewInt(10), big.NewInt(-expo), nil))
	} else {
		return new(big.Int).Div(x, new(big.Int).Exp(big.NewInt(10), big.NewInt(expo), nil))
	}
}

// Min returns the smaller of two big.Int values
func Min(x, y *big.Int) *big.Int {
	if x.Cmp(y) <= 0 {
		return x
	}
	return y
}

// Max returns the larger of two big.Int values
func Maxi(x, y *big.Int) *big.Int {
	if x.Cmp(y) >= 0 {
		return x
	}
	return y
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func FloatToBigInt(f float64) *big.Int {
	//Improvements - came up with the following implementations to avoid precision loss while converting float to big Int
	//		we have to come up with a cleaner way to deal with price and amount
	floatStr := fmt.Sprintf("%f", f)
	floatStr = strings.TrimRight(floatStr, "0")
	floatStr = strings.TrimRight(floatStr, ".")

	// Split the string into integer and fractional parts
	parts := strings.Split(floatStr, ".")
	if len(parts) != 2 {
		parts = append(parts, "0")
	}

	// Get the number of decimal places in the fractional part
	decimalPlaces := len(parts[1])
	if decimalPlaces > 18 {
		panic("more than 18 decimal places")
	}

	// Calculate the number of zeroes to add (18 - number of decimals)
	zeroesToAdd := 18 - decimalPlaces

	// Remove the decimal point and add the required number of zeroes
	combinedStr := parts[0] + parts[1] + strings.Repeat("0", zeroesToAdd)

	// Convert the resulting string to a big.Int
	result, success := new(big.Int).SetString(combinedStr, 10)
	if !success {
		panic("conversion to big.Int failed")
	}

	return result
}

func ConvertXCustToX18(tokenPriceX *big.Int, expo int64) *big.Int {
	scalex18 := new(big.Int).Mul(tokenPriceX, GetBigx18())
	if expo < 0 {
		return new(big.Int).Div(scalex18, new(big.Int).Exp(big.NewInt(10), big.NewInt(-expo), nil))
	} else {
		return new(big.Int).Mul(scalex18, new(big.Int).Exp(big.NewInt(10), big.NewInt(expo), nil))
	}
}

func MinBigInt(a, b *big.Int) *big.Int {
	if a.Cmp(b) == -1 {
		return a
	}
	return b
}

func MaxBigInt(a, b *big.Int) *big.Int {
	if a.Cmp(b) == 1 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}

func FloatStrToFloat64(floatValueStr string) float64 {
	floatValue, success := strconv.ParseFloat(floatValueStr, 64)
	if success != nil {
		log.Panicf("failed to convert float string to float64: error - %v | floatValueStr - %v", success, floatValueStr)
	}
	return floatValue
}

func FloatStrZeroCheck(str string) bool {
	return strings.Trim(str, "0.") == ""
}
