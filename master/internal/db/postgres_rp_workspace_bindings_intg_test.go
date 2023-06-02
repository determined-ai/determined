package db

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddAndRemoveBindings(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)
	// Test single insert/delete
	// Test bulk insert/delete
	err := AddRPWorkspaceBindings(ctx, []int32{1}, "test pool")
	if err != nil {
		t.Errorf("Error when inserting: %t", err)
	}

	var values []RPWorkspaceBinding
	count, err := Bun().NewSelect().Model(&values).ScanAndCount(ctx)
	require.NoError(t, err, "error when scanning DB: %t", err)
	require.Equal(t, 1, count, "expected 1 item in DB, found %d", count)
	require.Equal(t, 1, values[0].WorkspaceID, "expected workspaceID to be 1, but it is %d",
		values[0].WorkspaceID)
	require.Equal(t, "test pool", values[0].PoolName,
		"expected pool name to be 'test pool', but got %s", values[0].PoolName)

	err = RemoveRPWorkspaceBindings(ctx, []int32{1}, "test pool")
	require.NoError(t, err, "error when removing workspace from DB: %t", err)
	values = []RPWorkspaceBinding{}
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
	// Test overwrite bindings
	// Test overwrite pool that's not bound to anything currently
	return
}

func TestOverwriteFail(t *testing.T) {
	// Test overwrite adding workspace that doesn't exist
	// Test overwrite pool that doesn't exist
	return
}

func TestRemoveInvalidBinding(t *testing.T) {
	// remove binding that doesn't exist
	// bulk remove bindings that don't exist
	return
}
