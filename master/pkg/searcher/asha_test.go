package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestASHASearcher(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		TargetTrialSteps: 90,
		Divisor:          3,
		MaxTrials:        12,
	}
	expected := [][]Kind{
		toKinds("10S 1V"), toKinds("10S 1V"), toKinds("10S 1V"),
		toKinds("10S 1V"), toKinds("10S 1V"), toKinds("10S 1V"),
		toKinds("10S 1V"), toKinds("10S 1V"),
		toKinds("10S 1V 20S 1V"),
		toKinds("10S 1V 20S 1V"),
		toKinds("10S 1V 20S 1V"),
		toKinds("10S 1V 20S 1V 60S 1V"),
	}
	searchMethod := newAsyncHalvingSearch(actual, defaultBatchesPerStep)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 90, []int{10, 30, 90}, nil),
				newConstantPredefinedTrial(0.02, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.03, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.04, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.05, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.06, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.07, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.08, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.09, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.10, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.11, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.12, 10, []int{10}, nil),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					TargetTrialSteps:    90,
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 90, []int{10, 30, 90}, nil),
				newConstantPredefinedTrial(0.02, 30, []int{10, 30}, nil),
				newEarlyExitPredefinedTrial(0.03, 30, []int{10}, nil),
				newConstantPredefinedTrial(0.04, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.05, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.06, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.07, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.08, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.09, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.10, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.11, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.12, 10, []int{10}, nil),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					TargetTrialSteps:    90,
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.12, 90, []int{10, 30, 90}, nil),
				newConstantPredefinedTrial(0.11, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.10, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.09, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.08, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.07, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.06, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.05, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.04, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.03, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.02, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.01, 10, []int{10}, nil),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     false,
					TargetTrialSteps:    90,
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.12, 90, []int{10, 30, 90}, nil),
				newConstantPredefinedTrial(0.11, 30, []int{10, 30}, nil),
				newEarlyExitPredefinedTrial(0.10, 30, []int{10}, nil),
				newConstantPredefinedTrial(0.09, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.08, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.07, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.06, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.05, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.04, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.03, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.02, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.01, 10, []int{10}, nil),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     false,
					TargetTrialSteps:    90,
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
		{
			name: "async promotions",
			expectedTrials: []predefinedTrial{
				// The first trial is promoted due to asynchronous
				// promotions despite being below top 1/3 of trials in
				// base rung.
				newConstantPredefinedTrial(0.10, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.11, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.12, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.01, 90, []int{10, 30, 90}, nil),
				newConstantPredefinedTrial(0.02, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.03, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.04, 30, []int{10, 30}, nil),
				newConstantPredefinedTrial(0.05, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.06, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.07, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.08, 10, []int{10}, nil),
				newConstantPredefinedTrial(0.09, 10, []int{10}, nil),
			},
			config: model.SearcherConfig{
				AsyncHalvingConfig: &model.AsyncHalvingConfig{
					Metric:              "error",
					NumRungs:            3,
					SmallerIsBetter:     true,
					TargetTrialSteps:    90,
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
