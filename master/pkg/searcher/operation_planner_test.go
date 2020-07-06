package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

func TestOperationPlanner(t *testing.T) {
	rand := nprand.New(0)
	opPlanner := NewOperationPlanner(model.Batches, 100, 0)

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

	plannedOps := opPlanner.Plan(searcherOps)
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

	assert.Equal(t, len(plannedOps), len(expectedOps))

	for i := range plannedOps {
		plannedOp, expectedOp := plannedOps[i], expectedOps[i]
		assert.DeepEqual(t, plannedOp, expectedOp)
	}

	searcherOpsReturned := 1 // 1 because we skip the create
	for _, plannedOp := range plannedOps {
		if workloadOp, ok := plannedOp.(WorkloadOperation); ok {
			searcherOp, err := opPlanner.WorkloadCompleted(create.RequestID, Workload{
				Kind:       workloadOp.Kind,
				StepID:     workloadOp.StepID,
				NumBatches: workloadOp.NumBatches,
			})

			assert.NilError(t, err, "error passing completed workload to operation planner")
			if searcherOp != nil {
				assert.DeepEqual(t, searcherOp, searcherOps[searcherOpsReturned])
				searcherOpsReturned++
			}
		}
	}
	assert.Equal(t, searcherOpsReturned, len(searcherOps))
}
