package math

import (
	"strconv"
	"strings"
)

func CountDecimalPlaces(f float64) int {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	parts := strings.Split(s, ".")

	if len(parts) == 2 {
		return len(parts[1])
	}

	return 0
}

func CountTrailingZeros(n int) int {
	s := strconv.Itoa(n)
	return len(s) - len(strings.TrimRight(s, "0"))
}
