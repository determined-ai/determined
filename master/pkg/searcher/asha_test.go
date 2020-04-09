package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestASHASearcher(t *testing.T) {
	actual := model.AsyncHalvingConfig{
		Metric:           defaultMetric,
		NumRungs:         4,
		TargetTrialSteps: 800,
		StepBudget:       480,
		Divisor:          4,
		TrainStragglers:  true,
	}
	expected := [][]Kind{
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V"), toKinds("12S 1V"), toKinds("12S 1V"),
		toKinds("12S 1V 38S 1V"),
		toKinds("12S 1V 38S 1V 150S 1V 600S 1V"),
	}
	checkSimulation(t, newAsyncHalvingSearch(actual), nil, ConstantValidation, expected)
}
