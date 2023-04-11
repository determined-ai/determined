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
	"github.com/pkg/errors"
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

	require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
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
	require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
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

func TestBatchesProcessed(t *testing.T) {
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

	dbTr, err := db.TrialByID(tr.ID)
	require.NoError(t, err)
	require.Equal(t, 0, dbTr.TotalBatches)

	a := &model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-%d", tr.TaskID, 0)),
		TaskID:       tr.TaskID,
		StartTime:    ptrs.Ptr(time.Now()),
	}
	err = db.AddAllocation(a)
	require.NoError(t, err, "failed to add allocation")

	metrics, err := structpb.NewStruct(map[string]any{"loss": 10})
	require.NoError(t, err)

	testMetricReporting := func(typ string, trialRunId, batches, expectedTotalBatches int) error {
		require.NoError(t, db.UpdateTrialRunID(tr.ID, trialRunId))
		trialMetrics := &trialv1.TrialMetrics{
			TrialId:        int32(tr.ID),
			TrialRunId:     int32(trialRunId),
			StepsCompleted: int32(batches),
			Metrics:        &commonv1.Metrics{AvgMetrics: metrics},
		}
		switch typ {
		case "training":
			require.NoError(t, db.AddTrainingMetrics(ctx, trialMetrics))
		case "validation":
			require.NoError(t, db.AddValidationMetrics(ctx, trialMetrics))
		case "checkpoint":
			require.NoError(t, AddCheckpointMetadata(ctx, &model.CheckpointV2{
				UUID:         uuid.New(),
				TaskID:       task.TaskID,
				AllocationID: a.AllocationID,
				ReportTime:   time.Now(),
				State:        model.CompletedState,
				Metadata:     map[string]any{"steps_completed": batches},
			}))

		default:
			return errors.Errorf("unknown type %s", typ)
		}

		dbTr, err = db.TrialByID(tr.ID)
		require.NoError(t, err)
		require.Equal(t, expectedTotalBatches, dbTr.TotalBatches)
		return nil
	}

	cases := []struct {
		typ             string
		trialRunID      int
		batches         int
		expectedBatches int
	}{ // order matters.
		{"training", 0, 10, 10},
		{"validation", 0, 10, 10},
		{"training", 0, 20, 20},
		{"validation", 0, 20, 20},
		{"validation", 0, 30, 30}, // will be rolled back.
		{"training", 0, 25, 30},
		{"validation", 1, 25, 25}, // rollback via validations.
		{"validation", 1, 30, 30}, // will be rolled back.
		{"training", 1, 30, 30},   // will be rolled back.
		{"training", 2, 27, 27},   // rollback via training.
		{"checkpoint", 2, 30, 27}, // CHECK: do NOT account for steps_completed here.
		{"checkpoint", 3, 25, 27}, // CHECK: do NOT account for steps_completed here.
	}
	for _, c := range cases {
		require.NoError(t, testMetricReporting(c.typ, c.trialRunID, c.batches, c.expectedBatches))
	}

	// check rollbacks happened as expected.
	archivedSteps, err := Bun().NewSelect().Table("raw_steps").
		Where("trial_id = ?", tr.ID).Where("archived = true").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, archivedSteps, "trial id %d", tr.ID)

	archivedValidations, err := Bun().NewSelect().Table("raw_validations").
		Where("trial_id = ?", tr.ID).Where("archived = true").Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, archivedValidations, "trial id %d", tr.ID)
}
