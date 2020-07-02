package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestASHASearcherRecords(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInRecords(576000),
		Divisor:   3,
		MaxTrials: 12,
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
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestASHASearcherBatches(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInBatches(9000),
		Divisor:   3,
		MaxTrials: 12,
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
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestASHASearcherEpochs(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric: defaultMetric, NumRungs: 3,
		MaxLength: model.NewLengthInEpochs(12),
		Divisor:   3,
		MaxTrials: 12,
	}
	expected := [][]Kind{
		toKinds("8S 1V"), toKinds("8S 1V"), toKinds("8S 1V"),
		toKinds("8S 1V"), toKinds("8S 1V"), toKinds("8S 1V"),
		toKinds("8S 1V"), toKinds("8S 1V"),
		toKinds("8S 1V 23S 1V"),
		toKinds("8S 1V 23S 1V"),
		toKinds("8S 1V 23S 1V"),
		toKinds("8S 1V 23S 1V 60S 1V"),
	}
	searchMethod := newAsyncHalvingSearch(actual, defaultBatchesPerStep)
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected, 48000)
}

func TestASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 3
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			kind: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "early exit -- smaller is better",
			kind: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "smaller is not better",
			kind: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "early exit -- smaller is not better",
			kind: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(9000),
					MaxTrials:           12,
					Divisor:             3,
					MaxConcurrentTrials: maxConcurrentTrials,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
	}

	runValueSimulationTestCases(t, testCases)
}
