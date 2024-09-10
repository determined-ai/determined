//go:build integration
// +build integration

package db

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/stretchr/testify/require"
)

func TestSetExperimentConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		err := CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name    string
		expTCPs *model.ExperimentTaskConfigPolicies
		global  bool
		err     *string
	}{
		{
			"invalid workspace id",
			&model.ExperimentTaskConfigPolicies{
				WorkspaceID:     -1,
				LastUpdatedBy:   user.ID,
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultExperimentConfig(),
				Constraints:     model.Constraints{},
			},
			false,
			ptrs.Ptr("violates foreign key constraint"),
		},
		{
			"invalid user id",
			&model.ExperimentTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   -1,
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultExperimentConfig(),
				Constraints:     model.Constraints{},
			},
			false,
			ptrs.Ptr("violates foreign key constraint"),
		},
		{
			"valid config no constraint",
			&model.ExperimentTaskConfigPolicies{
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultExperimentConfig(),
				Constraints:     model.Constraints{},
			},
			false,
			nil,
		},
		// {
		// 	"valid constraint no config",
		// 	&model.ExperimentTaskConfigPolicies{
		// 		LastUpdatedBy:   user.ID,
		// 		LastUpdatedTime: time.Now().UTC(),
		// 		WorkloadType:    model.ExperimentType,
		// 		InvariantConfig: expconf.ExperimentConfig{},
		// 		Constraints:     DefaultConstraints(),
		// 	},
		// 	false,
		// 	nil,
		// },
		// {
		// 	"valid constraint valid config",
		// 	&model.ExperimentTaskConfigPolicies{
		// 		WorkspaceID:     0,
		// 		LastUpdatedBy:   user.ID,
		// 		LastUpdatedTime: time.Now().UTC(),
		// 		WorkloadType:    model.ExperimentType,
		// 		InvariantConfig: expconf.ExperimentConfig{},
		// 		Constraints:     DefaultConstraints(),
		// 	},
		// 	false,
		// 	nil,
		// },
		// {
		// 	"valid config no constraint",
		// 	&model.ExperimentTaskConfigPolicies{
		// 		WorkspaceID:     0,
		// 		LastUpdatedBy:   user.ID,
		// 		LastUpdatedTime: time.Now().UTC(),
		// 		WorkloadType:    model.ExperimentType,
		// 		InvariantConfig: DefaultExperimentConfig(),
		// 		Constraints:     model.Constraints{},
		// 	},
		// 	false,
		// 	nil,
		// },
		// {
		// 	"global valid constraint no config",
		// 	&model.ExperimentTaskConfigPolicies{
		// 		WorkspaceID:     0,
		// 		LastUpdatedBy:   user.ID,
		// 		LastUpdatedTime: time.Now().UTC(),
		// 		WorkloadType:    model.ExperimentType,
		// 		InvariantConfig: expconf.ExperimentConfig{},
		// 		Constraints:     DefaultConstraints(),
		// 	},
		// 	true,
		// 	nil,
		// },
		// {
		// 	"global valid constraint valid config",
		// 	&model.ExperimentTaskConfigPolicies{
		// 		WorkspaceID:     0,
		// 		LastUpdatedBy:   user.ID,
		// 		LastUpdatedTime: time.Now().UTC(),
		// 		WorkloadType:    model.ExperimentType,
		// 		InvariantConfig: DefaultExperimentConfig(),
		// 		Constraints:     DefaultConstraints(),
		// 	},
		// 	true,
		// 	nil,
		// },
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w model.Workspace
			if test.global {
				w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
				_, err := Bun().NewInsert().Model(w).Exec(ctx)
				require.NoError(t, err)
				workspaceIDs = append(workspaceIDs, int32(w.ID))
				test.expTCPs.WorkspaceID = w.ID
			}

			// Test add experiment task config policies.
			err := SetExperimentConfigPolicies(ctx, test.expTCPs)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, *test.err)
				return
			}

			// Test get experiment task config policies.
			expTCPs, err := GetExperimentConfigPolicies(ctx, test.expTCPs.WorkspaceID)
			require.NoError(t, err)
			require.Equal(t, test.expTCPs, expTCPs)
		})
	}
}

func TestSetNTSCConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		err := CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name     string
		ntscTCPs *model.NTSCTaskConfigPolicies
		global   bool
		err      *string
	}{
		{
			"invalid workspace id",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     -1,
				LastUpdatedBy:   user.ID,
				WorkloadType:    model.NTSCType,
				InvariantConfig: DefaultCommandConfig(),
				Constraints:     DefaultConstraints(),
			},
			false,
			ptrs.Ptr("violates foreign key constraint"),
		},
		{
			"invalid user id",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   -1,
				WorkloadType:    model.NTSCType,
				InvariantConfig: DefaultCommandConfig(),
				Constraints:     DefaultConstraints(),
			},
			false,
			ptrs.Ptr("violates foreign key constraint"),
		},
		{
			"valid config no constraint",
			&model.NTSCTaskConfigPolicies{
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultCommandConfig(),
				Constraints:     model.Constraints{},
			},
			false,
			nil,
		},
		{
			"valid constraint no config",
			&model.NTSCTaskConfigPolicies{
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: model.CommandConfig{},
				Constraints:     DefaultConstraints(),
			},
			false,
			nil,
		},
		{
			"valid constraint valid config",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: model.CommandConfig{},
				Constraints:     DefaultConstraints(),
			},
			false,
			nil,
		},
		{
			"valid config no constraint",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultCommandConfig(),
				Constraints:     model.Constraints{},
			},
			false,
			nil,
		},
		{
			"global valid constraint no config",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: model.CommandConfig{},
				Constraints:     DefaultConstraints(),
			},
			true,
			nil,
		},
		{
			"global valid constraint valid config",
			&model.NTSCTaskConfigPolicies{
				WorkspaceID:     0,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC(),
				WorkloadType:    model.ExperimentType,
				InvariantConfig: DefaultCommandConfig(),
				Constraints:     DefaultConstraints(),
			},
			true,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w model.Workspace
			if test.global {
				w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
				_, err := Bun().NewInsert().Model(w).Exec(ctx)
				require.NoError(t, err)
				workspaceIDs = append(workspaceIDs, int32(w.ID))
				test.ntscTCPs.WorkspaceID = w.ID
			}

			// Test add experiment task config policies.
			err := SetNTSCConfigPolicies(ctx, test.ntscTCPs)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, *test.err)
				return
			}

			// Test get experiment task config policies.
			ntscTCPs, err := GetExperimentConfigPolicies(ctx, test.ntscTCPs.WorkspaceID)
			require.NoError(t, err)
			require.Equal(t, test.ntscTCPs, ntscTCPs)
		})
	}
}

// Test the enforcement of the primary key on the task_config_polciies table.
func TestTaskConfigPoliciesUnique(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)

	// Global scope.
	w, expTCP, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, true, true, true)
	expTCP.Constraints = model.Constraints{}
	expInvariantConfig, err := json.Marshal(expTCP.InvariantConfig)
	require.NoError(t, err)

	count, err := Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	_, err = Bun().NewInsert().Model(expTCP).
		Where("workspace_id = ?", nil).
		Where("workspace_type = ?", model.ExperimentType).
		Value("invariant_config", "?", string(expInvariantConfig)).
		Exec(ctx)
	require.ErrorContains(t, err, "duplicate key value violates unique constraint")

	// Workspace-level.
	w, expTCP, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	expTCP.Constraints = model.Constraints{}
	expInvariantConfig, err = json.Marshal(expTCP.InvariantConfig)
	require.NoError(t, err)

	count, err = Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	_, err = Bun().NewInsert().Model(expTCP).
		Where("workspace_id = ?", w.ID).
		Where("workspace_type = ?", model.NTSCType).
		Value("invariant_config", "?", string(expInvariantConfig)).
		Exec(ctx)
	require.ErrorContains(t, err, "duplicate key value violates unique constraint")
}

func TestDeleteConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		err := CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name               string
		global             bool
		workloadType       model.WorkloadType
		hasInvariantConfig bool
		hasConstraints     bool
		err                *string
	}{
		{"exp config no constraint", false, model.ExperimentType, true, false, nil},
		{"exp config and constraint", false, model.ExperimentType, true, true, nil},
		{"exp no config has constraint", false, model.ExperimentType, false, true, nil},
		{"exp no config no constraint", false, model.ExperimentType, false, false, nil},
		{"ntsc config no constraint", false, model.NTSCType, true, false, nil},
		{"ntsc config and constraint", false, model.NTSCType, true, true, nil},
		{"ntsc no config has constraint", false, model.NTSCType, false, true, nil},
		{
			"unspecified workload type", false, model.UnknownType, true, true,
			ptrs.Ptr("invalid workload type"),
		},
		{"ntsc no config no constraint", false, model.NTSCType, false, false, nil},
		{"global exp config no constraint", true, model.ExperimentType, true, false, nil},
		{"global exp config no constraint", true, model.ExperimentType, true, true, nil},
		{"global ntsc config no constraint", true, model.NTSCType, true, false, nil},
		{"global ntsc config and constraint", true, model.NTSCType, true, true, nil},
		{
			"global unspecified workload type", true, model.UnknownType, true, true,
			ptrs.Ptr("invalid workload type"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.workloadType {
			case model.ExperimentType:
				w, expTCP, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, test.global,
					test.hasInvariantConfig, test.hasConstraints)
				if !test.global {
					workspaceIDs = append(workspaceIDs, int32(w.ID))
				}

				err := DeleteConfigPolicies(ctx, expTCP.WorkspaceID, test.workloadType)
				if test.err == nil {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, *test.err)
				}

			default:
				w, _, ntscTCP := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, test.global,
					test.hasInvariantConfig, test.hasConstraints)
				if !test.global {
					workspaceIDs = append(workspaceIDs, int32(w.ID))
				}

				err := DeleteConfigPolicies(ctx, ntscTCP.WorkspaceID, test.workloadType)
				if test.err == nil {
					require.NoError(t, err)
				} else {
					require.ErrorContains(t, err, *test.err)
				}
			}
		})
	}

	// Verify that trying to delete task config policies for a nonexistent scope doesn't error out.
	err := DeleteConfigPolicies(ctx, -1, model.ExperimentType)
	require.NoError(t, err)

	// Verify that trying to delete task config policies for a workspace with no set policies
	// doesn't error out.
	w := &model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err = Bun().NewInsert().Model(w).Exec(ctx)
	require.NoError(t, err)
	workspaceIDs = append(workspaceIDs, int32(w.ID))

	err = DeleteConfigPolicies(ctx, w.ID, model.ExperimentType)
	require.NoError(t, err)

	// Verify that we can create and delete task config policies individually for different
	// workspaces.
	w1, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w1.ID))
	q1 := Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w1.ID)
	count, err := q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	w2, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w2.ID))
	q2 := Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w2.ID)
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = DeleteConfigPolicies(ctx, w1.ID, model.ExperimentType)
	require.NoError(t, err)

	// Verify that exactly 1 task config policy was deleted from w1.
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify that both task config policies in w2 still exist.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = DeleteConfigPolicies(ctx, w2.ID, model.ExperimentType)
	require.NoError(t, err)

	// Verify that exactly 1 task config policy was deleted from w2.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = DeleteConfigPolicies(ctx, w1.ID, model.NTSCType)
	require.NoError(t, err)

	// Verify that no task config policies exist for w1.
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	err = DeleteConfigPolicies(ctx, w2.ID, model.NTSCType)
	require.NoError(t, err)

	// Verify that no task config policies exist for w2.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Test delete cascade on task config policies when a workspace is deleted.
	w1, _, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w1.ID))
	q1 = Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w1.ID)
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	w2, _, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w2.ID))
	q2 = Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w2.ID)
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Verify that only the task config policies of w1 were deleted when w1 was deleted.
	_, err = Bun().NewDelete().Model(&model.Workspace{}).Where("id = ?", w1.ID).Exec(ctx)
	require.NoError(t, err)

	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Verify that both task config policies of w2 are deleted when w2 is deleted.
	_, err = Bun().NewDelete().Model(&model.Workspace{}).Where("id = ?", w2.ID).Exec(ctx)
	require.NoError(t, err)

	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// CreateMockTaskConfigPolicies creates experiment and NTSC invariant configs and constraints as
// requested for the specified scope.
func CreateMockTaskConfigPolicies(ctx context.Context, t *testing.T,
	pgDB *PgDB, user model.User, global bool, hasInvariantConfig bool,
	hasConstraints bool) (*model.Workspace, *model.ExperimentTaskConfigPolicies,
	*model.NTSCTaskConfigPolicies,
) {
	var scope int
	var w model.Workspace
	var expConfig expconf.ExperimentConfig
	var ntscConfig model.CommandConfig

	var constraints model.Constraints

	if !global {
		w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := Bun().NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)
		scope = w.ID
	}
	if hasInvariantConfig {
		expConfig = DefaultExperimentConfig()
		ntscConfig = DefaultCommandConfig()
	}
	if hasConstraints {
		constraints = DefaultConstraints()
	}

	experimentTCP := &model.ExperimentTaskConfigPolicies{
		WorkspaceID:     scope,
		LastUpdatedBy:   user.ID,
		WorkloadType:    model.ExperimentType,
		InvariantConfig: expConfig,
		Constraints:     constraints,
	}
	err := SetExperimentConfigPolicies(ctx, experimentTCP)
	require.NoError(t, err)

	ntscTCP := &model.NTSCTaskConfigPolicies{
		WorkspaceID:     scope,
		LastUpdatedBy:   user.ID,
		WorkloadType:    model.NTSCType,
		InvariantConfig: ntscConfig,
		Constraints:     constraints,
	}
	err = SetNTSCConfigPolicies(ctx, ntscTCP)
	require.NoError(t, err)

	return &w, experimentTCP, ntscTCP
}

func DefaultExperimentConfig() expconf.ExperimentConfig {

	// Marshal and Unmarshal this and return the Unmarshaled version
	// var expConf expconf.ExperimentConfig
	// json.Unmarshal(bytes, &d.RawString)

	return expconf.ExperimentConfig{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr("path/to/config"),
			},
		},
		RawHyperparameters: expconf.HyperparametersV0{},
		RawName:            expconf.Name{RawString: ptrs.Ptr(uuid.NewString())},
		RawReproducibility: &expconf.ReproducibilityConfigV0{
			RawExperimentSeed: ptrs.Ptr[uint32](10),
		},
		RawSearcher: &expconf.SearcherConfigV0{
			RawMetric: ptrs.Ptr("training_test"),
			RawSingleConfig: &expconf.SingleConfigV0{
				RawMaxLength: &expconf.LengthV0{
					Unit:  expconf.Batches,
					Units: uint64(2),
				},
			},
		},
	}
}

func DefaultCommandConfig() model.CommandConfig {
	return model.CommandConfig{
		Description: "random description",
		Resources: model.ResourcesConfig{
			Slots:    4,
			MaxSlots: ptrs.Ptr(8),
		},
	}
}

func DefaultConstraints() model.Constraints {
	return model.Constraints{
		PriorityLimit: ptrs.Ptr[int](10),
		ResourceConstraints: &model.ResourceConstraints{
			MaxSlots: ptrs.Ptr(10),
		},
	}
}
