package db

import (
	"context"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddJob persists the existence of a job.
func (db *PgDB) AddJob(j *model.Job) error {
	return addJob(db.sql, j)
}

// JobByID retrieves a job by ID.
func (db *PgDB) JobByID(ctx context.Context, jID model.JobID) (*model.Job, error) {
	var j model.Job
	return &j, db.bun.NewSelect().
		Model(&j).
		Where("job_id = ?", jID).
		Scan(ctx)
}

// addJob persists the existence of a job from a tx.
func addJob(tx queryHandler, j *model.Job) error {
	if _, err := tx.NamedExec(`
INSERT INTO jobs (job_id, job_type, owner_id)
VALUES (:job_id, :job_type, :owner_id)
`, j); err != nil {
		return errors.Wrap(err, "adding job")
	}
	return nil
}
