package utils

import "strconv"

// Convert []string to []uint64

func ConvertStringSliceToUint64Slice(strSlice []string) *[]uint64 {
	uintSlice := make([]uint64, len(strSlice))
	for i, strValue := range strSlice {
		value, err := strconv.ParseUint(strValue, 10, 64)
		if err != nil {
			panic(err)
		}
		uintSlice[i] = value
	}
	return &uintSlice
}
