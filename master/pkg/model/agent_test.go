package model

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortableSlotIndex(t *testing.T) {
	require.Equal(t, "002", SortableSlotIndex(2))
	require.Equal(t, "016", SortableSlotIndex(16))

	// Do we actually sort?
	var gpuIndexes []string
	for i := 0; i <= 999; i++ {
		gpuIndexes = append(gpuIndexes, SortableSlotIndex(i))
	}
	require.True(t, slices.IsSorted(gpuIndexes))
}
