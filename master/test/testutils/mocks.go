// +build integration

package testutils

import (
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

var (
	defaultDeterminedUID model.UserID = 2
)

// ExperimentModelOption is an option that can be applied to update an experiment.
type ExperimentModelOption interface {
	apply(*model.Experiment)
}

// ExperimentModelOptionFunc is a type that implements ExperimentModelOption.
type ExperimentModelOptionFunc func(*model.Experiment)

func (f ExperimentModelOptionFunc) apply(experiment *model.Experiment) {
	f(experiment)
}

// ExperimentModel returns a new experiment with the specified options.
func ExperimentModel(opts ...ExperimentModelOption) *model.Experiment {
	maxLength := expconf.NewLengthInBatches(100)
	config := expconf.ExperimentConfig{
		RawSearcher: &expconf.SearcherConfig{
			RawMetric: ptrs.StringPtr("loss"),
			RawSingleConfig: &expconf.SingleConfig{
				RawMaxLength: &maxLength,
			},
		},
		RawEntrypoint: ptrs.StringPtr("model_def:SomeTrialClass"),
		RawHyperparameters: expconf.Hyperparameters{
			expconf.GlobalBatchSize: expconf.Hyperparameter{
				RawConstHyperparameter: &expconf.ConstHyperparameter{RawVal: 64},
			},
		},
		RawCheckpointStorage: &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath: ptrs.StringPtr("/"),
			},
		},
	}
	config = schemas.WithDefaults(config).(expconf.ExperimentConfig)
	internal.DefaultConfig().TaskContainerDefaults.MergeIntoExpConfig(&config)

	e := &model.Experiment{
		State:                model.ActiveState,
		Config:               config,
		StartTime:            time.Now(),
		OwnerID:              &defaultDeterminedUID,
		ModelDefinitionBytes: []byte{},
	}

	for _, o := range opts {
		o.apply(e)
	}
	return e
}

// TrialModelOption is an option that can be applied to a trial.
type TrialModelOption interface {
	apply(*model.Trial)
}

// TrialModelOptionFunc is a type that implements TrialModelOption.
type TrialModelOptionFunc func(*model.Trial)

func (f TrialModelOptionFunc) apply(trial *model.Trial) {
	f(trial)
}

// WithTrialState is a TrialModeOption that sets a trials state.
func WithTrialState(state model.State) TrialModelOption {
	return TrialModelOptionFunc(func(trial *model.Trial) {
		trial.State = state
	})
}

// TrialModel returns a new trial with the specified options.
func TrialModel(eID int, opts ...TrialModelOption) *model.Trial {
	t := &model.Trial{
		TaskID:       model.TaskID(uuid.New().String()),
		ExperimentID: eID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	for _, o := range opts {
		o.apply(t)
	}
	return t
}
