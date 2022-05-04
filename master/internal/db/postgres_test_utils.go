//go:build integration
// +build integration

package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/model"
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
	err = db.initAuthKeys()
	require.NoError(t, err, "failed to initAuthKeys")
}

// MustSetupTestPostgres returns a ready to use test postgres connection.
func MustSetupTestPostgres(t *testing.T) *PgDB {
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, migrationsFromDB)
	return pgDB
}

func RequireMockTask(t *testing.T, db *PgDB, userID *model.UserID) *model.Task {
	// Add a job.
	jID := model.NewJobID()
	jIn := &model.Job{
		JobID:   jID,
		JobType: model.JobTypeExperiment,
		OwnerID: userID,
		QPos:    decimal.New(0, 0),
	}
	err := db.AddJob(jIn)
	require.NoError(t, err, "failed to add job")

	// Add a task.
	tID := model.NewTaskID()
	tIn := &model.Task{
		TaskID:    tID,
		JobID:     &jID,
		TaskType:  model.TaskTypeTrial,
		StartTime: time.Now().UTC().Truncate(time.Millisecond),
	}
	err = db.AddTask(tIn)
	require.NoError(t, err, "failed to add task")

	return tIn
}
