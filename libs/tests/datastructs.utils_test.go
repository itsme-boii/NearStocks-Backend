package tests

import (
	"encoding/json"
	"fmt"
	"github/eugenix-io/logx-inf-backend/libs/cutils"
	"github/eugenix-io/logx-inf-backend/libs/db"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Recursively convert a map to a struct
func MapToStruct(data map[string]interface{}, result interface{}) error {
	val := reflect.ValueOf(result).Elem()
	for key, value := range data {
		field := val.FieldByName(key)
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		fieldType := field.Type()
		valueType := reflect.TypeOf(value)

		// Handle nested structs or pointers to structs
		if fieldType.Kind() == reflect.Struct || (fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct) {
			var newStruct reflect.Value
			if fieldType.Kind() == reflect.Ptr {
				newStruct = reflect.New(fieldType.Elem()).Elem()
			} else {
				newStruct = field
			}

			nestedMap, ok := value.(map[string]interface{})
			if ok {
				MapToStruct(nestedMap, newStruct.Addr().Interface())
				if fieldType.Kind() == reflect.Ptr {
					field.Set(newStruct.Addr())
				}
			}
		} else if fieldType.Kind() == reflect.Ptr {
			// Handle pointer fields
			if valueType.AssignableTo(fieldType.Elem()) {
				newVal := reflect.New(fieldType.Elem())
				newVal.Elem().Set(reflect.ValueOf(value))
				field.Set(newVal)
			}
		} else {
			// Handle non-pointer fields
			if valueType.AssignableTo(fieldType) {
				field.Set(reflect.ValueOf(value))
			}
		}
	}
	return nil
}
func TestMapToStruct2(t *testing.T) {
	// ADD YOUR CODE HERE
	data := map[string]any{
		"amt_quantum":   10,
		"broker_id":     2,
		"expiry_ts":     1729214862000,
		"market_id":     7,
		"party":         "SOLVER",
		"price_quantum": 2999000,
		"side":          "SELL",
		"signature":     "0xrandomSignature",
		"status":        "PARTIAL",
		"subaccount_id": "2_0x7f9082dF4d0FBfE668407b9b2066dd3AC06c9d80_1",
		"timestamp":     0,
	}

	var target db.OrderTable
	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		t.Fatalf("Expected nil, got %v", err)
	}

	err2 := json.Unmarshal(bytes, &target)

	if err2 != nil {
		fmt.Printf("Error: %v\n", err2)
		t.Fatalf("Expected nil, got %v", err2)
	}

	fmt.Printf("Target: %v\n", target)
}

func TestReverseSlice(t *testing.T) {
	// Case odd len
	rev := cutils.ReverseSlice([]int{1, 2, 3, 4, 5})
	expected := []int{5, 4, 3, 2, 1}
	assert.ElementsMatchf(t, expected, rev, "Expected %v, got %v", expected, rev)

	// Case even len
	rev = cutils.ReverseSlice([]int{1, 2, 3, 4})
	expected = []int{4, 3, 2, 1}
	assert.ElementsMatchf(t, expected, rev, "Expected %v, got %v", expected, rev)

	// Case empty slice
	rev = cutils.ReverseSlice([]int{})
	expected = []int{}
	assert.ElementsMatchf(t, expected, rev, "Expected %v, got %v", expected, rev)
}
