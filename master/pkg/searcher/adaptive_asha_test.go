//nolint:exhaustivestruct
package searcher

import (
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

func modePtr(x expconf.AdaptiveMode) *expconf.AdaptiveMode {
	return &x
}

func TestAdaptiveASHASearcherReproducibility(t *testing.T) {
	conf := expconf.AdaptiveASHAConfig{
		RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(6400)),
		RawMaxTrials: ptrs.Ptr(128),
	}
	conf = schemas.WithDefaults(conf)
	gen := func() SearchMethod { return newAdaptiveASHASearch(conf, true) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveASHASearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.5),
				newConstantPredefinedTrial(toOps("300B"), 0.4),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.2),
				newConstantPredefinedTrial(toOps("900B"), 0.1),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.5),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.4),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.2),
				newConstantPredefinedTrial(toOps("900B"), 0.1),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}

func TestAdaptiveASHAStoppingSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newConstantPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
					RawStopOnce:  ptrs.Ptr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(true),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
					RawStopOnce:  ptrs.Ptr(true),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newConstantPredefinedTrial(toOps("300B 900B"), 0.2),
				newConstantPredefinedTrial(toOps("300B 900B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
					RawStopOnce:  ptrs.Ptr(true),
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B 900B"), 0.1),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B 900B"), 0.3),
				newConstantPredefinedTrial(toOps("900B"), 0.4),
				newConstantPredefinedTrial(toOps("900B"), 0.5),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.Ptr(false),
				RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(900)),
					RawMaxTrials: ptrs.Ptr(5),
					RawMode:      modePtr(expconf.StandardMode),
					RawMaxRungs:  ptrs.Ptr(2),
					RawDivisor:   ptrs.Ptr[float64](3),
					RawStopOnce:  ptrs.Ptr(true),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
