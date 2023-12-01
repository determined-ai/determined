//go:build integration
// +build integration

package streamdata

import (
	"context"
	"database/sql"

	// embed is only used in comments.
	_ "embed"
	"strconv"
	"testing"
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/test/migrationutils"
)

const (
	latestMigration = 20231126215150
	testJob         = "test_job"
)

//go:embed stream_trials.sql
var streamTrialsSQL string

// StreamTrialsData holds the migration function and relevant information.
type StreamTrialsData struct {
	MustMigrate migrationutils.MustMigrateFn
	ExpID       int
	JobID       string
	TaskIDs     []string
	TrialIDs    []int
}

// GenerateStreamData fills the database with dummy experiment, trials, jobs, and tasks.
func GenerateStreamData() StreamTrialsData {
	mustMigrate := func(t *testing.T, pgdb *db.PgDB, migrationsPath string) {
		extra := migrationutils.MigrationExtra{
			When: latestMigration,
			SQL:  streamTrialsSQL,
		}
		migrationutils.MustMigrateWithExtras(t, pgdb, migrationsPath, extra)
	}
	return StreamTrialsData{
		MustMigrate: mustMigrate,
		ExpID:       1,
		JobID:       "test_job1",
		TaskIDs:     []string{"1.1", "1.2", "1.3"},
		TrialIDs:    []int{1, 2, 3},
	}
}

// Trial contains a subset of actual determined trial fields and is used for testing purposes
// without having to import anything from determined/internal.
type Trial struct {
	bun.BaseModel `bun:"table:trials"`
	ID            int            `bun:"id,pk"`
	ExperimentID  int            `bun:"experiment_id"`
	HParams       map[string]any `bun:"hparams"`
	State         model.State    `bun:"state"`
	StartTime     time.Time      `bun:"start_time"`
	WorkspaceID   int            `bun:"-"`
	Seq           int64          `bun:"seq"`
}

// ExecutableQuery an interface that requires queries of this type to have an exec function.
type ExecutableQuery interface {
	bun.Query
	Exec(ctx context.Context, dest ...interface{}) (sql.Result, error)
}

// GetUpdateTrialQuery constructs a query for modifying rows of a trial.
func GetUpdateTrialQuery(newTrial Trial) ExecutableQuery {
	return db.Bun().NewUpdate().Model(&newTrial).Where("id = ?", newTrial.ID)
}

func queryTrials(ctx context.Context) ([]Trial, error) {
	var trials []Trial
	err := db.Bun().NewSelect().
		Model(&trials).
		Scan(ctx, &trials)
	if err != nil {
		return nil, err
	}
	return trials, nil
}

// GetAddTrialQueries constructs the necessary queries
// to create a new trial under the given experimentID.
func GetAddTrialQueries(ctx context.Context, experimentID int) ([]ExecutableQuery, error) {
	queries := []ExecutableQuery{}
	trials, err := queryTrials(ctx)
	if err != nil {
		return nil, err
	}

	nextSeq := int64(0)
	numRelevantTrials := 0

	for _, t := range trials {
		if t.Seq > nextSeq {
			nextSeq = t.Seq
		}
		if t.ExperimentID == experimentID {
			numRelevantTrials++
		}
	}

	newTaskID := strconv.Itoa(experimentID) + strconv.Itoa(numRelevantTrials+1)
	newJobID := testJob + strconv.Itoa(experimentID)
	queries = append(queries,
		db.Bun().NewInsert().Model(
			&model.Task{
				TaskID:    model.TaskID(newTaskID),
				TaskType:  "TRIAL",
				StartTime: time.Now(),
				JobID:     (*model.JobID)(&newJobID),
			},
		),
	)

	// Insert into tasks, trials, and task id trial id
	startTime := time.Now()
	queries = append(queries,
		db.Bun().NewInsert().Model(
			&Trial{
				ID:           int(nextSeq + 1),
				ExperimentID: experimentID,
				HParams:      map[string]any{},
				State:        "ERROR",
				StartTime:    startTime,
				Seq:          nextSeq + 1,
			},
		),
	)

	insertMap := map[string]interface{}{
		"trial_id": nextSeq + 1,
		"task_id":  newTaskID,
	}
	queries = append(queries, db.Bun().NewInsert().Model(&insertMap).Table("trial_id_task_id"))
	return queries, nil
}

// Experiment contains a subset of actual determined experiment fields and is used to test
// streaming code without importing anything from determined/internal.
type Experiment struct {
	bun.BaseModel        `bun:"table:experiments"`
	ID                   int                  `bun:"id,pk"`
	JobID                string               `bun:"job_id"`
	State                model.State          `bun:"state"`
	Notes                string               `bun:"notes"`
	Config               expconf.LegacyConfig `bun:"config"`
	ModelDefinitionBytes []byte               `bun:"model_definition"`
	StartTime            time.Time            `bun:"start_time"`
	OwnerID              *model.UserID        `bun:"owner_id"`
	ProjectID            int                  `bin:"project_id"`
}

// GetAddExperimentQuery constructs the query to create a new experiment in the db.
func GetAddExperimentQuery(experiment *Experiment) ExecutableQuery {
	return db.Bun().NewInsert().Model(experiment)
}

// GetUpdateExperimentQuery constructs a query to update an experiment in the db.
func GetUpdateExperimentQuery(newExp Experiment) ExecutableQuery {
	return db.Bun().NewUpdate().Model(&newExp).OmitZero().Where("id = ?", newExp.ID)
}

// GetDeleteExperimentQuery constructs a query to delete an experiment with the specified id.
func GetDeleteExperimentQuery(id int) ExecutableQuery {
	return db.Bun().NewDelete().Table("experiments").Where("id = ?", id)
}

// Checkpoint contains a subset of checkpoint_v2 fields and is used to test streaming code
// without importing anything from determined/master/internal.
type Checkpoint struct {
	bun.BaseModel `bun:"table:checkpoints_v2"`
	ID            int         `bun:"id,pk"`
	TaskID        string      `bun:"task_id"`
	State         model.State `bun:"state"`
	ReportTime    time.Time   `bun:"report_time"`
}

// GetUpdateCheckpointQuery constructs a query for updating a checkpoint.
func GetUpdateCheckpointQuery(checkpoint Checkpoint) ExecutableQuery {
	return db.Bun().NewUpdate().Model(&checkpoint).OmitZero().WherePK()
}
