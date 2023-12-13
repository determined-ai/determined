//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
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
		require.NoError(t, db.Bun().NewSelect().Table("trials").
			ColumnExpr("to_jsonb(trials.*) AS trial_data").
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

	t.Run("experimentsView", func(t *testing.T) {
		var currentExperimentTableData []struct {
			ExperimentData map[string]any
		}
		require.NoError(t, db.Bun().NewSelect().Table("experiments").
			ColumnExpr("to_jsonb(experiments.*) AS experiment_data").
			Order("id").
			Scan(ctx, &currentExperimentTableData),
		)

		var actual []map[string]any
		for _, t := range currentExperimentTableData {
			actual = append(actual, t.ExperimentData)
		}

		require.Equal(t, preTrialData.PreRunExperimentsTable, actual)
	})

	t.Run("run_collection name", func(t *testing.T) {
		var nameData []struct {
			ID                      int
			ExternalRunCollectionID *string
			Name                    string
		}
		require.NoError(t, db.Bun().NewSelect().Table("run_collections").
			Column("id", "external_run_collection_id", "name").
			Order("id").
			Scan(ctx, &nameData),
		)

		seenNoExternalCase := false
		seenExternalCase := false

		for _, r := range nameData {
			var expected string
			if r.ExternalRunCollectionID == nil {
				expected = fmt.Sprintf("experiment_id:%d", r.ID)
				seenExternalCase = true
			} else {
				expected = fmt.Sprintf("experiment_id:%d, external_experiment_id:%s",
					r.ID, *r.ExternalRunCollectionID)
				seenNoExternalCase = true
			}

			require.Equal(t, expected, r.Name)
		}

		// Convince ourselves we are actually testing both cases.
		require.True(t, seenNoExternalCase)
		require.True(t, seenExternalCase)
	})
}
