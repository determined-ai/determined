//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"google.golang.org/grpc/codes"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAddAndGetExperimentConfigPolicy(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	tests := []struct {
		name          string
		experimentTCP *model.ExperimentTaskConfigPolicy
		err           error
	}{
		{
			"globasic add",
			&model.ExperimentTaskConfigPolicy{},
			nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := AddExperimentConfigPolicies(ctx, test.experimentTCP)
			require.ErrorIs(t, test.err, err)
		})
	}
	user := RequireMockUser(t, pgDB)
	workspaceIDs, err := MockWorkspaces([]string{"test1", "test2", "test3", "test4"}, user.ID)
	require.NoError(t, err, "failed creating workspaces: %t", err)
	defer func() {
		err = CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	// Things to test...

	// With this, we should get a primary key constraint violation
	// Test that we cannot add an invalid workspace id
	// Test with invalid constraints
	// Test with invalid config
	// Test with invalid workload type
	// Test that we cannot add an anything with a nonexistent user

	// Test that we can add a config and not a constraint
	// Test that we can add a constraint and null config
	// Test that we can add both a config and a constraint
	// Test all of the defaults

	// Do all of this for some experiments and some NTSC tasks

	// We should probably have the getter available so that we can retrieve from the database and verify
	// That everything is as we stored it.
	// ^ This could be a helper method that also gets used when testing the the Get function
	// Test that the result of the get is equal tothe

	// Test that we cannot add tcps for the same scope , including null scope
	// But we stil want to make sure that we violate the unique constraint if we try to creaet two
	// of the same workspace id or, for the same workspace id, we try to create two different workload types
}

func TestAddNTSCTaskConfigPolicy(t *testing.T) {
	ctx := context.Background()

	require.NoError(t, etc.SetRootPath(RootFromDB))
	pgDB, cleanup := MustResolveNewPostgresDatabase(t)
	defer cleanup()
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs, err := MockWorkspaces([]string{"test1", "test2", "test3", "test4"}, user.ID)
	require.NoError(t, err, "failed creating workspaces: %t", err)
	defer func() {
		err = CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()
}

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
	// TODO: Create a workspace and give it a valid task-config policy.
	var wkspID int
	expCP, ntscCP, err = getConfigPolicies(ctx, &wkspID, model.UnknownType)
	require.Nil(t, expCP)
	require.Nil(t, ntscCP)
	require.ErrorContains(t, err, codes.InvalidArgument.String())
}

// For the DELETE tests
// Add some experiment config and constraints and verify that we can delete them
// Add some NTSC config and constraints and verify that we can delete them
// Verify that non-esistent scope returns an error when trying to delete
// Verify that we cannot delete things for an invalid scope
// Verify tha twe cannot delete things with an invalid workload type
