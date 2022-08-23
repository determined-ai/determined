package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/jmoiron/sqlx"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

// initAllocationSessions purges sessions of all closed allocations.
func (db *PgDB) initAllocationSessions() error {
	_, err := db.sql.Exec(`
DELETE FROM allocation_sessions WHERE allocation_id in (
	SELECT allocation_id FROM allocations
	WHERE start_time IS NOT NULL AND end_time IS NOT NULL
)`)
	return err
}

// queryHandler is an interface for a query handler to use tx/db for same queries.
type queryHandler interface {
	sqlx.Queryer
	sqlx.Execer
	// Unfortunately database/sql doesn't expose an interface for this like sqlx.
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

// CheckTaskExists checks if the task exists.
func (db *PgDB) CheckTaskExists(id model.TaskID) (bool, error) {
	var exists bool
	err := db.sql.QueryRow(`
SELECT
EXISTS(
  SELECT task_id
  FROM tasks
  WHERE task_id = $1
)`, id).Scan(&exists)
	return exists, err
}

// AddTask UPSERT's the existence of a task.
func (db *PgDB) AddTask(t *model.Task) error {
	if _, err := db.sql.NamedExec(`
INSERT INTO tasks (task_id, task_type, start_time, job_id, log_version)
VALUES (:task_id, :task_type, :start_time, :job_id, :log_version)
ON CONFLICT (task_id) DO UPDATE SET
task_type=EXCLUDED.task_type, start_time=EXCLUDED.start_time,
job_id=EXCLUDED.job_id, log_version=EXCLUDED.log_version
`, t); err != nil {
		return errors.Wrap(err, "adding task")
	}
	return nil
}

// TaskByID returns a task by its ID.
func (db *PgDB) TaskByID(tID model.TaskID) (*model.Task, error) {
	var t model.Task
	if err := db.query(`
SELECT *
FROM tasks
WHERE task_id = $1
`, &t, tID); err != nil {
		return nil, errors.Wrap(err, "querying task")
	}
	return &t, nil
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

// AddAllocation upserts the existence of an allocation. Allocation IDs may conflict in the event
// the master restarts and the trial run ID increment is not persisted, but it is the same
// allocation so this is OK.
func (db *PgDB) AddAllocation(a *model.Allocation) error {
	return db.namedExecOne(`
INSERT INTO allocations
	(task_id, allocation_id, slots, resource_pool, agent_label, start_time, state)
VALUES
	(:task_id, :allocation_id, :slots, :resource_pool, :agent_label, :start_time, :state)
ON CONFLICT
	(allocation_id)
DO UPDATE SET
	task_id=EXCLUDED.task_id, slots=EXCLUDED.slots, resource_pool=EXCLUDED.resource_pool,
	agent_label=EXCLUDED.agent_label, start_time=EXCLUDED.start_time, state=EXCLUDED.state
`, a)
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) CompleteAllocation(a *model.Allocation) error {
	if a.StartTime == nil {
		a.StartTime = a.EndTime
	}

	_, err := db.sql.Exec(`
UPDATE allocations
SET start_time = $2, end_time = $3
WHERE allocation_id = $1`, a.AllocationID, a.StartTime, a.EndTime)

	return err
}

// CompleteAllocationTelemetry returns the analytics of an allocation for the telemetry.
func (db *PgDB) CompleteAllocationTelemetry(aID model.AllocationID) ([]byte, error) {
	return db.rawQuery(`
SELECT json_build_object(
	'allocation_id', a.allocation_id,
	'job_id', t.job_id,
	'task_type', t.task_type,
    'duration_sec', COALESCE(EXTRACT(EPOCH FROM (a.end_time - a.start_time)), 0)
)
FROM allocations as a JOIN tasks as t
ON a.task_id = t.task_id
WHERE a.allocation_id = $1;
`, aID)
}

// AllocationByID retrieves an allocation by its ID.
func (db *PgDB) AllocationByID(aID model.AllocationID) (*model.Allocation, error) {
	var a model.Allocation
	if err := db.query(`
SELECT *
FROM allocations
WHERE allocation_id = $1
`, &a, aID); err != nil {
		return nil, errors.Wrap(err, "querying allocation")
	}
	return &a, nil
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
		log.WithError(err).Debug("failed to verify allocation_session token")
		return nil, ErrNotFound
	}

	query := `SELECT * FROM allocation_sessions WHERE id=$1`
	if err := db.query(query, &session, session.ID); errors.Cause(err) == ErrNotFound {
		log.WithField("allocation_sessions.id", session.ID).Debug("allocation_session not found")
		return nil, ErrNotFound
	} else if err != nil {
		log.WithError(err).WithField("allocation_sessions.id", session.ID).
			Debug("failed to lookup allocation_session")
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

// UpdateAllocationState stores the latest task state and readiness.
func (db *PgDB) UpdateAllocationState(a model.Allocation) error {
	_, err := db.sql.Exec(`
		UPDATE allocations
		SET state=$2, is_ready=$3
		WHERE allocation_id=$1
	`, a.AllocationID, a.State, a.IsReady)
	return err
}

// UpdateAllocationStartTime stores the latest start time.
func (db *PgDB) UpdateAllocationStartTime(a model.Allocation) error {
	_, err := db.sql.Exec(`
		UPDATE allocations
		SET start_time = $2
		WHERE allocation_id = $1
	`, a.AllocationID, a.StartTime)
	return err
}

// CloseOpenAllocations finds all allocations that were open when the master crashed
// and adds an end time.
func (db *PgDB) CloseOpenAllocations(exclude []model.AllocationID) error {
	if _, err := db.sql.Exec(`
	UPDATE allocations
	SET start_time = cluster_heartbeat FROM cluster_id
	WHERE start_time is NULL`); err != nil {
		return errors.Wrap(err,
			"setting start time to cluster heartbeat when it's assigned to zero value")
	}

	excludedFilter := ""
	if len(exclude) > 0 {
		excludeStr := make([]string, 0, len(exclude))
		for _, v := range exclude {
			excludeStr = append(excludeStr, v.String())
		}

		excludedFilter = strings.Join(excludeStr, ",")
	}

	if _, err := db.sql.Exec(`
	UPDATE allocations
	SET end_time = greatest(cluster_heartbeat, start_time)
	FROM cluster_id
	WHERE end_time IS NULL AND
	($1 = '' OR allocation_id NOT IN (
		SELECT unnest(string_to_array($1, ','))))`, excludedFilter); err != nil {
		return errors.Wrap(err, "closing old allocations")
	}
	return nil
}

// taskLogsFieldMap is used to map fields in filters to expressions. This was used historically
// in trial logs to either read timestamps or regex them out of logs.
var taskLogsFieldMap = map[string]string{}

type taskLogsFollowState struct {
	// The last ID returned by the query. Historically the trial logs API when streaming
	// repeatedly made a request like SELECT ... FROM trial_logs ... ORDER BY k OFFSET N LIMIT M.
	// Since offset is less than optimal (no filtering is done during the initial
	// index scan), we at least pass Postgres the ID and let it begin after a certain ID rather
	// than offset N into the query.
	id int64
}

// TaskLogs takes a task ID and log offset, limit and filters and returns matching logs.
func (db *PgDB) TaskLogs(
	taskID model.TaskID, limit int, fs []api.Filter, order apiv1.OrderBy, followState interface{},
) ([]*model.TaskLog, interface{}, error) {
	if followState != nil {
		fs = append(fs, api.Filter{
			Field:     "id",
			Operation: api.FilterOperationGreaterThan,
			Values:    []int64{followState.(*taskLogsFollowState).id},
		})
	}

	params := []interface{}{taskID, limit}
	fragment, params := filtersToSQL(fs, params, taskLogsFieldMap)
	query := fmt.Sprintf(`
SELECT
    l.id,
    l.task_id,
    l.allocation_id,
    l.agent_id,
    l.container_id,
    l.rank_id,
    l.timestamp,
    l.level,
    l.stdtype,
    l.source,
    l.log
FROM task_logs l
WHERE l.task_id = $1
%s
ORDER BY l.id %s LIMIT $2
`, fragment, OrderByToSQL(order))

	var b []*model.TaskLog
	if err := db.queryRows(query, &b, params...); err != nil {
		return nil, nil, err
	}

	if len(b) > 0 {
		lastLog := b[len(b)-1]
		followState = &taskLogsFollowState{id: int64(*lastLog.ID)}
	}

	return b, followState, nil
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

		args = append(args, log.TaskID, log.AllocationID, []byte(log.Log), log.AgentID, log.ContainerID,
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

// RecordTaskStats record stats for tasks.
func (db *PgDB) RecordTaskStats(stats *model.TaskStats) error {
	return RecordTaskStatsBun(stats)
}

// RecordTaskStatsBun record stats for tasks with bun.
func RecordTaskStatsBun(stats *model.TaskStats) error {
	_, err := Bun().NewInsert().Model(stats).Exec(context.TODO())
	return err
}

// RecordTaskEndStats record end stats for tasks.
func (db *PgDB) RecordTaskEndStats(stats *model.TaskStats) error {
	return RecordTaskEndStatsBun(stats)
}

// RecordTaskEndStatsBun record end stats for tasks with bun.
func RecordTaskEndStatsBun(stats *model.TaskStats) error {
	_, err := Bun().NewUpdate().Model(stats).Column("end_time").Where(
		"allocation_id = ? AND event_type = ? AND end_time IS NULL", stats.AllocationID, stats.EventType,
	).Exec(context.TODO())
	return err
}

// EndAllTaskStats called at master starts, in case master previously crashed.
func (db *PgDB) EndAllTaskStats() error {
	_, err := db.sql.Exec(`
UPDATE task_stats SET end_time = greatest(cluster_heartbeat, task_stats.start_time)
FROM cluster_id, allocations
WHERE allocations.allocation_id = task_stats.allocation_id
AND allocations.end_time IS NOT NULL
AND task_stats.end_time IS NULL`)
	return err
}

// TaskLogsFields returns the unique fields that can be filtered on for the given task.
func (db *PgDB) TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error) {
	var fields apiv1.TaskLogsFieldsResponse
	err := db.QueryProto("get_task_logs_fields", &fields, taskID)
	return &fields, err
}

// MaxTerminationDelay is the max delay before a consumer can be sure all logs have been recevied.
// For Postgres, we don't need to wait very long at all; this is just a hypothetical cap on fluent
// to DB latency.
func (db *PgDB) MaxTerminationDelay() time.Duration {
	// TODO: K8s logs can take a bit to get to us, so much so we should investigate.
	return 5 * time.Second
}
