package utils

import (
	"fmt"
	"math"
)

// precision will be in 1, 2, 3, -1, -2 etc
// Take the float number upto the precision and returns stringified float
func FormatNumberWithPrecision(number float64, precision int) string {
	if precision >= 0 {
		return fmt.Sprintf(fmt.Sprintf("%%.%df", precision), number)
	} else {
		// Convert the float to int
		intNumber := int64(number)
		// Now multiply by 10^precision
		intNumber = intNumber / int64(math.Pow10(-precision))
		// Now divide by 10^precision
		intNumber = intNumber * int64(math.Pow10(-precision))
		return fmt.Sprintf("%d", intNumber)
	}
}
