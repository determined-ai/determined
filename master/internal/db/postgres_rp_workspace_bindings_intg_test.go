//go:build integration
// +build integration

package db

import (
    "context"
    "require"
    "testing"

    "github.com/determined-ai/determined/master/internal/config"
    "github.com/determined-ai/determined/master/pkg/etc"
)

func masterConfig() {
    config := config.GetMasterConfig()
    config.
}
func TestAddAndRemoveBindings(t *testing.T) {
    // Test single insert/delete
    // Test bulk insert/delete
    return
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

func TestListRPsBoundToWorkspace(t *testing.T) {
    // pretty straightforward
    // don't list binding that are invalid
    // return unbound pools too (we pull from config)
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
    ctx := context.TODO()
    user := RequireMockUser(t, db)

    // Test overwrite bindings
    poolName := "poolName1"
    workspaceNames := []string{"test1", "test2", "test3"}
    workspaceIDs, err := MockWorkspaces(workspaceNames, user.ID)
    require.NoError(err)
    AddRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    // call list bindings here to make sure it worked
    // Test overwrite pool that's not bound to anything currently
    poolName = "poolName2"
    OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    // call list bindings here to make sure it worked
    CleanupMockWorkspace(workspaceIds)
    return
}

func TestOverwriteFail(t *testing.T) {
    ctx := context.TODO()
    // Test overwrite adding workspace that doesn't exist
    workspaceIDs := []int32{4, 5, 6}
    poolName := "poolName"
    err := OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    require.Error(t, err)
    // Test overwrite pool that doesn't exist
    err = OverwriteRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    require.Error(t, err)
    return
}

func TestRemoveInvalidBinding(t *testing.T) {
    // remove binding that doesn't exist
    poolName := "poolName"
    workspaceIDs := []int32{1}
    err := RemoveRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    require.Error(t, err)
    // bulk remove bindings that don't exist
    poolName := "poolName"
    workspaceIDS := []int32{1, 2, 3}
    err := RemoveRPWorkspaceBindings(ctx, workspaceIDs, poolName)
    require.Error(t, err)
    return
}



