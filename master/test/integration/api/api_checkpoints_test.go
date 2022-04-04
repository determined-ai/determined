//go:build integration
// +build integration

package api

import (
	"context"
	"sort"
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

func TestGetExperimentCheckpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")
	testGetExperimentCheckpoints(t, creds, cl, pgDB)
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
			experiment, trial := createExperimentAndTrial(t, db)

			latestBatch := int32(10)
			if tc.validate {
				trialMetrics := trialv1.TrialMetrics{
					TrialId:     int32(trial.ID),
					TrialRunId:  int32(0),
					LatestBatch: latestBatch,
					Metrics:     nil,
				}

				err := db.AddValidationMetrics(context.Background(), &trialMetrics)
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
			err := db.AddCheckpointMetadata(context.Background(), &checkpointMeta)

			assert.NilError(t, err, "failed to add checkpoint meta")

			ctx, cancel := context.WithTimeout(creds, 10*time.Second)
			defer cancel()
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
		})
	}

	for idx, tc := range testCases {
		runTestCase(t, tc, idx)
	}
}

func testGetExperimentCheckpoints(
	t *testing.T, creds context.Context, cl apiv1.DeterminedClient, db *db.PgDB,
) {
	experiment, trial := createExperimentAndTrial(t, db)

	var uuids []string
	for i := 0; i < 5; i++ {
		checkpointUuid := uuid.NewString()
		uuids = append(uuids, checkpointUuid)
		checkpointMeta := trialv1.CheckpointMetadata{
			TrialId:           int32(trial.ID),
			TrialRunId:        int32(0),
			Uuid:              checkpointUuid,
			Resources:         map[string]int64{"ok": 1.0},
			Framework:         "some framework",
			Format:            "some format",
			DeterminedVersion: "1.0.0",
			LatestBatch:       int32(10 * i),
		}
		err := db.AddCheckpointMetadata(context.Background(), &checkpointMeta)
		assert.NilError(t, err, "failed to add checkpoint meta")
	}

	ctx, cancel := context.WithTimeout(creds, 10*time.Second)
	defer cancel()

	req := apiv1.GetExperimentCheckpointsRequest{
		Id: int32(experiment.ID),
	}

	resp, err := cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl := resp.Checkpoints

	// default sort order is unspecified
	assert.Equal(t, len(ckptsCl), 5)

	// check sorting by assending end time
	req.SortBy = apiv1.GetExperimentCheckpointsRequest_SORT_BY_END_TIME
	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	for j := 0; j < 5; j += 1 {
		assert.Equal(t, ckptsCl[j].Uuid, uuids[j])
	}

	// check sorting by assending uuid
	req.SortBy = apiv1.GetExperimentCheckpointsRequest_SORT_BY_UUID
	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	sort.Strings(uuids)
	for j := 0; j < 5; j += 1 {
		assert.Equal(t, ckptsCl[j].Uuid, uuids[j])
	}

	req.Limit = 3
	req.Offset = 2

	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	// ascending uuid
	assert.Equal(t, len(ckptsCl), 3)
	sort.Strings(uuids)
	for j := 2; j < 5; j += 1 {
		assert.Equal(t, ckptsCl[j-2].Uuid, uuids[j])
	}

}

func createExperimentAndTrial(t *testing.T, db *db.PgDB) (*model.Experiment, *model.Trial) {
	experiment := model.ExperimentModel()
	err := db.AddExperiment(experiment)
	assert.NilError(t, err, "failed to insert experiment")

	trial := model.TrialModel(
		experiment.ID, experiment.JobID, model.WithTrialState(model.ActiveState))
	err = db.AddTrial(trial)
	assert.NilError(t, err, "failed to insert trial")

	return experiment, trial
}
