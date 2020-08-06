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
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.1),
				newConstantPredefinedTrial(toOps("800B V"), 0.2),
				newConstantPredefinedTrial(toOps("3200B V"), 0.3),
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
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.1),
				newEarlyExitPredefinedTrial(toOps("800B"), 0.2),
				newConstantPredefinedTrial(toOps("3200B V"), 0.3),
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
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.3),
				newConstantPredefinedTrial(toOps("800B V"), 0.2),
				newConstantPredefinedTrial(toOps("3200B V"), 0.1),
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
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.1),
				newEarlyExitPredefinedTrial(toOps("800B"), 0.2),
				newConstantPredefinedTrial(toOps("3200B V"), 0.3),
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
		},
	}

	runValueSimulationTestCases(t, testCases)
}
