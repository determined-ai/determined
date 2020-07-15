package searcher

import (
	"fmt"
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
	expectedOps := []Operation{
		create,
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     1,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     2,
			NumBatches: 50,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    2,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     3,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     4,
			NumBatches: 27,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    4,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    4,
		},
	}

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
	expectedOps := []Operation{
		create,
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     1,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     2,
			NumBatches: 50,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       ComputeValidationMetrics,
			StepID:     2,
			NumBatches: 0,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     3,
			NumBatches: 50,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       ComputeValidationMetrics,
			StepID:     3,
			NumBatches: 0,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     4,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       CheckpointModel,
			StepID:     4,
			NumBatches: 0,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     5,
			NumBatches: 50,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       ComputeValidationMetrics,
			StepID:     5,
			NumBatches: 0,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     6,
			NumBatches: 50,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       ComputeValidationMetrics,
			StepID:     6,
			NumBatches: 0,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       CheckpointModel,
			StepID:     6,
			NumBatches: 0,
		},
	}

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
	expectedOps := []Operation{
		create,
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     1,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     2,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    2,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    2,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     3,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     4,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    4,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    4,
		},
	}

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
	expectedOps := []Operation{
		create,
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     1,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     2,
			NumBatches: 56,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    2,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    2,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     3,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     4,
			NumBatches: 56,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    4,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    4,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     5,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     6,
			NumBatches: 56,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    6,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    6,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     7,
			NumBatches: 100,
		},
		WorkloadOperation{
			RequestID:  create.RequestID,
			Kind:       RunStep,
			StepID:     8,
			NumBatches: 56,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      ComputeValidationMetrics,
			StepID:    8,
		},
		WorkloadOperation{
			RequestID: create.RequestID,
			Kind:      CheckpointModel,
			StepID:    8,
		},
	}

	simulationOperationPlanner(t, searcherOps, expectedOps, opPlanner, create.RequestID)
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
