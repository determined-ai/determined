package db

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// addJob persists the existence of a task from a tx.
func addJob(tx *sqlx.Tx, t *model.Job) error {
	if _, err := tx.NamedExec(`
INSERT INTO jobs (job_id, job_type)
VALUES (:job_id, :job_type)
`, t); err != nil {
		return errors.Wrap(err, "adding job")
	}
	return nil
}

// updateJob propagates the new queue position to the job
func (db *PgDB) UpdateJob(job *model.Job) error {
	if job.JobID == "" {
		return errors.Errorf("error modifying job with empty id")
	}
	query := `
UPDATE jobs
SET priority_value = :priority_value
WHERE job_id = :job_id`
	return db.namedExecOne(query, job)
}