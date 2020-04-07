package internal

import (
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
searcher:
  name: single
  metric: loss
  max_steps: 1
reproducibility:
  experiment_seed: 42
checkpoint_policy: none
`

	expConfig := model.DefaultExperimentConfig()
	assert.NilError(t, yaml.Unmarshal([]byte(yam), &expConfig, yaml.DisallowUnknownFields))

	experiment := &model.Experiment{ID: 1, State: model.ActiveState, Config: expConfig}

	rand := nprand.New(0)
	create := searcher.NewCreate(rand, nil, model.TrialWorkloadSequencerType)

	// Sequencer input messages.
	trainOperation1 := searcher.NewTrain(create.RequestID, 1)
	trainOperation2 := searcher.NewTrain(create.RequestID, 2)
	trainOperation3 := searcher.NewTrain(create.RequestID, 3)
	trainOperation4 := searcher.NewTrain(create.RequestID, 4)
	trainOperation5 := searcher.NewTrain(create.RequestID, 5)
	checkpointOperation2 := searcher.NewCheckpoint(create.RequestID, 2)
	validateOperation2 := searcher.NewValidate(create.RequestID, 2)

	trainWorkload1 := searcher.Workload{
		Kind:         searcher.RunStep,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       1,
	}
	trainWorkload2 := searcher.Workload{
		Kind:         searcher.RunStep,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       2,
	}
	trainWorkload3 := searcher.Workload{
		Kind:         searcher.RunStep,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       3,
	}
	trainWorkload4 := searcher.Workload{
		Kind:         searcher.RunStep,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       4,
	}
	trainWorkload5 := searcher.Workload{
		Kind:         searcher.RunStep,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       5,
	}
	checkpointWorkload1 := searcher.Workload{
		Kind:         searcher.CheckpointModel,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       1,
	}
	checkpointWorkload2 := searcher.Workload{
		Kind:         searcher.CheckpointModel,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       2,
	}
	validationWorkload2 := searcher.Workload{
		Kind:         searcher.ComputeValidationMetrics,
		ExperimentID: 1,
		TrialID:      1,
		StepID:       2,
	}

	// Sequencer output messages.
	trainCompleted1 := searcher.CompletedMessage{
		Workload: trainWorkload1,
	}
	trainCompleted2 := searcher.CompletedMessage{
		Workload: trainWorkload2,
	}
	trainCompleted3 := searcher.CompletedMessage{
		Workload: trainWorkload3,
	}
	trainCompleted4 := searcher.CompletedMessage{
		Workload: trainWorkload4,
	}
	checkpointCompleted2 := searcher.CompletedMessage{
		Workload:          checkpointWorkload2,
		CheckpointMetrics: &searcher.CheckpointMetrics{UUID: uuid.New()},
	}
	validationCompleted2 := searcher.CompletedMessage{
		Workload: validationWorkload2,
	}

	s := newTrialWorkloadSequencer(experiment, create, nil)

	// Check that upToDate() returns true as soon as sequencer is created.
	assert.Assert(t, s.UpToDate())

	// Request a few operations so the sequencer builds its internal desired state.
	assert.NilError(t, s.OperationRequested(trainOperation1))
	assert.Assert(t, !s.UpToDate())
	assert.NilError(t, s.OperationRequested(trainOperation2))
	assert.NilError(t, s.OperationRequested(checkpointOperation2))
	assert.NilError(t, s.OperationRequested(validateOperation2))

	// Check that workload() returns an error before setTrialID is set
	_, err := s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() before sequencer.SetTrialID()")

	s.SetTrialID(1)

	w, err := s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload1)

	// Check that before training has completed, there is nothing to checkpoint.
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	assert.NilError(t, s.WorkloadCompleted(trainCompleted1, nil))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload1)

	assert.NilError(t, s.WorkloadCompleted(trainCompleted2, nil))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, validationWorkload2)
	assert.Equal(t, *s.PrecloseCheckpointWorkload(), checkpointWorkload2)

	assert.NilError(t, s.WorkloadCompleted(validationCompleted2, nil))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, checkpointWorkload2)

	assert.NilError(t, s.WorkloadCompleted(checkpointCompleted2, nil))
	assert.Assert(t, s.PrecloseCheckpointWorkload() == nil)

	assert.Assert(t, s.UpToDate())

	// Check that workload() returns an error when upToDate returns true
	_, err = s.Workload()
	assert.Error(t, err, "cannot call sequencer.Workload() with sequencer.UpToDate() == true")

	// Check that rollBackSequencer() affects nothing after a completed checkpoint.
	assert.Equal(t, s.RollBackSequencer(), 2)
	assert.Assert(t, s.UpToDate())

	// Check that rollBackSequencer() causes workload() to replay workloads.
	assert.NilError(t, s.OperationRequested(trainOperation3))
	assert.NilError(t, s.OperationRequested(trainOperation4))
	assert.NilError(t, s.OperationRequested(trainOperation5))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload3)
	assert.NilError(t, s.WorkloadCompleted(trainCompleted3, nil))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload4)
	assert.NilError(t, s.WorkloadCompleted(trainCompleted4, nil))
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload5)
	assert.Equal(t, s.RollBackSequencer(), 2)
	w, err = s.Workload()
	assert.NilError(t, err)
	assert.Equal(t, w, trainWorkload3)
}
