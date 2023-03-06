//go:build integration
// +build integration

package olddata

import (
	"sort"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

type migrationExtra struct {
	// When in the migration process the SQL should be executed.
	When int64
	// SQL to inject the old data.
	SQL string
}

// A MustMigrateFn in an olddata collection should take an empty database and migrate it fully, with
// any necessary pauses and data injections.
type MustMigrateFn func(t *testing.T, pgdb *db.PgDB, migrationsPath string)

// mustMigrateWithExtras migrates a database, with extra sql statements to inject data at arbitrary
// points in the migration process.
//
// It would be neat if mustMigrateWithExtras were useful directly in tests, and test could pick and
// choose from a wide selection of migrationExtras.  But in practice, creating migrationExtras which
// are disjoint enough to be combined arbitrarily is really hard.  So instead, it is expected that
// mustMigrateWithExtras is only used inside the olddata module to create curated collections of old
// data.
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
