//go:build integration
// +build integration

package api

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/test/testutils"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/google/uuid"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils/protoconverter"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func TestGetCheckpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	telemetry.MockTelemetry()
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")
	testGetCheckpoint(creds, t, cl, pgDB)
}

func TestGetExperimentCheckpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")
	testGetExperimentCheckpoints(creds, t, cl, pgDB)
}

func TestGetTrialCheckpoints(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
	defer cancel()
	assert.NilError(t, err, "failed to start master")
	testGetTrialCheckpoints(creds, t, cl, pgDB)
}

func testGetCheckpoint(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
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

	conv := &protoconverter.ProtoConverter{}

	runTestCase := func(t *testing.T, tc testCase, id int) {
		t.Run(tc.name, func(t *testing.T) {
			experiment, trial, allocation := createPrereqs(t, pgDB)

			stepsCompleted := int32(10)
			if tc.validate {
				trialMetrics := trialv1.TrialMetrics{
					TrialId:        int32(trial.ID),
					TrialRunId:     int32(0),
					StepsCompleted: stepsCompleted,
					Metrics: &commonv1.Metrics{
						AvgMetrics: &structpb.Struct{
							Fields: map[string]*structpb.Value{
								"okness": {
									Kind: &structpb.Value_NumberValue{
										NumberValue: float64(0.5),
									},
								},
							},
						},
					},
				}

				err := pgDB.AddValidationMetrics(context.Background(), &trialMetrics)
				assert.NilError(t, err, "failed to add validation metrics")
			}

			checkpointUUID := uuid.NewString()
			checkpointMeta := model.CheckpointV2{
				UUID:         conv.ToUUID(checkpointUUID),
				TaskID:       allocation.TaskID,
				AllocationID: &allocation.AllocationID,
				ReportTime:   timestamppb.Now().AsTime(),
				State:        conv.ToCheckpointState(checkpointv1.State_STATE_COMPLETED),
				Resources:    map[string]int64{"ok": 1.0},
				Metadata: map[string]interface{}{
					"steps_completed":    stepsCompleted,
					"framework":          "some framework",
					"determined_version": "1.0.0",
				},
			}
			err := db.AddCheckpointMetadata(context.Background(), &checkpointMeta)

			assert.NilError(t, err, "failed to add checkpoint meta")

			ctx, cancel := context.WithTimeout(creds, 10*time.Second)
			defer cancel()
			req := apiv1.GetCheckpointRequest{CheckpointUuid: checkpointUUID}

			ckptResp, err := cl.GetCheckpoint(ctx, &req)
			assert.NilError(t, err, "failed to get checkpoint from api")
			ckptCl := ckptResp.Checkpoint
			assert.Equal(t, ckptCl.Uuid, checkpointUUID)

			entrypoint := ckptCl.Training.ExperimentConfig.GetFields()["entrypoint"].GetStringValue()
			assert.Equal(t, entrypoint, "model_def:SomeTrialClass")

			assert.Equal(t, ckptCl.Training.ExperimentId.Value, int32(experiment.ID))
			assert.Equal(t, ckptCl.Training.TrialId.Value, int32(trial.ID))

			actualFramework := ckptCl.Metadata.GetFields()["framework"].GetStringValue()
			assert.Equal(t, actualFramework, "some framework")

			t.Logf("validationMetrics: %v", ckptCl.Training.ValidationMetrics)

			if tc.validate {
				assert.Assert(t, ckptCl.Training.ValidationMetrics.AvgMetrics != nil)
			} else {
				assert.Assert(t, ckptCl.Training.ValidationMetrics.AvgMetrics == nil)
			}
			assert.Equal(t, ckptCl.State, checkpointv1.State_STATE_COMPLETED)
		})
	}

	for idx, tc := range testCases {
		runTestCase(t, tc, idx)
	}
}

func testGetExperimentCheckpoints(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	experiment, trial, allocation := createPrereqs(t, pgDB)
	conv := &protoconverter.ProtoConverter{}

	var uuids []string
	for i := 0; i < 5; i++ {
		checkpointUUID := uuid.NewString()
		uuids = append(uuids, checkpointUUID)
		stepsCompleted := 10 * i
		checkpointMeta := model.CheckpointV2{
			UUID:         conv.ToUUID(checkpointUUID),
			TaskID:       allocation.TaskID,
			AllocationID: &allocation.AllocationID,
			ReportTime:   timestamppb.Now().AsTime(),
			State:        conv.ToCheckpointState(checkpointv1.State_STATE_COMPLETED),
			Resources:    map[string]int64{"ok": 1.0},
			Metadata: map[string]interface{}{
				"steps_completed":    stepsCompleted,
				"framework":          "some framework",
				"determined_version": "1.0.0",
			},
		}

		err := db.AddCheckpointMetadata(context.Background(), &checkpointMeta)
		assert.NilError(t, err, "failed to add checkpoint meta")

		trialMetrics := trialv1.TrialMetrics{
			TrialId:        int32(trial.ID),
			TrialRunId:     int32(0),
			StepsCompleted: int32(stepsCompleted),
			Metrics: &commonv1.Metrics{
				AvgMetrics: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"loss": {
							Kind: &structpb.Value_NumberValue{
								NumberValue: float64(i) * (4.5 - float64(i)),
							},
						},
					},
				},
			},
		}

		err = pgDB.AddValidationMetrics(context.Background(), &trialMetrics)
		assert.NilError(t, err, "failed to add validation metrics")
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
	req.OrderBy = apiv1.OrderBy_ORDER_BY_ASC
	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	for j := 0; j < 5; j++ {
		assert.Equal(t, ckptsCl[j].Uuid, uuids[j])
	}

	// check sorting by searcher metric
	req.SortBy = apiv1.GetExperimentCheckpointsRequest_SORT_BY_SEARCHER_METRIC
	req.OrderBy = apiv1.OrderBy_ORDER_BY_UNSPECIFIED
	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	// the metric is 4.5*i - i^2
	assert.Equal(t, ckptsCl[0].Uuid, uuids[0]) // metric(0) = 0
	assert.Equal(t, ckptsCl[1].Uuid, uuids[4]) // metric(4) = 2
	assert.Equal(t, ckptsCl[2].Uuid, uuids[1]) // metric(1) = 3.5

	// check sorting by assending uuid
	req.SortBy = apiv1.GetExperimentCheckpointsRequest_SORT_BY_UUID
	resp, err = cl.GetExperimentCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetExperimentCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	sort.Strings(uuids)
	for j := 0; j < 5; j++ {
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
	for j := 2; j < 5; j++ {
		assert.Equal(t, ckptsCl[j-2].Uuid, uuids[j])
	}
}

func testGetTrialCheckpoints(
	creds context.Context, t *testing.T, cl apiv1.DeterminedClient, pgDB *db.PgDB,
) {
	_, trial, allocation := createPrereqs(t, pgDB)
	conv := &protoconverter.ProtoConverter{}

	var uuids []string
	for i := 0; i < 5; i++ {
		checkpointUUID := uuid.NewString()
		uuids = append(uuids, checkpointUUID)
		stepsCompleted := 10 * i
		checkpointMeta := model.CheckpointV2{
			UUID:         conv.ToUUID(checkpointUUID),
			TaskID:       allocation.TaskID,
			AllocationID: &allocation.AllocationID,
			ReportTime:   timestamppb.Now().AsTime(),
			State:        conv.ToCheckpointState(checkpointv1.State_STATE_COMPLETED),
			Resources:    map[string]int64{"ok": 1.0},
			Metadata: map[string]interface{}{
				"steps_completed":    stepsCompleted,
				"framework":          "some framework",
				"determined_version": "1.0.0",
			},
		}
		err := db.AddCheckpointMetadata(context.Background(), &checkpointMeta)
		assert.NilError(t, err, "failed to add checkpoint meta")
	}

	ctx, cancel := context.WithTimeout(creds, 10*time.Second)
	defer cancel()

	req := apiv1.GetTrialCheckpointsRequest{
		Id: int32(trial.ID),
	}

	resp, err := cl.GetTrialCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetTrialCheckpoints error")
	ckptsCl := resp.Checkpoints

	// default sort order is unspecified
	assert.Equal(t, len(ckptsCl), 5)

	// check sorting by assending end time
	req.SortBy = apiv1.GetTrialCheckpointsRequest_SORT_BY_END_TIME
	resp, err = cl.GetTrialCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetTrialCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	for j := 0; j < 5; j++ {
		assert.Equal(t, ckptsCl[j].Uuid, uuids[j])
	}

	// check sorting by assending uuid
	req.SortBy = apiv1.GetTrialCheckpointsRequest_SORT_BY_UUID
	resp, err = cl.GetTrialCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetTrialCheckpoints error")
	ckptsCl = resp.Checkpoints

	assert.Equal(t, len(ckptsCl), 5)
	sort.Strings(uuids)
	for j := 0; j < 5; j++ {
		assert.Equal(t, ckptsCl[j].Uuid, uuids[j])
	}

	req.Limit = 3
	req.Offset = 2

	resp, err = cl.GetTrialCheckpoints(ctx, &req)
	assert.NilError(t, err, "GetTrialCheckpoints error")
	ckptsCl = resp.Checkpoints

	// ascending uuid
	assert.Equal(t, len(ckptsCl), 3)
	sort.Strings(uuids)
	for j := 2; j < 5; j++ {
		assert.Equal(t, ckptsCl[j-2].Uuid, uuids[j])
	}
}

func createPrereqs(t *testing.T, pgDB *db.PgDB) (
	*model.Experiment, *model.Trial, *model.Allocation,
) {
	experiment, activeConfig := model.ExperimentModel()
	err := pgDB.AddExperiment(experiment, activeConfig)
	assert.NilError(t, err, "failed to insert experiment")

	task := db.RequireMockTask(t, pgDB, experiment.OwnerID)
	trial := &model.Trial{
		ExperimentID: experiment.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}

	err = db.AddTrial(context.TODO(), trial, task.TaskID)
	assert.NilError(t, err, "failed to insert trial")
	t.Logf("Created trial=%v", trial)

	startTime := time.Now().UTC()
	a := &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", task.TaskID, 1)),
		TaskID:       task.TaskID,
		StartTime:    ptrs.Ptr(startTime),
		EndTime:      ptrs.Ptr(startTime.Add(time.Duration(1) * time.Second)),
	}
	err = pgDB.AddAllocation(a)
	assert.NilError(t, err, "failed to add allocation")
	err = pgDB.CompleteAllocation(a)
	assert.NilError(t, err, "failed to complete allocation")

	return experiment, trial, a
}
