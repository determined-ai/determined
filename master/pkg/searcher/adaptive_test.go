package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestConservativeMode(t *testing.T) {
	assert.DeepEqual(t, conservativeMode(1), []int{1})
	assert.DeepEqual(t, conservativeMode(2), []int{1, 2})
	assert.DeepEqual(t, conservativeMode(3), []int{1, 2, 3})
	assert.DeepEqual(t, conservativeMode(4), []int{1, 2, 3, 4})
	assert.DeepEqual(t, conservativeMode(5), []int{1, 2, 3, 4, 5})
}

func TestStandardMode(t *testing.T) {
	assert.DeepEqual(t, standardMode(1), []int{1})
	assert.DeepEqual(t, standardMode(2), []int{1, 2})
	assert.DeepEqual(t, standardMode(3), []int{2, 3})
	assert.DeepEqual(t, standardMode(4), []int{2, 3, 4})
	assert.DeepEqual(t, standardMode(5), []int{3, 4, 5})
}

func TestAggressiveMode(t *testing.T) {
	assert.DeepEqual(t, aggressiveMode(1), []int{1})
	assert.DeepEqual(t, aggressiveMode(2), []int{2})
	assert.DeepEqual(t, aggressiveMode(3), []int{3})
	assert.DeepEqual(t, aggressiveMode(4), []int{4})
	assert.DeepEqual(t, aggressiveMode(5), []int{5})
}

func TestAdaptiveSearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(6400), Budget: model.NewLengthInBatches(102400),
		Divisor: 4, TrainStragglers: true, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveSearch(conf) }
	checkReproducibility(t, gen, defaultHyperparameters(), defaultMetric)
}

func TestAdaptiveSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			kind: model.Batches,
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.3, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveConfig: &model.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(3200),
					Budget:          model.NewLengthInBatches(6400),
					Mode:            model.StandardMode,
					MaxRungs:        2,
					Divisor:         4,
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
				newConstantPredefinedTrial(0.1, 32, []int{8, 32}, nil),
				newEarlyExitPredefinedTrial(0.2, 8, nil, nil),
				newConstantPredefinedTrial(0.3, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveConfig: &model.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					MaxLength:       model.NewLengthInBatches(3200),
					Budget:          model.NewLengthInBatches(6400),
					Mode:            model.StandardMode,
					MaxRungs:        2,
					Divisor:         4,
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
				newConstantPredefinedTrial(0.3, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.2, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveConfig: &model.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(3200),
					Budget:          model.NewLengthInBatches(6400),
					Mode:            model.StandardMode,
					MaxRungs:        2,
					Divisor:         4,
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
				newConstantPredefinedTrial(0.3, 32, []int{8, 32}, nil),
				newEarlyExitPredefinedTrial(0.2, 8, nil, nil),
				newConstantPredefinedTrial(0.1, 32, []int{32}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveConfig: &model.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					MaxLength:       model.NewLengthInBatches(3200),
					Budget:          model.NewLengthInBatches(6400),
					Mode:            model.StandardMode,
					MaxRungs:        2,
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
