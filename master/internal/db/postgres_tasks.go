package db

import (
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// initTaskSessions creates a row in the task_sessions table.
func (db *PgDB) initTaskSessions() error {
	_, err := db.sql.Exec("DELETE FROM task_sessions")
	return err
}

// StartTaskSession creates a row in the task_sessions table.
func (db *PgDB) StartTaskSession(taskID string) (string, error) {
	taskSession := &model.TaskSession{
		TaskID: taskID,
	}

	query := "INSERT INTO task_sessions (task_id) VALUES (:task_id) RETURNING id"
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

// TaskSessionByToken returns a task session given an authentication token.
func (db *PgDB) TaskSessionByToken(token string) (*model.TaskSession, error) {
	v2 := paseto.NewV2()

	var session model.TaskSession
	err := v2.Verify(token, db.tokenKeys.PublicKey, &session, nil)
	if err != nil {
		return nil, ErrNotFound
	}

	query := `SELECT * FROM task_sessions WHERE id=$1`
	if err := db.query(query, &session, session.ID); errors.Cause(err) == ErrNotFound {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &session, nil
}

// DeleteTaskSessionByTaskID deletes the task session with the given ID.
func (db *PgDB) DeleteTaskSessionByTaskID(taskID string) error {
	_, err := db.sql.Exec("DELETE FROM task_sessions WHERE task_id=$1", taskID)
	return err
}
