package cutils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
)

// NOTE: This is experimental and may fail in some cases.
// MapToStruct converts a map into a struct.
// The map keys should match the struct field names.
// Returns an error if the conversion fails.
func MapToStruct(data map[string]interface{}, result interface{}) error {
	return fillStruct(reflect.ValueOf(result).Elem(), data)
}

func fillStruct(v reflect.Value, data map[string]interface{}) error {
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected a struct, but got %v", v.Kind())
	}

	for key, value := range data {
		field := v.FieldByName(key)
		if !field.IsValid() {
			continue // skip if the field is not found
		}

		fieldType := field.Type()

		if field.Kind() == reflect.Ptr {
			if value == nil {
				continue // skip nil values for pointer fields
			}
			field.Set(reflect.New(fieldType.Elem()))
			field = field.Elem()
			fieldType = field.Type()
		}

		if fieldType.Kind() == reflect.Struct {
			if subMap, ok := value.(map[string]interface{}); ok {
				err := fillStruct(field, subMap)
				if err != nil {
					return err
				}
			}
		} else {
			val := reflect.ValueOf(value)
			if !val.Type().AssignableTo(fieldType) {
				if val.Type().ConvertibleTo(fieldType) {
					val = val.Convert(fieldType)
				} else {
					continue // skip if the value is not assignable or convertible
				}
			}
			field.Set(val)
		}
	}
	return nil
}

func ConvertMapStringToInterface(data map[string]string) map[string]interface{} {
	converted := make(map[string]interface{}, len(data))
	for key, value := range data {
		// Try to convert string to int
		if intValue, err := strconv.Atoi(value); err == nil {
			converted[key] = intValue
		} else {
			converted[key] = value
		}
	}
	return converted
}

// NOTE: USE WITH CAUTION as this is slow: https://stackoverflow.com/a/69652489/16574003
func CloneStruct[T interface{}](orig *T) *T {
	origJSON, err := json.Marshal(orig)
	if err != nil {
		panic(err)
	}
	clone := new(T)
	if err = json.Unmarshal(origJSON, clone); err != nil {
		panic(err)
	}
	return clone
}

func ConvertMapToStructViaJson[T interface{}](data interface{}, target *T) {
	// Convert map to json
	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(jsonData, target)
	if err != nil {
		panic(err)
	}
}

// UniqueSliceToMap converts a slice to a map using the provided keyFunc to generate the key.
func UniqueSliceToMap[K comparable, V any](slice []V, keyFunc func(V) K) map[K]V {
	m := make(map[K]V)
	for _, v := range slice {
		k := keyFunc(v)
		if _, exists := m[k]; exists {
			panic(
				fmt.Sprintf(
					"UniqueSliceToMap: duplicate value: %+v",
					v,
				),
			)
		}
		m[k] = v
	}
	return m
}

func MapSlice[K, V any](slice []V, mappingFunc func(V) K) []K {
	result := make([]K, len(slice))
	for i, v := range slice {
		result[i] = mappingFunc(v)
	}
	return result
}

// FilterSlice takes a function that returns a boolean on whether to include the element in the final
// result, and returns a slice of elements where the function returned true when called with each element.
func FilterSlice[V any](values []V, filterFunc func(V) bool) []V {
	filteredValues := make([]V, 0, len(values))
	for _, value := range values {
		if filterFunc(value) {
			filteredValues = append(filteredValues, value)
		}
	}

	return filteredValues
}

// Returns a slice of values from a map.
func MapValues[W comparable, V any](m map[W]V) []V {
	values := make([]V, 0, len(m))
	for _, value := range m {
		values = append(values, value)
	}
	return values
}

// Returns a slice of keys from a map
func MapKeys[W comparable, V any](m map[W]V) []W {
	keys := make([]W, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// MergeMaps merges all the maps into a single map.
// Does not require maps to have distinct keys.
func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	combinedMap := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			combinedMap[k] = v
		}
	}
	return combinedMap
}

func SliceExists[V comparable](slice []V, value V) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func ReverseSlice[V any](slice []V) []V {
	reversed := make([]V, len(slice))
	for i, j := 0, len(slice)-1; i < len(slice); i, j = i+1, j-1 {
		reversed[i] = slice[j]
	}
	return reversed
}

// NOTE: This is only written for adding map values for now. This can be modified to support other operations.
func GenericMergeMaps[K comparable, V any](mapA map[K]V, mapB map[K]V, fn func(V, V) V) map[K]V {
	result := make(map[K]V)
	for k, v := range mapA {
		result[k] = v
	}
	for k, v := range mapB {
		if existing, ok := result[k]; ok {
			result[k] = fn(existing, v)
		} else {
			result[k] = v
		}
	}
	return result
}

func ModifyMapValues[K comparable, V any, W any](m map[K]V, fn func(V) W) map[K]W {
	result := make(map[K]W)
	for k, v := range m {
		result[k] = fn(v)
	}
	return result
}

// Ptr returns a pointer to the value passed.
func Ptr[T any](value T) *T {
	return &value
}
