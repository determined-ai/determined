//go:build integration
// +build integration

package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	rootFromDB       = "../../static/srv"
	migrationsFromDB = "file://../../static/migrations"
)

// ResolveTestPostgres resolves a connection to a postgres database. To debug tests that use this
// (or otherwise run the tests outside of the Makefile), make sure to set
// DET_INTEGRATION_POSTGRES_URL.
func ResolveTestPostgres() (*PgDB, error) {
	pgDB, err := ConnectPostgres(os.Getenv("DET_INTEGRATION_POSTGRES_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	return pgDB, nil
}

// MustResolveTestPostgres is the same as ResolveTestPostgres but with panics on errors.
func MustResolveTestPostgres(t *testing.T) *PgDB {
	pgDB, err := ResolveTestPostgres()
	require.NoError(t, err, "failed to connect to postgres")
	return pgDB
}

// MustMigrateTestPostgres ensures the integrations DB has migrations applied.
func MustMigrateTestPostgres(t *testing.T, db *PgDB, migrationsPath string) {
	err := db.Migrate(migrationsPath, []string{"up"})
	require.NoError(t, err, "failed to migrate postgres")
}

// PostTestTeardown deletes our bun singleton, which we normally don't allow at all, but which is
// necessary during testing.
func PostTestTeardown() {
	theOneBun = nil
}
