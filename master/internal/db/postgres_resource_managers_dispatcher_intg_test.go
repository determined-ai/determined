//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func TestDispatchPersistence(t *testing.T) {
	err := etc.SetRootPath(RootFromDB)
	require.NoError(t, err)

	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)

	u := RequireMockUser(t, db)
	tk := RequireMockTask(t, db, &u.ID)
	a := RequireMockAllocation(t, db, tk.TaskID)

	// Hack, to avoid circular imports.
	rID := sproto.ResourcesID(uuid.NewString())
	_, err = db.sql.Exec(`
INSERT INTO allocation_resources (allocation_id, resource_id)
VALUES ($1, $2)
	`, a.AllocationID, rID)
	require.NoError(t, err)

	d := Dispatch{
		DispatchID:       uuid.NewString(),
		ResourceID:       rID,
		AllocationID:     a.AllocationID,
		ImpersonatedUser: uuid.NewString(),
	}
	err = InsertDispatch(context.TODO(), &d)
	require.NoError(t, err)

	ds, _ := ListDispatchesByAllocationID(context.TODO(), d.AllocationID)
	require.Len(t, ds, 1)
	require.Equal(t, &d, ds[0])

	ds, _ = ListAllDispatches(context.TODO())
	require.Len(t, ds, 1)
	require.Equal(t, &d, ds[0])

	byID, err := DispatchByID(context.TODO(), d.DispatchID)
	require.NoError(t, err)
	require.Equal(t, &d, byID)

	count, err := DeleteDispatch(context.TODO(), d.DispatchID)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	ds, _ = ListDispatchesByAllocationID(context.TODO(), d.AllocationID)
	require.Len(t, ds, 0)

	ds, _ = ListAllDispatches(context.TODO())
	require.Len(t, ds, 0)
}
