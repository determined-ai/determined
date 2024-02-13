//go:build integration
// +build integration

package command

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestMain(m *testing.M) {
	pgDB, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestGetCommandOwnerID(t *testing.T) {
	ctx := context.Background()

	_, err := GetCommandOwnerID(ctx, model.TaskID(uuid.New().String()))
	require.ErrorIs(t, err, db.ErrNotFound)

	user := db.RequireMockUser(t, db.SingleDB())
	commandID := requireMockCommandID(t, user.ID)

	userID, err := GetCommandOwnerID(ctx, commandID)
	require.NoError(t, err)
	require.Equal(t, user.ID, userID)
}

func TestIdentifyTask(t *testing.T) {
	ctx := context.Background()
	pgDB := db.SingleDB()

	_, err := IdentifyTask(ctx, model.TaskID(uuid.New().String()))
	require.ErrorIs(t, err, db.ErrNotFound)

	// Command.
	user := db.RequireMockUser(t, pgDB)
	commandID := requireMockCommandID(t, user.ID)

	meta, err := IdentifyTask(ctx, commandID)
	require.NoError(t, err)
	require.Equal(t, TaskMetadata{
		WorkspaceID: 1,
		TaskType:    model.TaskTypeCommand,
	}, meta)

	// Tensorboard.
	exp0 := db.RequireMockExperiment(t, pgDB, user)
	t0, trialTask := db.RequireMockTrial(t, pgDB, exp0)
	t1, _ := db.RequireMockTrial(t, pgDB, exp0)
	exp1 := db.RequireMockExperiment(t, pgDB, user)

	expIDs := []int{exp0.ID, exp1.ID}
	trialIDs := []int{t0.ID, t1.ID}
	tensorboardID := requireMockTensorboardID(t, user.ID, expIDs, trialIDs)

	meta, err = IdentifyTask(ctx, tensorboardID)
	require.NoError(t, err)
	require.Equal(t, TaskMetadata{
		WorkspaceID:   1,
		TaskType:      model.TaskTypeTensorboard,
		ExperimentIDs: []int32{int32(expIDs[0]), int32(expIDs[1])},
		TrialIDs:      []int32{int32(trialIDs[0]), int32(trialIDs[1])},
	}, meta)

	// Experiment task.
	// This always is not found and is probably a footgun from function name / comment.
	// TODO(DET-10005) remove this footgun.
	_, err = IdentifyTask(ctx, trialTask.TaskID)
	require.ErrorIs(t, err, db.ErrNotFound)
}

// requireMockCommandID creates a mock command and returns a command ID.
func requireMockCommandID(t *testing.T, userID model.UserID) model.TaskID {
	pgDB := db.SingleDB()

	task := db.RequireMockTask(t, pgDB, &userID)
	alloc := db.RequireMockAllocation(t, pgDB, task.TaskID)

	mockCommand := struct {
		bun.BaseModel `bun:"table:command_state"`

		TaskID             model.TaskID
		AllocationID       model.AllocationID
		GenericCommandSpec map[string]any
	}{
		TaskID:       task.TaskID,
		AllocationID: alloc.AllocationID,
		GenericCommandSpec: map[string]any{
			"TaskType": model.TaskTypeCommand,
			"Metadata": map[string]any{
				"workspace_id": 1,
			},
			"Base": map[string]any{
				"Owner": map[string]any{
					"id": userID,
				},
			},
		},
	}
	_, err := db.Bun().NewInsert().Model(&mockCommand).Exec(context.TODO())
	require.NoError(t, err)

	return task.TaskID
}

// requireMockTensorboardID creates a mock tensorboard and returns a tensorboard ID.
func requireMockTensorboardID(
	t *testing.T, userID model.UserID, expIDs, trialIDs []int,
) model.TaskID {
	pgDB := db.SingleDB()

	task := db.RequireMockTask(t, pgDB, &userID)
	alloc := db.RequireMockAllocation(t, pgDB, task.TaskID)

	mockTensorboard := struct {
		bun.BaseModel `bun:"table:command_state"`

		TaskID             model.TaskID
		AllocationID       model.AllocationID
		GenericCommandSpec map[string]any
	}{
		TaskID:       task.TaskID,
		AllocationID: alloc.AllocationID,
		GenericCommandSpec: map[string]any{
			"TaskType": model.TaskTypeTensorboard,
			"Metadata": map[string]any{
				"workspace_id":   1,
				"experiment_ids": expIDs,
				"trial_ids":      trialIDs,
			},
			"Base": map[string]any{
				"Owner": map[string]any{
					"id": userID,
				},
			},
		},
	}
	_, err := db.Bun().NewInsert().Model(&mockTensorboard).Exec(context.TODO())
	require.NoError(t, err)

	return task.TaskID
}
