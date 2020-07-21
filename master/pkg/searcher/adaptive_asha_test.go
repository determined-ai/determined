package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
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
	conf := model.AdaptiveASHAConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		TargetTrialSteps: 64, MaxTrials: 128, Divisor: 4,
		Mode: model.AggressiveMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveASHASearch(conf, defaultBatchesPerStep) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 5
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 9, []int{3, 9}, nil),
				newConstantPredefinedTrial(0.2, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.3, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.4, 9, []int{9}, nil),
				newConstantPredefinedTrial(0.5, 9, []int{9}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     true,
					TargetTrialSteps:    9,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 9, []int{3, 9}, nil),
				newEarlyExitPredefinedTrial(0.2, 3, nil, nil),
				newConstantPredefinedTrial(0.3, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.4, 9, []int{9}, nil),
				newConstantPredefinedTrial(0.5, 9, []int{9}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     true,
					TargetTrialSteps:    9,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.5, 9, []int{3, 9}, nil),
				newConstantPredefinedTrial(0.4, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.3, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.2, 9, []int{9}, nil),
				newConstantPredefinedTrial(0.1, 9, []int{9}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     false,
					TargetTrialSteps:    9,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.5, 9, []int{3, 9}, nil),
				newEarlyExitPredefinedTrial(0.4, 3, nil, nil),
				newConstantPredefinedTrial(0.3, 3, []int{3}, nil),
				newConstantPredefinedTrial(0.2, 9, []int{9}, nil),
				newConstantPredefinedTrial(0.1, 9, []int{9}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     false,
					TargetTrialSteps:    9,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
