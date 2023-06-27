//go:build integration
// +build integration

package db

import (
	"context"
	"sort"
	"testing"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/etc"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestAddAndRemoveBindings(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
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

	var int32IDs []int32
	for _, workspaceID := range workspaceIDs {
		int32IDs = append(int32IDs, workspaceID)
	}
	// Test single insert/delete
	// Test bulk insert/delete
	testPoolName := "test pool"
	testPool2Name := "test pool too"
	err = AddRPWorkspaceBindings(ctx, []int32{int32IDs[0]}, testPoolName, []config.ResourcePoolConfig{
		{PoolName: testPoolName}, {PoolName: testPool2Name},
	})
	require.NoError(t, err, "failed to add bindings: %t", err)

	var values []RPWorkspaceBinding
	count, err := Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 1, count, "expected 1 item in DB, found %d", count)
	require.Equal(t, int(workspaceIDs[0]), values[0].WorkspaceID,
		"expected workspaceID to be %d, but it is %d", workspaceIDs[0], values[0].WorkspaceID)
	require.Equal(t, testPoolName, values[0].PoolName,
		"expected pool name to be '%s', but got %s", testPoolName, values[0].PoolName)

	err = RemoveRPWorkspaceBindings(ctx, []int32{int32IDs[0]}, testPoolName)
	require.NoError(t, err, "failed to remove bindings: %t", err)

	count, err = Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 0, count, "expected 0 items in DB, found %d", count)

	err = AddRPWorkspaceBindings(ctx, int32IDs, testPool2Name, []config.ResourcePoolConfig{
		{PoolName: testPoolName}, {PoolName: testPool2Name},
	})
	require.NoError(t, err, "failed to add bindings: %t", err)

	count, err = Bun().NewSelect().Model(&values).Order("workspace_id ASC").ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 4, count, "expected 4 items in DB, found %d", count)
	for i := 0; i < 4; i++ {
		require.Equal(t, int(workspaceIDs[i]), values[i].WorkspaceID,
			"expected workspaceID to be %d, but it is %d", i+1, values[i].WorkspaceID)
		require.Equal(t, testPool2Name, values[i].PoolName,
			"expected pool name to be '%s', but got %s", testPool2Name, values[i].PoolName)
	}

	err = RemoveRPWorkspaceBindings(ctx, int32IDs, testPool2Name)
	require.NoError(t, err, "failed to remove bindings: %t", err)

	count, err = Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 0, count, "expected 0 items in DB, found %d", count)
}

func TestBindingFail(t *testing.T) {
	// Test add the same binding multiple times - should fail
	// Test add same binding among bulk transaction - should fail the entire transaction
	// Test add workspace that doesn't exist
	// Test add RP that doesn't exist
	return
}

func TestListWorkspacesBindingRP(t *testing.T) {
	// pretty straightforward
	// don't list bindings that are invalid
	// if RP is unbound, return nothing
	return
}

func TestListRPsAvailableToWorkspace(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	ctx := context.Background()
	user := RequireMockUser(t, db)

	workspaceNames := []string{"workspace1", "workspace2"}
	workspaceIDs, err := MockWorkspaces(workspaceNames, user.ID)
	require.NoError(t, err)
	existingPools := []config.ResourcePoolConfig{
		{PoolName: "poolName1"},
		{PoolName: "poolName2"},
		{PoolName: "poolName3"},
	}

	// A workspace has no bounded RP
	rpNames, _, err := ReadRPsAvailableToWorkspace(ctx, workspaceIDs[0], 0, 0, existingPools)
	require.NoError(t, err)

	expectedRPNames := []string{}
	for _, rpName := range existingPools {
		expectedRPNames = append(expectedRPNames, rpName.PoolName)
	}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	// Workspace 1 has no bounded RP and a RP is bound to workspace 2
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName2", existingPools)
	require.NoError(t, err)

	rpNames, _, err = ReadRPsAvailableToWorkspace(ctx, workspaceIDs[0], 0, 0, existingPools)
	require.NoError(t, err)
	expectedRPNames = []string{"poolName1", "poolName3"}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName2")
	require.NoError(t, err)

	// A workspace has some bounded RP and some unbound RP
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName1", existingPools)
	require.NoError(t, err)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName2", existingPools)
	require.NoError(t, err)

	rpNames, _, err = ReadRPsAvailableToWorkspace(ctx, workspaceIDs[0], 0, 0, existingPools)
	require.NoError(t, err)
	expectedRPNames = []string{"poolName1", "poolName3"}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName1")
	require.NoError(t, err)
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName2")
	require.NoError(t, err)

	// A workspace only has bounded RP.
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName1", existingPools)
	require.NoError(t, err)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs, "poolName2", existingPools)
	require.NoError(t, err)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName3", existingPools)
	require.NoError(t, err)

	rpNames, _, err = ReadRPsAvailableToWorkspace(ctx, workspaceIDs[0], 0, 0, existingPools)
	require.NoError(t, err)
	expectedRPNames = []string{"poolName1", "poolName2"}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName1")
	require.NoError(t, err)
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs, "poolName2")
	require.NoError(t, err)
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[1:2], "poolName3")
	require.NoError(t, err)

	// Test pagination
	morePools := []config.ResourcePoolConfig{
		{PoolName: "poolName4"},
		{PoolName: "poolName5"},
		{PoolName: "poolName6"},
		{PoolName: "poolName7"},
		{PoolName: "poolName8"},
	}
	existingPools = append(existingPools, morePools...)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName2", existingPools)
	require.NoError(t, err)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName3", existingPools)
	require.NoError(t, err)

	rpNames, pagination, err := ReadRPsAvailableToWorkspace(ctx, workspaceIDs[0], 0, 3, existingPools)
	require.NoError(t, err)
	expectedRPNames = []string{}
	for _, rpName := range existingPools[0:3] {
		expectedRPNames = append(expectedRPNames, rpName.PoolName)
	}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	rpNames, pagination, err = ReadRPsAvailableToWorkspace(
		ctx, workspaceIDs[0], pagination.EndIndex, pagination.Limit, existingPools,
	)
	require.NoError(t, err)
	expectedRPNames = []string{}
	for _, rpName := range existingPools[3:6] {
		expectedRPNames = append(expectedRPNames, rpName.PoolName)
	}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	rpNames, _, err = ReadRPsAvailableToWorkspace(
		ctx, workspaceIDs[0], pagination.EndIndex, pagination.Limit, existingPools,
	)
	require.NoError(t, err)
	expectedRPNames = []string{}
	for _, rpName := range existingPools[6:8] {
		expectedRPNames = append(expectedRPNames, rpName.PoolName)
	}
	sort.Strings(rpNames)
	require.Equal(t, expectedRPNames, rpNames)

	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName2")
	require.NoError(t, err)
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs[0:1], "poolName3")
	require.NoError(t, err)

	err = CleanupMockWorkspace(workspaceIDs)
	require.NoError(t, err)
	return
}

func TestListAllBindings(t *testing.T) {
	// pretty straightforward
	// list ALL bindings, even invalid ones
	// make sure to return unbound pools too (we pull from config)
	return
}

func TestOverwriteBindings(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	ctx := context.Background()
	user := RequireMockUser(t, db)
	var existingPools []config.ResourcePoolConfig
	pool := config.ResourcePoolConfig{PoolName: "poolName1"}
	existingPools = append(existingPools, pool)
	// Test overwrite bindings
	poolName := "poolName1" //nolint:goconst
	workspaceNames := []string{"test1", "test2", "test3"}
	workspaceIDs, err := MockWorkspaces(workspaceNames, user.ID)
	require.NoError(t, err)
	err = AddRPWorkspaceBindings(ctx, workspaceIDs, poolName, existingPools)
	require.NoError(t, err)
	err = OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName, existingPools)
	require.NoError(t, err)
	// TODO: call list bindings here to make sure it worked
	// Test overwrite pool that's not bound to anything currently
	pool = config.ResourcePoolConfig{PoolName: "poolName2"}
	existingPools = append(existingPools, pool)
	poolName = "poolName2"
	err = OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName, existingPools)
	require.NoError(t, err)
	// TODO: call list bindings here to make sure it worked
	err = CleanupMockWorkspace(workspaceIDs)
	require.NoError(t, err)
	return
}

func TestOverwriteFail(t *testing.T) {
	ctx := context.Background()
	var existingPools []config.ResourcePoolConfig
	pool := config.ResourcePoolConfig{PoolName: "poolName1"}
	existingPools = append(existingPools, pool)
	// Test overwrite adding workspace that doesn't exist
	workspaceIDs := []int32{100, 102, 103}
	poolName := "poolName1"
	err := OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName, existingPools)
	// db Error that workspace doesn't exist
	require.ErrorContains(t, err, "violates foreign key constraint")
	// Test overwrite pool that doesn't exist
	poolName = "poolNameDoesntExist"
	err = OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName, existingPools)
	require.ErrorContains(t, err, "doesn't exist in config")
	return
}

func TestRemoveInvalidBinding(t *testing.T) {
	ctx := context.Background()
	// remove binding that doesn't exist
	poolName := "poolName" //nolint:goconst
	workspaceIDs := []int32{1}
	err := RemoveRPWorkspaceBindings(ctx, workspaceIDs, poolName)
	require.ErrorContains(t, err, "binding doesn't exist")
	// bulk remove bindings that don't exist
	poolName = "poolName"
	workspaceIDs = []int32{1, 2, 3}
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs, poolName)
	require.ErrorContains(t, err, " binding doesn't exist")
	return
}
