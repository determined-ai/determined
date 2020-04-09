package searcher

import (
	"math/rand"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"
	"gotest.tools/assert/cmp"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

const defaultMetric = "metric"

type validatedTrial struct {
	validation float64
	steps      int
}

func CheckValidationFunction(trials []validatedTrial) ValidationFunction {
	return func(_ *rand.Rand, trialID, stepID int) float64 {
		trial := trials[trialID-1]
		check.Panic(check.LessThanOrEqualTo(stepID, trial.steps))
		return trial.validation
	}
}

func isExpected(actual, expected []Kind) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, v := range actual {
		if v != expected[i] {
			return false
		}
	}
	return true
}

func checkSimulation(
	t *testing.T,
	method SearchMethod,
	params model.Hyperparameters,
	validation ValidationFunction,
	expected [][]Kind,
) {
	search := NewSearcher(0, method, params)
	actual, err := Simulate(search, new(int64), validation, true, defaultMetric)
	assert.NilError(t, err)

	assert.Equal(t, len(actual.Results), len(expected))
	for _, actualTrial := range actual.Results {
		actualKinds := make([]Kind, 0, len(actualTrial))
		for _, msg := range actualTrial {
			actualKinds = append(actualKinds, msg.Workload.Kind)
		}
		found := false
		for i, expectedTrial := range expected {
			if isExpected(actualKinds, expectedTrial) {
				expected = append(expected[:i], expected[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected trial %+v not in %+v", actualKinds, expected)
		}
	}
}

// checkReproducibility creates two searchers with the same seed and the given config, simulates
// them, and checks that they produce the same trials and the same sequence of workloads for each
// trial.
func checkReproducibility(
	t assert.TestingT, methodGen func() SearchMethod, hparams model.Hyperparameters, metric string,
) {
	seed := int64(17)
	searcher1 := NewSearcher(uint32(seed), methodGen(), hparams)
	searcher2 := NewSearcher(uint32(seed), methodGen(), hparams)

	results1, err1 := Simulate(searcher1, &seed, ConstantValidation, true, metric)
	assert.NilError(t, err1)
	results2, err2 := Simulate(searcher2, &seed, ConstantValidation, true, metric)
	assert.NilError(t, err2)

	assert.Equal(t, len(results1.Results), len(results2.Results),
		"searchers had different number of trials")
	for requestID := range results1.Results {
		w1 := results1.Results[requestID]
		w2 := results2.Results[requestID]

		assert.Equal(t, len(w1), len(w2), "trial had different numbers of workloads between searchers")
		for i := range w1 {
			// We want to ignore the start and end time fields, so check the rest individually.
			assert.Equal(t, w1[i].Type, w2[i].Type, "message type differed between searchers")
			assert.Assert(t, cmp.DeepEqual(w1[i].RawMetrics, w2[i].RawMetrics),
				"message metrics differed between searchers")
			assert.Equal(t, w1[i].Workload, w2[i].Workload,
				"message workload differed between searchers")
		}
	}
}

func toKinds(types string) (kinds []Kind) {
	for _, unparsed := range strings.Fields(types) {
		var kind Kind
		switch char := string(unparsed[len(unparsed)-1]); char {
		case "C":
			kind = CheckpointModel
		case "S":
			kind = RunStep
		case "V":
			kind = ComputeValidationMetrics
		}
		count, err := strconv.Atoi(unparsed[:len(unparsed)-1])
		if err != nil {
			panic(err)
		}
		for i := 0; i < count; i++ {
			kinds = append(kinds, kind)
		}
	}
	return kinds
}
