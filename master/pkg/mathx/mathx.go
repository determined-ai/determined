package mathx

import (
	"fmt"

	"golang.org/x/exp/constraints"
)

// Min returns the smallest value of all provided values.
func Min[T constraints.Ordered](values ...T) T {
	minValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}

// Max returns the largest value of all provided values.
func Max[T constraints.Ordered](values ...T) T {
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}

// Clamp returns the val, clamped between min and max. Clamp panics if max is less than min.
func Clamp[T constraints.Ordered](min, val, max T) T {
	if min > max {
		panic(fmt.Sprintf("cannot call clamp with %v (min) !<= %v (max)", max, min))
	}
	return Min(Max(min, val), max)
}
