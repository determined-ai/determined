package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestAdaptiveSimpleConservativeCornerCase(t *testing.T) {
	config := expconf.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		MaxLength: expconf.NewLengthInBatches(100),
		MaxTrials: 1,
		Divisor:   ptrs.Float64Ptr(4),
		Mode:      expconf.AdaptiveModePtr(expconf.ConservativeMode),
		MaxRungs:  ptrs.IntPtr(3),
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("100B V"),
		toOps("25B V 75B V"),
		toOps("6B V 19B V 75B V"),
	}
	searchMethod := newAdaptiveSimpleSearch(config)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleAggressiveCornerCase(t *testing.T) {
	config := expconf.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		MaxLength: expconf.NewLengthInBatches(100), MaxTrials: 1,
		Divisor:  ptrs.Float64Ptr(4),
		Mode:     expconf.AdaptiveModePtr(expconf.AggressiveMode),
		MaxRungs: ptrs.IntPtr(3),
	}
	schemas.FillDefaults(&config)
	expected := [][]Runnable{
		toOps("6B V 19B V 75B V"),
	}
	searchMethod := newAdaptiveSimpleSearch(config)
	checkSimulation(t, searchMethod, nil, ConstantValidation, expected)
}

func TestAdaptiveSimpleSearcherReproducibility(t *testing.T) {
	config := expconf.AdaptiveSimpleConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		MaxLength: expconf.NewLengthInBatches(6400), MaxTrials: 50,
		Divisor:  ptrs.Float64Ptr(4),
		Mode:     expconf.AdaptiveModePtr(expconf.ConservativeMode),
		MaxRungs: ptrs.IntPtr(3),
	}
	schemas.FillDefaults(&config)
	gen := func() SearchMethod { return newAdaptiveSimpleSearch(config) }
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
			config: expconf.SearcherConfig{
				AdaptiveSimpleConfig: &expconf.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxTrials:       8,
					MaxLength:       expconf.NewLengthInBatches(3200),
					MaxRungs:        ptrs.IntPtr(5),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveSimpleConfig: &expconf.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxTrials:       8,
					MaxLength:       expconf.NewLengthInBatches(3200),
					MaxRungs:        ptrs.IntPtr(5),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveSimpleConfig: &expconf.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxTrials:       8,
					MaxLength:       expconf.NewLengthInBatches(3200),
					MaxRungs:        ptrs.IntPtr(5),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveSimpleConfig: &expconf.AdaptiveSimpleConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxTrials:       8,
					MaxLength:       expconf.NewLengthInBatches(3200),
					MaxRungs:        ptrs.IntPtr(5),
					Divisor:         ptrs.Float64Ptr(4),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
