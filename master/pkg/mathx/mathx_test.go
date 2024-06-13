package mathx

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMin(t *testing.T) {
	require.Equal(t, 1, Min(1, 2, 3))
	require.Equal(t, math.MinInt64, Min(math.MinInt64, 2, 3))
	require.Equal(t, 2, Min(math.MaxInt64, 2, 3))
}

func TestMax(t *testing.T) {
	require.Equal(t, 3, Max(1, 2, 3))
	require.Equal(t, 3, Max(math.MinInt64, 2, 3))
	require.Equal(t, math.MaxInt64, Max(math.MaxInt64, 2, 3))
}

func TestClamp(t *testing.T) {
	require.Equal(t, 2, Clamp(1, 2, 3))
	require.Equal(t, 3, Clamp(1, 4, 3))
	require.Equal(t, 1, Clamp(1, 0, 3))
	require.Panics(t, func() { Clamp(3, 0, 1) })
}
