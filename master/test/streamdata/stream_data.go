//go:build integration
// +build integration

package streamdata

import (
	// embed is only used in comments.
	_ "embed"
	"sort"
	"strconv"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/stretchr/testify/require"
)

//go:embed stream_trials.sql
var streamTrialsSQL string

const latestMigration = 20230830174810

// ripped from olddata/utils.go
type migrationExtra struct {
	// When in the migration process the SQL should be executed.
	When int64
	// SQL to inject the old data.
	SQL string
}

// MustMigrateFn is ripped from olddata/utils.go
type MustMigrateFn func(t *testing.T, pgdb *db.PgDB, migrationsPath string)

// StreamTrialsData holds the migration function and relevant information
type StreamTrialsData struct {
	MustMigrate MustMigrateFn
	ExpID       int
	JobID       string
	TaskIDs     []string
	TrialIDs    []int
}

// ripped from olddata/utils.go
func mustMigrateWithExtras(
	t *testing.T, pgdb *db.PgDB, migrationsPath string, extras ...migrationExtra,
) {
	// Require extras to be pre-sorted to improve readability of calling code.
	lessFn := func(i, j int) bool {
		return extras[i].When < extras[j].When
	}
	require.True(t, sort.SliceIsSorted(extras, lessFn), "extras slice is not presorted by .When")

	// Run each extra at its approriate time in the overall migration.
	for _, extra := range extras {
		db.MustMigrateTestPostgres(t, pgdb, migrationsPath, "up", strconv.FormatInt(extra.When, 10))
		_ = pgdb.MustExec(t, extra.SQL)
	}

	// Finish the rest of the migrations.
	db.MustMigrateTestPostgres(t, pgdb, migrationsPath, "up")
}

// GenerateStreamTrials fills the database with dummy experiment, trials, jobs, and tasks.
func GenerateStreamTrials() StreamTrialsData {
	mustMigrate := func(t *testing.T, pgdb *db.PgDB, migrationsPath string) {
		extra := migrationExtra{
			When: latestMigration,
			SQL:  streamTrialsSQL,
		}
		mustMigrateWithExtras(t, pgdb, migrationsPath, extra)
	}
	return StreamTrialsData{
		MustMigrate: mustMigrate,
		ExpID:       0,
		JobID:       "test_job",
		TaskIDs:     []string{"1.1", "1.2", "1.3"},
		TrialIDs:    []int{1, 2, 3},
	}
}
