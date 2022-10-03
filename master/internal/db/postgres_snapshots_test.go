//go:build integration
// +build integration

package db

import (
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/searcher"
)

func TestCustomSearcherSnapshot(t *testing.T) {
	etc.SetRootPath(RootFromDB)
	db := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, db, MigrationsFromDB)
	user := RequireMockUser(t, db)
	exp := RequireMockExperiment(t, db, user)

	config := expconf.SearcherConfig{
		RawCustomConfig: &expconf.CustomConfig{},
	}

	customSearchMethod := searcher.NewSearchMethod(config)
	cSearcher1 := searcher.NewSearcher(3, customSearchMethod, nil)

	// Add initialOperations
	_, err := cSearcher1.InitialOperations()
	require.NoError(t, err)
	// Add trialExitedEarly
	requestID := model.RequestID(uuid.New())
	exitedReason := model.Errored
	_, err = cSearcher1.TrialExitedEarly(requestID, exitedReason)

	// Save the snapshot to database.
	snapshot, err := cSearcher1.Snapshot()
	require.NoError(t, err)
	err = db.SaveSnapshot(exp.ID, 2, snapshot)
	require.NoError(t, err)

	// Retrieve snapshot from database.
	restored_snapshotSearcher1, _, err1 := db.ExperimentSnapshot(exp.ID)
	require.NoError(t, err1)

	// Restore snapshot from custom searcher 1 to custom searcher 2 to
	// verify Restore of customSearchMethod.
	customSearchMethod2 := searcher.NewSearchMethod(config)
	cSearcher2 := searcher.NewSearcher(4, customSearchMethod2, nil)
	err2 := cSearcher2.Restore(restored_snapshotSearcher1)
	require.NoError(t, err2)
	queue1, err := cSearcher1.GetCustomSearcherEventQueue()
	require.NoError(t, err)
	queue2, err := cSearcher2.GetCustomSearcherEventQueue()
	require.NoError(t, err)
	require.Equal(t, queue1.GetEvents(), queue2.GetEvents())
	db.DeleteSnapshotsForExperiment(exp.ID)
	db.DeleteExperiment(exp.ID)
}
