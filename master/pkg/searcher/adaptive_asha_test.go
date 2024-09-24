//nolint:exhaustruct
package searcher

import (
	"github.com/stretchr/testify/require"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestBracketMaxTrials(t *testing.T) {
	assert.DeepEqual(t, getBracketMaxTrials(20, 3., []int{3, 2, 1}), []int{12, 5, 3})
	assert.DeepEqual(t, getBracketMaxTrials(50, 3., []int{4, 3}), []int{35, 15})
	assert.DeepEqual(t, getBracketMaxTrials(50, 4., []int{3, 2}), []int{37, 13})
	assert.DeepEqual(t, getBracketMaxTrials(100, 4., []int{4, 3, 2}), []int{70, 22, 8})
}

func TestBracketMaxConcurrentTrials(t *testing.T) {
	assert.DeepEqual(t, getBracketMaxConcurrentTrials(0, 3., []int{9, 3, 1}), []int{3, 3, 3})
	assert.DeepEqual(t, getBracketMaxConcurrentTrials(11, 3., []int{9, 3, 1}), []int{4, 4, 3})
	// We try to take advantage of the max degree of parallelism for the narrowest bracket.
	assert.DeepEqual(t, getBracketMaxConcurrentTrials(0, 4., []int{40, 10}), []int{10, 10})
}

func TestMakeBrackets(t *testing.T) {
	cases := []struct {
		conf        expconf.AdaptiveASHAConfig
		expBrackets []bracket
	}{
		{
			conf: expconf.AdaptiveASHAConfig{
				RawMode:                expconf.AdaptiveModePtr(expconf.StandardMode),
				RawMaxLength:           &expconf.LengthV0{Units: 100},
				RawMaxConcurrentTrials: ptrs.Ptr(2),
				RawMaxTrials:           ptrs.Ptr(10),
			},
			expBrackets: []bracket{
				{
					numRungs:            2,
					maxTrials:           7,
					maxConcurrentTrials: 1,
				},
				{
					numRungs:            1,
					maxTrials:           3,
					maxConcurrentTrials: 1,
				},
			},
		},
		{
			conf: expconf.AdaptiveASHAConfig{
				RawMode:                expconf.AdaptiveModePtr(expconf.ConservativeMode),
				RawMaxLength:           &expconf.LengthV0{Units: 1000},
				RawDivisor:             ptrs.Ptr(3.0),
				RawMaxConcurrentTrials: ptrs.Ptr(5),
				RawMaxTrials:           ptrs.Ptr(10),
			},
			expBrackets: []bracket{
				{
					numRungs:            3,
					maxTrials:           7,
					maxConcurrentTrials: 2,
				},
				{
					numRungs:            2,
					maxTrials:           2,
					maxConcurrentTrials: 2,
				},
				{
					numRungs:            1,
					maxTrials:           1,
					maxConcurrentTrials: 1,
				},
			},
		},
	}
	for _, c := range cases {
		brackets := makeBrackets(schemas.WithDefaults(c.conf))
		require.Equal(t, len(c.expBrackets), len(brackets))
		require.Equal(t, c.expBrackets, brackets)
	}
}

func TestAdaptiveASHA(t *testing.T) {
	// xxx: write this
}
