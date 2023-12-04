//go:build integration
// +build integration

package db

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
}
