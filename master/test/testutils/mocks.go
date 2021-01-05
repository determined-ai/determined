// +build integration

package testutils

import (
	"time"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/pkg/model"
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
	c := model.DefaultExperimentConfig(&internal.DefaultConfig().TaskContainerDefaults)
	c.Entrypoint = "model_def:SomeTrialClass"
	c.Searcher = model.SearcherConfig{
		Metric: "loss",
		SingleConfig: &model.SingleConfig{
			MaxLength: model.NewLengthInBatches(100),
		},
	}
	c.Hyperparameters = model.Hyperparameters{
		model.GlobalBatchSize: model.Hyperparameter{
			ConstHyperparameter: &model.ConstHyperparameter{Val: 64},
		},
	}
	c.CheckpointStorage.SharedFSConfig.HostPath = "/"

	e := &model.Experiment{
		State:                model.ActiveState,
		Config:               c,
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
		ExperimentID: eID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	for _, o := range opts {
		o.apply(t)
	}
	return t
}
