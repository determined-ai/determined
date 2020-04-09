package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestRandomSearcher(t *testing.T) {
	actual := model.RandomConfig{MaxTrials: 4, MaxSteps: 3}
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, newRandomSearch(actual), nil, ConstantValidation, expected)
}

func TestRandomSearcherReproducibility(t *testing.T) {
	conf := model.RandomConfig{MaxTrials: 4, MaxSteps: 3}
	gen := func() SearchMethod { return newRandomSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}
