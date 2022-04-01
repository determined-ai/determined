//go:build integration
// +build integration

package api

import (
	"context"
	"testing"
	"time"

	"github.com/determined-ai/determined/proto/pkg/checkpointv1"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/google/uuid"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestGetCheckpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")
	testGetCheckpoint(t, creds, cl, pgDB)
}

func testGetCheckpoint(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, db *db.PgDB,
) {
	type testCase struct {
		name     string
		validate bool
	}

	testCases := []testCase{
		{
			name:     "checkpoint with validation",
			validate: true,
		},
		{
			name:     "checkpoint without validation",
			validate: false,
		},
	}

	runTestCase := func(t *testing.T, tc testCase, id int) {
		t.Run(tc.name, func(t *testing.T) {
			experiment := model.ExperimentModel()
			err := db.AddExperiment(experiment)
			assert.NilError(t, err, "failed to insert experiment")

			trial := model.TrialModel(
				experiment.ID, experiment.JobID, model.WithTrialState(model.ActiveState))
			err = db.AddTrial(trial)
			assert.NilError(t, err, "failed to insert trial")

			latestBatch := int32(10)
			if tc.validate {
				trialMetrics := trialv1.TrialMetrics{
					TrialId:     int32(trial.ID),
					TrialRunId:  int32(0),
					LatestBatch: latestBatch,
					Metrics:     nil,
				}

				err = db.AddValidationMetrics(context.Background(), &trialMetrics)
				assert.NilError(t, err, "failed to add validation metrics")
			}

			checkpointUuid := uuid.NewString()
			checkpointMeta := trialv1.CheckpointMetadata{
				TrialId:           int32(trial.ID),
				TrialRunId:        int32(0),
				Uuid:              checkpointUuid,
				Resources:         map[string]int64{"ok": 1.0},
				Framework:         "some framework",
				Format:            "some format",
				DeterminedVersion: "1.0.0",
				LatestBatch:       latestBatch,
			}
			err = db.AddCheckpointMetadata(context.Background(), &checkpointMeta)

			// TODO remove
			t.Logf("checkpoint meta=%v", checkpointMeta)

			assert.NilError(t, err, "failed to add checkpoint meta")

			ctx, _ := context.WithTimeout(creds, 10*time.Second)
			req := apiv1.GetCheckpointRequest{CheckpointUuid: checkpointUuid}

			ckptResp, err := cl.GetCheckpoint(ctx, &req)
			assert.NilError(t, err, "failed to get checkpoint from api")
			ckptCl := *ckptResp.Checkpoint
			assert.Equal(t, ckptCl.Uuid, checkpointUuid)

			entrypoint := ckptCl.ExperimentConfig.GetFields()["entrypoint"].GetStringValue()
			assert.Equal(t, entrypoint, "model_def:SomeTrialClass")

			assert.Equal(t, ckptCl.ExperimentId, int32(experiment.ID))
			assert.Equal(t, ckptCl.TrialId, int32(trial.ID))
			assert.Equal(t, ckptCl.Framework, "some framework")
			assert.Equal(t, ckptCl.Format, "some format")

			if tc.validate {
				assert.Equal(t, ckptCl.ValidationState, checkpointv1.State_STATE_COMPLETED)
			} else {
				assert.Equal(t, ckptCl.ValidationState, checkpointv1.State_STATE_UNSPECIFIED)
			}
			assert.Equal(t, ckptCl.State, checkpointv1.State_STATE_COMPLETED)
			t.Logf("Checkpoint from api: %v", ckptCl)
			t.Logf("Uuid=%s", ckptCl.Uuid)
			t.Logf("ExperimentConfig=%v", ckptCl.ExperimentConfig)
		})
	}

	for idx, tc := range testCases {
		runTestCase(t, tc, idx)
	}
}
