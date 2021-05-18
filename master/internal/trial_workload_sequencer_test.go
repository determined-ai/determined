package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/google/uuid"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func defaultExperimentConfig() (expconf.ExperimentConfig, error) {
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
entrypoint: model_def:MyTrial
`
	expConfig, err := expconf.ParseAnyExperimentConfigYAML([]byte(yam))
	if err != nil {
		return expconf.ExperimentConfig{}, nil
	}
	return schemas.WithDefaults(expConfig).(expconf.ExperimentConfig), nil
}

func TestTrialWorkloadSequencer(t *testing.T) {
	expConfig, err := defaultExperimentConfig()
	assert.NilError(t, err)

	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	schedulingUnit := expConfig.SchedulingUnit()
	train := searcher.NewValidateAfter(create.RequestID, expconf.NewLength(expconf.Batches, 500))

	trainWorkload1 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                1,
		NumBatches:            schedulingUnit,
		PriorBatchesProcessed: 0,
	}
	trainWorkload2 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		NumBatches:            schedulingUnit,
		PriorBatchesProcessed: schedulingUnit,
	}
	trainWorkload3 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                3,
		NumBatches:            schedulingUnit,
		PriorBatchesProcessed: 2 * schedulingUnit,
	}
	trainWorkload4 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		NumBatches:            schedulingUnit,
		PriorBatchesProcessed: 3 * schedulingUnit,
	}
	trainWorkload5 := workload.Workload{
		Kind:                  workload.RunStep,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		NumBatches:            schedulingUnit,
		PriorBatchesProcessed: 4 * schedulingUnit,
	}
	checkpointWorkload1 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                1,
		PriorBatchesProcessed: schedulingUnit,
	}
	checkpointWorkload2 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		PriorBatchesProcessed: schedulingUnit * 2,
	}
	checkpointWorkload4 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		PriorBatchesProcessed: schedulingUnit * 4,
	}
	checkpointWorkload5 := workload.Workload{
		Kind:                  workload.CheckpointModel,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		PriorBatchesProcessed: schedulingUnit * 5,
	}
	validationWorkload2 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                2,
		PriorBatchesProcessed: schedulingUnit * 2,
	}
	validationWorkload4 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                4,
		PriorBatchesProcessed: schedulingUnit * 4,
	}
	validationWorkload5 := workload.Workload{
		Kind:                  workload.ComputeValidationMetrics,
		ExperimentID:          1,
		TrialID:               1,
		StepID:                5,
		PriorBatchesProcessed: schedulingUnit * 5,
	}

	s := newTrialWorkloadSequencer(experiment, create, nil)

	// Check that upToDate() returns true as soon as sequencer is created.
	assert.Assert(t, s.UpToDate())

	// Request a few operations so the sequencer builds its internal desired state.
	s.OperationRequested(train)
	assert.Assert(t, !s.UpToDate())

	// Check that workload() returns an error before setTrialID is set
	_, err = s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() before sequencer.SetTrialID()")

	s.SetTrialID(1)

	w, err := s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload1)

	// Check that before training has completed, there is nothing to checkpoint.
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	isNotBestVal := func() bool { return false }

	// Complete first RUN_STEP.
	completedOp, err := s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload1,
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload1)

	// Complete second RUN_STEP.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload2,
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload2)

	// Complete first COMPUTE_VALIDATION_METRICS.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          validationWorkload2,
		ValidationMetrics: &workload.ValidationMetrics{},
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload3)

	// Complete third and fourth RUN_STEP.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload3,
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload4)

	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload4,
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload4)

	// Complete second COMPUTE_VALIDATION_METRICS.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          validationWorkload4,
		ValidationMetrics: &workload.ValidationMetrics{},
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload4)

	// Complete first CHECKPOINT_MODEL.
	fakeCheckpointMetrics := workload.CheckpointMetrics{UUID: uuid.New()}
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          checkpointWorkload4,
		CheckpointMetrics: &fakeCheckpointMetrics,
	}, isNotBestVal)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	// Complete last RUN_STEP.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload5,
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload5)

	// Check that rollBackSequencer() affects nothing before a completed checkpoint.
	totalBatches, err := s.RollBackSequencer()
	assert.NilError(t, err)
	assert.Equal(t, totalBatches, 400)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload5)

	// Replay last RUN_STEP after rollback.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: trainWorkload5,
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload5)

	// Complete last CHECKPOINT_MODEL.
	fakeCheckpointMetrics = workload.CheckpointMetrics{UUID: uuid.New()}
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          checkpointWorkload5,
		CheckpointMetrics: &fakeCheckpointMetrics,
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload5)

	// Complete last COMPUTE_VALIDATION_METRICS.
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload:          validationWorkload5,
		ValidationMetrics: &workload.ValidationMetrics{},
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp != nil, "expected to report a validation")
	assert.Equal(t, *completedOp, train, "reported incorrect validation") // nolint:staticcheck
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	// Check that we are up to date now.
	assert.Assert(t, s.UpToDate())

	// Check that workload() returns an error when upToDate returns true
	_, err = s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() with sequencer.UpToDate() == true")
}

func TestTrialWorkloadSequencerFailedWorkloads(t *testing.T) {
	expConfig, err := defaultExperimentConfig()
	assert.NilError(t, err)
	expConfig.SetMinCheckpointPeriod(expconf.NewLengthInBatches(100))
	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	s := newTrialWorkloadSequencer(experiment, create, nil)
	s.SetTrialID(1)

	train := searcher.NewValidateAfter(create.RequestID, expconf.NewLength(expconf.Batches, 500))
	s.OperationRequested(train)

	msg := workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.RunStep,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			NumBatches:            expConfig.SchedulingUnit(),
			PriorBatchesProcessed: 0,
		},
	}
	completedOp, err := s.WorkloadCompleted(msg, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)

	exitedReason := workload.ExitedReason("not ok")
	immediateExit, err := s.WorkloadFailed(msg, exitedReason)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)
	assert.Equal(t, immediateExit, true, "should have been exiting early, immediately")
}

func TestTrialWorkloadSequencerOperationLessThanBatchSize(t *testing.T) {
	expConfig, err := defaultExperimentConfig()
	assert.NilError(t, err)
	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, map[string]interface{}{
		model.GlobalBatchSize: 64,
	}, model.TrialWorkloadSequencerType)

	s := newTrialWorkloadSequencer(experiment, create, nil)
	s.SetTrialID(1)

	train := searcher.NewValidateAfter(create.RequestID, expconf.NewLength(expconf.Records, 24))
	s.OperationRequested(
		train,
	)

	completedOp, err := s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.RunStep,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			NumBatches:            1,
			PriorBatchesProcessed: 0,
		},
	}, nil)
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)

	fakeCheckpointMetrics := &workload.CheckpointMetrics{UUID: uuid.New()}
	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.CheckpointModel,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			PriorBatchesProcessed: s.TotalBatchesProcessed,
		},
		CheckpointMetrics: fakeCheckpointMetrics,
	}, func() bool { return false })
	assert.NilError(t, err)
	assert.Assert(t, completedOp == nil, "should not have finished %v yet", train)

	completedOp, err = s.WorkloadCompleted(workload.CompletedMessage{
		Workload: workload.Workload{
			Kind:                  workload.ComputeValidationMetrics,
			ExperimentID:          1,
			TrialID:               1,
			StepID:                1,
			PriorBatchesProcessed: s.TotalBatchesProcessed,
		},
		ValidationMetrics: &workload.ValidationMetrics{},
	}, func() bool { return false })
	assert.NilError(t, err)
	assert.Assert(t, completedOp != nil, "expected to report a validation")
	assert.Equal(t, *completedOp, train, "reported incorrect validation") // nolint:staticcheck
}
