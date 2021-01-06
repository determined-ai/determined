package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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
	config := expconf.AdaptiveConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		MaxLength: expconf.NewLengthInBatches(6400), Budget: expconf.NewLengthInBatches(102400),
		Divisor: ptrs.Float64Ptr(4), TrainStragglers: ptrs.BoolPtr(true), Mode: expconf.AdaptiveModePtr(expconf.AggressiveMode), MaxRungs: ptrs.IntPtr(3),
	}
	schemas.FillDefaults(&config)
	gen := func() SearchMethod { return newAdaptiveSearch(config) }
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
			config: expconf.SearcherConfig{
				AdaptiveConfig: &expconf.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					MaxLength:       expconf.NewLengthInBatches(3200),
					Budget:          expconf.NewLengthInBatches(6400),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:        ptrs.IntPtr(2),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveConfig: &expconf.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					MaxLength:       expconf.NewLengthInBatches(3200),
					Budget:          expconf.NewLengthInBatches(6400),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:        ptrs.IntPtr(2),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveConfig: &expconf.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					MaxLength:       expconf.NewLengthInBatches(3200),
					Budget:          expconf.NewLengthInBatches(6400),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:        ptrs.IntPtr(2),
					Divisor:         ptrs.Float64Ptr(4),
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
			config: expconf.SearcherConfig{
				AdaptiveConfig: &expconf.AdaptiveConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					MaxLength:       expconf.NewLengthInBatches(3200),
					Budget:          expconf.NewLengthInBatches(6400),
					Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
					MaxRungs:        ptrs.IntPtr(2),
					Divisor:         ptrs.Float64Ptr(4),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
