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

	expected := &Run{
		ID:                    trial.ID,
		RequestID:             trial.RequestID,
		ExperimentID:          trial.ExperimentID,
		State:                 trial.State,
		StartTime:             trial.StartTime,
		EndTime:               trial.EndTime,
		HParams:               trial.HParams,
		WarmStartCheckpointID: trial.WarmStartCheckpointID,
		Seed:                  trial.Seed,
		TotalBatches:          trial.TotalBatches,
		ExternalRunID:         trial.ExternalTrialID,
		RestartID:             19,
		LastActivity:          trial.LastActivity,
	}

	require.Equal(t, expected, trial.ToRun())
}
