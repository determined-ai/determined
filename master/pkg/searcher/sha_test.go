package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestSHASearcherWithRecords(t *testing.T) {
	actual := model.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       model.NewLengthInRecords(5120050),
		Budget:          model.NewLengthInRecords(3072050),
		Divisor:         4,
		TrainStragglers: true,
	}
	expected := [][]Kind{
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V 38S 1V"),
		toKinds("13S 1V 38S 1V 150S 1V 600S 1V"),
	}
	searchMethod := newSyncHalvingSearch(actual)
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestSHASearcherWithBatches(t *testing.T) {
	actual := model.SyncHalvingConfig{
		Metric:          defaultMetric,
		NumRungs:        4,
		MaxLength:       model.NewLengthInBatches(80000),
		Budget:          model.NewLengthInBatches(48000),
		Divisor:         4,
		TrainStragglers: true,
	}
	expected := [][]Kind{
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V"), toKinds("13S 1V"), toKinds("13S 1V"),
		toKinds("13S 1V 38S 1V"),
		toKinds("13S 1V 38S 1V 150S 1V 600S 1V"),
	}
	searchMethod := newSyncHalvingSearch(actual)
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected, 0)
}

func TestSHASearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			kind: model.Batches,
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 801, []int{13, 51, 201, 801}, nil),
				newConstantPredefinedTrial(0.02, 51, []int{13, 51}, nil),
				newConstantPredefinedTrial(0.03, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.04, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.05, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.06, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.07, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.08, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.09, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.10, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.11, 13, []int{13}, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
				newConstantPredefinedTrial(0.01, 801, []int{13, 51, 201, 801}, nil),
				newEarlyExitPredefinedTrial(0.02, 50, []int{13}, nil),
				newConstantPredefinedTrial(0.03, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.04, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.05, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.06, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.07, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.08, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.09, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.10, 13, []int{13}, nil),
				newEarlyExitPredefinedTrial(0.11, 11, nil, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
				newConstantPredefinedTrial(0.11, 801, []int{13, 51, 201, 801}, nil),
				newConstantPredefinedTrial(0.10, 51, []int{13, 51}, nil),
				newConstantPredefinedTrial(0.09, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.08, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.07, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.06, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.05, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.04, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.03, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.02, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.01, 13, []int{13}, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
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
				newConstantPredefinedTrial(0.11, 801, []int{13, 51, 201, 801}, nil),
				newEarlyExitPredefinedTrial(0.10, 50, []int{13}, nil),
				newConstantPredefinedTrial(0.09, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.08, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.07, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.06, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.05, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.04, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.03, 13, []int{13}, nil),
				newConstantPredefinedTrial(0.02, 13, []int{13}, nil),
				newEarlyExitPredefinedTrial(0.01, 11, nil, nil),
			},
			config: model.SearcherConfig{
				SyncHalvingConfig: &model.SyncHalvingConfig{
					Metric:          "error",
					NumRungs:        4,
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(80000),
					Budget:          model.NewLengthInBatches(48000),
					Divisor:         4,
					TrainStragglers: true,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
	}

	runValueSimulationTestCases(t, testCases)
}
