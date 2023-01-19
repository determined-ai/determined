//go:build integration
// +build integration

package model

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

var defaultDeterminedUID UserID = 2

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
//nolint: exhaustivestruct
func ExperimentModel(opts ...ExperimentModelOption) (*Experiment, expconf.ExperimentConfig) {
	maxLength := expconf.NewLengthInBatches(100)
	activeConfig := expconf.ExperimentConfig{
		RawSearcher: &expconf.SearcherConfig{
			RawMetric: ptrs.Ptr("loss"),
			RawSingleConfig: &expconf.SingleConfig{
				RawMaxLength: &maxLength,
			},
		},
		RawEntrypoint:      &expconf.Entrypoint{RawEntrypoint: "model_def:SomeTrialClass"},
		RawHyperparameters: expconf.Hyperparameters{},
		RawCheckpointStorage: &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath: ptrs.Ptr("/"),
			},
		},
	}
	activeConfig = schemas.WithDefaults(activeConfig)
	DefaultTaskContainerDefaults().MergeIntoExpConfig(&activeConfig)

	e := &Experiment{
		JobID:                NewJobID(),
		State:                ActiveState,
		Config:               activeConfig.AsLegacy(),
		StartTime:            time.Now(),
		OwnerID:              &defaultDeterminedUID,
		ModelDefinitionBytes: []byte{},
		ProjectID:            1,
	}

	for _, o := range opts {
		o.apply(e)
	}
	return e, activeConfig
}
