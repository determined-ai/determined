//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/configpolicy"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/stretchr/testify/require"
)

func TestCannotGetUnspecifiedWorkloadConfigPolicy(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	// Global scope.
	expCP, ntscCP, err := getConfigPolicies(ctx, nil, model.UnknownType)
	require.Nil(t, expCP)
	require.Nil(t, ntscCP)
	require.ErrorContains(t, err, codes.InvalidArgument.String())

	// Workspace-level scope.
	// (Setup) Create a workspace and give it a valid constraints policy.
	w, _, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, false)

	// Verify that we can get the workspace's constraints policy for experiments.
	expCP, _, err = getConfigPolicies(ctx, &w.ID, model.ExperimentType)
	require.NoError(t, err)
	require.NotNil(t, expCP)

	var wkspID int
	expCP, ntscCP, err = getConfigPolicies(ctx, &wkspID, model.UnknownType)
	require.Nil(t, expCP)
	require.Nil(t, ntscCP)
	require.ErrorContains(t, err, codes.InvalidArgument.String())
}

func TestDeleteConfigPolicies(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	tests := []struct {
		name         string
		global       bool
		workloadType model.WorkloadType
		err          error
	}{
		{"invariant config no constraint", false, model.ExperimentType, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			switch test.workloadType {
			case model.ExperimentType:
				_, expTCP, _ := CreateMockTaskConfigPolicies(ctx, t, pgDB, test.global)
				err := DeleteConfigPolicies(ctx, expTCP.WorkspaceID, test.workloadType)
				require.Equal(t, test.err, err)
			case model.NTSCType:
				_, _, ntscTCP := CreateMockTaskConfigPolicies(ctx, t, pgDB, test.global)
				err := DeleteConfigPolicies(ctx, ntscTCP.WorkspaceID, test.workloadType)
				require.Equal(t, test.err, err)
			}

		})
	}

	// Cleanup
}

func CreateMockTaskConfigPolicies(ctx context.Context, t *testing.T,
	pgDB *PgDB, global bool) (*model.Workspace, *model.ExperimentTaskConfigPolicies,
	*model.NTSCTaskConfigPolicies) {
	user := RequireMockUser(t, pgDB)

	var scope *int
	var w model.Workspace
	if !global {
		w = model.Workspace{Name: uuid.NewString(), UserID: user.ID}
		_, err := Bun().NewInsert().Model(&w).Exec(ctx)
		require.NoError(t, err)
		scope = &w.ID
	}
	experimentTCP := &model.ExperimentTaskConfigPolicies{
		WorkspaceID:   scope,
		LastUpdatedBy: user.ID,
		WorkloadType:  model.ExperimentType,
		InvariantConfig: &expconf.ExperimentConfigV0{
			RawWorkspace:   &w.Name,
			RawDescription: ptrs.Ptr("random description"),
		},
		Constraints: &configpolicy.Constraints{PriorityLimit: ptrs.Ptr(10)},
	}
	err := SetExperimentConfigPolicies(ctx, experimentTCP)
	require.NoError(t, err)

	ntscTCP := &model.NTSCTaskConfigPolicies{
		WorkspaceID:   scope,
		LastUpdatedBy: user.ID,
		WorkloadType:  model.ExperimentType,
		InvariantConfig: &model.CommandConfig{
			Description: "random description",
			Resources: model.ResourcesConfig{
				Slots:    4,
				MaxSlots: ptrs.Ptr(8),
			},
		},
		Constraints: &configpolicy.Constraints{PriorityLimit: ptrs.Ptr[int](10),
			ResourceConstraints: &configpolicy.ResourceConstraints{
				MaxSlots: ptrs.Ptr(10),
			}},
	}
	err = SetNTSCConfigPolicies(ctx, ntscTCP)
	require.NoError(t, err)

	return &w, experimentTCP, ntscTCP
}
