package searcher

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

func TestOperationPlanner(t *testing.T) {
	rand := nprand.New(0)
	opPlanner := NewOperationPlanner(100, 0, model.Batches, model.NewLengthInBatches(0),
		model.NewLengthInBatches(0), model.NoneCheckpointPolicy)

	create := NewCreate(
		rand, sampleAll(defaultHyperparameters(), rand), model.TrialWorkloadSequencerType)
	searcherOps := []Operation{
		create,
		NewTrain(create.RequestID, model.NewLengthInBatches(150)),
		NewValidate(create.RequestID),
		NewTrain(create.RequestID, model.NewLengthInBatches(127)),
		NewValidate(create.RequestID),
		NewCheckpoint(create.RequestID),
	}
	expectedOps := append([]Operation{create}, toExpected(create.RequestID, `
		R100 R50 V
		R100 R27 V C
	`)...)

	simulationOperationPlanner(t, searcherOps, expectedOps, opPlanner, create.RequestID)
}

func TestOperationPlannerWithMinValsAndMinCkpts(t *testing.T) {
	rand := nprand.New(0)
	opPlanner := NewOperationPlanner(100, 0, model.Batches, model.NewLengthInBatches(150),
		model.NewLengthInBatches(300), model.NoneCheckpointPolicy)
	create := NewCreate(
		rand, sampleAll(defaultHyperparameters(), rand), model.TrialWorkloadSequencerType)
	searcherOps := []Operation{
		create,
		NewTrain(create.RequestID, model.NewLengthInBatches(200)),
		NewValidate(create.RequestID),
		NewTrain(create.RequestID, model.NewLengthInBatches(200)),
		NewValidate(create.RequestID),
		NewCheckpoint(create.RequestID),
	}
	expectedOps := append([]Operation{create}, toExpected(create.RequestID, `
		R100 R50 V
		R50 V
		R100 C
		R50 V
		R50 V C
	`)...)

	simulationOperationPlanner(t, searcherOps, expectedOps, opPlanner, create.RequestID)
}

func TestOperationPlannerWithCheckpointPolicy(t *testing.T) {
	rand := nprand.New(0)
	opPlanner := NewOperationPlanner(100, 0, model.Batches, model.NewLengthInBatches(0),
		model.NewLengthInBatches(0), model.AllCheckpointPolicy)
	create := NewCreate(
		rand, sampleAll(defaultHyperparameters(), rand), model.TrialWorkloadSequencerType)
	searcherOps := []Operation{
		create,
		NewTrain(create.RequestID, model.NewLengthInBatches(200)),
		NewValidate(create.RequestID),
		NewTrain(create.RequestID, model.NewLengthInBatches(200)),
		NewValidate(create.RequestID),
		NewCheckpoint(create.RequestID),
	}
	expectedOps := append([]Operation{create}, toExpected(create.RequestID, `
		R100 R100 V C
		R100 R100 V C
	`)...)

	simulationOperationPlanner(t, searcherOps, expectedOps, opPlanner, create.RequestID)
}

func TestOperationPlannerWithAllConfigurations(t *testing.T) {
	rand := nprand.New(0)
	opPlanner := NewOperationPlanner(100, 0, model.Records, model.NewLengthInRecords(10000),
		model.NewLengthInRecords(20000), model.AllCheckpointPolicy)
	create := NewCreate(
		rand, sampleAll(defaultHyperparameters(), rand), model.TrialWorkloadSequencerType)
	searcherOps := []Operation{
		create,
		NewTrain(create.RequestID, model.NewLengthInRecords(20000)),
		NewValidate(create.RequestID),
		NewTrain(create.RequestID, model.NewLengthInRecords(20000)),
		NewCheckpoint(create.RequestID),
	}
	expectedOps := append([]Operation{create}, toExpected(create.RequestID, `
		R100 R56 V C
		R100 R56 V C
		R100 R56 V C
		R100 R56 V C
	`)...)

	simulationOperationPlanner(t, searcherOps, expectedOps, opPlanner, create.RequestID)
}

func toExpected(requestID RequestID, repr string) (ops []Operation) {
	curStep := 0
	for _, field := range strings.Fields(repr) {
		switch field[0] {
		case 'R':
			batches, err := strconv.Atoi(field[1:])
			if err != nil {
				panic(err)
			}
			curStep++
			ops = append(ops, WorkloadOperation{
				RequestID:  requestID,
				Kind:       RunStep,
				StepID:     curStep,
				NumBatches: batches,
			})
		case 'V':
			ops = append(ops, WorkloadOperation{
				RequestID: requestID,
				Kind:      ComputeValidationMetrics,
				StepID:    curStep,
			})
		case 'C':
			ops = append(ops, WorkloadOperation{
				RequestID: requestID,
				Kind:      CheckpointModel,
				StepID:    curStep,
			})
		}
	}
	return ops
}

func simulationOperationPlanner(
	t *testing.T, searcherOps []Operation, expectedOps []Operation, opPlanner OperationPlanner,
	requestID RequestID,
) {
	plannedOps := opPlanner.Plan(searcherOps)
	completedWorkloads := make(map[Operation]bool)
	searcherOpsReturned := 1 // 1 because we skip the create
	for len(plannedOps) > 0 {
		next := plannedOps[0]
		plannedOps = plannedOps[1:]

		if _, ok := next.(WorkloadOperation); ok && completedWorkloads[next] {
			continue
		}

		if len(expectedOps) == 0 {
			t.Fatalf("ran out of expectedOps trying to finish %+v", plannedOps)
		}
		expectedOp := expectedOps[0]
		expectedOps = expectedOps[1:]
		assert.DeepEqual(t, next, expectedOp)

		if workloadOp, ok := next.(WorkloadOperation); ok {
			searcherOp, newOps, err := opPlanner.WorkloadCompleted(requestID, Workload{
				Kind:       workloadOp.Kind,
				StepID:     workloadOp.StepID,
				NumBatches: workloadOp.NumBatches,
			}, false)

			completedWorkloads[workloadOp] = true
			plannedOps = append(newOps, plannedOps...)

			assert.NilError(t, err, fmt.Sprintf("error passing completed workload: %s", workloadOp))
			if searcherOp != nil {
				assert.DeepEqual(t, searcherOp, searcherOps[searcherOpsReturned])
				searcherOpsReturned++
			}
		}
	}

	assert.Equal(t, searcherOpsReturned, len(searcherOps))
}
