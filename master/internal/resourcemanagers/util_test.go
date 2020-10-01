package resourcemanagers

import (
	"math"
	"testing"

	"gotest.tools/assert"
)

func TestMin(t *testing.T) {
	assert.Equal(t, min(1, 2, 3), 1)
	assert.Equal(t, min(math.MinInt64, 2, 3), math.MinInt64)
	assert.Equal(t, min(math.MaxInt64, 2, 3), 2)
}

func TestMax(t *testing.T) {
	assert.Equal(t, max(1, 2, 3), 3)
	assert.Equal(t, max(math.MinInt64, 2, 3), 3)
	assert.Equal(t, max(math.MaxInt64, 2, 3), math.MaxInt64)
}
