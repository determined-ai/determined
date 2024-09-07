//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/require"
)

func TestSetAndGetExperimentConfigPolicies(t *testing.T) {
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
		name       string
		expTCP     *model.ExperimentTaskConfigPolicies
		updatedTCP *model.ExperimentTaskConfigPolicies
		expected   model.ExperimentTaskConfigPolicies
		err        *string
	}{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w = &model.Workspace{Name: uuid.NewString(), UserID: user.ID}
			_, err := Bun().NewInsert().Model(w).Exec(ctx)
			require.NoError(t, err)
			// Test add
			err = SetExperimentConfigPolicies(ctx, test.expTCP)
			if test.err == nil {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, *test.err)
			}

			// Test Get

			// Test update.

		})
	}
}

func TestSetAndGetNTSCConfigPolicies(t *testing.T) {
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
		ntscTCP  model.NTSCTaskConfigPolicies
		expected model.NTSCTaskConfigPolicies
		err      *string
	}{}

	for _, test := range tests {

	}
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
		{"unspecified workload type", false, model.UnknownType, true, true,
			ptrs.Ptr("invalid workload type")},
		{"ntsc no config no constraint", false, model.NTSCType, false, false, nil},
		{"global exp config no constraint", true, model.ExperimentType, true, false, nil},
		{"global exp config no constraint", true, model.ExperimentType, true, true, nil},
		{"global ntsc config no constraint", true, model.NTSCType, true, false, nil},
		{"global ntsc config and constraint", true, model.NTSCType, true, true, nil},
		{"global unspecified workload type", true, model.UnknownType, true, true,
			ptrs.Ptr("invalid workload type")},
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
					require.Nil(t, err)
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
					require.Nil(t, err)
				} else {
					require.ErrorContains(t, err, *test.err)
				}
			}
		})
	}

	// Verify that trying to delete task config policies for a nonexistent scope doesn't error out.
	err := DeleteConfigPolicies(ctx, ptrs.Ptr(-1), model.ExperimentType)
	require.NoError(t, err)

	// Verify that trying to delete task config policies for a workspace with no set policies
	// doesn't error out.
	w := &model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err = Bun().NewInsert().Model(w).Exec(ctx)
	require.NoError(t, err)
	workspaceIDs = append(workspaceIDs, int32(w.ID))

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that we can delete both the experiment and NTSC task config policy for the same
	// workspace.
	w, _, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w.ID))

	q := Bun().NewSelect().Table("task_config_policies")
	count, err := q.Where("workspace_id = ?", w.ID).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w.ID), model.ExperimentType)
	require.NoError(t, err)

	count, err = q.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w.ID), model.NTSCType)
	require.NoError(t, err)
	count, err = q.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Verify that we can create and delete task config policies individually for different
	// workspaces.
	w1, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w1.ID))
	count, err = Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w1.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	w2, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w2.ID))
	count, err = Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w2.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w1.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that the task config policy in the other workspace still exists.
	count, err = Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w2.ID).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w2.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that the task config policy in the other workspace still exists.
	count, err = Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w2.ID).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

// CreateMockTaskConfigPolicies creates experiment and NTSC invariant configs and constraints as
// requested for the specified scope.
func CreateMockTaskConfigPolicies(ctx context.Context, t *testing.T,
	pgDB *PgDB, user model.User, global bool, hasInvariantConfig bool,
	hasConstraints bool) (*model.Workspace, *model.ExperimentTaskConfigPolicies,
	*model.NTSCTaskConfigPolicies) {

	var scope *int
	var w *model.Workspace
	var expConfig expconf.ExperimentConfig
	var ntscConfig model.CommandConfig

	var constraints model.Constraints

	if !global {
		w = &model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := Bun().NewInsert().Model(w).Exec(ctx)
		require.NoError(t, err)
		scope = &w.ID
	}
	if hasInvariantConfig {
		expConfig = DefaultExperimentConfig()
		ntscConfig = model.CommandConfig{
			Description: "random description",
			Resources: model.ResourcesConfig{
				Slots:    4,
				MaxSlots: ptrs.Ptr(8),
			},
		}
	}
	if hasConstraints {
		constraints = model.Constraints{PriorityLimit: ptrs.Ptr[int](10),
			ResourceConstraints: &model.ResourceConstraints{
				MaxSlots: ptrs.Ptr(10),
			}}
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

	return w, experimentTCP, ntscTCP
}

func DefaultExperimentConfig() expconf.ExperimentConfigV0 {
	return expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr("path/to/config")},
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
