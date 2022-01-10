package db

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// AddJob persists the existence of a job.
func (db *PgDB) AddJob(j *model.Job) error {
	return addJob(db.sql, j)
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
