package utils

import "math"

func Clamp(x, lower, upper float64) float64 {
	return math.Max(lower, math.Min(upper, x))
}

// FF function based on the given piecewise linear function
func FF(x float64) float64 {
	absX := math.Abs(x)
	signX := math.Copysign(1, x) // returns 1 if x is positive, -1 if x is negative

	if absX < 0.005 {
		return x
	} else if absX < 0.015 {
		return 0.005*signX + ((absX-0.005)*signX)*2
	} else {
		return (0.005+(0.015-0.005)*2)*signX + ((absX-0.015)*signX)*4
	}
}
