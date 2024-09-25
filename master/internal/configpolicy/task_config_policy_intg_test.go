package configpolicy

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
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
	w := addWorkspacePriorityLimit(t, user, wkspLimit)

	// Priority is outside workspace limit.
	smallerValueIsHigherPriority := true
	ok, err = PriorityAllowed(w.ID, model.NTSCType, wkspLimit-1, smallerValueIsHigherPriority)
	require.NoError(t, err)
	require.False(t, ok)

	globalLimit := 42
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit))

	// Priority is within global limit.
	ok, err = PriorityAllowed(w.ID, model.NTSCType, wkspLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority is outside global limit.
	ok, err = PriorityAllowed(w.ID+1, model.NTSCType, globalLimit-1, true)
	require.NoError(t, err)
	require.False(t, ok)
}

func TestValidateNTSCConstraints(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	wkspPriorityLimit := 7
	user := db.RequireMockUser(t, pgDB)

	t.Run("no constraints set - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), 1, config, &resourceManager)
		require.NoError(t, err)
		require.True(t, ok)
	})

	t.Run("running in wksp with constraints - not ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit)
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.False(t, ok)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requested priority")
	})

	t.Run("running in wksp without constraints - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.True(t, ok)
		require.NoError(t, err)
	})

	t.Run("exceeds max slots - not ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)
		addConstraints(t, user, &w.ID, *constraints)

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.False(t, ok)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requested resources.max_slots")
	})

	t.Run("rm priority not supported - ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit)
		rm1 := mocks.ResourceManager{}
		rm1.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil).Once()

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), w.ID, config, &rm1)
		require.False(t, ok)
		require.Error(t, err)
		require.Contains(t, err.Error(), "requested priority")

		// Validate constraints again. This time, the RM does not support priority.
		rmNoPriority := mocks.ResourceManager{}
		rmNoPriority.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, fmt.Errorf("not supported")).Once()
		ok, err = ValidateNTSCConstraints(context.Background(), w.ID, config, &rmNoPriority)
		require.True(t, ok)
		require.NoError(t, err)
	})

	t.Run("no config set - ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		addConstraints(t, user, nil, *constraints) // add global constraints

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultConfig()
		ok, err := ValidateNTSCConstraints(context.Background(), 1, config, &resourceManager)
		require.False(t, ok)
		require.Error(t, err)

		emptyConfig := model.CommandConfig{}
		ok, err = ValidateNTSCConstraints(context.Background(), 1, emptyConfig, &resourceManager)
		require.True(t, ok)
		require.NoError(t, err)
	})
}

func TestGetMergedConstraints(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// When no constraints present, all values are nil.
	constraints, err := GetMergedConstraints(context.Background(), 0, model.NTSCType)
	require.NoError(t, err)
	require.Nil(t, constraints.PriorityLimit)
	require.Nil(t, constraints.ResourceConstraints)

	// Workspace priority limit set.
	wkspLimit := 42
	user := db.RequireMockUser(t, pgDB)
	w := addWorkspacePriorityLimit(t, user, wkspLimit)
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Nil(t, constraints.ResourceConstraints)
	require.Equal(t, wkspLimit, *constraints.PriorityLimit)

	// Global limit overrides workspace limit.
	globalLimit := 25
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit))
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Nil(t, constraints.ResourceConstraints)
	require.Equal(t, globalLimit, *constraints.PriorityLimit)

	// Workspace max slots set.
	addConstraints(t, user, &w.ID, *DefaultConstraints())
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Equal(t, 8, *constraints.ResourceConstraints.MaxSlots) // defined in DefaultConstraintsStr
	require.Equal(t, globalLimit, *constraints.PriorityLimit)      // global constraint overrides workspace value
}

func addWorkspacePriorityLimit(t *testing.T, user model.User, limit int) model.Workspace {
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

func addConstraints(t *testing.T, user model.User, wkspID *int, constraints string) {
	ctx := context.Background()

	input := model.TaskConfigPolicies{
		WorkloadType:  model.NTSCType,
		WorkspaceID:   wkspID,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err := SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)
}

func defaultConfig() model.CommandConfig {
	config := model.DefaultConfig(nil)

	configPriority := 50
	configMaxSlots := 12
	config.Resources.Priority = &configPriority
	config.Resources.MaxSlots = &configMaxSlots

	return config
}
