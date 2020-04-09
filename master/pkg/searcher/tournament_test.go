package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomTournamentSearcher(t *testing.T) {
	actual := newTournamentSearch(
		newRandomSearch(model.RandomConfig{MaxTrials: 2, MaxSteps: 3}),
		newRandomSearch(model.RandomConfig{MaxTrials: 3, MaxSteps: 2}),
	)
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, actual, nil, ConstantValidation, expected)
}

func TestRandomTournamentSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 5, MaxSteps: 8}
	gen := func() SearchMethod {
		return newTournamentSearch(newRandomSearch(conf), newRandomSearch(conf))
	}
	checkReproducibility(t, gen, nil, defaultMetric)
}
