package model

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

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
		ProjectID:             12,
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

	actualRun, actualTrial := trial.ToRunAndTrialV2(expectedRun.ProjectID)
	require.Equal(t, expectedRun, actualRun)
	require.Equal(t, expectedTrial, actualTrial)
}
