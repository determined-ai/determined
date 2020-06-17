package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAdaptiveASHASearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveASHAConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		TargetTrialSteps: 64, MaxTrials: 128, Divisor: 4,
		Mode: model.AggressiveMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveASHASearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 5
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.4, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.5, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     true,
					TargetTrialSteps:    32,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             4,
					MaxConcurrentTrials: &maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
				newEarlyExitPredefinedTrial(0.2, 8, nil, nil),
				newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.4, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.5, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     true,
					TargetTrialSteps:    32,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             4,
					MaxConcurrentTrials: &maxConcurrentTrials,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.5, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.4, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     false,
					TargetTrialSteps:    32,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             4,
					MaxConcurrentTrials: &maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.5, 32, []int{8, 32}, nil),
				newEarlyExitPredefinedTrial(0.4, 8, nil, nil),
				newConstantPredefinedTrial(0.3, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveASHAConfig: &model.AdaptiveASHAConfig{
					Metric:              "error",
					SmallerIsBetter:     false,
					TargetTrialSteps:    32,
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
					Divisor:             4,
					MaxConcurrentTrials: &maxConcurrentTrials,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
