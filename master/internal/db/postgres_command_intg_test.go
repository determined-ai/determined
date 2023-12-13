//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestGetCommandOwnerID(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	_, err := GetCommandOwnerID(ctx, model.TaskID(uuid.New().String()))
	require.ErrorIs(t, err, ErrNotFound)

	user := RequireMockUser(t, db)
	commandID := RequireMockCommandID(t, db, user.ID)

	userID, err := GetCommandOwnerID(ctx, commandID)
	require.NoError(t, err)
	require.Equal(t, user.ID, userID)
}

func TestIdentifyTask(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	_, err := IdentifyTask(ctx, model.TaskID(uuid.New().String()))
	require.ErrorIs(t, err, ErrNotFound)

	// Command.
	user := RequireMockUser(t, db)
	commandID := RequireMockCommandID(t, db, user.ID)

	meta, err := IdentifyTask(ctx, commandID)
	require.NoError(t, err)
	require.Equal(t, TaskMetadata{
		WorkspaceID: 1,
		// TODO(DET-10004) remove these double quotes.
		TaskType: model.TaskType(fmt.Sprintf(`"%s"`, model.TaskTypeCommand)),
	}, meta)

	// Tensorboard.
	exp0 := RequireMockExperiment(t, db, user)
	t0, trialTask := RequireMockTrial(t, db, exp0)
	t1, _ := RequireMockTrial(t, db, exp0)
	exp1 := RequireMockExperiment(t, db, user)

	expIDs := []int{exp0.ID, exp1.ID}
	trialIDs := []int{t0.ID, t1.ID}
	tensorboardID := RequireMockTensorboardID(t, db, user.ID, expIDs, trialIDs)

	meta, err = IdentifyTask(ctx, tensorboardID)
	require.NoError(t, err)
	require.Equal(t, TaskMetadata{
		WorkspaceID: 1,
		// TODO(DET-10004) remove these double quotes.
		TaskType:      model.TaskType(fmt.Sprintf(`"%s"`, model.TaskTypeTensorboard)),
		ExperimentIDs: []int32{int32(expIDs[0]), int32(expIDs[1])},
		TrialIDs:      []int32{int32(trialIDs[0]), int32(trialIDs[1])},
	}, meta)

	// Experiment task.
	// This always is not found and is probably a footgun from function name / comment.
	// TODO(DET-10005) remove this footgun.
	_, err = IdentifyTask(ctx, trialTask.TaskID)
	require.ErrorIs(t, err, ErrNotFound)
}
