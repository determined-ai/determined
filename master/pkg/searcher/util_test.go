package searcher

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const defaultMetric = "metric"

func isExpected(actual, expected []ValidateAfter) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, act := range actual {
		if expected[i].Length != act.Length {
			return false
		}
	}
	return true
}

func checkSimulation(
	t *testing.T,
	method SearchMethod,
	params expconf.Hyperparameters,
	validation ValidationFunction,
	expected [][]ValidateAfter,
) {
	search := NewSearcher(0, method, params)
	actual, err := Simulate(search, new(int64), validation, true, defaultMetric)
	assert.NilError(t, err)

	assert.Equal(t, len(actual.Results), len(expected))
	for _, actualTrial := range actual.Results {
		found := false
		for i, expectedTrial := range expected {
			if isExpected(actualTrial, expectedTrial) {
				expected = append(expected[:i], expected[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unexpected trial %+v not in %+v", actualTrial, expected)
		}
	}
}

// checkReproducibility creates two searchers with the same seed and the given config, simulates
// them, and checks that they produce the same trials and the same sequence of workloads for each
// trial.
func checkReproducibility(
	t assert.TestingT, methodGen func() SearchMethod, hparams expconf.Hyperparameters, metric string,
) {
	hparams = schemas.WithDefaults(hparams)
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
			assert.Equal(t, w1[i], w2[i], "workload differed between searchers")
		}
	}
}

func toOps(types string) (ops []ValidateAfter) {
	for _, unparsed := range strings.Fields(types) {
		count, err := strconv.ParseUint(unparsed[:len(unparsed)-1], 10, 64)
		if err != nil {
			panic(err)
		}
		switch unit := string(unparsed[len(unparsed)-1]); unit {
		case "R":
			ops = append(ops, ValidateAfter{Length: count})
		case "B":
			ops = append(ops, ValidateAfter{Length: count})
		case "E":
			ops = append(ops, ValidateAfter{Length: count})
		}
	}
	return ops
}

type predefinedTrial struct {
	Ops        []ValidateAfter
	ValMetrics []float64
	EarlyExit  *int
}

func newPredefinedTrial(ops []ValidateAfter, earlyExit *int, valMetrics []float64) predefinedTrial {
	return predefinedTrial{
		Ops:        ops,
		EarlyExit:  earlyExit,
		ValMetrics: valMetrics,
	}
}

func newEarlyExitPredefinedTrial(ops []ValidateAfter, valMetric float64) predefinedTrial {
	var valMetrics []float64
	for range ops {
		valMetrics = append(valMetrics, valMetric)
	}
	exitEarly := len(ops) - 1
	return newPredefinedTrial(ops, &exitEarly, valMetrics)
}

func newConstantPredefinedTrial(ops []ValidateAfter, valMetric float64) predefinedTrial {
	var valMetrics []float64
	for range ops {
		valMetrics = append(valMetrics, valMetric)
	}
	return newPredefinedTrial(ops, nil, valMetrics)
}

func (t *predefinedTrial) Train(length uint64, opIndex int) error {
	if opIndex >= len(t.Ops) {
		return fmt.Errorf("ran out of expected ops trying to train")
	}
	op := t.Ops[opIndex]
	if op.Length != length {
		return fmt.Errorf("wanted %v got %v", op.Length, length)
	}
	return nil
}

func (t *predefinedTrial) CheckComplete(opIndex int) error {
	return check.Equal(len(t.Ops), opIndex, "had ops %s left", t.Ops[opIndex:])
}

// checkValueSimulation will run a SearchMethod until completion, using predefinedTrials.
func checkValueSimulation(
	t *testing.T,
	method SearchMethod,
	params expconf.Hyperparameters,
	expectedTrials []predefinedTrial,
) error {
	// Create requests are assigned a predefinedTrial in order.
	var nextTrialID int
	var pending []Operation

	trialIDs := map[model.RequestID]int{}
	trialOpIdx := map[model.RequestID]int{}
	trialEarlyExits := map[model.RequestID]bool{}

	ctx := context{
		rand:    nprand.New(0),
		hparams: params,
	}

	ops, err := method.initialOperations(ctx)
	if err != nil {
		return fmt.Errorf("initialOperations: %w", err)
	}

	pending = append(pending, ops...)

	for len(pending) > 0 {
		var requestID model.RequestID
		operation := pending[0]
		pending = pending[1:]

		switch operation := operation.(type) {
		case Create:
			requestID = operation.RequestID
			if nextTrialID >= len(expectedTrials) {
				return fmt.Errorf("search method created too many trials")
			}
			trialIDs[requestID] = nextTrialID
			trialOpIdx[requestID] = 0

			ops, err = method.trialCreated(ctx, requestID)
			if err != nil {
				return fmt.Errorf("trialCreated: %w", err)
			}
			nextTrialID++

		case ValidateAfter:
			requestID = operation.GetRequestID()
			if trialEarlyExits[requestID] {
				continue
			}

			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			if trial.EarlyExit != nil && trialOpIdx[requestID] == *trial.EarlyExit {
				trialEarlyExits[requestID] = true
			}
			ops, err = simulateOperationComplete(ctx, method, trial, operation, trialOpIdx[requestID])
			if err != nil {
				return fmt.Errorf("simulateOperationComplete for trial %v: %w", trialID+1, err)
			}
			trialOpIdx[requestID]++
			if err = saveAndReload(method); err != nil {
				return fmt.Errorf("snapshot failed: %w", err)
			}

		case Close:
			requestID = operation.RequestID
			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			err = trial.CheckComplete(trialOpIdx[requestID])
			if err != nil {
				return fmt.Errorf("trial %v closed before completion: %w", trialID+1, err)
			}

			ops, err = method.trialClosed(ctx, requestID)
			if err != nil {
				return fmt.Errorf("trialClosed: %w", err)
			}

		default:
			return fmt.Errorf("unexpected searcher operation: %T", operation)
		}

		pending = append(pending, ops...)
	}

	for requestID, trialID := range trialIDs {
		if err = expectedTrials[trialID].CheckComplete(trialOpIdx[requestID]); err != nil {
			return fmt.Errorf("incomplete trial %v: %w", trialID+1, err)
		}
	}

	return nil
}

func runValueSimulationTestCases(t *testing.T, testCases []valueSimulationTestCase) {
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			// Apply WithDefaults in one place to make tests easyto write.
			config := schemas.WithDefaults(tc.config)
			hparams := schemas.WithDefaults(tc.hparams)
			method := NewSearchMethod(config)
			err := checkValueSimulation(t, method, hparams, tc.expectedTrials)
			assert.NilError(t, err)
		})
	}
}

type valueSimulationTestCase struct {
	name           string
	expectedTrials []predefinedTrial
	hparams        expconf.Hyperparameters
	config         expconf.SearcherConfig
}

func simulateOperationComplete(
	ctx context,
	method SearchMethod,
	trial predefinedTrial,
	operation ValidateAfter,
	opIndex int,
) ([]Operation, error) {
	if err := trial.Train(operation.Length, opIndex); err != nil {
		return nil, fmt.Errorf("error checking ValidateAfter with predefinedTrial: %w", err)
	}

	if trial.EarlyExit != nil && opIndex == *trial.EarlyExit {
		ops, err := method.trialExitedEarly(ctx, operation.RequestID, model.UserRequestedStop)
		if err != nil {
			return nil, fmt.Errorf("trainCompleted: %w", err)
		}
		return ops, nil
	}

	ops, err := method.validationCompleted(
		ctx, operation.RequestID, trial.ValMetrics[opIndex], operation,
	)
	if err != nil {
		return nil, fmt.Errorf("validationCompleted: %w", err)
	}

	return ops, nil
}

func saveAndReload(method SearchMethod) error {
	// take the state back and forth through a round of serialization to test.
	if state, err := method.Snapshot(); err != nil {
		return err
	} else if err := method.Restore(state); err != nil {
		return err
	} else if state2, err := method.Snapshot(); err != nil { // Test restore is correct.
		return err
	} else if !bytes.Equal(state, state2) {
		unmarshaledState := method.Restore(state)
		unmarshaledState2 := method.Restore(state2)
		fmt.Printf("%+v\n", unmarshaledState)  //nolint: forbidigo
		fmt.Printf("%+v\n", unmarshaledState2) //nolint: forbidigo
		return errors.New("successive snapshots were not identical")
	}
	return nil
}
