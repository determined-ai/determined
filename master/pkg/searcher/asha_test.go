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
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}

func TestASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 1
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
					Metric:           "error",
					NumRungs:         3,
					SmallerIsBetter:  true,
					TargetTrialSteps: 90,
					MaxTrials:        12,
					Divisor:          3,
				},
			},
		},
		{
			name: "smaller is not better",
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
					Metric:           "error",
					NumRungs:         3,
					SmallerIsBetter:  false,
					TargetTrialSteps: 90,
					MaxTrials:        12,
					Divisor:          3,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
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
					Metric:           "error",
					NumRungs:         3,
					SmallerIsBetter:  false,
					TargetTrialSteps: 90,
					MaxTrials:        12,
					Divisor:          3,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
