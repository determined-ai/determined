//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
)

func TestProtoGetTrial(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	exp, activeConfig := model.ExperimentModel()
	err := db.AddExperiment(exp, activeConfig)
	require.NoError(t, err, "failed to add experiment")

	task := RequireMockTask(t, db, exp.OwnerID)
	tr := model.Trial{
		TaskID:       task.TaskID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	err = db.AddTrial(&tr)
	require.NoError(t, err, "failed to add trial")

	startTime := time.Now().UTC()
	for i := 0; i < 3; i++ {
		a := &model.Allocation{
			AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, i)),
			TaskID:       tr.TaskID,
			StartTime:    ptrs.Ptr(startTime.Add(time.Duration(i) * time.Second)),
			EndTime:      ptrs.Ptr(startTime.Add(time.Duration(i+1) * time.Second)),
		}
		err = db.AddAllocation(a)
		require.NoError(t, err, "failed to add allocation")
		err = db.CompleteAllocation(a)
		require.NoError(t, err, "failed to complete allocation")
	}

	var trResp trialv1.Trial
	err = db.QueryProtof(
		"proto_get_trials_plus",
		[]any{"($1::int, $2::int)"},
		&trResp,
		tr.ID,
		1,
	)
	require.NoError(t, err, "failed to query trial")
	require.Equal(t, trResp.WallClockTime, float64(3), "wall clock time is wrong")
}

// Covers an issue where checkpoint_view returned multiple records per checkpoint
// due to the LEFT JOIN raw_steps ON total_batches AND trial_id.
// This is in this file because AddValidationMetrics broke the assumption the join uses.
// That assumption being that for each trial_id and total_batches that there
// is at most one unarchived result.
func TestAddValidationMetricsDupeCheckpoints(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	exp, activeConfig := model.ExperimentModel()
	require.NoError(t, db.AddExperiment(exp, activeConfig))
	task := RequireMockTask(t, db, exp.OwnerID)
	tr := model.Trial{
		TaskID:       task.TaskID,
		JobID:        exp.JobID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
	}
	require.NoError(t, db.AddTrial(&tr))

	trainMetrics, err := structpb.NewStruct(map[string]any{"loss": 10})
	require.NoError(t, err)
	valMetrics, err := structpb.NewStruct(map[string]any{"loss": 50})
	require.NoError(t, err)

	// First trial run.
	a := &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 0)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	require.NoError(t, db.AddAllocation(a))

	// Report training metrics.
	require.NoError(t, db.AddTrainingMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     0,
		StepsCompleted: 50,
		Metrics:        &commonv1.Metrics{AvgMetrics: trainMetrics},
	}))

	require.NoError(t, db.AddCheckpointMetadata(ctx, &model.CheckpointV2{
		UUID:         uuid.New(),
		TaskID:       task.TaskID,
		AllocationID: a.AllocationID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Metadata:     map[string]any{"steps_completed": 50},
	}))

	// Trial gets interrupted and starts in the future with a new trial run ID.
	a = &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 1)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	require.NoError(t, db.AddAllocation(a))
	require.NoError(t, db.UpdateTrialRunID(tr.ID, 1))

	// Now trial runs validation.
	require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     1,
		StepsCompleted: 50,
		Metrics:        &commonv1.Metrics{AvgMetrics: valMetrics},
	}))

	checkpoints := []*checkpointv1.Checkpoint{}
	require.NoError(t, db.QueryProto("get_checkpoints_for_experiment", &checkpoints, exp.ID))
	require.Len(t, checkpoints, 1)
	require.Equal(t, 10.0, checkpoints[0].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 50.0, checkpoints[0].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])

	// Dummy metrics still happen if no other results at given total_batches.
	checkpoint2UUID := uuid.New()
	valMetrics2, err := structpb.NewStruct(map[string]any{"loss": 1.5})
	require.NoError(t, err)
	require.NoError(t, db.AddValidationMetrics(ctx, &trialv1.TrialMetrics{
		TrialId:        int32(tr.ID),
		TrialRunId:     1,
		StepsCompleted: 400,
		Metrics:        &commonv1.Metrics{AvgMetrics: valMetrics2},
	}))
	require.NoError(t, db.AddCheckpointMetadata(ctx, &model.CheckpointV2{
		UUID:         checkpoint2UUID,
		TaskID:       task.TaskID,
		AllocationID: a.AllocationID,
		ReportTime:   time.Now(),
		State:        model.ActiveState,
		Metadata:     map[string]any{"steps_completed": 400},
	}))
	checkpoints = []*checkpointv1.Checkpoint{}
	require.NoError(t, db.QueryProto("get_checkpoints_for_experiment", &checkpoints, exp.ID))
	require.Len(t, checkpoints, 2)
	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Uuid != checkpoint2UUID.String() // Have second checkpoint later.
	})

	require.Equal(t, 10.0, checkpoints[0].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 50.0, checkpoints[0].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])

	require.Equal(t, nil, checkpoints[1].Training.TrainingMetrics.AvgMetrics.AsMap()["loss"])
	require.Equal(t, 1.5, checkpoints[1].Training.ValidationMetrics.AvgMetrics.AsMap()["loss"])
}
