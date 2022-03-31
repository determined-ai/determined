//go:build integration
// +build integration

package api

import (
	"context"
	"testing"
	"time"

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
	t.Run("a", func(t *testing.T) {
		experiment := model.ExperimentModel()
		err := db.AddExperiment(experiment)
		assert.NilError(t, err, "failed to insert experiment")

		trial := model.TrialModel(
			experiment.ID, experiment.JobID, model.WithTrialState(model.ActiveState))
		err = db.AddTrial(trial)
		assert.NilError(t, err, "failed to insert trial")

		checkpointUuid := uuid.NewString()
		latestBatch := int32(10)
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
		t.Logf("checkpoint meta=%v", checkpointMeta)
		assert.NilError(t, err, "failed to add checkpoint meta")

		ctx, _ := context.WithTimeout(creds, 10*time.Second)
		req := apiv1.GetCheckpointRequest{CheckpointUuid: checkpointUuid}

		ckptResp, err := cl.GetCheckpoint(ctx, &req)
		assert.NilError(t, err, "failed to get checkpoint from api")
		ckptCl := *ckptResp.Checkpoint
		fields := ckptCl.ExperimentConfig.GetFields()
		entrypoint := fields["entrypoint"].GetStringValue()
		assert.Equal(t, ckptCl.Uuid, checkpointUuid)
		t.Logf("Entrypoint: %s", entrypoint)
		// assert.Equal(t, ckptCl.ExperimentConfig.Entrypoint.ModelDef, "model_def:SomeTrialClass")
		assert.Equal(t, ckptCl.ExperimentId, int32(experiment.ID))
		assert.Equal(t, ckptCl.TrialId, int32(trial.ID))
		t.Logf("Checkpoint from api: %v", ckptCl)
		t.Logf("Uuid=%s", ckptCl.Uuid)
		t.Logf("ExperimentConfig=%v", ckptCl.ExperimentConfig)
	})
}
