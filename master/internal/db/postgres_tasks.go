package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

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

// queryHandler is an interface for a query handler to use tx/db for same queries.
type queryHandler interface {
	sqlx.Queryer
	sqlx.Execer
	// Unfortunately database/sql doesn't expose an interface for this like sqlx.
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

// TaskByID returns a task by its ID.
func (db *PgDB) TaskByID(taskID model.TaskID) (model.Task, error) {
	task := model.Task{}
	if err := db.sql.QueryRowx(`
SELECT task_id, task_type, start_time, end_time, log_version
FROM tasks
WHERE task_id = $1
`, taskID).StructScan(&task); err != nil {
		return model.Task{}, errors.Wrap(err, "querying task")
	}
	return task, nil
}

// CheckTaskExists checks if the task exists.
func (db *PgDB) CheckTaskExists(id model.TaskID) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  select task_id
  FROM tasks
  WHERE task_id = $1
)`, id).Scan(&exists)
	return exists, err
}

// AddTask persists the existence of a task.
func (db *PgDB) AddTask(t *model.Task) error {
	return addTask(db.sql, t)
}

func addTask(q queryHandler, t *model.Task) error {
	if _, err := q.NamedExec(`
INSERT INTO tasks (task_id, task_type, start_time, job_id, log_version)
VALUES (:task_id, :task_type, :start_time, :job_id, :log_version)
`, t); err != nil {
		return errors.Wrap(err, "adding task")
	}
	return nil
}

// CompleteTask persists the completion of a task.
func (db *PgDB) CompleteTask(tID model.TaskID, endTime time.Time) error {
	return completeTask(db.sql, tID, endTime)
}

func completeTask(ex sqlx.Execer, tID model.TaskID, endTime time.Time) error {
	if _, err := ex.Exec(`
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

// CloseOpenAllocations finds all allocations that were open when the master crashed
// and adds an end time.
func (db *PgDB) CloseOpenAllocations() error {
	if _, err := db.sql.Exec(`
UPDATE allocations
SET end_time = current_timestamp AT TIME ZONE 'UTC'
WHERE end_time IS NULL
`); err != nil {
		return errors.Wrap(err, "closing old allocations")
	}
	return nil
}

var taskLogsFieldMap = map[string]string{}

// TaskLogs takes a task ID and log offset, limit and filters and returns matching logs.
func (db *PgDB) TaskLogs(
	taskID model.TaskID, limit int, fs []api.Filter, order apiv1.OrderBy, followState interface{},
) ([]*model.TaskLog, interface{}, error) {
	var offset int
	if followState == nil {
		offset = 0
	} else {
		offset = followState.(int)
	}

	params := []interface{}{taskID, offset, limit}
	fragment, params := filtersToSQL(fs, params, taskLogsFieldMap)
	query := fmt.Sprintf(`
SELECT
    l.id,
    l.task_id,
    l.allocation_id,
    l.agent_id,
    l.container_id,
    l.timestamp,
    l.level,
    l.stdtype,
    l.source,
	l.log
FROM task_logs l
WHERE l.task_id = $1
%s
ORDER BY timestamp %s OFFSET $2 LIMIT $3
`, fragment, orderByToSQL(order))

	var b []*model.TaskLog
	if err := db.queryRows(query, &b, params...); err != nil {
		return nil, nil, err
	}

	for _, l := range b {
		l.Resolve()
	}

	return b, offset + len(b), nil
}

// AddTaskLogs adds a list of *model.TaskLog objects to the database with automatic IDs.
func (db *PgDB) AddTaskLogs(logs []*model.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}

	var text strings.Builder
	text.WriteString(`
INSERT INTO task_logs
  (task_id, allocation_id, log, agent_id, container_id, rank_id, timestamp, level, stdtype, source)
VALUES
`)

	args := make([]interface{}, 0, len(logs)*10)

	for i, log := range logs {
		if i > 0 {
			text.WriteString(",")
		}
		// TODO(brad): We can do better.
		fmt.Fprintf(&text, " ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*10+1, i*10+2, i*10+3, i*10+4, i*10+5, i*10+6, i*10+7, i*10+8, i*10+9, i*10+10)

		args = append(args, log.TaskID, log.AllocationID, log.Log, log.AgentID, log.ContainerID,
			log.RankID, log.Timestamp, log.Level, log.StdType, log.Source)
	}

	if _, err := db.sql.Exec(text.String(), args...); err != nil {
		return errors.Wrapf(err, "error inserting %d task logs", len(logs))
	}

	return nil
}

// DeleteTaskLogs deletes the logs for the given tasks.
func (db *PgDB) DeleteTaskLogs(ids []model.TaskID) error {
	if _, err := db.sql.Exec(`
DELETE FROM task_logs
WHERE task_id IN (SELECT unnest($1::text [])::text);
`, ids); err != nil {
		return errors.Wrapf(err, "error deleting task logs for task %v", ids)
	}
	return nil
}

// TaskLogsCount returns the number of logs in postgres for the given task.
func (db *PgDB) TaskLogsCount(taskID model.TaskID, fs []api.Filter) (int, error) {
	params := []interface{}{taskID}
	fragment, params := filtersToSQL(fs, params, taskLogsFieldMap)
	query := fmt.Sprintf(`
SELECT count(*)
FROM task_logs
WHERE task_id = $1
%s
`, fragment)
	var count int
	if err := db.sql.QueryRow(query, params...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// TaskLogsFields returns the unique fields that can be filtered on for the given task.
func (db *PgDB) TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error) {
	var fields apiv1.TaskLogsFieldsResponse
	err := db.QueryProto("get_task_logs_fields", &fields, taskID)
	return &fields, err
}
