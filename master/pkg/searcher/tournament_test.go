package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
)

const RandomTournamentSearch SearchMethodType = "random_tournament"

func TestRandomTournamentSearcher(t *testing.T) {
	actual := newTournamentSearch(
		RandomTournamentSearch,
		newRandomSearch(model.RandomConfig{
			MaxTrials: 2,
			MaxLength: model.NewLengthInBatches(300),
		}),
		newRandomSearch(model.RandomConfig{
			MaxTrials: 3,
			MaxLength: model.NewLengthInBatches(200),
		}),
	)
	expected := [][]Runnable{
		toOps("300B V"),
		toOps("300B V"),
		toOps("200B V"),
		toOps("200B V"),
		toOps("200B V"),
	}
	checkSimulation(t, actual, nil, ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 5, MaxLength: model.NewLengthInBatches(800)}
	gen := func() SearchMethod {
		return newTournamentSearch(
			RandomTournamentSearch,
			newRandomSearch(conf),
			newRandomSearch(conf),
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

	adaptiveConfig1 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: true,
			MaxLength:       model.NewLengthInBatches(3200),
			Budget:          model.NewLengthInBatches(6400),
			Mode:            model.StandardMode,
			MaxRungs:        2,
			Divisor:         4,
		},
	}
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1)

	adaptiveConfig2 := model.SearcherConfig{
		AdaptiveConfig: &model.AdaptiveConfig{
			Metric:          "error",
			SmallerIsBetter: false,
			MaxLength:       model.NewLengthInBatches(3200),
			Budget:          model.NewLengthInBatches(6400),
			Mode:            model.StandardMode,
			MaxRungs:        2,
			Divisor:         4,
		},
	}
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2)

	params := model.Hyperparameters{}

	method := newTournamentSearch(AdaptiveSearch, adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, params, expectedTrials)
	assert.NilError(t, err)
}
