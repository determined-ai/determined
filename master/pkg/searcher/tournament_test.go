//nolint:exhaustivestruct
package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const RandomTournamentSearch SearchMethodType = "random_tournament"
const MetricName string = "tournamentMetric"

func TestRandomTournamentSearcher(t *testing.T) {
	actual := newTournamentSearch(
		RandomTournamentSearch,
		newRandomSearch(schemas.WithDefaults(expconf.RandomConfig{
			RawMaxTrials: ptrs.Ptr(2),
			RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(300)),
		})),
		newRandomSearch(schemas.WithDefaults(expconf.RandomConfig{
			RawMaxTrials: ptrs.Ptr(3),
			RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(200)),
		})),
	)
	expected := [][]ValidateAfter{
		toOps("300B"),
		toOps("300B"),
		toOps("200B"),
		toOps("200B"),
		toOps("200B"),
	}
	checkSimulation(t, actual, nil, ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	conf := expconf.RandomConfig{
		RawMaxTrials: ptrs.Ptr(5), RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(800)),
	}
	conf = schemas.WithDefaults(conf)
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
		newConstantPredefinedTrial(toOps("1000B 3000B"), 0.1),
		newConstantPredefinedTrial(toOps("1000B"), 0.2),
		newConstantPredefinedTrial(toOps("1000B"), 0.3),

		newConstantPredefinedTrial(toOps("1000B"), 0.3),
		newConstantPredefinedTrial(toOps("1000B"), 0.2),
		newConstantPredefinedTrial(toOps("1000B 3000B"), 0.1),
	}

	adaptiveConfig1 := expconf.SearcherConfig{
		RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
			RawNumRungs:  ptrs.Ptr(3),
			RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
			RawMaxTrials: ptrs.Ptr(3),
			RawDivisor:   ptrs.Ptr[float64](3),
		},
		RawMetric: ptrs.Ptr(MetricName),
	}
	adaptiveConfig1 = schemas.WithDefaults(adaptiveConfig1)
	adaptiveMethod1 := NewSearchMethod(adaptiveConfig1)

	adaptiveConfig2 := expconf.SearcherConfig{
		RawAsyncHalvingConfig: &expconf.AsyncHalvingConfig{
			RawNumRungs:  ptrs.Ptr(3),
			RawMaxLength: ptrs.Ptr(expconf.NewLengthInBatches(9000)),
			RawMaxTrials: ptrs.Ptr(3),
			RawDivisor:   ptrs.Ptr[float64](3),
		},
		RawMetric: ptrs.Ptr(MetricName),
	}
	adaptiveConfig2 = schemas.WithDefaults(adaptiveConfig2)
	adaptiveMethod2 := NewSearchMethod(adaptiveConfig2)

	params := expconf.Hyperparameters{}

	method := newTournamentSearch(AdaptiveSearch, adaptiveMethod1, adaptiveMethod2)

	err := checkValueSimulation(t, method, params, expectedTrials, MetricName)
	assert.NilError(t, err)
}
