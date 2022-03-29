package resourcemanagers

import (
	"math"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/mmath"
)

func TestMin(t *testing.T) {
	assert.Equal(t, mmath.Min(1, 2, 3), 1)
	assert.Equal(t, mmath.Min(math.MinInt64, 2, 3), math.MinInt64)
	assert.Equal(t, mmath.Min(math.MaxInt64, 2, 3), 2)
}

func TestMax(t *testing.T) {
	assert.Equal(t, mmath.Max(1, 2, 3), 3)
	assert.Equal(t, mmath.Max(math.MinInt64, 2, 3), 3)
	assert.Equal(t, mmath.Max(math.MaxInt64, 2, 3), math.MaxInt64)
}
