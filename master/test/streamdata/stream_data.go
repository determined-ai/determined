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
	latestMigration = 20230830174810
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

// GenerateStreamTrials fills the database with dummy experiment, trials, jobs, and tasks.
func GenerateStreamTrials() StreamTrialsData {
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
	ID                   int                  `bun:"id, pk"`
	JobID                string               `bun:"job_id"`
	State                string               `bun:"state"`
	Notes                string               `bun:"notes"`
	Config               expconf.LegacyConfig `bun:"config"`
	ModelDefinitionBytes []byte               `bun:"model_definition"`
	StartTime            time.Time            `bun:"start_time"`
	OwnerID              *model.UserID        `bun:"owner_id"`
}

// GetAddExperimentQueries constructs the necessary queries
// to create a new experiment to the db.
func GetAddExperimentQueries(experiment *Experiment) (
	queries []ExecutableQuery,
	experimentID int,
	err error,
) {
	ctx := context.TODO()
	var ids []int
	err = db.Bun().NewSelect().
		Table("experiments").
		Column("ID").
		Order("id DESC").
		Scan(ctx, &ids)
	if err != nil {
		return nil, 0, err
	}

	newJobID := testJob + strconv.Itoa(ids[0]+1)
	ownerID := model.UserID(1)
	queries = append(queries,
		db.Bun().NewInsert().Model(
			&model.Job{
				JobID:   model.JobID(newJobID),
				JobType: "EXPERIMENT",
				OwnerID: &ownerID,
			},
		),
	)

	// we have to control id and jobId generation
	experiment.ID = ids[0] + 1
	experiment.JobID = newJobID

	queries = append(queries, db.Bun().NewInsert().Model(experiment))
	// XXX (eliu): example experiment:
	// Experiment{
	//			ID:                   ids[0] + 1,
	//			JobID:                newJobID,
	//			State:                "ERROR",
	//			Notes:                "",
	//			Config:               expconf.LegacyConfig{},
	//			ModelDefinitionBytes: nil,
	//			StartTime:            time.Now(),
	//			OwnerID:              &ownerID,
	//		}

	return queries, ids[0] + 1, nil
}
