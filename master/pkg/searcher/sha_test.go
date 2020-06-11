package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestSHASearcher(t *testing.T) {
	actual := model.SyncHalvingConfig{
		Metric:           defaultMetric,
		NumRungs:         4,
		TargetTrialSteps: 800,
		StepBudget:       480,
		Divisor:          4,
		TrainStragglers:  true,
	}
	expected := [][]Kind{
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V 38S 1V"),
		toKinds("12S 1V 38S 1V 150S 1V 600S 1V"),
	}
	checkSimulation(t, newSyncHalvingSearch(actual, defaultBatchesPerStep), nil, ConstantValidation, expected)
}

func TestSHASearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 800, []int{12, 50, 200, 800}, nil),
				newConstantPredefinedTrial(0.02, 50, []int{12, 50}, nil),
				newConstantPredefinedTrial(0.03, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.04, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.05, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.06, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.07, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.08, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.09, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.10, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.11, 12, []int{12}, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:           "error",
					NumRungs:         4,
					SmallerIsBetter:  true,
					TargetTrialSteps: 800,
					StepBudget:       480,
					Divisor:          4,
					TrainStragglers:  true,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 800, []int{12, 50, 200, 800}, nil),
				newEarlyExitPredefinedTrial(0.02, 50, []int{12}, nil),
				newConstantPredefinedTrial(0.03, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.04, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.05, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.06, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.07, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.08, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.09, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.10, 12, []int{12}, nil),
				newEarlyExitPredefinedTrial(0.11, 11, nil, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:           "error",
					NumRungs:         4,
					SmallerIsBetter:  true,
					TargetTrialSteps: 800,
					StepBudget:       480,
					Divisor:          4,
					TrainStragglers:  true,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.11, 800, []int{12, 50, 200, 800}, nil),
				newConstantPredefinedTrial(0.10, 50, []int{12, 50}, nil),
				newConstantPredefinedTrial(0.09, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.08, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.07, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.06, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.05, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.04, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.03, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.02, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.01, 12, []int{12}, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:           "error",
					NumRungs:         4,
					SmallerIsBetter:  false,
					TargetTrialSteps: 800,
					StepBudget:       480,
					Divisor:          4,
					TrainStragglers:  true,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.11, 800, []int{12, 50, 200, 800}, nil),
				newEarlyExitPredefinedTrial(0.10, 50, []int{12}, nil),
				newConstantPredefinedTrial(0.09, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.08, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.07, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.06, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.05, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.04, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.03, 12, []int{12}, nil),
				newConstantPredefinedTrial(0.02, 12, []int{12}, nil),
				newEarlyExitPredefinedTrial(0.01, 11, nil, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:           "error",
					NumRungs:         4,
					SmallerIsBetter:  false,
					TargetTrialSteps: 800,
					StepBudget:       480,
					Divisor:          4,
					TrainStragglers:  true,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
