//go:build integration
// +build integration

package run

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/test/olddata"
)

func TestMigrateTrials(t *testing.T) {
	ctx := context.Background()
	require.NoError(t, etc.SetRootPath("../../static/srv"))

	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()

	preTrialData := olddata.MigrateToPreRunTrialsData(t, pgDB, "file://../../static/migrations")

	t.Run("trialView", func(t *testing.T) {
		var currentTrialsViewData []struct {
			TrialData map[string]any
		}
		// get all trial info, excluding additional fields added after the transition to runs.
		require.NoError(t, db.Bun().NewSelect().Table("trials").
			ColumnExpr("to_jsonb(trials.*)-'metadata'-'log_signal' AS trial_data").
			Order("id").
			Scan(ctx, &currentTrialsViewData),
		)

		var actual []map[string]any
		for _, t := range currentTrialsViewData {
			actual = append(actual, t.TrialData)
		}

		require.Equal(t, preTrialData.PreRunTrialsTable, actual)
	})

	t.Run("checkpointsView", func(t *testing.T) {
		var currentCheckpointViewData []struct {
			CheckpointData map[string]any
		}
		require.NoError(t, db.Bun().NewSelect().Table("checkpoints_view").
			ColumnExpr("to_jsonb(checkpoints_view.*) AS checkpoint_data").
			Order("id").
			Scan(ctx, &currentCheckpointViewData),
		)

		var actual []map[string]any
		for _, t := range currentCheckpointViewData {
			actual = append(actual, t.CheckpointData)
		}

		require.Equal(t, preTrialData.PreRunCheckpointsView, actual)
	})
}
