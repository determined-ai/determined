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
	expectedTrials := []predefinedTrial{
		newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.1),
		newConstantPredefinedTrial(toOps("1000B V"), 0.2),
		newConstantPredefinedTrial(toOps("1000B V"), 0.3),

		newConstantPredefinedTrial(toOps("1000B V"), 0.3),
		newConstantPredefinedTrial(toOps("1000B V"), 0.2),
		newConstantPredefinedTrial(toOps("1000B V 2000B V"), 0.1),
	}

	adaptiveConfig1 := model.SearcherConfig{
		AsyncHalvingConfig: &model.AsyncHalvingConfig{
			Metric:          "error",
			NumRungs:        3,
			SmallerIsBetter: true,
			MaxLength:       model.NewLengthInBatches(9000),
			MaxTrials:       3,
			Divisor:         3,
		},
	}
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1)

	adaptiveConfig2 := model.SearcherConfig{
		AsyncHalvingConfig: &model.AsyncHalvingConfig{
			Metric:          "error",
			NumRungs:        3,
			SmallerIsBetter: true,
			MaxLength:       model.NewLengthInBatches(9000),
			MaxTrials:       3,
			Divisor:         3,
		},
	}
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2)

	params := model.Hyperparameters{}

	method := newTournamentSearch(AdaptiveSearch, adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, params, expectedTrials)
	assert.NilError(t, err)
}
