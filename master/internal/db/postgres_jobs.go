package db

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddJob persists the existence of a job.
func (db *PgDB) AddJob(j *model.Job) error {
	return addJob(db.sql, j)
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
