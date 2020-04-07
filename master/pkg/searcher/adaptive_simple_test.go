package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestAdaptiveSimpleConservativeCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 1, MaxTrials: 1,
		Divisor: 4, Mode: model.ConservativeMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V"),
		toKinds("1S 1V 1S 1V"),
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	checkSimulation(t, newAdaptiveSimpleSearch(actual), nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 1, MaxTrials: 1,
		Divisor: 4, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V 1S 1V 1S 1V"),
	}
	checkSimulation(t, newAdaptiveSimpleSearch(actual), nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleSearcherReproducibility(t *testing.T) {
	conf := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 64, MaxTrials: 50,
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
				newConstantPredefinedTrial(0.1, 32, []int{1, 2, 3, 4, 32}, nil),
				newConstantPredefinedTrial(0.2, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.4, 32, []int{1, 2, 3, 32}, nil),
				newConstantPredefinedTrial(0.5, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.7, 32, []int{1, 2, 32}, nil),
				newConstantPredefinedTrial(0.8, 1, []int{1}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxSteps:        32,
					MaxRungs:        5,
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.8, 32, []int{1, 2, 3, 4, 32}, nil),
				newConstantPredefinedTrial(0.7, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.6, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.5, 32, []int{1, 2, 3, 32}, nil),
				newConstantPredefinedTrial(0.4, 1, []int{1}, nil),
				newConstantPredefinedTrial(0.3, 1, []int{1}, nil),

				newConstantPredefinedTrial(0.2, 32, []int{1, 2, 32}, nil),
				newConstantPredefinedTrial(0.1, 1, []int{1}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					Mode:            model.StandardMode,
					MaxTrials:       8,
					MaxSteps:        32,
					MaxRungs:        5,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
