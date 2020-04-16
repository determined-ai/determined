package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestBayesSearcher(t *testing.T) {
	actual := model.BayesConfig{
		Metric: defaultMetric, MaxTrials: 4, MaxSteps: 3, ConcurrentTrials: 1}
	expected := [][]Kind{
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
		{RunStep, RunStep, RunStep, ComputeValidationMetrics},
	}
	checkSimulation(t, newBayesSearch(actual),
		generateHyperparameters([]int{1}), ConstantValidation, expected)
}

func TestBayesSearcherReproducibility(t *testing.T) {
	conf := model.BayesConfig{Metric: defaultMetric, MaxTrials: 4, MaxSteps: 3, ConcurrentTrials: 1}
	gen := func() SearchMethod { return newBayesSearch(conf) }
	checkReproducibility(t, gen, generateHyperparameters([]int{1}), defaultMetric)
}
