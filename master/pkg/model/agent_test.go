package model

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSortableSlotIndex(t *testing.T) {
	require.Equal(t, SortableSlotIndex(2), "002")
	require.Equal(t, SortableSlotIndex(16), "016")

	// Do we actually sort?
	var gpuIndexes []string
	for i := 0; i <= 999; i++ {
		gpuIndexes = append(gpuIndexes, SortableSlotIndex(i))
	}
	require.True(t, slices.IsSorted(gpuIndexes))
}
