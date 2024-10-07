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
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestFindAllowedPriority(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// No priority limit to find.
	_, found, err := findAllowedPriority(nil, model.ExperimentType)
	require.NoError(t, err)
	require.False(t, found)

	// Priority limit set.
	globalLimit := 10
	user := db.RequireMockUser(t, pgDB)
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit), model.ExperimentType)
	limit, found, err := findAllowedPriority(nil, model.ExperimentType)
	require.Equal(t, globalLimit, limit)
	require.True(t, found)
	require.NoError(t, err)

	// NTSC priority set.
	configPriority := 15
	invariantConfig := fmt.Sprintf(`{"resources": {"priority": %d}}`, configPriority)
	addConfigs(t, user, nil, invariantConfig, model.NTSCType)
	limit, _, err = findAllowedPriority(nil, model.NTSCType)
	require.ErrorIs(t, err, errPriorityImmutable)
	require.Equal(t, configPriority, limit)

	// Experiment priority set.
	configPriority = 7
	invariantConfig = fmt.Sprintf(`{"resources": {"priority": %d}}`, configPriority)
	addConfigs(t, user, nil, invariantConfig, model.ExperimentType)
	limit, _, err = findAllowedPriority(nil, model.ExperimentType)
	require.ErrorIs(t, err, errPriorityImmutable)
	require.Equal(t, configPriority, limit)
}

func TestPriorityUpdateAllowed(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// When no constraints are present, any priority is allowed.
	ok, err := PriorityUpdateAllowed(1, model.NTSCType, 0, true)
	require.NoError(t, err)
	require.True(t, ok)

	wkspLimit := 50
	user := db.RequireMockUser(t, pgDB)
	w := addWorkspacePriorityLimit(t, user, wkspLimit, model.NTSCType)

	// Priority is outside workspace limit.
	smallerValueIsHigherPriority := true
	ok, err = PriorityUpdateAllowed(w.ID, model.NTSCType, wkspLimit-1, smallerValueIsHigherPriority)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = PriorityUpdateAllowed(w.ID, model.ExperimentType, wkspLimit-1, smallerValueIsHigherPriority)
	require.NoError(t, err)
	require.True(t, ok)

	globalLimit := 42
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit), model.NTSCType)

	// Priority is within global limit.
	ok, err = PriorityUpdateAllowed(w.ID, model.NTSCType, wkspLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = PriorityUpdateAllowed(w.ID, model.ExperimentType, wkspLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority is outside global limit.
	ok, err = PriorityUpdateAllowed(w.ID+1, model.NTSCType, globalLimit-1, true)
	require.NoError(t, err)
	require.False(t, ok)
	ok, err = PriorityUpdateAllowed(w.ID+1, model.ExperimentType, globalLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority cannot be updated if invariant_config.resources.priority is set.
	invariantConfig := `{"resources": {"priority": 7}}`
	addConfigs(t, user, &w.ID, invariantConfig, model.NTSCType)
	_, err = PriorityUpdateAllowed(w.ID, model.NTSCType, globalLimit, true)
	require.Error(t, errPriorityImmutable)
	ok, err = PriorityUpdateAllowed(w.ID, model.ExperimentType, globalLimit, true)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCheckNTSCConstraints(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	wkspPriorityLimit := 7
	user := db.RequireMockUser(t, pgDB)

	t.Run("no constraints set - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		config := defaultNTSCConfig()
		err := CheckNTSCConstraints(context.Background(), 1, config, &resourceManager)
		require.NoError(t, err)
	})

	t.Run("running in wksp with constraints - not ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit, model.NTSCType)
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		config := defaultNTSCConfig()
		err := CheckNTSCConstraints(ctx, w.ID, config, &resourceManager)
		require.Error(t, err)
		require.ErrorIs(t, err, errPriorityConstraintFailure)
	})

	t.Run("running in wksp without constraints - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)

		config := defaultNTSCConfig()
		err = CheckNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.NoError(t, err)
	})

	t.Run("exceeds max slots - not ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)
		addConstraints(t, user, &w.ID, *constraints, model.NTSCType)

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultNTSCConfig()
		err = CheckNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.Error(t, err)
		require.ErrorIs(t, err, errResourceConstraintFailure)
	})

	t.Run("exceeds slots - not ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)
		addConstraints(t, user, &w.ID, *constraints, model.NTSCType)

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultNTSCConfig()
		config.Resources.Slots = *config.Resources.MaxSlots
		config.Resources.MaxSlots = nil // ensure only slots is set
		err = CheckNTSCConstraints(context.Background(), w.ID, config, &resourceManager)
		require.Error(t, err)
		require.ErrorIs(t, err, errResourceConstraintFailure)
	})

	t.Run("rm priority not supported - ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit, model.NTSCType)
		rm1 := mocks.ResourceManager{}
		rm1.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil).Once()

		config := defaultNTSCConfig()
		err := CheckNTSCConstraints(context.Background(), w.ID, config, &rm1)
		require.Error(t, err)
		require.ErrorIs(t, err, errPriorityConstraintFailure)

		// Validate constraints again. This time, the RM does not support priority.
		rmNoPriority := mocks.ResourceManager{}
		rmNoPriority.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, fmt.Errorf("not supported")).Once()
		err = CheckNTSCConstraints(context.Background(), w.ID, config, &rmNoPriority)
		require.NoError(t, err)
	})

	t.Run("no config set - ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		addConstraints(t, user, nil, *constraints, model.NTSCType) // add global constraints

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultNTSCConfig()
		err := CheckNTSCConstraints(context.Background(), 1, config, &resourceManager)
		require.Error(t, err)

		emptyConfig := model.CommandConfig{}
		err = CheckNTSCConstraints(context.Background(), 1, emptyConfig, &resourceManager)
		require.NoError(t, err)
	})
}

func TestCheckExperimentConstraints(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	wkspPriorityLimit := 7
	user := db.RequireMockUser(t, pgDB)

	t.Run("no constraints set - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		config := defaultExperimentConfig()
		err := CheckExperimentConstraints(context.Background(), 1, config, &resourceManager)
		require.NoError(t, err)
	})

	t.Run("running in wksp with constraints - not ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit, model.ExperimentType)
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		config := defaultExperimentConfig()
		err := CheckExperimentConstraints(ctx, w.ID, config, &resourceManager)
		require.Error(t, err)
		require.ErrorIs(t, err, errPriorityConstraintFailure)
	})

	t.Run("running in wksp without constraints - ok", func(t *testing.T) {
		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil)
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)

		config := defaultExperimentConfig()
		err = CheckExperimentConstraints(context.Background(), w.ID, config, &resourceManager)
		require.NoError(t, err)
	})

	t.Run("exceeds max slots - not ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(context.Background())
		require.NoError(t, err)
		addConstraints(t, user, &w.ID, *constraints, model.ExperimentType)

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultExperimentConfig()
		err = CheckExperimentConstraints(context.Background(), w.ID, config, &resourceManager)
		require.Error(t, err)
		require.ErrorIs(t, err, errResourceConstraintFailure)
	})

	t.Run("rm priority not supported - ok", func(t *testing.T) {
		w := addWorkspacePriorityLimit(t, user, wkspPriorityLimit, model.ExperimentType)
		rm1 := mocks.ResourceManager{}
		rm1.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, nil).Once()

		config := defaultExperimentConfig()
		err := CheckExperimentConstraints(context.Background(), w.ID, config, &rm1)
		require.Error(t, err)
		require.ErrorIs(t, err, errPriorityConstraintFailure)

		// Validate constraints again. This time, the RM does not support priority.
		rmNoPriority := mocks.ResourceManager{}
		rmNoPriority.On("SmallerValueIsHigherPriority", mock.Anything).Return(false, fmt.Errorf("not supported")).Once()
		err = CheckExperimentConstraints(context.Background(), w.ID, config, &rmNoPriority)
		require.NoError(t, err)
	})

	t.Run("no config set - ok", func(t *testing.T) {
		constraints := DefaultConstraints()
		addConstraints(t, user, nil, *constraints, model.ExperimentType) // add global constraints

		resourceManager := mocks.ResourceManager{}
		resourceManager.On("SmallerValueIsHigherPriority", mock.Anything).Return(true, nil)

		config := defaultExperimentConfig()
		err := CheckExperimentConstraints(context.Background(), 1, config, &resourceManager)
		require.Error(t, err)

		emptyResources := expconf.ResourcesConfigV0{}
		emptyConfig := expconf.ExperimentConfigV0{}
		emptyConfig.SetResources(emptyResources)
		err = CheckExperimentConstraints(context.Background(), 1, emptyConfig, &resourceManager)
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
	w := addWorkspacePriorityLimit(t, user, wkspLimit, model.NTSCType)
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Nil(t, constraints.ResourceConstraints)
	require.Equal(t, wkspLimit, *constraints.PriorityLimit)

	// Global limit overrides workspace limit.
	globalLimit := 25
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit), model.NTSCType)
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Nil(t, constraints.ResourceConstraints)
	require.Equal(t, globalLimit, *constraints.PriorityLimit)

	// Workspace max slots set.
	addConstraints(t, user, &w.ID, *DefaultConstraints(), model.NTSCType)
	constraints, err = GetMergedConstraints(context.Background(), w.ID, model.NTSCType)
	require.NoError(t, err)
	require.Equal(t, 8, *constraints.ResourceConstraints.MaxSlots) // defined in DefaultConstraintsStr
	require.Equal(t, globalLimit, *constraints.PriorityLimit)      // global constraint overrides workspace value
}

func addWorkspacePriorityLimit(t *testing.T, user model.User, limit int, workloadType string) model.Workspace {
	ctx := context.Background()

	// add a workspace to use
	w := model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
	require.NoError(t, err)

	constraints := fmt.Sprintf(`{"priority_limit": %d}`, limit)
	input := model.TaskConfigPolicies{
		WorkloadType:  workloadType,
		WorkspaceID:   &w.ID,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err = SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)

	return w
}

func addConstraints(t *testing.T, user model.User, wkspID *int, constraints string, workloadType string) {
	ctx := context.Background()

	input := model.TaskConfigPolicies{
		WorkloadType:  workloadType,
		WorkspaceID:   wkspID,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err := SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)
}

func addConfigs(t *testing.T, user model.User, wkspID *int, configs string, workloadType string) {
	ctx := context.Background()

	input := model.TaskConfigPolicies{
		WorkloadType:    workloadType,
		WorkspaceID:     wkspID,
		InvariantConfig: &configs,
		LastUpdatedBy:   user.ID,
	}
	err := SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)
}

func defaultNTSCConfig() model.CommandConfig {
	config := model.DefaultConfig(nil)

	configPriority := 50
	configMaxSlots := 12
	config.Resources.Priority = &configPriority
	config.Resources.MaxSlots = &configMaxSlots

	return config
}

func defaultExperimentConfig() expconf.ExperimentConfigV0 {
	config := expconf.ExperimentConfigV0{}

	configPriority := 50
	configMaxSlots := 12
	resources := expconf.ResourcesConfigV0{
		RawMaxSlots: &configMaxSlots,
		RawPriority: &configPriority,
	}
	config.SetResources(resources)

	return config
}
