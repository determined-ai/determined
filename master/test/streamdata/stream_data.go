//go:build integration
// +build integration

package streamdata

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/uptrace/bun"

	// embed is only used in comments.
	_ "embed"

	"github.com/determined-ai/determined/master/internal/db"

	"github.com/determined-ai/determined/master/test/migrationutils"
)

const (
	latestMigration = 20230830174810
	testJob         = "test_job"
)

//go:embed stream_trials.sql
var streamTrialsSQL string

// StreamTrialsData holds the migration function and relevant information
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

type trial struct {
	bun.BaseModel `bun:"table:trials"`
	ID            int            `bun:"id,pk"`
	ExperimentID  int            `bun:"experiment_id"`
	HParams       map[string]any `bun:"hparams"`
	State         model.State    `bun:"state"`
	StartTime     time.Time      `bun:"start_time"`
	Seq           int64          `bun:"seq"`
}

func ModTrial(ctx context.Context, trialID, experimentID int, changeStart bool, changeSeq bool, changeState string) error {
	trials, err := queryTrials(ctx)
	if err != nil {
		return err
	}
	startTime := time.Time{}
	seq := int64(0)
	maxSeq := int64(0)
	state := model.State("")
	for _, t := range trials {
		if t.ID == trialID {
			startTime = t.StartTime
			seq = t.Seq
			state = t.State
		}
		if t.Seq > maxSeq {
			maxSeq = t.Seq
		}
	}

	if changeStart {
		startTime = time.Now()
	}
	if changeSeq {
		seq = maxSeq + 1
	}
	if changeState != "" {
		state = model.State(changeState)
	}

	_, err = db.Bun().NewUpdate().Model(&trial{
		ID:           trialID,
		ExperimentID: experimentID,
		HParams:      nil,
		State:        state,
		StartTime:    startTime,
		Seq:          seq,
	}).Where("id = ?", trialID).Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func queryTrials(ctx context.Context) ([]trial, error) {
	var trials []trial
	err := db.Bun().NewSelect().
		Model(&trials).
		Scan(ctx, &trials)
	if err != nil {
		return nil, err
	}
	return trials, nil
}

func AddTrial(ctx context.Context, experimentID int) error {
	var trials []trial
	err := db.Bun().NewSelect().
		Model(&trials).
		Scan(ctx, &trials)
	if err != nil {
		return err
	}

	nextSeq := int64(0)
	numRelevantTrials := 0

	for _, t := range trials {
		if t.Seq > nextSeq {
			nextSeq = t.Seq
		}
		if t.ExperimentID == experimentID {
			numRelevantTrials += 1
		}
	}

	newTaskID := strconv.Itoa(experimentID) + strconv.Itoa(numRelevantTrials+1)
	newJobID := testJob + strconv.Itoa(experimentID)
	_, err = db.Bun().NewInsert().Model(
		&model.Task{
			TaskID:    model.TaskID(newTaskID),
			TaskType:  "TRIAL",
			StartTime: time.Now(),
			JobID:     (*model.JobID)(&newJobID),
		}).Exec(ctx)
	if err != nil {
		return err
	}

	startTime := time.Now()
	// Insert into tasks, trials, and task id trial id
	_, err = db.Bun().NewInsert().Model(
		&trial{
			ID:           int(nextSeq + 1),
			ExperimentID: experimentID,
			HParams:      map[string]any{},
			State:        "ERROR",
			StartTime:    startTime,
			Seq:          nextSeq + 1,
		}).Exec(ctx)

	if err != nil {
		return err
	}

	insertMap := map[string]interface{}{
		"trial_id": nextSeq + 1,
		"task_id":  newTaskID,
	}
	_, err = db.Bun().NewInsert().Model(&insertMap).Table("trial_id_task_id").Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

func AddExperiment(pgDB *db.PgDB) (int, error) {
	type experiment struct {
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

	ctx := context.TODO()
	var ids []int
	err := db.Bun().NewSelect().
		Table("experiments").
		Column("ID").
		Order("id DESC").
		Scan(ctx, &ids)
	if err != nil {
		return 0, err
	}

	newJobID := testJob + strconv.Itoa(ids[0]+1)
	ownerID := model.UserID(1)
	_, err = db.Bun().NewInsert().Model(
		&model.Job{
			JobID:   model.JobID(newJobID),
			JobType: "EXPERIMENT",
			OwnerID: &ownerID,
		}).Exec(ctx)

	if err != nil {
		return 0, err
	}

	_, err = db.Bun().NewInsert().Model(
		&experiment{
			ID:                   ids[0] + 1,
			JobID:                newJobID,
			State:                "ERROR",
			Notes:                "",
			Config:               expconf.LegacyConfig{},
			ModelDefinitionBytes: nil,
			StartTime:            time.Now(),
			OwnerID:              &ownerID,
		}).Exec(ctx)

	if err != nil {
		return 0, err
	}

	return ids[0] + 1, nil
}
