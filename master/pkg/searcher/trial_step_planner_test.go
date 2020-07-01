package searcher

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"gotest.tools/assert"
)

func TestTrialStepPlanner(t *testing.T) {
	trialStepPlanner := newTrialStepPlanner(defaultBatchesPerStep, 0)
	ctx := context{
		rand:    nprand.New(0),
		hparams: defaultHyperparameters(),
	}

	create := trialStepPlanner.create(ctx, sampleAll(ctx.hparams, ctx.rand))
	ops, trunc := trialStepPlanner.trainAndValidate(create.RequestID, model.NewLengthInRecords(65))
	ops = append(ops, trialStepPlanner.checkpoint(create.RequestID))

	assert.Equal(t, len(ops), 3)
	assert.Equal(t, ops[0], WorkloadOperation{create.RequestID, RunStep, 1, 1})
	assert.Equal(t, ops[1], WorkloadOperation{create.RequestID, ComputeValidationMetrics, 1, 0})
	assert.Equal(t, ops[2], WorkloadOperation{create.RequestID, CheckpointModel, 1, 0})
	assert.Equal(t, trunc, model.NewLengthInRecords(1))

	ops, trunc = trialStepPlanner.trainAndValidate(create.RequestID, model.NewLengthInRecords(66))
	ops = append(ops, trialStepPlanner.checkpoint(create.RequestID))

	assert.Equal(t, len(ops), 3)
	assert.Equal(t, ops[0], WorkloadOperation{create.RequestID, RunStep, 2, 1})
	assert.Equal(t, ops[1], WorkloadOperation{create.RequestID, ComputeValidationMetrics, 2, 0})
	assert.Equal(t, ops[2], WorkloadOperation{create.RequestID, CheckpointModel, 2, 0})
	assert.Equal(t, trunc, model.NewLengthInRecords(2))
}
