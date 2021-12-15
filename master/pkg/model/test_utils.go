//go:build integration
// +build integration

package model

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

var (
	defaultDeterminedUID UserID = 2
)

// ExperimentModelOption is an option that can be applied to update an experiment.
type ExperimentModelOption interface {
	apply(*Experiment)
}

// ExperimentModelOptionFunc is a type that implements ExperimentModelOption.
type ExperimentModelOptionFunc func(*Experiment)

func (f ExperimentModelOptionFunc) apply(experiment *Experiment) {
	f(experiment)
}

// ExperimentModel returns a new experiment with the specified options.
func ExperimentModel(opts ...ExperimentModelOption) *Experiment {
	maxLength := expconf.NewLengthInBatches(100)
	eConf := expconf.ExperimentConfig{
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
	eConf = schemas.WithDefaults(eConf).(expconf.ExperimentConfig)
	DefaultTaskContainerDefaults().MergeIntoExpConfig(&eConf)

	e := &Experiment{
		JobID:                NewJobID(),
		State:                ActiveState,
		Config:               eConf,
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
	apply(*Trial)
}

// TrialModelOptionFunc is a type that implements TrialModelOption.
type TrialModelOptionFunc func(*Trial)

func (f TrialModelOptionFunc) apply(trial *Trial) {
	f(trial)
}

// WithTrialState is a TrialModeOption that sets a trials state.
func WithTrialState(state State) TrialModelOption {
	return TrialModelOptionFunc(func(trial *Trial) {
		trial.State = state
	})
}

// TrialModel returns a new trial with the specified options.
func TrialModel(eID int, jobID JobID, opts ...TrialModelOption) *Trial {
	t := &Trial{
		TaskID:       NewTaskID(),
		JobID:        jobID,
		ExperimentID: eID,
		State:        ActiveState,
		StartTime:    time.Now(),
	}
	for _, o := range opts {
		o.apply(t)
	}
	return t
}
