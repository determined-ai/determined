//go:build integration
// +build integration

package olddata

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

// migration001213 is the last migration before runs were started to be added.
const migrationBeforeRuns = 20231031103358

// PreRunTrialsData holds the migration and useful data for pre run trials test.
type PreRunTrialsData struct {
	PreRunTrialsTable []map[string]any
}

// MigrateToPreRunTrialsData sets the database to where trials were migrated.
func MigrateToPreRunTrialsData(t *testing.T, pgdb *db.PgDB, migrationsPath string) PreRunTrialsData {
	addTrialData := migrationExtra{
		When: migration001213,
		SQL:  preRemoveStepsSQL,
	}

	saveTrialDataAsJSONTable := migrationExtra{
		When: migrationBeforeRuns,
		SQL: `CREATE TABLE trial_json_data AS
	SELECT to_jsonb(trials.*) AS trial_data FROM trials ORDER BY id;`,
	}

	mustMigrateWithExtras(t, pgdb, migrationsPath, addTrialData, saveTrialDataAsJSONTable)

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

	return PreRunTrialsData{
		PreRunTrialsTable: trialsJSON,
	}
}
