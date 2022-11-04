//go:build integration
// +build integration

package db

import (
	"archive/tar"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/archive"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"

	"github.com/determined-ai/determined/master/pkg/schemas"
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
//nolint: exhaustivestruct
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
	}).(expconf.ExperimentConfigV0)

	exp := model.Experiment{
		JobID:                model.NewJobID(),
		State:                model.ActiveState,
		Config:               cfg,
		ModelDefinitionBytes: ReadTestModelDefiniton(t, DefaultTestSrcPath),
		StartTime:            time.Now().Add(-time.Hour),
		OwnerID:              &user.ID,
		Username:             user.Username,
		ProjectID:            1,
	}
	err := db.AddExperiment(&exp)
	require.NoError(t, err, "failed to add experiment")
	return &exp
}

// ReadTestModelDefiniton reads a test model definition into a []byte.
func ReadTestModelDefiniton(t *testing.T, folderPath string) []byte {
	path, err := filepath.Abs(folderPath)
	require.NoError(t, err)
	files, err := ioutil.ReadDir(path)
	require.NoError(t, err)
	var arcs []archive.Item
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		var bytes []byte
		bytes, err = ioutil.ReadFile(filepath.Join(path, name)) //nolint: gosec
		require.NoError(t, err)
		arcs = append(arcs, archive.UserItem(name, bytes, tar.TypeReg, byte(file.Mode()), 0, 0))
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
		AllocationID: a.AllocationID,
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
