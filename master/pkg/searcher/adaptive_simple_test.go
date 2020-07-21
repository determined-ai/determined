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
	expected := [][]Runnable{
		toOps("100B V"),
		toOps("25B V 75B V"),
		toOps("6B V 19B V 75B V"),
	}
	searchMethod := newAdaptiveSimpleSearch(actual)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(100), MaxTrials: 1,
		Divisor: 4, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	expected := [][]Runnable{
		toOps("6B V 19B V 75B V"),
	}
	searchMethod := newAdaptiveSimpleSearch(actual)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleSearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		MaxLength: model.NewLengthInBatches(6400), MaxTrials: 50,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveSimpleSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestAdaptiveSimpleSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("12B V 38B V 150B V 600B V 2400B V"), 0.1),
				newConstantPredefinedTrial(toOps("12B V"), 0.2),
				newConstantPredefinedTrial(toOps("12B V"), 0.3),

				newConstantPredefinedTrial(toOps("50B V 150B V 600B V 2400B V"), 0.4),
				newConstantPredefinedTrial(toOps("50B V"), 0.5),
				newConstantPredefinedTrial(toOps("50B V"), 0.6),

				newConstantPredefinedTrial(toOps("200B V 600B V 2400B V"), 0.7),
				newConstantPredefinedTrial(toOps("200B V"), 0.8),
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
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{

				newConstantPredefinedTrial(toOps("12B V 38B V 150B V 600B V 2400B V"), 0.1),
				newConstantPredefinedTrial(toOps("12B V"), 0.2),
				newConstantPredefinedTrial(toOps("12B V"), 0.3),

				newEarlyExitPredefinedTrial(toOps("50B"), 0.4),
				newConstantPredefinedTrial(toOps("50B V 150B V 600B V 2400B V"), 0.5),
				newConstantPredefinedTrial(toOps("50B V"), 0.6),

				newConstantPredefinedTrial(toOps("200B V 600B V 2400B V"), 0.7),
				newConstantPredefinedTrial(toOps("200B V"), 0.8),
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
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("12B V 38B V 150B V 600B V 2400B V"), 0.8),
				newConstantPredefinedTrial(toOps("12B V"), 0.7),
				newConstantPredefinedTrial(toOps("12B V"), 0.6),

				newConstantPredefinedTrial(toOps("50B V 150B V 600B V 2400B V"), 0.5),
				newConstantPredefinedTrial(toOps("50B V"), 0.4),
				newConstantPredefinedTrial(toOps("50B V"), 0.3),

				newConstantPredefinedTrial(toOps("200B V 600B V 2400B V"), 0.2),
				newConstantPredefinedTrial(toOps("200B V"), 0.1),
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
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(toOps("12B V 38B V 150B V 600B V 2400B V"), 0.8),
				newConstantPredefinedTrial(toOps("12B V"), 0.7),
				newConstantPredefinedTrial(toOps("12B V"), 0.6),

				newConstantPredefinedTrial(toOps("50B V 150B V 600B V 2400B V"), 0.5),
				newEarlyExitPredefinedTrial(toOps("50B"), 0.4),
				newConstantPredefinedTrial(toOps("50B V"), 0.3),

				newConstantPredefinedTrial(toOps("200B V 600B V 2400B V"), 0.2),
				newConstantPredefinedTrial(toOps("200B V"), 0.1),
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
		},
	}

	runValueSimulationTestCases(t, testCases)
}
