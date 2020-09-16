package searcher

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

const defaultMetric = "metric"

func isExpected(actual, expected []Runnable) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, act := range actual {
		switch act := act.(type) {
		case Train:
			op, ok := expected[i].(Train)
			if !ok || op.Length != act.Length {
				return false
			}
		case Validate:
			_, ok := expected[i].(Validate)
			if !ok {
				return false
			}
		case Checkpoint:
			_, ok := expected[i].(Checkpoint)
			if !ok {
				return false
			}
		default:
			panic("trial had unexpected operation type")
		}
	}
	return true
}

func checkSimulation(
	t *testing.T,
	method SearchMethod,
	params model.Hyperparameters,
	validation ValidationFunction,
	expected [][]Runnable,
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
			assert.Equal(t, w1[i], w2[i], "workload differed between searchers")
		}
	}
}

func toOps(types string) (ops []Runnable) {
	for _, unparsed := range strings.Fields(types) {
		switch char := string(unparsed[0]); char {
		case "C":
			if len(unparsed) > 1 {
				panic("invalid short-form op")
			}
			ops = append(ops, Checkpoint{})
		case "V":
			if len(unparsed) > 1 {
				panic("invalid short-form op")
			}
			ops = append(ops, Validate{})
		default:
			count, err := strconv.Atoi(unparsed[:len(unparsed)-1])
			if err != nil {
				panic(err)
			}
			switch unit := string(unparsed[len(unparsed)-1]); unit {
			case "R":
				ops = append(ops, Train{Length: model.NewLengthInRecords(count)})
			case "B":
				ops = append(ops, Train{Length: model.NewLengthInBatches(count)})
			case "E":
				ops = append(ops, Train{Length: model.NewLengthInEpochs(count)})
			}
		}
	}
	return ops
}

type predefinedTrial struct {
	Ops        []Runnable
	ValMetrics []float64
	EarlyExit  *int
}

func newPredefinedTrial(ops []Runnable, earlyExit *int, valMetrics []float64) predefinedTrial {
	return predefinedTrial{
		Ops:        ops,
		EarlyExit:  earlyExit,
		ValMetrics: valMetrics,
	}
}

func newEarlyExitPredefinedTrial(ops []Runnable, valMetric float64) predefinedTrial {
	var valMetrics []float64
	for _, op := range ops {
		if _, ok := op.(Validate); ok {
			valMetrics = append(valMetrics, valMetric)
		}
	}
	exitEarly := len(ops) - 1
	return newPredefinedTrial(ops, &exitEarly, valMetrics)
}

func newConstantPredefinedTrial(ops []Runnable, valMetric float64) predefinedTrial {
	var valMetrics []float64
	for _, op := range ops {
		if _, ok := op.(Validate); ok {
			valMetrics = append(valMetrics, valMetric)
		}
	}
	return newPredefinedTrial(ops, nil, valMetrics)
}

func (t *predefinedTrial) Train(length model.Length, opIndex int) error {
	if opIndex >= len(t.Ops) {
		return errors.Errorf("ran out of expected ops trying to train")
	}
	tOp, ok := t.Ops[opIndex].(Train)
	if !ok {
		return errors.Errorf("wanted %v", t.Ops[0])
	}
	if tOp.Length != length {
		return errors.Errorf("wanted %s got %s", tOp.Length, length)
	}
	return nil
}

func (t *predefinedTrial) Validate(opIndex int) (float64, error) {
	if opIndex >= len(t.Ops) {
		return 0, errors.Errorf("ran out of expected ops trying to validate")
	}
	if _, ok := t.Ops[opIndex].(Validate); !ok {
		return 0, errors.Errorf("wanted %v", t.Ops[0])
	}
	valsSeen := 0
	for idx := range t.Ops {
		if idx == opIndex {
			return t.ValMetrics[valsSeen], nil
		}
		if _, ok := t.Ops[idx].(Validate); ok {
			valsSeen++
		}
	}
	return 0, errors.New("ran out of metrics to return for validations")
}

func (t *predefinedTrial) Checkpoint(opIndex int) error {
	if opIndex >= len(t.Ops) {
		return errors.Errorf("ran out of expected ops trying to checkpoint")
	}
	if _, ok := t.Ops[opIndex].(Checkpoint); !ok {
		return errors.Errorf("wanted %v", t.Ops[0])
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
	params model.Hyperparameters,
	expectedTrials []predefinedTrial,
) error {
	// Create requests are assigned a predefinedTrial in order.
	var nextTrialID int
	var pending []Operation

	trialIDs := map[RequestID]int{}
	trialOpIdx := map[RequestID]int{}

	ctx := context{
		rand:    nprand.New(0),
		hparams: params,
	}

	ops, err := method.initialOperations(ctx)
	if err != nil {
		return errors.Wrap(err, "initialOperations")
	}

	pending = append(pending, ops...)

	for len(pending) > 0 {
		var exitEarly bool
		var requestID RequestID
		operation := pending[0]
		pending = pending[1:]

		switch operation := operation.(type) {
		case Create:
			requestID = operation.RequestID
			if nextTrialID >= len(expectedTrials) {
				return errors.Errorf("search method created too many trials")
			}
			trialIDs[requestID] = nextTrialID
			trialOpIdx[requestID] = 0

			ops, err = method.trialCreated(ctx, requestID)
			if err != nil {
				return errors.Wrap(err, "trialCreated")
			}
			nextTrialID++

		case Runnable:
			requestID = operation.GetRequestID()
			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			if trial.EarlyExit != nil && trialOpIdx[requestID] == *trial.EarlyExit {
				exitEarly = true
			}
			ops, err = simulateOperationComplete(ctx, method, trial, operation, trialOpIdx[requestID])
			if err != nil {
				return errors.Wrapf(err, "simulateOperationComplete for trial %v", trialID+1)
			}
			trialOpIdx[requestID]++

		case Close:
			requestID = operation.RequestID
			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			err = trial.CheckComplete(trialOpIdx[requestID])
			if err != nil {
				return errors.Wrapf(err, "trial %v closed before completion", trialID+1)
			}

			ops, err = method.trialClosed(ctx, requestID)
			if err != nil {
				return errors.Wrap(err, "trialClosed")
			}

		default:
			return errors.Errorf("unexpected searcher operation: %T", operation)
		}

		pending = append(pending, ops...)
		if exitEarly {
			var newPending []Operation
			for _, op := range pending {
				switch op := op.(type) {
				case Requested:
					if op.GetRequestID() != requestID {
						newPending = append(newPending, op)
					}
				default:
					newPending = append(newPending, op)
				}
			}
			pending = newPending
		}
	}

	for requestID, trialID := range trialIDs {
		if err = expectedTrials[trialID].CheckComplete(trialOpIdx[requestID]); err != nil {
			return errors.Wrapf(err, "incomplete trial %v", trialID+1)
		}
	}

	return nil
}

func runValueSimulationTestCases(t *testing.T, testCases []valueSimulationTestCase) {
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			method := NewSearchMethod(tc.config)
			err := checkValueSimulation(t, method, tc.hparams, tc.expectedTrials)
			assert.NilError(t, err)
		})
	}
}

type valueSimulationTestCase struct {
	name           string
	expectedTrials []predefinedTrial
	hparams        model.Hyperparameters
	config         model.SearcherConfig
}

func simulateOperationComplete(
	ctx context,
	method SearchMethod,
	trial predefinedTrial,
	operation Runnable,
	opIndex int,
) ([]Operation, error) {
	var ops []Operation
	var err error

	switch operation := operation.(type) {
	case Train:
		if err = trial.Train(operation.Length, opIndex); err != nil {
			return nil, errors.Wrap(err, "error checking Train with predefinedTrial")
		}

		if trial.EarlyExit != nil && opIndex == *trial.EarlyExit {
			ops, err = method.trialExitedEarly(ctx, operation.RequestID)
		} else {
			ops, err = method.trainCompleted(ctx, operation.RequestID, operation)
		}
		if err != nil {
			return nil, errors.Wrap(err, "trainCompleted")
		}

	case Validate:
		val, vErr := trial.Validate(opIndex)
		if vErr != nil {
			return nil, errors.Wrap(err, "error checking Validate with predefinedTrial")
		}
		metrics := workload.ValidationMetrics{
			Metrics: map[string]interface{}{
				"error": val,
			},
		}
		ops, err = method.validationCompleted(ctx, operation.RequestID, operation, metrics)
		if err != nil {
			return nil, errors.Wrap(err, "validationCompleted")
		}

	case Checkpoint:
		if err = trial.Checkpoint(opIndex); err != nil {
			return nil, errors.Wrap(err, "error checking Checkpoint with predefinedTrial")
		}
		metrics := workload.CheckpointMetrics{}
		ops, err = method.checkpointCompleted(ctx, operation.RequestID, operation, metrics)
		if err != nil {
			return nil, errors.Wrap(err, "checkpointCompleted")
		}

	default:
		return nil, errors.Errorf("invalid runnable %q", operation)
	}

	return ops, nil
}
