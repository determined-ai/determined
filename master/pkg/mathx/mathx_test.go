package mathx

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMin(t *testing.T) {
	require.Equal(t, Min(1, 2, 3), 1)
	require.Equal(t, Min(math.MinInt64, 2, 3), math.MinInt64)
	require.Equal(t, Min(math.MaxInt64, 2, 3), 2)
}

func TestMax(t *testing.T) {
	require.Equal(t, Max(1, 2, 3), 3)
	require.Equal(t, Max(math.MinInt64, 2, 3), 3)
	require.Equal(t, Max(math.MaxInt64, 2, 3), math.MaxInt64)
}

func TestClamp(t *testing.T) {
	require.Equal(t, Clamp(1, 2, 3), 2)
	require.Equal(t, Clamp(1, 4, 3), 3)
	require.Equal(t, Clamp(1, 0, 3), 1)
	require.Panics(t, func() { Clamp(3, 0, 1) })
}
