//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/etc"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

const (
	testPoolName  = "test pool"
	testPool2Name = "test pool too"
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

	// Test single insert/delete
	// Test bulk insert/delete
	err = AddRPWorkspaceBindings(ctx,
		[]int32{workspaceIDs[0]}, testPoolName, []config.ResourcePoolConfig{
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

	err = RemoveRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPoolName)
	require.NoError(t, err, "failed to remove bindings: %t", err)

	count, err = Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 0, count, "expected 0 items in DB, found %d", count)

	err = AddRPWorkspaceBindings(ctx, workspaceIDs, testPool2Name, []config.ResourcePoolConfig{
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

	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs, testPool2Name)
	require.NoError(t, err, "failed to remove bindings: %t", err)

	count, err = Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 0, count, "expected 0 items in DB, found %d", count)
}

func TestBindingFail(t *testing.T) {
	// Test add the same binding multiple times - should fail
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	user := RequireMockUser(t, pgDB)
	workspaceIDs, err := MockWorkspaces([]string{"test1", "test2", "test3"}, user.ID)
	require.NoError(t, err, "failed creating workspaces: %t", err)
	defer func() {
		err = CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	nonExistentWorkspaceID := 9999999
	poolConfigs := []config.ResourcePoolConfig{{PoolName: testPoolName}, {PoolName: testPool2Name}}
	err = AddRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPoolName, poolConfigs)
	require.NoError(t, err, "failed to add bindings: %t", err)

	err = AddRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPoolName, poolConfigs)
	require.Errorf(t, err, "expected error adding binding for workspace %d", workspaceIDs[0])
	// Test add same binding among bulk transaction - should fail the entire transaction
	err = AddRPWorkspaceBindings(ctx,
		[]int32{workspaceIDs[1], workspaceIDs[1]}, testPoolName, poolConfigs)
	require.Errorf(t, err,
		"expected error adding binding for workspaces %d, %d", workspaceIDs[0], workspaceIDs[1])
	// Test add workspace that doesn't exist
	err = AddRPWorkspaceBindings(
		ctx, []int32{int32(nonExistentWorkspaceID)}, testPool2Name, poolConfigs)
	require.Errorf(t, err,
		"expected error adding binding for non-existent workspaces %d", nonExistentWorkspaceID)
	// Test add RP that doesn't exist
	err = AddRPWorkspaceBindings(ctx, []int32{workspaceIDs[2]}, "NonexistentPool", poolConfigs)
	require.Errorf(t, err, "expected error adding binding for workspaces %d on non-existent pool",
		workspaceIDs[2], workspaceIDs[1])

	return
}

func TestListWorkspacesBindingRP(t *testing.T) {
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
	// no bindings
	bindings, _, err := ReadWorkspacesBoundToRP(ctx, testPoolName, 0, 0)
	require.NoError(t, err, "error when reading workspaces bound to RP %s", testPoolName)
	require.Equal(t, 0, len(bindings),
		"expected length of bindings to be 0, but got %d", len(bindings))

	// one binding
	poolConfigs := []config.ResourcePoolConfig{{PoolName: testPoolName}, {PoolName: testPool2Name}}
	err = AddRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPoolName, poolConfigs)
	require.NoError(t, err, "failed to add bindings: %t", err)

	bindings, _, err = ReadWorkspacesBoundToRP(ctx, testPoolName, 0, 0)
	require.NoError(t, err, "error when reading workspaces bound to RP %s", testPoolName)
	require.Equal(t, 1, len(bindings),
		"expected length of bindings to be 0, but got %d", len(bindings))
	require.Equal(t, int(workspaceIDs[0]), bindings[0].WorkspaceID,
		"expected workspaceID %d, but got %d", workspaceIDs[0], bindings[0].WorkspaceID)

	// don't list bindings that are invalid
	bindingsToInsert := []RPWorkspaceBinding{
		{WorkspaceID: int(workspaceIDs[0]), PoolName: "invalid pool", Valid: false},
		{WorkspaceID: int(workspaceIDs[1]), PoolName: testPoolName, Valid: false},
	}
	_, err = Bun().NewInsert().Model(&bindingsToInsert).Exec(ctx)
	require.NoError(t, err, "error inserting invalid entries into rp_workspace_bindings table")

	bindings, _, err = ReadWorkspacesBoundToRP(ctx, testPoolName, 0, 0)
	require.NoError(t, err, "error when reading workspaces bound to RP %s", testPoolName)
	require.Equal(t, 1, len(bindings),
		"expected length of bindings to be 0, but got %d", len(bindings))
	require.Equal(t, int(workspaceIDs[0]), bindings[0].WorkspaceID,
		"expected workspaceID %d, but got %d", workspaceIDs[0], bindings[0].WorkspaceID)

	return
}

func TestListRPsBoundToWorkspace(t *testing.T) {
	// pretty straightforward
	// don't list binding that are invalid
	// return unbound pools too (we pull from config)
	return
}

func bindingsEqual(a, b []int32) bool {
	aMap := map[int32]bool{}
	for _, val := range a {
		aMap[val] = false
	}

	for _, val := range b {
		found, ok := aMap[val]
		if !ok || found {
			return false
		}
		aMap[val] = true
	}
	return true
}

func TestOverwriteBindings(t *testing.T) {
	require.NoError(t, etc.SetRootPath(RootFromDB))
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	ctx := context.Background()
	user := RequireMockUser(t, db)

	existingPools := []config.ResourcePoolConfig{{PoolName: testPoolName}}
	// Test overwrite bindings
	workspaceNames := []string{"test1", "test2", "test3"}
	workspaceIDs, err := MockWorkspaces(workspaceNames, user.ID)
	require.NoError(t, err, "failed to add bindings: %t", err)
	defer func() {
		err = CleanupMockWorkspace(workspaceIDs)
		if err != nil {
			log.Errorf("error when cleaning up mock workspaces")
		}
	}()

	err = AddRPWorkspaceBindings(ctx, workspaceIDs, testPoolName, existingPools)
	require.NoError(t, err, "failed to add ")
	bindings, _, err := ReadWorkspacesBoundToRP(ctx, testPoolName, 0, 0)
	require.NoError(t, err, "failed to read bindings: %t", err)
	require.Equal(t, 3, len(bindings),
		"expected bindings length 3, but got length %d", len(bindings))
	var bindingsIDs []int32
	for _, binding := range bindings {
		require.Equal(t, testPoolName, binding.PoolName,
			"expected bound pool to be %s, but was %s", testPoolName, binding.PoolName)
		bindingsIDs = append(bindingsIDs, int32(binding.WorkspaceID))
	}
	require.True(t, bindingsEqual(workspaceIDs, bindingsIDs),
		"workspace IDs %t do not match bound workspace IDs %t", workspaceIDs, bindingsIDs)

	err = OverwriteRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPoolName, existingPools)
	require.NoError(t, err, "failed to overwrite bindings: %t", err)
	bindings, _, err = ReadWorkspacesBoundToRP(ctx, testPoolName, 0, 0)
	require.NoError(t, err, "failed to read bindings: %t", err)
	require.Equal(t, 1, len(bindings),
		"expected bindings length 1, but got length %d", len(bindings))
	require.Equal(t, int(workspaceIDs[0]), bindings[0].WorkspaceID,
		"workspaceID %d does not match bound workspace ID %d",
		workspaceIDs[0], bindings[0].WorkspaceID,
	)
	require.Equal(t, testPoolName, bindings[0].PoolName,
		"expected bound pool to be %s, but was %s", testPoolName, bindings[0].PoolName)
	// Test overwrite pool that's not bound to anything currently
	existingPools = append(existingPools, config.ResourcePoolConfig{PoolName: testPool2Name})
	err = OverwriteRPWorkspaceBindings(ctx, []int32{workspaceIDs[0]}, testPool2Name, existingPools)
	require.NoError(t, err)
	// TODO: call list bindings here to make sure it worked
	bindings, _, err = ReadWorkspacesBoundToRP(ctx, testPool2Name, 0, 0)
	require.NoError(t, err, "failed to read bindings: %t", err)
	require.Equal(t, 1, len(bindings),
		"expected bindings length 1, but got length %d", len(bindings))
	require.Equal(t, int(workspaceIDs[0]), bindings[0].WorkspaceID,
		"workspaceID %d does not match bound workspace ID %d",
		workspaceIDs[0], bindings[0].WorkspaceID,
	)
	require.Equal(t, testPool2Name, bindings[0].PoolName,
		"expected bound pool to be %s, but was %s", testPool2Name, bindings[0].PoolName)

	return
}

func TestOverwriteFail(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	existingPools := []config.ResourcePoolConfig{{PoolName: testPoolName}}
	// Test overwrite adding workspace that doesn't exist
	workspaceIDs := []int32{100, 102, 103}
	err := OverwriteRPWorkspaceBindings(ctx, workspaceIDs, testPoolName, existingPools)
	// db Error that workspace doesn't exist
	require.ErrorContains(t, err, "violates foreign key constraint")
	// Test overwrite pool that doesn't exist
	nonExistentPoolName := "poolNameDoesntExist"
	err = OverwriteRPWorkspaceBindings(ctx, workspaceIDs, nonExistentPoolName, existingPools)
	require.ErrorContains(t, err, "doesn't exist in config")
	return
}

func TestRemoveInvalidBinding(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)
	// remove binding that doesn't exist
	workspaceIDs := []int32{1}
	err := RemoveRPWorkspaceBindings(ctx, workspaceIDs, testPoolName)
	require.ErrorContains(t, err, "binding doesn't exist")
	// bulk remove bindings that don't exist
	workspaceIDs = []int32{1, 2, 3}
	err = RemoveRPWorkspaceBindings(ctx, workspaceIDs, testPoolName)
	require.ErrorContains(t, err, " binding doesn't exist")
	return
}
