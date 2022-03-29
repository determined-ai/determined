package mmath

import "golang.org/x/exp/constraints"

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
