package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAdaptiveASHASearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveASHAConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(6400), MaxTrials: 128, Divisor: 4,
		Mode: model.AggressiveMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveASHASearch(conf) }
	checkReproducibility(t, gen, defaultHyperparameters(), defaultMetric)
}

func TestAdaptiveASHASearchMethod(t *testing.T) {
	maxConcurrentTrials := 5
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			unit: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
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
			unit: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
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
			unit: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
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
			unit: model.Batches,
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
					MaxLength:           model.NewLengthInBatches(900),
					MaxTrials:           5,
					Mode:                model.StandardMode,
					MaxRungs:            2,
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
