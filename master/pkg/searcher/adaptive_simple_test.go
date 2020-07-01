package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAdaptiveSimpleConservativeCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(100), MaxTrials: 1,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V"),
		toKinds("1S 1V 1S 1V"),
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	searchMethod := newAdaptiveSimpleSearch(actual, defaultBatchesPerStep, 0)
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(100), MaxTrials: 1,
		Divisor: 4, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	searchMethod := newAdaptiveSimpleSearch(actual, defaultBatchesPerStep, 0)
	checkSimulation(t, searchMethod, defaultHyperparameters(), ConstantValidation, expected)
}

func TestAdaptiveSimpleSearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(6400), MaxTrials: 50,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	gen := func() SearchMethod {
		return newAdaptiveSimpleSearch(conf, defaultBatchesPerStep, 0)
	}
	checkReproducibility(t, gen, defaultHyperparameters(), defaultMetric)
}

func TestAdaptiveSimpleSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 34, []int{1, 2, 4, 10, 34}, nil),
				newConstantPredefinedTrial(0.2, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.4, 33, []int{1, 3, 9, 33}, nil),
				newConstantPredefinedTrial(0.5, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.7, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.8, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxLength:       model.NewLengthInBatches(3200),
					MaxRungs:        5,
					Divisor:         4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 34, []int{1, 2, 4, 10, 34}, nil),
				newConstantPredefinedTrial(0.2, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newEarlyExitPredefinedTrial(0.4, 1, nil, nil),
				newConstantPredefinedTrial(0.5, 33, []int{1, 3, 9, 33}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.7, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.8, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxLength:       model.NewLengthInBatches(3200),
					MaxRungs:        5,
					Divisor:         4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.8, 34, []int{1, 2, 4, 10, 34}, nil),
				newConstantPredefinedTrial(0.7, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.5, 33, []int{1, 3, 9, 33}, nil),
				newConstantPredefinedTrial(0.4, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.2, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.1, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxLength:       model.NewLengthInBatches(3200),
					MaxRungs:        5,
					Divisor:         4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.8, 34, []int{1, 2, 4, 10, 34}, nil),
				newConstantPredefinedTrial(0.7, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.5, 33, []int{1, 3, 9, 33}, nil),
				newEarlyExitPredefinedTrial(0.4, 1, nil, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.2, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.1, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxLength:       model.NewLengthInBatches(3200),
					MaxRungs:        5,
					Divisor:         4,
				},
			},
			hparams:         defaultHyperparameters(),
			batchesPerStep:  defaultBatchesPerStep,
			recordsPerEpoch: 0,
		},
	}

	runValueSimulationTestCases(t, testCases)
}
