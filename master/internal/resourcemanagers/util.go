package resourcemanagers

import "golang.org/x/exp/constraints"

// min returns the smallest value of all provided values.
func min[T constraints.Ordered](values ...T) T {
	minValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
	}
	return minValue
}

// max returns the largest value of all provided values.
func max(values ...int) int {
	maxValue := values[0]
	for _, value := range values[1:] {
		if value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
