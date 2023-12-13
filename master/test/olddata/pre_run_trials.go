//go:build integration
// +build integration

package olddata

import (
	"context"
	// embed is only used in comments.
	_ "embed"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

// migration001213 is the last migration before runs were started to be added.
const migrationBeforeRuns = 20231031103358

// PreRunTrialsData holds the migration and useful data for pre run trials test.
type PreRunTrialsData struct {
	PreRunTrialsTable      []map[string]any
	PreRunCheckpointsView  []map[string]any
	PreRunExperimentsTable []map[string]any
}

//go:embed modern_experiments_add.sql
var modernExperimentsAdd string

// MigrateToPreRunTrialsData sets the database to where trials were migrated.
func MigrateToPreRunTrialsData(t *testing.T, pgdb *db.PgDB, migrationsPath string) PreRunTrialsData {
	addTrialData := migrationExtra{
		When: migration001213,
		SQL:  preRemoveStepsSQL,
	}

	addMoreModernExperiments := migrationExtra{
		When: migrationBeforeRuns,
		SQL:  modernExperimentsAdd,
	}

	saveTrialDataAsJSONTable := migrationExtra{
		When: migrationBeforeRuns,
		SQL: `CREATE TABLE trial_json_data AS
	SELECT to_jsonb(trials.*) AS trial_data FROM trials ORDER BY id;`,
	}

	saveCheckpointsViewAsJSONTable := migrationExtra{
		When: migrationBeforeRuns,
		SQL: `CREATE TABLE checkpoints_view_json_data AS
	SELECT to_jsonb(checkpoints_view.*) AS checkpoint_data FROM checkpoints_view ORDER BY id;`,
	}

	saveExperimentsDataAsJSONTable := migrationExtra{
		When: migrationBeforeRuns,
		SQL: `CREATE TABLE experiment_json_data AS
	SELECT to_jsonb(experiments.*) AS experiment_data FROM experiments ORDER BY id;`,
	}

	mustMigrateWithExtras(t, pgdb, migrationsPath,
		addTrialData,
		addMoreModernExperiments,
		saveTrialDataAsJSONTable,
		saveCheckpointsViewAsJSONTable,
		saveExperimentsDataAsJSONTable,
	)

	var trialJSONDataRow []struct {
		TrialData map[string]any
	}
	require.NoError(t,
		db.Bun().NewSelect().Table("trial_json_data").Scan(context.TODO(), &trialJSONDataRow),
		"getting trial json data",
	)
	var trialsJSON []map[string]any
	for _, t := range trialJSONDataRow {
		trialsJSON = append(trialsJSON, t.TrialData)
	}

	var checkpointJSONDataRow []struct {
		CheckpointData map[string]any
	}
	require.NoError(t, db.Bun().NewSelect().Table("checkpoints_view_json_data").
		Scan(context.TODO(), &checkpointJSONDataRow),
		"getting checkpoint json data",
	)
	var checkpointsJSON []map[string]any
	for _, t := range checkpointJSONDataRow {
		checkpointsJSON = append(checkpointsJSON, t.CheckpointData)
	}

	var experimentJSONDataRow []struct {
		ExperimentData map[string]any
	}
	require.NoError(t,
		db.Bun().NewSelect().Table("experiment_json_data").Scan(context.TODO(), &experimentJSONDataRow),
		"getting experiment json data",
	)
	var experimentsJSON []map[string]any
	for _, t := range experimentJSONDataRow {
		experimentsJSON = append(experimentsJSON, t.ExperimentData)
	}

	return PreRunTrialsData{
		PreRunTrialsTable:      trialsJSON,
		PreRunCheckpointsView:  checkpointsJSON,
		PreRunExperimentsTable: experimentsJSON,
	}
}
