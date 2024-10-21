package configpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestFindAllowedPriority(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// No priority limit to find.
	_, exists, err := findAllowedPriority(nil, model.ExperimentType)
	require.NoError(t, err)
	require.False(t, exists)

	// Priority limit set.
	globalLimit := 10
	user := db.RequireMockUser(t, pgDB)
	addConstraints(t, user, nil, fmt.Sprintf(`{"priority_limit": %d}`, globalLimit), model.ExperimentType)
	limit, exists, err := findAllowedPriority(nil, model.ExperimentType)
	require.Equal(t, globalLimit, limit)
	require.True(t, exists)
	require.NoError(t, err)

	// NTSC priority set.
	configPriority := 15
	invariantConfig := fmt.Sprintf(`{"resources": {"priority": %d}}`, configPriority)
	addConfig(t, user, nil, invariantConfig, model.NTSCType)
	limit, _, err = findAllowedPriority(nil, model.NTSCType)
	require.ErrorIs(t, err, errPriorityImmutable)
	require.Equal(t, configPriority, limit)

	// Experiment priority set.
	configPriority = 7
	invariantConfig = fmt.Sprintf(`{"resources": {"priority": %d}}`, configPriority)
	addConfig(t, user, nil, invariantConfig, model.ExperimentType)
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
	w := createWorkspaceWithPriorityLimit(t, user, wkspLimit, model.NTSCType)

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
	// No config policies set for experiments.
	ok, err = PriorityUpdateAllowed(w.ID+1, model.ExperimentType, globalLimit-1, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority cannot be updated if invariant_config.resources.priority is set.
	invariantConfig := `{"resources": {"priority": 1}}`
	addConfig(t, user, &w.ID, invariantConfig, model.NTSCType)
	_, err = PriorityUpdateAllowed(w.ID, model.NTSCType, 11, true)
	require.ErrorIs(t, err, errPriorityImmutable)
	ok, err = PriorityUpdateAllowed(w.ID, model.ExperimentType, 10, true)
	require.NoError(t, err)
	require.True(t, ok)

	// Priority can be updated if it's the same value as the one set by invariant config.
	ok, err = PriorityUpdateAllowed(w.ID, model.NTSCType, 1, true)
	require.NoError(t, err)
	require.True(t, ok)

	globalLimit = 15
	globalConfig := fmt.Sprintf(`{"resources": {"priority": %d}}`, globalLimit)
	addConfig(t, user, nil, globalConfig, model.NTSCType)
	ok, err = PriorityUpdateAllowed(w.ID, model.NTSCType, globalLimit, true)
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
		w := createWorkspaceWithPriorityLimit(t, user, wkspPriorityLimit, model.NTSCType)
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
		w := createWorkspaceWithPriorityLimit(t, user, wkspPriorityLimit, model.NTSCType)
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
		w := createWorkspaceWithPriorityLimit(t, user, wkspPriorityLimit, model.ExperimentType)
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
		w := createWorkspaceWithPriorityLimit(t, user, wkspPriorityLimit, model.ExperimentType)
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
	w := createWorkspaceWithPriorityLimit(t, user, wkspLimit, model.NTSCType)
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

func createWorkspaceWithUser(ctx context.Context, t *testing.T, userID model.UserID) model.Workspace {
	w := model.Workspace{Name: uuid.NewString(), UserID: userID}
	_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
	require.NoError(t, err)
	return w
}

func addWorkspacePriorityLimit(ctx context.Context, t *testing.T, user model.User,
	w model.Workspace, limit int, workloadType string,
) {
	constraints := fmt.Sprintf(`{"priority_limit": %d}`, limit)
	input := model.TaskConfigPolicies{
		WorkloadType:  workloadType,
		WorkspaceID:   &w.ID,
		Constraints:   &constraints,
		LastUpdatedBy: user.ID,
	}
	err := SetTaskConfigPolicies(ctx, &input)
	require.NoError(t, err)
}

func createWorkspaceWithPriorityLimit(t *testing.T, user model.User, wkspLimit int,
	workloadType string,
) model.Workspace {
	ctx := context.Background()
	w := createWorkspaceWithUser(ctx, t, user.ID)
	addWorkspacePriorityLimit(ctx, t, user, w, wkspLimit, workloadType)
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

func addConfig(t *testing.T, user model.User, wkspID *int, config string, workloadType string) {
	ctx := context.Background()

	input := model.TaskConfigPolicies{
		WorkloadType:    workloadType,
		WorkspaceID:     wkspID,
		InvariantConfig: &config,
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

var defaultConfig = schemas.WithDefaults(expconf.ExperimentConfigV0{})

func TestMergeWithInvariantExperimentConfigs(t *testing.T) {
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)
	ctx := context.Background()
	w := createWorkspaceWithUser(ctx, t, user.ID)

	wkspDefaultConfig := `{
	"description": "random description workspace",
	"resources": {
		"slots": 5
	}, 
	"preemption_timeout": 2000,
	"bind_mounts": [
		{
		  "host_path": "random/path/wksp",
		  "container_path": "random/container/path/wksp",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	   "log_policies": [
		{
			"pattern": ".*CUDA out of memory.*",
			"actions": [
				{
					"signal": "CUDA OOM"
				}
			]
		},
		{
			"pattern": ".*uncorrectable ECC error encountered.*",
			"actions": [
				{
					"signal": "ECC Error"
				}
			]
		}
	  ]
}`

	var defaultInvariantConfig expconf.ExperimentConfigV0
	err := json.Unmarshal([]byte(*DefaultInvariantConfig()), &defaultInvariantConfig)
	require.NoError(t, err)

	var wkspInvariantConfig expconf.ExperimentConfigV0
	err = json.Unmarshal([]byte(wkspDefaultConfig), &wkspInvariantConfig)
	require.NoError(t, err)

	wkspConfigNoDefaults := wkspInvariantConfig
	// We assign the config name because it is otherwise auto-generated randomly (and this will not
	// be equal to defaultConfig Name).
	wkspInvariantConfig.RawName = defaultConfig.RawName
	wkspInvariantConfig = schemas.WithDefaults(wkspInvariantConfig)

	wkspConfigMergedWithGlobal := schemas.WithDefaults(wkspInvariantConfig)
	err = json.Unmarshal([]byte(*DefaultInvariantConfig()), &wkspConfigMergedWithGlobal)
	require.NoError(t, err)

	defaultInvariantConfig.RawName = defaultConfig.RawName
	defaultInvariantConfig = schemas.WithDefaults(defaultInvariantConfig)

	tests := []struct {
		name           string
		globalTCPs     *model.TaskConfigPolicies
		workspaceTCPs  *model.TaskConfigPolicies
		expectedConfig *expconf.ExperimentConfigV0
	}{
		{
			name: "constraint policy no config",
			workspaceTCPs: &model.TaskConfigPolicies{
				WorkspaceID:   &w.ID,
				WorkloadType:  model.ExperimentType,
				LastUpdatedBy: user.ID,
				Constraints:   DefaultConstraints(),
			},
			expectedConfig: &defaultConfig,
		},
		{
			name: "constraint policy no config global",
			globalTCPs: &model.TaskConfigPolicies{
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			workspaceTCPs: &model.TaskConfigPolicies{
				WorkspaceID:   &w.ID,
				WorkloadType:  model.ExperimentType,
				LastUpdatedBy: user.ID,
				Constraints:   DefaultConstraints(),
			},
			expectedConfig: &defaultInvariantConfig,
		},
		{
			name:           "no config policies for wksp",
			expectedConfig: &defaultConfig,
		},
		{
			name: "no config policies for wksp global",
			globalTCPs: &model.TaskConfigPolicies{
				WorkspaceID:     nil,
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			expectedConfig: &defaultInvariantConfig,
		},
		{
			name: "invariant config policy wksp only",
			workspaceTCPs: &model.TaskConfigPolicies{
				WorkspaceID:     &w.ID,
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				InvariantConfig: &wkspDefaultConfig,
			},
			expectedConfig: &wkspInvariantConfig,
		},
		{
			name: "invariant config policy wksp and global",
			globalTCPs: &model.TaskConfigPolicies{
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			workspaceTCPs: &model.TaskConfigPolicies{
				WorkspaceID:     &w.ID,
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				InvariantConfig: &wkspDefaultConfig,
				Constraints:     DefaultConstraints(),
			},
			expectedConfig: &wkspConfigMergedWithGlobal,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setConfigPolicies(ctx, t, &w.ID, test.workspaceTCPs, test.globalTCPs)

			config := &expconf.ExperimentConfigV0{}
			config.RawName = defaultConfig.RawName
			config.RawReproducibility = defaultConfig.RawReproducibility
			config.RawReproducibility.RawExperimentSeed = test.expectedConfig.
				RawReproducibility.RawExperimentSeed
			config = schemas.WithDefaults(config)

			config, err := MergeWithInvariantExperimentConfigs(context.Background(), w.ID, *config)
			require.NoError(t, err)

			require.Equal(t, *test.expectedConfig, *config)
		})
	}

	t.Run("test empty user config", func(t *testing.T) {
		setConfigPolicies(ctx, t, &w.ID,
			&model.TaskConfigPolicies{
				WorkspaceID:     &w.ID,
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   w.UserID,
				InvariantConfig: &wkspDefaultConfig,
			},
			nil)

		conf, err := MergeWithInvariantExperimentConfigs(context.Background(), w.ID,
			expconf.ExperimentConfigV0{})
		require.NoError(t, err)
		require.Equal(t, wkspConfigNoDefaults, *conf)
	})

	t.Run("test invalid workspace id", func(t *testing.T) {
		conf, err := MergeWithInvariantExperimentConfigs(context.Background(), -1, defaultConfig)
		require.NoError(t, err)
		require.Equal(t, defaultConfig, *conf)
	})

	testMergeSlicesAndMaps(t)
}

func testMergeSlicesAndMaps(t *testing.T) {
	t.Run("test merge slices and maps", func(t *testing.T) {
		require.NoError(t, etc.SetRootPath(db.RootFromDB))
		pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
		defer cleanup()
		db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

		user := db.RequireMockUser(t, pgDB)
		ctx := context.Background()
		w := createWorkspaceWithUser(ctx, t, user.ID)

		userPartialConfig := `{
		"description": "random description workspace",
		"raw_data": { "1" : "data point 1" },
		  "resources": {
		"slots": 5,
		"devices": [
		  {
			"host_path": "random/path",
			"container_path": "random/container/path",
			"mode": "slow"
		  }
		]
	  },
	  "preemption_timeout": 2000,
	  "bind_mounts": [
		{
		  "host_path": "random/path",
		  "container_path": "random/container/path",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	  "environment": {
		"environment_variables": {
		  "cpu": [
			"cpu1:cpuval1"
		  ],
		   "cuda": [
			"cuda1:cudaval"
		  ],
		   "rocm": [
			"rocm1:rocmval1"
		  ],
		  "proxy_ports": [
			{
			  "proxy_port": 91950,
			  "proxy_tcp": true
			}
		  ]
		}
	  },
	  "log_policies": [
		{
		  "pattern": "nonrepeat"
		}
	  ]
	}
	`

		adminPartialConfig := `{
	  "raw_data": { "2" : "data point 2" },
	  "resources": {
		"devices": [
		  {
			"host_path": "random/path/wksp",
			"container_path": "random/container/path/wksp",
			"mode": "fast"
		  } ]
		}, 
	  "bind_mounts": [
		{
		  "host_path": "random/path/wksp",
		  "container_path": "random/container/path/wksp",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	  "environment": {
		"environment_variables": {
		  "cpu": [
			"cpu2:cpuval2"
		  ],
		  "proxy_ports": [
			{
			  "proxy_port": 9195,
			  "proxy_tcp": true
			}
		  ]
		}
	  },
	  "log_policies": [
		{
		  "pattern": "repeat"
		}
	  ]
	}`

		mergedWkspConfig := `
	{
	  "raw_data": { 
			  "1" : "data point 1",
			 "2" : "data point 2"
		},
	  "description": "random description workspace",
	  "resources": {
		"slots": 5,
		"devices": [
		 {
			"host_path": "random/path/wksp",
			"container_path": "random/container/path/wksp",
			"mode": "fast"
		  },
		  {
			"host_path": "random/path",
			"container_path": "random/container/path",
			"mode": "slow"
		  }
		]
	  },
	  "preemption_timeout": 2000,
	  "bind_mounts": [
		{
		  "host_path": "random/path/wksp",
		  "container_path": "random/container/path/wksp",
		  "read_only": true,
		  "propagation": "cluster-wide"
		},
		 {
		  "host_path": "random/path",
		  "container_path": "random/container/path",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	  "environment": {
		"environment_variables": {
		  "cpu": [
			"cpu1:cpuval1", "cpu2:cpuval2"
		  ],
		   "cuda": [
			"cuda1:cudaval"
		  ],
		   "rocm": [
			"rocm1:rocmval1"
		  ],
		  "proxy_ports": [
			{
			  "proxy_port": 9195,
			  "proxy_tcp": true
			},
			{
			  "proxy_port": 91950,
			  "proxy_tcp": true
			}
		  ]
		}
	  },
	  "log_policies": [
		{
		  "pattern": "nonrepeat"
		},
		{
		  "pattern": "repeat"    
		}
	  ]
	}
	`

		globalPartialConfig := `{
	  "raw_data": { "3" : "data point 3" },
	  "resources": {
		"devices": [
		  {
			"host_path": "random/path/global",
			"container_path": "random/container/path/global",
			"mode": "fast"
		  } ]
		}, 
	  "bind_mounts": [
		{
		  "host_path": "random/path/global",
		  "container_path": "random/container/path/global",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	  "environment": {
		"environment_variables": {
		  "cpu": [
			"cpu3:cpuval3"
		  ],
		  "proxy_ports": [
			{
			  "proxy_port": 10195,
			  "proxy_tcp": true
			}
		  ]
		}
	  },
	  "log_policies": [
		{
		  "pattern": "gloablrepeat"
		}
	  ]
	}`

		mergedGlobalConfig := `
	{
	  "raw_data": { 
			  "1" : "data point 1",
			 "2" : "data point 2",
			 "3" : "data point 3"
		},
	  "description": "random description workspace",
	  "resources": {
		"slots": 5,
		"devices": [
		{
			"host_path": "random/path/global",
			"container_path": "random/container/path/global",
			"mode": "fast"
		  },
		 {
			"host_path": "random/path/wksp",
			"container_path": "random/container/path/wksp",
			"mode": "fast"
		  },
		  {
			"host_path": "random/path",
			"container_path": "random/container/path",
			"mode": "slow"
		  }
		]
	  },
	  "preemption_timeout": 2000,
	  "bind_mounts": [
	   {
		  "host_path": "random/path/global",
		  "container_path": "random/container/path/global",
		  "read_only": true,
		  "propagation": "cluster-wide"
		},
		{
		  "host_path": "random/path/wksp",
		  "container_path": "random/container/path/wksp",
		  "read_only": true,
		  "propagation": "cluster-wide"
		},
		 {
		  "host_path": "random/path",
		  "container_path": "random/container/path",
		  "read_only": true,
		  "propagation": "cluster-wide"
		}
	  ],
	  "environment": {
		"environment_variables": {
		  "cpu": [
			"cpu1:cpuval1", "cpu2:cpuval2", "cpu3:cpuval3"
		  ],
		   "cuda": [
			"cuda1:cudaval"
		  ],
		   "rocm": [
			"rocm1:rocmval1"
		  ],
		  "proxy_ports": [
			{
			  "proxy_port": 9195,
			  "proxy_tcp": true
			},
			{
			  "proxy_port": 91950,
			  "proxy_tcp": true
			},
			{
			  "proxy_port": 10195,
			  "proxy_tcp": true
			}
		  ]
		}
	  },
	  "log_policies": [
		{
		  "pattern": "nonrepeat"
		},
		{
		  "pattern": "repeat"    
		},
		{
		  "pattern": "gloablrepeat"
		}
	  ]
	}
	`
		var adminPartialInvariantConfig expconf.ExperimentConfigV0
		err := json.Unmarshal([]byte(adminPartialConfig), &adminPartialInvariantConfig)
		require.NoError(t, err)

		adminPartialInvariantConfig.RawName = defaultConfig.RawName
		adminPartialInvariantConfig = schemas.WithDefaults(adminPartialInvariantConfig)

		var userPartialInvariantConfig expconf.ExperimentConfigV0
		err = json.Unmarshal([]byte(userPartialConfig), &userPartialInvariantConfig)
		require.NoError(t, err)

		userPartialInvariantConfig.RawName = defaultConfig.RawName
		userPartialInvariantConfig = schemas.WithDefaults(userPartialInvariantConfig)

		var mergedWkspInvariantConfig expconf.ExperimentConfigV0
		err = json.Unmarshal([]byte(mergedWkspConfig), &mergedWkspInvariantConfig)
		require.NoError(t, err)

		// Test merging user-submitted config with workspace config policies only.
		setConfigPolicies(ctx, t, &w.ID, &model.TaskConfigPolicies{
			WorkspaceID:     &w.ID,
			WorkloadType:    model.ExperimentType,
			LastUpdatedBy:   w.UserID,
			InvariantConfig: &adminPartialConfig,
		}, nil)

		mergedWkspInvariantConfig.RawName = adminPartialInvariantConfig.RawName
		mergedWkspInvariantConfig = schemas.WithDefaults(mergedWkspInvariantConfig)
		res, err := MergeWithInvariantExperimentConfigs(context.Background(), w.ID,
			userPartialInvariantConfig)

		require.NoError(t, err)
		require.ElementsMatch(t, mergedWkspInvariantConfig.Data(), res.Data())
		require.Equal(t, mergedWkspInvariantConfig.Description(), res.Description())
		require.Equal(t, mergedWkspInvariantConfig.Resources().Slots(), res.Resources().Slots())
		require.ElementsMatch(t, mergedWkspInvariantConfig.Resources().Devices(),
			res.Resources().Devices())
		require.Equal(t, mergedWkspInvariantConfig.PreemptionTimeout(),
			res.PreemptionTimeout())
		require.ElementsMatch(t, mergedWkspInvariantConfig.BindMounts(), res.BindMounts())
		require.ElementsMatch(t, mergedWkspInvariantConfig.Environment().EnvironmentVariables().CPU(),
			res.Environment().EnvironmentVariables().CPU())
		require.ElementsMatch(t, mergedWkspInvariantConfig.Environment().EnvironmentVariables().CUDA(),
			res.Environment().EnvironmentVariables().CUDA())
		require.ElementsMatch(t, mergedWkspInvariantConfig.Environment().EnvironmentVariables().ROCM(),
			res.Environment().EnvironmentVariables().ROCM())
		require.ElementsMatch(t, mergedWkspInvariantConfig.BindMounts(), res.BindMounts())
		require.ElementsMatch(t, mergedWkspInvariantConfig.Environment().ProxyPorts(),
			res.Environment().ProxyPorts())
		require.ElementsMatch(t, mergedWkspInvariantConfig.LogPolicies(), res.LogPolicies())

		// Merge user-submitted config with workspace and global config policies.
		setConfigPolicies(ctx, t, &w.ID, &model.TaskConfigPolicies{
			WorkspaceID:     &w.ID,
			WorkloadType:    model.ExperimentType,
			LastUpdatedBy:   w.UserID,
			InvariantConfig: &adminPartialConfig,
		}, &model.TaskConfigPolicies{
			WorkspaceID:     nil,
			WorkloadType:    model.ExperimentType,
			LastUpdatedBy:   w.UserID,
			InvariantConfig: &globalPartialConfig,
		})

		var globalInvariantConfig expconf.ExperimentConfigV0
		err = json.Unmarshal([]byte(globalPartialConfig), &globalInvariantConfig)
		require.NoError(t, err)

		var mergedGlobalInvariantConfig expconf.ExperimentConfigV0
		err = json.Unmarshal([]byte(mergedGlobalConfig), &mergedGlobalInvariantConfig)
		require.NoError(t, err)

		mergedGlobalInvariantConfig.RawName = defaultConfig.RawName
		mergedGlobalInvariantConfig = schemas.WithDefaults(mergedGlobalInvariantConfig)
		res, err = MergeWithInvariantExperimentConfigs(context.Background(), w.ID,
			userPartialInvariantConfig)

		require.NoError(t, err)
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Data(), res.Data())
		require.Equal(t, mergedGlobalInvariantConfig.Description(), res.Description())
		require.Equal(t, mergedGlobalInvariantConfig.Resources().Slots(), res.Resources().Slots())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Resources().Devices(),
			res.Resources().Devices())
		require.Equal(t, mergedGlobalInvariantConfig.PreemptionTimeout(),
			res.PreemptionTimeout())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.BindMounts(), res.BindMounts())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Environment().EnvironmentVariables().CPU(),
			res.Environment().EnvironmentVariables().CPU())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Environment().EnvironmentVariables().CUDA(),
			res.Environment().EnvironmentVariables().CUDA())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Environment().EnvironmentVariables().ROCM(),
			res.Environment().EnvironmentVariables().ROCM())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.BindMounts(), res.BindMounts())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.Environment().ProxyPorts(),
			res.Environment().ProxyPorts())
		require.ElementsMatch(t, mergedGlobalInvariantConfig.LogPolicies(), res.LogPolicies())
	})
}

func setConfigPolicies(ctx context.Context, t *testing.T, wID *int, workspaceTCPs,
	globalTCPs *model.TaskConfigPolicies,
) {
	if workspaceTCPs == nil {
		err := DeleteConfigPolicies(ctx, wID, model.ExperimentType)
		require.NoError(t, err)
	} else {
		err := SetTaskConfigPolicies(ctx, workspaceTCPs)
		require.NoError(t, err)
	}

	if globalTCPs == nil {
		err := DeleteConfigPolicies(ctx, nil, model.ExperimentType)
		require.NoError(t, err)
	} else {
		err := SetTaskConfigPolicies(ctx, globalTCPs)
		require.NoError(t, err)
	}
}
