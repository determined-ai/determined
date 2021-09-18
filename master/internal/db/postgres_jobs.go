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
