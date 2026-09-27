package ctypes

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/xlog"
	"math/big"
)

type BigInt struct {
	Val *big.Int
}

func (b *BigInt) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		// Try to parse the string value as a big.Int
		var success bool
		b.Val, success = new(big.Int).SetString(v, 10)
		if !success {
			return fmt.Errorf("failed to parse BigInt value: %s", value)
		}
		return nil
	}
	xlog.Errorf("BigInt Scan: received value of type %T and value %v", value, value)
	return fmt.Errorf("invalid type of value: %v", value)
}

func (b BigInt) Value() (driver.Value, error) {
	if b.Val == nil {
		return nil, nil
	}
	return b.Val.String(), nil
}

// If the value is nil, return 0
// MarshalJSON implements the json.Marshaler interface.
func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.Val == nil {
		return json.Marshal("0")
	}
	return json.Marshal(b.Val.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (b *BigInt) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	var ok bool
	b.Val, ok = new(big.Int).SetString(str, 10)
	if !ok {
		return fmt.Errorf("failed to parse BigInt value: %s", str)
	}
	return nil
}

func (ot *BigInt) UnmarshalBinary(data []byte) error {
	// Convert bytes to string
	*ot = NewBigInt(new(big.Int).SetBytes(data))
	return nil
}

// Set default value to 0 if the value is nil
func (ot BigInt) MarshalBinary() (data []byte, err error) {
	if ot.Val == nil {
		ot.Val = big.NewInt(0)
	}
	valStr := ot.Val.String()
	return []byte(valStr), nil
}

func (b *BigInt) Cmp(other BigInt) int {
	return b.Val.Cmp(other.Val)
}

func (b BigInt) Sign() int {
	return b.Val.Sign()
}

// Note that this doesn't modify the original value
func (b *BigInt) Add(other BigInt) BigInt {
	return BigInt{Val: new(big.Int).Add(b.Val, other.Val)}
}

// Note that this doesn't modify the original value
func (b *BigInt) Sub(other BigInt) BigInt {
	return BigInt{Val: new(big.Int).Sub(b.Val, other.Val)}
}

// Note that this doesn't modify the original value
func (b *BigInt) Mul(other BigInt) BigInt {
	return BigInt{Val: new(big.Int).Mul(b.Val, other.Val)}
}

// Note that this doesn't modify the original value
func (b *BigInt) Div(other BigInt) BigInt {
	return BigInt{Val: new(big.Int).Div(b.Val, other.Val)}
}

func NewBigInt(val *big.Int) BigInt {
	return BigInt{Val: val}
}

func NewBigIntFromString(val string) BigInt {
	if val == "" {
		val = "0"
	}
	bigInt, success := new(big.Int).SetString(val, 10)
	if !success {
		xlog.Errorf("Error while parsing string: %s", val)
		return BigInt{Val: nil}
	}
	return NewBigInt(bigInt)
}

func (b BigInt) Copy() BigInt {
	return NewBigInt(new(big.Int).Set(b.Val))
}
func (b *BigInt) IsNil() bool {
	return b.Val == nil
}

func (b *BigInt) String() string {
	return b.Val.String()
}

func ConvertStringAndNegate(val string) string {
	// Convert the string to a BigInt
	bigInt := NewBigIntFromString(val)

	// Check if the conversion was successful
	if bigInt.IsNil() {
		xlog.Errorf("Failed to parse string to BigInt: %s", val)
		return ""
	}

	// Negate the BigInt value
	negatedBigInt := bigInt.Copy()
	negatedBigInt.Val.Neg(negatedBigInt.Val)

	// Return the negated value as a string
	return negatedBigInt.String()
}
