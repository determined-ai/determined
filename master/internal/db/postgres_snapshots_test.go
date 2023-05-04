//go:build integration
// +build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func TestCustomSearcherSnapshot(t *testing.T) {
	err := etc.SetRootPath(RootFromDB)
	require.NoError(t, err)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	//nolint:exhaustivestruct
	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	// Create a searcher and add some operations to it.
	searcher1 := searcher.NewSearcher(3, searcher.NewSearchMethod(config), nil)
	_, err = searcher1.InitialOperations()
	require.NoError(t, err)
	_, err = searcher1.TrialExitedEarly(model.RequestID(uuid.New()), model.Errored)
	require.NoError(t, err)

	// Save snapshot to database.
	snapshot, err := searcher1.Snapshot()
	require.NoError(t, err)
	err = db.SaveSnapshot(exp.ID, 2, snapshot)
	require.NoError(t, err)

	// Retrieve snapshot from database.
	restoredSnapshot, _, err := db.ExperimentSnapshot(exp.ID)
	require.NoError(t, err)

	// Verify that restoring the snapshot yields a searcher in the same state as before.
	searcher2 := searcher.NewSearcher(4, searcher.NewSearchMethod(config), nil)
	err = searcher2.Restore(restoredSnapshot)
	require.NoError(t, err)
	queue1, err := searcher1.GetCustomSearcherEventQueue()
	require.NoError(t, err)
	queue2, err := searcher2.GetCustomSearcherEventQueue()
	require.NoError(t, err)
	require.Equal(t, queue1.GetEvents(), queue2.GetEvents())

	err = db.DeleteSnapshotsForExperiment(exp.ID)
	require.NoError(t, err)
	ctx := context.Background()
	err = db.DeleteExperiments(ctx, []int{exp.ID})
	require.NoError(t, err)
}
