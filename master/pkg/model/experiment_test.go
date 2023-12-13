package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRunCollectionNameFromExperiment(t *testing.T) {
	t.Run("no externalID", func(t *testing.T) {
		actual := runCollectionNameFromExperiment(&Experiment{
			ID: 123,
		})

		require.Equal(t, "experiment_id:123", actual)
	})

	t.Run("externalID", func(t *testing.T) {
		actual := runCollectionNameFromExperiment(&Experiment{
			ID:                   123,
			ExternalExperimentID: ptrs.Ptr("uuid123"),
		})

		require.Equal(t, "experiment_id:123, external_experiment_id:uuid123", actual)
	})
}

func TestToRunCollection(t *testing.T) {
	exp := &Experiment{
		ID:    4,
		JobID: "job-id",
		State: CompletedState,
		Notes: "notes",
		Config: expconf.LegacyConfig{
			Searcher: expconf.LegacySearcher{
				Name: "fake-searcher",
			},
		},
		OriginalConfig:       "orig config",
		ModelDefinitionBytes: []byte{155},
		StartTime:            time.Now(),
		EndTime:              ptrs.Ptr(time.Now()),
		ParentID:             ptrs.Ptr(18),
		Archived:             true,
		OwnerID:              ptrs.Ptr(UserID(99)),
		Username:             "username",
		ProjectID:            13,
		Unmanaged:            true,
		ExternalExperimentID: ptrs.Ptr("external"),
		Progress:             ptrs.Ptr(31.0),
	}

	expectedRunCollection := &RunCollection{
		ID:                      exp.ID,
		Name:                    runCollectionNameFromExperiment(exp),
		State:                   exp.State,
		Notes:                   exp.Notes,
		ProjectID:               exp.ProjectID,
		OwnerID:                 exp.OwnerID,
		Progress:                exp.Progress,
		Archived:                exp.Archived,
		StartTime:               exp.StartTime,
		EndTime:                 exp.EndTime,
		ExternalRunCollectionID: exp.ExternalExperimentID,
	}

	expectedExpV2 := &ExperimentV2{
		RunCollectionID:      exp.ID,
		JobID:                exp.JobID,
		Config:               exp.Config,
		OriginalConfig:       exp.OriginalConfig,
		ModelDefinitionBytes: exp.ModelDefinitionBytes,
		ParentID:             exp.ParentID,
		Username:             exp.Username,
		Unmanaged:            exp.Unmanaged,
	}

	actualRunCollection, actualExpV2 := exp.ToRunCollectionAndExperimentV2()
	require.Equal(t, expectedRunCollection, actualRunCollection)
	require.Equal(t, expectedExpV2, actualExpV2)
}

func TestToRun(t *testing.T) {
	trial := &Trial{
		ID:                    3,
		RequestID:             ptrs.Ptr(RequestID(uuid.New())),
		ExperimentID:          4,
		State:                 CompletedState,
		StartTime:             time.Now(),
		EndTime:               ptrs.Ptr(time.Now()),
		HParams:               map[string]any{"test": "test"},
		WarmStartCheckpointID: ptrs.Ptr(2),
		Seed:                  12,
		TotalBatches:          15,
		ExternalTrialID:       ptrs.Ptr("ext"),
		RunID:                 19,
		LastActivity:          ptrs.Ptr(time.Now()),
	}

	expectedRun := &Run{
		ID:                    trial.ID,
		ExperimentID:          trial.ExperimentID,
		State:                 trial.State,
		StartTime:             trial.StartTime,
		EndTime:               trial.EndTime,
		HParams:               trial.HParams,
		WarmStartCheckpointID: trial.WarmStartCheckpointID,
		TotalBatches:          trial.TotalBatches,
		ExternalRunID:         trial.ExternalTrialID,
		RestartID:             19,
		LastActivity:          trial.LastActivity,
	}
	expectedTrial := &TrialV2{
		RunID:     trial.ID,
		RequestID: trial.RequestID,
		Seed:      trial.Seed,
	}

	actualRun, actualTrial := trial.ToRunAndTrialV2()
	require.Equal(t, expectedRun, actualRun)
	require.Equal(t, expectedTrial, actualTrial)
}
