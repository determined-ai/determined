package configpolicy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestGetPriorityLimitPrecedence(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// when no constraints are present, do not find constraints
	_, found, err := GetPriorityLimitPrecedence(ctx, 10, model.NTSCType)
	require.NoError(t, err)
	require.False(t, found)

	// set priority limit for a workspace
	wkspLimit := 50
	w := addWorkspacePriorityLimit(t, pgDB, wkspLimit)

	// workspace but none for global - get workspace limit
	limit, found, err := GetPriorityLimitPrecedence(ctx, w.ID, model.NTSCType)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, wkspLimit, limit)

	// set global priority limit
	globalLimit := 42
	addGlobalPriorityLimit(t, pgDB, globalLimit)

	// global and workspace - get global limit
	limit, found, err = GetPriorityLimitPrecedence(ctx, w.ID, model.NTSCType)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, globalLimit, limit)

	// global but none for workspace - get global limit
	limit, found, err = GetPriorityLimitPrecedence(ctx, w.ID+1, model.NTSCType)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, globalLimit, limit)
}

func addWorkspacePriorityLimit(t *testing.T, pgDB *db.PgDB, limit int) model.Workspace {
	ctx := context.Background()
	user := db.RequireMockUser(t, pgDB)

	// add a workspace to use
	w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
	require.NoError(t, err)

	input := model.NTSCTaskConfigPolicies{
		WorkloadType: model.NTSCType,
		WorkspaceID:  &w.ID,
		Constraints: model.Constraints{
			PriorityLimit: &limit,
		},
		LastUpdatedBy: user.ID,
	}
	err = SetNTSCConfigPolicies(ctx, &input)
	require.NoError(t, err)

	return w
}

func addGlobalPriorityLimit(t *testing.T, pgDB *db.PgDB, limit int) {
	ctx := context.Background()
	user := db.RequireMockUser(t, pgDB)

	input := model.NTSCTaskConfigPolicies{
		WorkloadType: model.NTSCType,
		WorkspaceID:  nil,
		Constraints: model.Constraints{
			PriorityLimit: &limit,
		},
		LastUpdatedBy: user.ID,
	}
	err := SetNTSCConfigPolicies(ctx, &input)
	require.NoError(t, err)
}
