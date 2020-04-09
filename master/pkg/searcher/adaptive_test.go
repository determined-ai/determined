package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
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
	conf := model.AdaptiveConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		TargetTrialSteps: 64, StepBudget: 1024, Divisor: 4, TrainStragglers: true,
		Mode: model.AggressiveMode, MaxRungs: 3,
	}
	gen := func() SearchMethod { return newAdaptiveSearch(conf) }
	checkReproducibility(t, gen, nil, defaultMetric)
}
