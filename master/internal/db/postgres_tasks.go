package db

import (
	"time"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// initAllocationSessions creates a row in the allocation_sessions table.
func (db *PgDB) initAllocationSessions() error {
	_, err := db.sql.Exec("DELETE FROM allocation_sessions")
	return err
}

// AddAllocation persists a the existence of an allocation.
func (db *PgDB) AddAllocation(
	taskID model.TaskID, allocationID model.AllocationID, rp string) error {
	_, err := db.sql.Exec(`
INSERT INTO allocations (task_id, allocation_id, resource_pool, start_time)
VALUES ($1, $2, $3, $4)
`, taskID, allocationID, rp, time.Now().UTC())
	return err
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) CompleteAllocation(allocationID model.AllocationID) error {
	_, err := db.sql.Exec(`
UPDATE allocations
SET end_time = $2
WHERE allocation_id = $1`, allocationID, time.Now().UTC())
	return err
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
