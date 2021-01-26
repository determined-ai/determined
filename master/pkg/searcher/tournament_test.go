package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRandomTournamentSearcher(t *testing.T) {
	subConfig1 := expconf.RandomConfig{
		MaxTrials: 2,
		MaxLength: expconf.NewLengthInBatches(300),
	}
	schemas.FillDefaults(&subConfig1)
	subConfig2 := expconf.RandomConfig{
		MaxTrials: 3,
		MaxLength: expconf.NewLengthInBatches(200),
	}
	schemas.FillDefaults(&subConfig2)

	search := newTournamentSearch(
		newRandomSearch(subConfig1),
		newRandomSearch(subConfig2),
	)
	schemas.FillDefaults(&search)
	expected := [][]Runnable{
		toOps("300B V"),
		toOps("300B V"),
		toOps("200B V"),
		toOps("200B V"),
		toOps("200B V"),
	}
	checkSimulation(t, search, nil, ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	config := expconf.RandomConfig{MaxTrials: 5, MaxLength: expconf.NewLengthInBatches(800)}
	schemas.FillDefaults(&config)
	gen := func() SearchMethod {
		return newTournamentSearch(
			newRandomSearch(config),
			newRandomSearch(config),
		)
	}
	checkReproducibility(t, gen, nil, defaultMetric)
}

func TestTournamentSearchMethod(t *testing.T) {
	// Run both of the tests from adaptive_test.go side by side.
	expectedTrials := []predefinedTrial{
		newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.1),
		newConstantPredefinedTrial(toOps("800B V"), 0.2),
		newConstantPredefinedTrial(toOps("3200B V"), 0.3),

		newConstantPredefinedTrial(toOps("800B V 2400B V"), 0.3),
		newConstantPredefinedTrial(toOps("800B V"), 0.2),
		newConstantPredefinedTrial(toOps("3200B V"), 0.1),
	}

	adaptiveConfig1 := expconf.SearcherConfig{
		AdaptiveConfig: &expconf.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: ptrs.BoolPtr(true),
			MaxLength:       expconf.NewLengthInBatches(3200),
			Budget:          expconf.NewLengthInBatches(6400),
			Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
			MaxRungs:        ptrs.IntPtr(2),
			Divisor:         ptrs.Float64Ptr(4),
		},
	}
	schemas.FillDefaults(&adaptiveConfig1)
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1)

	adaptiveConfig2 := expconf.SearcherConfig{
		AdaptiveConfig: &expconf.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: ptrs.BoolPtr(false),
			MaxLength:       expconf.NewLengthInBatches(3200),
			Budget:          expconf.NewLengthInBatches(6400),
			Mode:            expconf.AdaptiveModePtr(expconf.StandardMode),
			MaxRungs:        ptrs.IntPtr(2),
			Divisor:         ptrs.Float64Ptr(4),
		},
	}
	schemas.FillDefaults(&adaptiveConfig2)
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2)

	params := expconf.Hyperparameters{}

	method := newTournamentSearch(adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, params, expectedTrials)
	assert.NilError(t, err)
}
