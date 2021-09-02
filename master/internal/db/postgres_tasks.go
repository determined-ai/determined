package db

import (
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// initAllocationSessions creates a row in the allocation_sessions table.
func (db *PgDB) initAllocationSessions() error {
	_, err := db.sql.Exec("DELETE FROM allocation_sessions")
	return err
}

// addTask persists the existence of a task from a tx.
func addTask(tx *sqlx.Tx, t *model.Task) error {
	if _, err := tx.NamedExec(`
INSERT INTO tasks (task_id, task_type, start_time)
VALUES (:task_id, :task_type, :start_time)
`, t); err != nil {
		return errors.Wrap(err, "adding task")
	}
	return nil
}

func completeTask(tx *sqlx.Tx, tID model.TaskID, endTime *time.Time) error {
	if _, err := tx.Exec(`
UPDATE tasks
SET end_time = $2
WHERE task_id = $1
	`, tID, endTime); err != nil {
		return errors.Wrap(err, "completing task")
	}
	return nil
}

// AddAllocation persists the existence of an allocation.
func (db *PgDB) AddAllocation(a *model.Allocation) error {
	return db.namedExecOne(`
INSERT INTO allocations (task_id, allocation_id, resource_pool, start_time)
VALUES (:task_id, :allocation_id, :resource_pool, :start_time)
`, a)
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) CompleteAllocation(a *model.Allocation) error {
	return db.namedExecOne(`
UPDATE allocations
SET end_time = :end_time
WHERE allocation_id = :allocation_id
`, a)
}

// StartAllocationSession creates a row in the allocation_sessions table.
func (db *PgDB) StartAllocationSession(allocationID model.AllocationID) (string, error) {
	taskSession := &model.AllocationSession{
		AllocationID: allocationID,
	}

	query := "INSERT INTO allocation_sessions (allocation_id) VALUES (:allocation_id) RETURNING id"
	if err := db.namedGet(&taskSession.ID, query, *taskSession); err != nil {
		return "", err
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(db.tokenKeys.PrivateKey, taskSession, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate task authentication token")
	}
	return token, nil
}

// AllocationSessionByToken returns a task session given an authentication token.
func (db *PgDB) AllocationSessionByToken(token string) (*model.AllocationSession, error) {
	v2 := paseto.NewV2()

	var session model.AllocationSession
	err := v2.Verify(token, db.tokenKeys.PublicKey, &session, nil)
	if err != nil {
		return nil, ErrNotFound
	}

	query := `SELECT * FROM allocation_sessions WHERE id=$1`
	if err := db.query(query, &session, session.ID); errors.Cause(err) == ErrNotFound {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &session, nil
}

// DeleteAllocationSession deletes the task session with the given AllocationID.
func (db *PgDB) DeleteAllocationSession(allocationID model.AllocationID) error {
	_, err := db.sql.Exec(
		"DELETE FROM allocation_sessions WHERE allocation_id=$1", allocationID)
	return err
}
