//go:build integration
// +build integration

package db

import (
	"archive/tar"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	// RootFromDB returns the relative path from db to root.
	RootFromDB = "../../static/srv"
	// MigrationsFromDB returns the relative path to migrations folder.
	MigrationsFromDB      = "file://../../static/migrations"
	defaultSearcherMetric = "okness"
	// DefaultTestSrcPath returns src to the mnsit_pytorch model example.
	DefaultTestSrcPath = "../../../examples/tutorials/mnist_pytorch"
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

// ResolveNewPostgresDatabase returns a connection to a randomly-named, newly-created database, and
// a function you should defer for deleting it afterwards.
func ResolveNewPostgresDatabase() (*PgDB, func(), error) {
	baseURL := os.Getenv("DET_INTEGRATION_POSTGRES_URL")
	if baseURL == "" {
		return nil, nil, errors.New("no DET_INTEGRATION_POSTGRES_URL detected")
	}

	url, err := url.Parse(baseURL)
	if err != nil {
		return nil, nil, errors.Wrapf(
			err, "failed to parse DET_INTEGRATION_POSTGRES_URL (%q):", baseURL,
		)
	}

	// Connect to the db server without selecting a database.
	url.Path = ""
	sql, err := sqlx.Connect("pgx", url.String())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to connect to postgres at %q", url)
	}

	randomSuffix := make([]byte, 16)
	_, err = rand.Read(randomSuffix)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed pick a random name")
	}

	dbname := fmt.Sprintf("intg-%x", randomSuffix)
	_, err = sql.Exec(fmt.Sprintf("CREATE DATABASE %q", dbname))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to create new database %q", dbname)
	}

	// Remember the connection we return to the newly-created database, because if we don't close it
	// we can't drop the database.  When we require postgres>=13, we can use DROP DATABASE ... FORCE
	// instead of manually closing this connection.
	var dbConn *sqlx.DB

	cleanup := func() {
		if dbConn != nil {
			if err := dbConn.Close(); err != nil {
				log.WithError(err).Errorf("failed to close sql")
			}
		}
		if _, err := sql.Exec(fmt.Sprintf("DROP DATABASE %q", dbname)); err != nil {
			log.WithError(err).Errorf("failed to delete temp database %q", dbname)
		}
	}

	success := false

	defer func() {
		if !success {
			cleanup()
		}
	}()

	// Connect to the new database.
	url.Path = fmt.Sprintf("/%v", dbname)
	pgDB, err := ConnectPostgres(url.String())
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to connect to new database %q", dbname)
	}

	dbConn = pgDB.sql

	success = true
	return pgDB, cleanup, nil
}

// MustResolveNewPostgresDatabase is the same as ResolveNewPostgresDatabase but panics on errors.
func MustResolveNewPostgresDatabase(t *testing.T) (*PgDB, func()) {
	pgDB, cleanup, err := ResolveNewPostgresDatabase()
	require.NoError(t, err, "failed to create new database")
	return pgDB, cleanup
}

// MustMigrateTestPostgres ensures the integrations DB has migrations applied.
func MustMigrateTestPostgres(t *testing.T, db *PgDB, migrationsPath string, actions ...string) {
	if len(actions) == 0 {
		actions = []string{"up"}
	}
	err := db.Migrate(migrationsPath, actions)
	require.NoError(t, err, "failed to migrate postgres")
	err = db.initAuthKeys()
	require.NoError(t, err, "failed to initAuthKeys")
}

// MustSetupTestPostgres returns a ready to use test postgres connection.
func MustSetupTestPostgres(t *testing.T) *PgDB {
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)
	return pgDB
}

// RequireMockTask returns a mock task.
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

// RequireMockUser requires a mock model.
func RequireMockUser(t *testing.T, db *PgDB) model.User {
	user := model.User{
		Username:     uuid.NewString(),
		PasswordHash: null.NewString("", false),
		Active:       true,
	}
	_, err := db.AddUser(&user, nil)
	require.NoError(t, err, "failed to add user")
	return user
}

// RequireMockExperiment returns a mock experiment.
// nolint: exhaustivestruct
func RequireMockExperiment(t *testing.T, db *PgDB, user model.User) *model.Experiment {
	cfg := schemas.WithDefaults(expconf.ExperimentConfigV0{
		RawCheckpointStorage: &expconf.CheckpointStorageConfigV0{
			RawSharedFSConfig: &expconf.SharedFSConfigV0{
				RawHostPath: ptrs.Ptr("/home/ckpts"),
			},
		},
		RawEntrypoint: &expconf.EntrypointV0{
			RawEntrypoint: ptrs.Ptr("model.Classifier"),
		},
		RawHyperparameters: map[string]expconf.HyperparameterV0{
			"global_batch_size": {
				RawConstHyperparameter: &expconf.ConstHyperparameterV0{
					RawVal: ptrs.Ptr(1),
				},
			},
		},
		RawSearcher: &expconf.SearcherConfigV0{
			RawSingleConfig: &expconf.SingleConfigV0{
				RawMaxLength: &expconf.LengthV0{
					Unit:  expconf.Batches,
					Units: 1,
				},
			},
			RawMetric: ptrs.Ptr(defaultSearcherMetric),
		},
	})

	exp := model.Experiment{
		JobID:                model.NewJobID(),
		State:                model.ActiveState,
		Config:               cfg.AsLegacy(),
		ModelDefinitionBytes: ReadTestModelDefiniton(t, DefaultTestSrcPath),
		StartTime:            time.Now().Add(-time.Hour),
		OwnerID:              &user.ID,
		Username:             user.Username,
		ProjectID:            1,
	}
	err := db.AddExperiment(&exp, cfg)
	require.NoError(t, err, "failed to add experiment")
	return &exp
}

// ReadTestModelDefiniton reads a test model definition into a []byte.
func ReadTestModelDefiniton(t *testing.T, folderPath string) []byte {
	path, err := filepath.Abs(folderPath)
	require.NoError(t, err)
	files, err := os.ReadDir(path)
	require.NoError(t, err)
	var arcs []archive.Item
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		var bytes []byte
		bytes, err = os.ReadFile(filepath.Join(path, name)) //nolint: gosec
		require.NoError(t, err)
		info, err := file.Info()
		require.NoError(t, err)
		arcs = append(arcs, archive.UserItem(name, bytes, tar.TypeReg, byte(info.Mode()), 0, 0))
	}
	targz, err := archive.ToTarGz(archive.Archive(arcs))
	require.NoError(t, err)
	return targz
}

// RequireMockTrial returns a mock trial.
func RequireMockTrial(t *testing.T, db *PgDB, exp *model.Experiment) *model.Trial {
	task := RequireMockTask(t, db, exp.OwnerID)
	rqID := model.NewRequestID(rand.Reader)
	tr := model.Trial{
		TaskID:       task.TaskID,
		RequestID:    &rqID,
		ExperimentID: exp.ID,
		State:        model.ActiveState,
		StartTime:    time.Now(),
		HParams:      model.JSONObj{"global_batch_size": 1},
		JobID:        exp.JobID,
	}
	err := db.AddTrial(&tr)
	require.NoError(t, err, "failed to add trial")
	return &tr
}

// RequireMockAllocation returns a mock allocation.
func RequireMockAllocation(t *testing.T, db *PgDB, tID model.TaskID) *model.Allocation {
	a := model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-1", tID)),
		TaskID:       tID,
		StartTime:    ptrs.Ptr(time.Now().UTC()),
		State:        ptrs.Ptr(model.AllocationStateTerminated),
	}
	err := db.AddAllocation(&a)
	require.NoError(t, err, "failed to add allocation")
	return &a
}

// MockModelCheckpoint returns a mock model checkpoint.
func MockModelCheckpoint(
	ckptUUID uuid.UUID, tr *model.Trial, a *model.Allocation,
) model.CheckpointV2 {
	stepsCompleted := int32(10)
	ckpt := model.CheckpointV2{
		UUID:         ckptUUID,
		TaskID:       tr.TaskID,
		AllocationID: &a.AllocationID,
		ReportTime:   time.Now().UTC(),
		State:        model.CompletedState,
		Resources: map[string]int64{
			"ok": 1.0,
		},
		Metadata: map[string]interface{}{
			"framework":          "some framework",
			"determined_version": "1.0.0",
			"steps_completed":    float64(stepsCompleted),
		},
	}

	return ckpt
}

// MustExec allows integration tests to run raw queries directly against a PgDB.
func (db *PgDB) MustExec(t *testing.T, sql string, args ...any) sql.Result {
	out, err := db.sql.Exec(sql, args...)
	require.NoError(t, err, "failed to run query")
	return out
}

// MockWorkspaces creates as many new workspaces as in workspaceNames and
// returns a list of workspaceIDs.
func MockWorkspaces(workspaceNames []string, userID model.UserID) ([]int, error) {
	ctx := context.Background()
	var workspaceIDs []int
	var workspaces []model.Workspace

	for _, workspaceName := range workspaceNames {
		workspaces = append(workspaces, model.Workspace{
			Name:   workspaceName,
			UserID: userID,
		})
	}

	_, err := Bun().NewInsert().Model(&workspaces).Exec(ctx)
	if err != nil {
		return nil, err
	}

	workspaces = []model.Workspace{}
	err = Bun().NewSelect().Model(&workspaces).
		Where("name IN (?)", bun.In(workspaceNames)).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	for _, workspace := range workspaces {
		workspaceIDs = append(workspaceIDs, workspace.ID)
	}

	return workspaceIDs, nil
}

// CleanupMockWorkspace removes the specified workspaceIDs from the workspaces table.
func CleanupMockWorkspace(workspaceIDs []int) error {
	var workspaces []model.Workspace
	_, err := Bun().NewDelete().Model(&workspaces).
		Where("id IN (?)", bun.In(workspaceIDs)).
		Exec(context.Background())

	return err
}
