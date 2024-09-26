//go:build integration
// +build integration

package configpolicy

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/stretchr/testify/require"
)

func TestSetTaskConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		if len(workspaceIDs) > 0 {
			err := db.CleanupMockWorkspace(workspaceIDs)
			if err != nil {
				log.Errorf("error when cleaning up mock workspaces")
			}
		}
	}()

	tests := []struct {
		name   string
		tcps   *model.TaskConfigPolicies
		global bool
		err    *string
	}{
		{
			"invalid user id",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   -1,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			false,
			ptrs.Ptr("violates foreign key constraint"),
		},
		{
			"valid config no constraint",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     nil,
			},
			false,
			nil,
		},
		{
			"valid constraint no config",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: nil,
				Constraints:     DefaultConstraints(),
			},
			false,
			nil,
		},
		{
			"valid constraint valid config",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			false,
			nil,
		},
		{
			"global valid constraint no config",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				WorkspaceID:     nil,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: nil,
				Constraints:     DefaultConstraints(),
			},
			true,
			nil,
		},
		{
			"global valid config no constraint",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     nil,
			},
			true,
			nil,
		},
		{
			"global valid constraint valid config",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: DefaultInvariantConfig(),
				Constraints:     DefaultConstraints(),
			},
			true,
			nil,
		},
		{
			"global no constraint no config",
			&model.TaskConfigPolicies{
				WorkloadType:    model.NTSCType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: nil,
				Constraints:     nil,
			},
			true,
			nil,
		},
		{
			"experiment workload type for TCP policies",
			&model.TaskConfigPolicies{
				WorkloadType:    model.ExperimentType,
				LastUpdatedBy:   user.ID,
				LastUpdatedTime: time.Now().UTC().Truncate(time.Second),
				InvariantConfig: nil,
				Constraints:     nil,
			},
			true,
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var w model.Workspace
			if !test.global {
				w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
				_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
				require.NoError(t, err)
				workspaceIDs = append(workspaceIDs, int32(w.ID))
				test.tcps.WorkspaceID = ptrs.Ptr(w.ID)
			}

			// Test add NTSC task config policies.
			err := SetTaskConfigPolicies(ctx, test.tcps)
			if test.err != nil {
				require.ErrorContains(t, err, *test.err)
				return
			}
			require.NoError(t, err)

			// Test get NTSC task config policies.
			tcps, err := GetTaskConfigPolicies(ctx, test.tcps.WorkspaceID, test.tcps.WorkloadType)
			require.NoError(t, err)
			tcps.LastUpdatedTime = tcps.LastUpdatedTime.UTC()
			requireEqualTaskPolicy(t, test.tcps, tcps)

			// Test update NTSC task config policies.
			test.tcps.InvariantConfig = ptrs.Ptr(
				`{"description":"random description","resources":{"slots":4,"max_slots":8},"notebook_idle_type":"activity"}`)
			err = SetTaskConfigPolicies(ctx, test.tcps)
			require.NoError(t, err)

			// Test get NTSC task config policies.
			tcps, err = GetTaskConfigPolicies(ctx, test.tcps.WorkspaceID, test.tcps.WorkloadType)
			require.NoError(t, err)
			tcps.LastUpdatedTime = tcps.LastUpdatedTime.UTC().Truncate(time.Second)
			requireEqualTaskPolicy(t, test.tcps, tcps)
		})
	}

	// Test invalid workspace ID.
	err := SetTaskConfigPolicies(ctx, &model.TaskConfigPolicies{
		WorkspaceID:     ptrs.Ptr(-1),
		LastUpdatedBy:   user.ID,
		WorkloadType:    model.NTSCType,
		InvariantConfig: DefaultInvariantConfig(),
		Constraints:     DefaultConstraints(),
	})
	require.ErrorContains(t, err, "violates foreign key constraint")
}

// Test the enforcement of the primary key on the task_config_polciies table.
func TestTaskConfigPoliciesUnique(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)

	// Global scope.
	_, _, ntscTCPs := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, true, true, true)
	ntscTCPs.Constraints = nil
	expInvariantConfig, err := json.Marshal(ntscTCPs.InvariantConfig)
	require.NoError(t, err)

	count, err := db.Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id IS NULL").
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = db.Bun().NewInsert().Model(ntscTCPs).
		Where("workspace_id = ?", nil).
		Where("workspace_type = ?", model.ExperimentType).
		Value("invariant_config", "?", string(expInvariantConfig)).
		Exec(ctx)
	require.ErrorContains(t, err, "duplicate key value violates unique constraint")

	// Workspace-level.
	w, _, ntscTCPs := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	ntscTCPs.Constraints = nil
	expInvariantConfig, err = json.Marshal(ntscTCPs.InvariantConfig)
	require.NoError(t, err)

	count, err = db.Bun().NewSelect().
		Table("task_config_policies").
		Where("workspace_id = ?", w.ID).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	_, err = db.Bun().NewInsert().Model(ntscTCPs).
		Where("workspace_id = ?", w.ID).
		Where("workspace_type = ?", model.NTSCType).
		Value("invariant_config", "?", string(expInvariantConfig)).
		Exec(ctx)
	require.ErrorContains(t, err, "duplicate key value violates unique constraint")
}

func TestDeleteConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(db.RootFromDB))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	user := db.RequireMockUser(t, pgDB)
	workspaceIDs := []int32{}

	defer func() {
		err := db.CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	tests := []struct {
		name               string
		global             bool
		workloadType       string
		hasInvariantConfig bool
		hasConstraints     bool
		err                *string
	}{
		{"ntsc config no constraint", false, model.NTSCType, true, false, nil},
		{"ntsc config and constraint", false, model.NTSCType, true, true, nil},
		{"ntsc no config has constraint", false, model.NTSCType, false, true, nil},
		{
			"unspecified workload type", false, model.UnknownType, true, true,
			ptrs.Ptr("invalid input value for enum"),
		},
		{"ntsc no config no constraint", false, model.NTSCType, false, false, nil},
		{"global ntsc config no constraint", true, model.NTSCType, true, false, nil},
		{"global ntsc config and constraint", true, model.NTSCType, true, true, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
		})
	}

	// Verify that trying to delete task config policies for a nonexistent scope doesn't error out.
	err := DeleteConfigPolicies(ctx, ptrs.Ptr(-1), model.ExperimentType)
	require.NoError(t, err)

	// Verify that trying to delete task config policies for a workspace with no set policies
	// doesn't error out.
	w := &model.Workspace{Name: uuid.NewString(), UserID: user.ID}
	_, err = db.Bun().NewInsert().Model(w).Exec(ctx)
	require.NoError(t, err)
	workspaceIDs = append(workspaceIDs, int32(w.ID))

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that we can create and delete task config policies individually for different
	// workspaces.
	w1, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w1.ID))
	q1 := db.Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w1.ID)
	count, err := q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	w2, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w2.ID))
	q2 := db.Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w2.ID)
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w1.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that exactly 1 task config policy was deleted from w1.
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify that both task config policies in w2 still exist.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w2.ID), model.ExperimentType)
	require.NoError(t, err)

	// Verify that exactly 1 task config policy was deleted from w2.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w1.ID), model.NTSCType)
	require.NoError(t, err)

	// Verify that no task config policies exist for w1.
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	err = DeleteConfigPolicies(ctx, ptrs.Ptr(w2.ID), model.NTSCType)
	require.NoError(t, err)

	// Verify that no task config policies exist for w2.
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Test delete cascade on task config policies when a workspace is deleted.
	w1, _, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w1.ID))
	q1 = db.Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w1.ID)
	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	w2, _, _ = CreateMockTaskConfigPolicies(ctx, t, pgDB, user, false, true, true)
	workspaceIDs = append(workspaceIDs, int32(w2.ID))
	q2 = db.Bun().NewSelect().Table("task_config_policies").Where("workspace_id = ?", w2.ID)
	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify that only the task config policies of w1 were deleted when w1 was deleted.
	_, err = db.Bun().NewDelete().Model(&model.Workspace{}).Where("id = ?", w1.ID).Exec(ctx)
	require.NoError(t, err)

	count, err = q1.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify that both task config policies of w2 are deleted when w2 is deleted.
	_, err = db.Bun().NewDelete().Model(&model.Workspace{}).Where("id = ?", w2.ID).Exec(ctx)
	require.NoError(t, err)

	count, err = q2.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

// CreateMockTaskConfigPolicies creates experiment and NTSC invariant configs and constraints as
// requested for the specified scope.
func CreateMockTaskConfigPolicies(ctx context.Context, t *testing.T,
	pgDB *db.PgDB, user model.User, global bool, hasInvariantConfig bool,
	hasConstraints bool) (*model.Workspace, *model.TaskConfigPolicies,
	*model.TaskConfigPolicies,
) {
	var scope *int
	var w model.Workspace
	var ntscConfig *string

	var constraints *string

	if !global {
		w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := db.Bun().NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)
		scope = ptrs.Ptr(w.ID)
	}
	if hasInvariantConfig {
		ntscConfig = DefaultInvariantConfig()
	}
	if hasConstraints {
		constraints = DefaultConstraints()
	}

	ntscTCP := &model.TaskConfigPolicies{
		WorkspaceID:     scope,
		WorkloadType:    model.NTSCType,
		LastUpdatedBy:   user.ID,
		InvariantConfig: ntscConfig,
		Constraints:     constraints,
	}
	err := SetTaskConfigPolicies(ctx, ntscTCP)
	require.NoError(t, err)

	return &w, nil, ntscTCP
}

func DefaultInvariantConfig() *string {
	return ptrs.Ptr(DefaultInvariantConfigStr)
}

func DefaultInvariantConfigImage() *string {
	return ptrs.Ptr(DefaultInvariantConfigStrImage)
}

func DefaultConstraints() *string {
	return ptrs.Ptr(DefaultConstraintsStr)
}

func requireEqualTaskPolicy(t *testing.T, exp *model.TaskConfigPolicies, act *model.TaskConfigPolicies) {
	require.Equal(t, exp.LastUpdatedBy, act.LastUpdatedBy)
	require.Equal(t, exp.LastUpdatedTime, act.LastUpdatedTime)
	require.Equal(t, exp.WorkloadType, act.WorkloadType)
	require.Equal(t, exp.WorkspaceID, act.WorkspaceID)

	if exp.Constraints == nil {
		require.Nil(t, exp.Constraints, act.Constraints)
	} else {
		var expJSONMap, actJSONMap map[string]interface{}
		err := json.Unmarshal([]byte(*exp.Constraints), &expJSONMap)
		require.NoError(t, err)
		err = json.Unmarshal([]byte(*act.Constraints), &actJSONMap)
		require.NoError(t, err)
		require.Equal(t, expJSONMap, actJSONMap)
	}
	if exp.InvariantConfig == nil {
		require.Nil(t, exp.InvariantConfig, act.InvariantConfig)
	} else {
		var expJSONMap, actJSONMap map[string]interface{}
		err := json.Unmarshal([]byte(*exp.InvariantConfig), &expJSONMap)
		require.NoError(t, err)
		err = json.Unmarshal([]byte(*act.InvariantConfig), &actJSONMap)
		require.NoError(t, err)
		require.Equal(t, expJSONMap, actJSONMap)
	}
}
