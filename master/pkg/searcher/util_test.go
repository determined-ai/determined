package searcher

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"gotest.tools/assert"
	"gotest.tools/assert/cmp"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

const (
	defaultMetric          = "metric"
	defaultGlobalBatchSize = 64.0
)

var defaultBatchesPerStep = model.DefaultExperimentConfig().BatchesPerStep

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
	recordsPerEpoch int,
) {
	search := NewSearcher(0, method, params, defaultBatchesPerStep, recordsPerEpoch,
		model.NewLength(method.Unit(), 0), model.NewLength(method.Unit(), 0),
		model.DefaultExperimentConfig().CheckpointPolicy)
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
			t.Errorf("unexpected trial %+v", kindsToShort(actualKinds))
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
	method1 := methodGen()
	searcher1 := NewSearcher(uint32(seed), method1, hparams, defaultBatchesPerStep, 0,
		model.NewLength(method1.Unit(), 0), model.NewLength(method1.Unit(), 0),
		model.NoneCheckpointPolicy)
	method2 := methodGen()
	searcher2 := NewSearcher(uint32(seed), methodGen(), hparams, defaultBatchesPerStep, 0,
		model.NewLength(method2.Unit(), 0), model.NewLength(method2.Unit(), 0),
		model.NoneCheckpointPolicy)

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

func kindsToShort(kinds []Kind) string {
	grouped := []struct {
		kind  Kind
		count int
	}{
		{kind: kinds[0], count: 1},
	}
	curIndex := 0
	for _, kind := range kinds[1:] {
		if kind == grouped[curIndex].kind {
			grouped[curIndex].count++
		} else {
			grouped = append(grouped, struct {
				kind  Kind
				count int
			}{
				kind:  kind,
				count: 1,
			})
			curIndex++
		}
	}

	repr := ""
	for _, grouped := range grouped {
		switch grouped.kind {
		case RunStep:
			repr += fmt.Sprintf("%d%s ", grouped.count, "S")
		case CheckpointModel:
			repr += fmt.Sprintf("%d%s ", grouped.count, "C")
		case ComputeValidationMetrics:
			repr += fmt.Sprintf("%d%s ", grouped.count, "V")
		}
	}

	return repr
}

type predefinedTrial struct {
	Trains      map[int]bool
	Validations map[int]float64
	Checkpoints map[int]bool
	EarlyExit   int
}

func newPredefinedTrial(
	nsteps int, validations map[int]float64, checkpoints []int, earlyExit int,
) predefinedTrial {
	trainsMap := make(map[int]bool)
	for i := 1; i <= nsteps; i++ {
		trainsMap[i] = true
	}

	checkpointsMap := make(map[int]bool)
	for _, i := range checkpoints {
		checkpointsMap[i] = true
	}

	return predefinedTrial{
		Trains:      trainsMap,
		Validations: validations,
		Checkpoints: checkpointsMap,
		EarlyExit:   earlyExit,
	}
}

func newEarlyExitPredefinedTrial(
	validation float64, nsteps int, validations []int, checkpoints []int,
) predefinedTrial {
	validationsMap := make(map[int]float64)
	for _, i := range validations {
		validationsMap[i] = validation
	}
	return newPredefinedTrial(nsteps, validationsMap, checkpoints, nsteps)
}

func newConstantPredefinedTrial(
	validation float64, nsteps int, validations []int, checkpoints []int,
) predefinedTrial {
	validationsMap := make(map[int]float64)
	for _, i := range validations {
		validationsMap[i] = validation
	}
	return newPredefinedTrial(nsteps, validationsMap, checkpoints, 0)
}

func (t *predefinedTrial) TrainForStep(stepID int) error {
	if ok := t.Trains[stepID]; !ok {
		return errors.Errorf("unexpected TrainForStep at step %v", stepID)
	}
	delete(t.Trains, stepID)
	return nil
}

func (t *predefinedTrial) ComputeValidationMetrics(stepID int) (float64, error) {
	validation, ok := t.Validations[stepID]
	if !ok {
		return 0, errors.Errorf("unexpected ComputeValidationMetrics at step %v", stepID)
	}
	delete(t.Validations, stepID)
	return validation, nil
}

func (t *predefinedTrial) CheckpointModel(stepID int) error {
	if ok := t.Checkpoints[stepID]; !ok {
		return errors.Errorf("unexpected CheckpointModel at step %v", stepID)
	}
	delete(t.Checkpoints, stepID)
	return nil
}

func (t *predefinedTrial) CheckComplete() error {
	for stepID := range t.Trains {
		return errors.Errorf("did not receive TrainForStep expected at step %v", stepID)
	}
	for stepID := range t.Validations {
		return errors.Errorf("did not receive ComputeValidationMetrics expected at step %v", stepID)
	}
	for stepID := range t.Checkpoints {
		return errors.Errorf("did not receive CheckpointModel expected at step %v", stepID)
	}
	return nil
}

// checkValueSimulation will run a SearchMethod until completion, using predefinedTrials.
func checkValueSimulation(
	t *testing.T,
	method SearchMethod,
	operationPlanner OperationPlanner,
	params model.Hyperparameters,
	expectedTrials []predefinedTrial,
) error {
	// Create requests are assigned a predefinedTrial in order.
	var nextTrialID int
	var pending []Operation

	trialIDs := map[RequestID]int{}

	ctx := context{
		rand:    nprand.New(0),
		hparams: params,
	}

	ops, err := method.initialOperations(ctx)
	if err != nil {
		return errors.Wrap(err, "initialOperations")
	}
	ops = operationPlanner.Plan(ops)

	pending = append(pending, ops...)

	for len(pending) > 0 {
		var earlyExit bool
		operation := pending[0]
		pending = pending[1:]

		switch operation := operation.(type) {
		case Create:
			requestID := operation.RequestID
			if nextTrialID >= len(expectedTrials) {
				return errors.Errorf("search method created too many trials")
			}
			trialIDs[requestID] = nextTrialID

			ops, err = method.trialCreated(ctx, requestID)
			if err != nil {
				return errors.Wrap(err, "trialCreated")
			}
			nextTrialID++

		case WorkloadOperation:
			requestID := operation.RequestID
			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			ops, err = simulateWorkloadComplete(
				ctx, method, operationPlanner, trial, operation, requestID)
			ops = operationPlanner.Plan(ops)
			if trial.EarlyExit == operation.StepID {
				earlyExit = true
			}
			if err != nil {
				return errors.Wrapf(err, "simulateWorkloadComplete for trial %v", trialID+1)
			}

		case Close:
			requestID := operation.RequestID
			trialID := trialIDs[requestID]
			trial := expectedTrials[trialID]
			if err = trial.CheckComplete(); err != nil {
				return errors.Wrapf(err, "trial %v/%v closed before completion", trialID+1, requestID)
			}

			ops, err = method.trialClosed(ctx, requestID)
			if err != nil {
				return errors.Wrap(err, "trialClosed")
			}

		default:
			return errors.Errorf("unexpected searcher operation: %T", operation)
		}

		pending = append(pending, ops...)
		if earlyExit {
			var requestID RequestID
			switch operation := operation.(type) {
			case WorkloadOperation:
				requestID = operation.RequestID
			default:
				return errors.Errorf("unexpected early exit for: %s", operation)
			}

			var newPending []Operation
			for _, op := range pending {
				switch op := op.(type) {
				case WorkloadOperation:
					if op.RequestID != requestID {
						newPending = append(newPending, op)
					}
				case Close:
					if op.RequestID != requestID {
						newPending = append(newPending, op)
					}
				default:
					newPending = append(newPending, op)
				}
			}
			pending = newPending
		}
	}

	for i, trial := range expectedTrials {
		if err = trial.CheckComplete(); err != nil {
			return errors.Wrapf(err, "incomplete trial %v", i+1)
		}
	}

	return nil
}

func runValueSimulationTestCases(t *testing.T, testCases []valueSimulationTestCase) {
	for _, testCase := range testCases {
		tc := testCase
		t.Run(tc.name, func(t *testing.T) {
			method := NewSearchMethod(tc.config)
			operationPlanner := NewOperationPlanner(tc.batchesPerStep, tc.recordsPerEpoch,
				model.NewLength(tc.unit, 0), model.NewLength(tc.unit, 0), model.NoneCheckpointPolicy)
			err := checkValueSimulation(t, method, operationPlanner, tc.hparams, tc.expectedTrials)
			assert.NilError(t, err)
		})
	}
}

type valueSimulationTestCase struct {
	name            string
	expectedTrials  []predefinedTrial
	hparams         model.Hyperparameters
	config          model.SearcherConfig
	batchesPerStep  int
	recordsPerEpoch int
	unit            model.Unit
}

func simulateWorkloadComplete(
	ctx context,
	method SearchMethod,
	operationPlanner OperationPlanner,
	trial predefinedTrial,
	operation WorkloadOperation,
	requestID RequestID,
) ([]Operation, error) {
	var ops []Operation
	var err error

	switch operation.Kind {
	case RunStep:
		if err = trial.TrainForStep(operation.StepID); err != nil {
			return nil, errors.Wrap(err, "TrainForStep")
		}
		w := Workload{
			Kind:       RunStep,
			StepID:     operation.StepID,
			NumBatches: operation.NumBatches,
		}
		op, _, pErr := operationPlanner.WorkloadCompleted(requestID, w, false)
		if pErr != nil {
			return nil, errors.Wrap(pErr, "trainCompleted")
		}
		if trial.EarlyExit == operation.StepID {
			ops, err = method.trialExitedEarly(ctx, requestID)
		} else if op == nil {
		} else if tOp, ok := op.(Train); !ok {
			return nil, errors.New("op wasn't Train")
		} else {
			ops, err = method.trainCompleted(ctx, requestID, tOp)
		}
		if err != nil {
			return nil, errors.Wrap(err, "trainCompleted")
		}

	case ComputeValidationMetrics:
		var val float64
		val, err = trial.ComputeValidationMetrics(operation.StepID)
		if err != nil {
			return nil, errors.Wrap(err, "ComputeValidationMetrics")
		}
		w := Workload{
			Kind:   ComputeValidationMetrics,
			StepID: operation.StepID,
		}
		metrics := ValidationMetrics{
			Metrics: map[string]interface{}{
				"error": val,
			},
		}
		op, _, pErr := operationPlanner.WorkloadCompleted(requestID, w, false)
		if pErr != nil {
			return nil, errors.Wrap(pErr, "validationCompleted")
		} else if tOp, ok := op.(Validate); !ok {
			return nil, errors.New("op wasn't Validate")
		} else {
			ops, err = method.validationCompleted(ctx, requestID, tOp, metrics)
			if err != nil {
				return nil, errors.Wrap(err, "validationCompleted")
			}
		}

	case CheckpointModel:
		if err = trial.CheckpointModel(operation.StepID); err != nil {
			return nil, errors.Wrap(err, "CheckpointModel")
		}
		w := Workload{
			Kind:   CheckpointModel,
			StepID: operation.StepID,
		}
		metrics := CheckpointMetrics{}

		op, _, err := operationPlanner.WorkloadCompleted(requestID, w, false)
		if err != nil {
			return nil, errors.Wrap(err, "checkpointCompleted")
		} else if tOp, ok := op.(Checkpoint); !ok {
			return nil, errors.New("op wasn't Validate")
		} else {
			ops, err = method.checkpointCompleted(ctx, requestID, tOp, metrics)
			if err != nil {
				return nil, errors.Wrap(err, "checkpointCompleted")
			}
		}

	default:
		return nil, errors.Errorf("invalid workload operation of kind %q", operation.Kind)
	}

	return ops, nil
}

func defaultHyperparameters() model.Hyperparameters {
	return model.Hyperparameters{
		GlobalBatchSize: model.Hyperparameter{
			ConstHyperparameter: &model.ConstHyperparameter{Val: defaultGlobalBatchSize},
		},
	}
}
