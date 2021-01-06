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

func TestAdaptiveASHASearcherReproducibility(t *testing.T) {
	conf := expconf.AdaptiveASHAConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		MaxLength: expconf.NewLengthInBatches(6400), MaxTrials: 128, Divisor: ptrs.Float64Ptr(4),
		Mode: expconf.AdaptiveModePtr(expconf.AggressiveMode), MaxRungs: ptrs.IntPtr(3),
	}
	schemas.FillDefaults(&conf)
	gen := func() SearchMethod { return newAdaptiveASHASearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 5
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B V 600B V"), 0.1),
				newConstantPredefinedTrial(toOps("300B V"), 0.2),
				newConstantPredefinedTrial(toOps("300B V"), 0.3),
				newConstantPredefinedTrial(toOps("900B V"), 0.4),
				newConstantPredefinedTrial(toOps("900B V"), 0.5),
			},
			config: expconf.SearcherConfig{
				AdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:            ptrs.IntPtr(2),
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B V 600B V"), 0.1),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.2),
				newConstantPredefinedTrial(toOps("300B V"), 0.3),
				newConstantPredefinedTrial(toOps("900B V"), 0.4),
				newConstantPredefinedTrial(toOps("900B V"), 0.5),
			},
			config: expconf.SearcherConfig{
				AdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     ptrs.BoolPtr(true),
					MaxLength:           expconf.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:            ptrs.IntPtr(2),
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B V 600B V"), 0.5),
				newConstantPredefinedTrial(toOps("300B V"), 0.4),
				newConstantPredefinedTrial(toOps("300B V"), 0.3),
				newConstantPredefinedTrial(toOps("900B V"), 0.2),
				newConstantPredefinedTrial(toOps("900B V"), 0.1),
			},
			config: expconf.SearcherConfig{
				AdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     ptrs.BoolPtr(false),
					MaxLength:           expconf.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:            ptrs.IntPtr(2),
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("300B V 600B V"), 0.5),
				newEarlyExitPredefinedTrial(toOps("300B"), 0.4),
				newConstantPredefinedTrial(toOps("300B V"), 0.3),
				newConstantPredefinedTrial(toOps("900B V"), 0.2),
				newConstantPredefinedTrial(toOps("900B V"), 0.1),
			},
			config: expconf.SearcherConfig{
				AdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     ptrs.BoolPtr(false),
					MaxLength:           expconf.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:            ptrs.IntPtr(2),
					Divisor:             ptrs.Float64Ptr(3),
					MaxConcurrentTrials: ptrs.IntPtr(maxConcurrentTrials),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
