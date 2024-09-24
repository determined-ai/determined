package configpolicy

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestPriorityAllowed(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// When no constraints are present, any priority is allowed.
	ok, err := PriorityAllowed(1, model.NTSCType, 0, true)
	require.NoError(t, err)
	require.True(t, ok)

	wkspLimit := 50
	user := db.RequireMockUser(t, pgDB)
	w := addWorkspacePriorityLimit(t, pgDB, user, wkspLimit)

	// Priority is outside workspace limit.
	smallerValueIsHigherPriority := true
	ok, err = PriorityAllowed(w.ID, model.NTSCType, wkspLimit-1, smallerValueIsHigherPriority)
	require.NoError(t, err)
	require.False(t, ok)

	globalLimit := 42
	addGlobalPriorityLimit(t, pgDB, user, globalLimit)

	// Priority is within global limit.
	ok, err = PriorityAllowed(w.ID, model.NTSCType, wkspLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority is outside global limit.
	ok, err = PriorityAllowed(w.ID+1, model.NTSCType, globalLimit-1, true)
	require.NoError(t, err)
	require.False(t, ok)
}

func addWorkspacePriorityLimit(t *testing.T, pgDB *db.PgDB, user model.User, limit int) model.Workspace {
	ctx := context.Background()

	// add a workspace to use
	w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
	require.NoError(t, err)

	constraints := fmt.Sprintf(`{"priority_limit": %d}`, limit)
	input := model.TaskConfigPolicies{
		WorkloadType:  model.NTSCType,
		WorkspaceID:   &w.ID,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err = SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)

	return w
}

func addGlobalPriorityLimit(t *testing.T, pgDB *db.PgDB, user model.User, limit int) {
	ctx := context.Background()

	constraints := fmt.Sprintf(`{"priority_limit": %d}`, limit)
	input := model.TaskConfigPolicies{
		WorkloadType:  model.NTSCType,
		WorkspaceID:   nil,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err := SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)
}
