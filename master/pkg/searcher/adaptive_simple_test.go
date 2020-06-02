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
		toKinds("1S 1V"),
		toKinds("1S 1V"),
	}
	checkSimulation(t, newAdaptiveSimpleSearch(actual), nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	actual := model.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: true, MaxSteps: 1, MaxTrials: 1,
		Divisor: 4, Mode: model.AggressiveMode, MaxRungs: 3,
	}
	expected := [][]Kind{
		toKinds("1S 1V"),
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
		//nolint:dupl
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.02, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.03, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.04, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.05, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.06, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.07, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.08, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.09, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.10, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.11, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.12, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.13, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.14, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.15, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.16, 2, []int{2}, nil),

				newConstantPredefinedTrial(0.17, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.18, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.19, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.20, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.21, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.22, 8, []int{8}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					Mode:            model.StandardMode,
					MaxTrials:       22,
					MaxSteps:        32,
					MaxRungs:        3,
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.01, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.02, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.03, 8, []int{2, 8}, nil),
				newEarlyExitPredefinedTrial(0.04, 8, []int{2}, nil),
				newConstantPredefinedTrial(0.05, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.06, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.07, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.08, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.09, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.10, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.11, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.12, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.13, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.14, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.15, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.16, 2, []int{2}, nil),

				newConstantPredefinedTrial(0.17, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.18, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.19, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.20, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.21, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.22, 8, []int{8}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					Mode:            model.StandardMode,
					MaxTrials:       22,
					MaxSteps:        32,
					MaxRungs:        3,
				},
			},
		},
		//nolint:dupl
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.22, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.21, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.20, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.19, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.18, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.17, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.16, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.15, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.14, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.13, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.12, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.11, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.10, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.09, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.08, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.07, 2, []int{2}, nil),

				newConstantPredefinedTrial(0.06, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.05, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.04, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.03, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.02, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.01, 8, []int{8}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					Mode:            model.StandardMode,
					MaxTrials:       22,
					MaxSteps:        32,
					MaxRungs:        3,
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				newConstantPredefinedTrial(0.22, 32, []int{2, 8, 32}, nil),
				newConstantPredefinedTrial(0.21, 8, []int{2, 8}, nil),
				newConstantPredefinedTrial(0.20, 8, []int{2, 8}, nil),
				newEarlyExitPredefinedTrial(0.19, 8, []int{2}, nil),
				newConstantPredefinedTrial(0.18, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.17, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.16, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.15, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.14, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.13, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.12, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.11, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.10, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.09, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.08, 2, []int{2}, nil),
				newConstantPredefinedTrial(0.07, 2, []int{2}, nil),

				newConstantPredefinedTrial(0.06, 32, []int{8, 32}, nil),
				newConstantPredefinedTrial(0.05, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.04, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.03, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.02, 8, []int{8}, nil),
				newConstantPredefinedTrial(0.01, 8, []int{8}, nil),
			},
			config: model.SearcherConfig{
				AdaptiveSimpleConfig: &model.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					Mode:            model.StandardMode,
					MaxTrials:       22,
					MaxSteps:        32,
					MaxRungs:        3,
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
