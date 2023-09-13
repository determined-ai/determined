//go:build integration
// +build integration

package streamdata

import (
	// embed is only used in comments.
	_ "embed"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/test/testutils"
)

//go:embed stream_trials.sql
var streamTrialsSQL string

const latestMigration = 20230830174810

// StreamTrialsData holds the migration function and relevant information
type StreamTrialsData struct {
	MustMigrate testutils.MustMigrateFn
	ExpID       int
	JobID       string
	TaskIDs     []string
	TrialIDs    []int
}

// GenerateStreamTrials fills the database with dummy experiment, trials, jobs, and tasks.
func GenerateStreamTrials() StreamTrialsData {
	mustMigrate := func(t *testing.T, pgdb *db.PgDB, migrationsPath string) {
		extra := testutils.MigrationExtra{
			When: latestMigration,
			SQL:  streamTrialsSQL,
		}
		testutils.MustMigrateWithExtras(t, pgdb, migrationsPath, extra)
	}
	return StreamTrialsData{
		MustMigrate: mustMigrate,
		ExpID:       0,
		JobID:       "test_job",
		TaskIDs:     []string{"1.1", "1.2", "1.3"},
		TrialIDs:    []int{1, 2, 3},
	}
}
