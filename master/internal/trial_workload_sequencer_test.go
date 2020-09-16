package internal

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func TestTrialWorkloadSequencer(t *testing.T) {
	yam := `
checkpoint_storage:
  type: s3
  access_key: my key
  secret_key: my secret
  bucket: my bucket
hyperparameters:
  global_batch_size: 64
min_validation_period:
  batches: 200
min_checkpoint_period:
  batches: 400
searcher:
  name: single
  metric: loss
  max_length:
    batches: 500
reproducibility:
  experiment_seed: 42
checkpoint_policy: none
`

	expConfig := model.DefaultExperimentConfig(nil)
	assert.NilError(t, yaml.Unmarshal([]byte(yam), &expConfig, yaml.DisallowUnknownFields))

	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	schedulingUnit := model.DefaultExperimentConfig(nil).SchedulingUnit
	train := searcher.NewTrain(create.RequestID, model.NewLength(model.Batches, 500))
	validate := searcher.NewValidate(create.RequestID)
	checkpoint := searcher.NewCheckpoint(create.RequestID)

	trainWorkload1 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                1,
		NumBatches:            schedulingUnit,
		TotalBatchesProcessed: 0,
	}
	trainWorkload2 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		NumBatches:            schedulingUnit,
		TotalBatchesProcessed: schedulingUnit,
	}
	trainWorkload3 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                3,
		NumBatches:            schedulingUnit,
		TotalBatchesProcessed: 2 * schedulingUnit,
	}
	trainWorkload4 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		NumBatches:            schedulingUnit,
		TotalBatchesProcessed: 3 * schedulingUnit,
	}
	trainWorkload5 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		NumBatches:            schedulingUnit,
		TotalBatchesProcessed: 4 * schedulingUnit,
	}
	checkpointWorkload1 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                1,
		TotalBatchesProcessed: schedulingUnit,
	}
	checkpointWorkload2 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		TotalBatchesProcessed: schedulingUnit * 2,
	}
	checkpointWorkload4 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		TotalBatchesProcessed: schedulingUnit * 4,
	}
	checkpointWorkload5 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		TotalBatchesProcessed: schedulingUnit * 5,
	}
	validationWorkload2 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		TotalBatchesProcessed: schedulingUnit * 2,
	}
	validationWorkload4 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		TotalBatchesProcessed: schedulingUnit * 4,
	}
	validationWorkload5 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		TotalBatchesProcessed: schedulingUnit * 5,
	}

	s := newTrialWorkloadSequencer(experiment, create, nil)

	// Check that upToDate() returns true as soon as sequencer is created.
	assert.Assert(t, s.UpToDate())

	// Request a few operations so the sequencer builds its internal desired state.
	assert.NilError(t, s.OperationRequested(train))
	assert.Assert(t, !s.UpToDate())
	assert.NilError(t, s.OperationRequested(validate))
	assert.NilError(t, s.OperationRequested(checkpoint))

	// Check that workload() returns an error before setTrialID is set
	_, err := s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() before sequencer.SetTrialID()")

	s.SetTrialID(1)

	w, err := s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload1)

	// Check that before training has completed, there is nothing to checkpoint.
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	// Complete first RUN_STEP.
	op, _, err := s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload1}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload1)

	// Complete second RUN_STEP.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload2}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload2)

	// Complete first COMPUTE_VALIDATION_METRICS.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: validationWorkload2}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload3)

	// Complete third and fourth RUN_STEP.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload3}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload4)

	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload4}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload4)

	// Complete second COMPUTE_VALIDATION_METRICS.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          validationWorkload4,
		ValidationMetrics: &workload.ValidationMetrics{},
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload4)

	// Complete first CHECKPOINT_MODEL.
	fakeCheckpointMetrics := workload.CheckpointMetrics{UUID: uuid.New()}
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          checkpointWorkload4,
		CheckpointMetrics: &fakeCheckpointMetrics,
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	// Complete last RUN_STEP.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload5}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, train, "expected searcher op to be returned")
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload5)

	// Check that rollBackSequencer() affects nothing before a completed checkpoint.
	assert.Equal(t, s.RollBackSequencer(), 4)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload5)

	// Replay last RUN_STEP after rollback.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{Workload: trainWorkload5}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, train, "expected searcher op to be returned")
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload5)

	// Complete last CHECKPOINT_MODEL.
	fakeCheckpointMetrics = workload.CheckpointMetrics{UUID: uuid.New()}
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          checkpointWorkload5,
		CheckpointMetrics: &fakeCheckpointMetrics,
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload5)

	// Complete last COMPUTE_VALIDATION_METRICS.
	op, _, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          validationWorkload5,
		ValidationMetrics: &workload.ValidationMetrics{},
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, validate, "expected searcher op to be returned")
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	// Complete cached CHECKPOINT_MODEL.
	op, _, err = s.CompleteCachedCheckpoints()
	assert.NilError(t, err)
	assert.Equal(t, op, checkpoint, "expected searcher op to be returned")

	// Check that we are up to date now.
	assert.Assert(t, s.UpToDate())

	// Check that workload() returns an error when upToDate returns true
	_, err = s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() with sequencer.UpToDate() == true")
}

func TestTrialWorkloadSequencerFailedWorkloads(t *testing.T) {
	expConfig := model.DefaultExperimentConfig(nil)
	expConfig.MinCheckpointPeriod = model.NewLengthInBatches(100)
	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	s := newTrialWorkloadSequencer(experiment, create, nil)
	s.SetTrialID(1)

	assert.NilError(t, s.OperationRequested(
		searcher.NewTrain(create.RequestID, model.NewLength(model.Batches, 500)),
	))

	_, _, err := s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.RunStep,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			NumBatches:            expConfig.SchedulingUnit,
			TotalBatchesProcessed: 0,
		},
	}, nil)
	assert.NilError(t, err)

	exitedReason := workload.ExitedReason("not ok")
	op, _, err := s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.CheckpointModel,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			TotalBatchesProcessed: expConfig.SchedulingUnit,
		},
		CheckpointMetrics: nil,
		ExitedReason:      &exitedReason,
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, nil, "should not have finished %v yet", op)
	assert.Equal(t, s.exitingEarly, true, "should have been exiting early")
}

func TestTrialWorkloadSequencerOperationLessThanBatchSize(t *testing.T) {
	expConfig := model.DefaultExperimentConfig(nil)
	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	s := newTrialWorkloadSequencer(experiment, create, nil)
	s.SetTrialID(1)

	train := searcher.NewTrain(create.RequestID, model.NewLength(model.Records, 24))
	assert.NilError(t, s.OperationRequested(
		train,
	))

	op, _, err := s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.RunStep,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			NumBatches:            1,
			TotalBatchesProcessed: 0,
		},
	}, nil)
	assert.NilError(t, err)
	assert.Equal(t, op, train)
}
