package db

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddJobTx persists the existence of a job with a transaction.
func AddJobTx(ctx context.Context, idb bun.IDB, j *model.Job) error {
	if idb == nil {
		idb = Bun()
	}

	if _, err := idb.NewInsert().Model(j).Exec(ctx); err != nil {
		return fmt.Errorf("adding job: %w", err)
	}

	return nil
}

// AddJob persists the existence of a job.
func (db *PgDB) AddJob(j *model.Job) error {
	return AddJobTx(context.TODO(), Bun(), j)
}

// JobByID retrieves a job by ID.
func (db *PgDB) JobByID(jID model.JobID) (*model.Job, error) {
	var j model.Job
	if err := db.query(`
SELECT *
FROM jobs
WHERE job_id = $1
`, &j, jID); err != nil {
		return nil, errors.Wrap(err, "querying job")
	}
	return &j, nil
}

// UpdateJobPosition propagates the new queue position to the job.
func (db *PgDB) UpdateJobPosition(jobID model.JobID, position decimal.Decimal) error {
	if jobID.String() == "" {
		return errors.Errorf("error modifying job with empty id")
	}
	_, err := db.sql.Exec(`
UPDATE jobs
SET q_position = $2
WHERE job_id = $1`, jobID, position)
	return err
}
