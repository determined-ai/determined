package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/o1egl/paseto"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// initAllocationSessions purges sessions of all closed allocations.
func initAllocationSessions(ctx context.Context) error {
	subq := Bun().NewSelect().Table("allocations").
		Column("allocation_id").Where("start_time IS NOT NULL AND end_time IS NOT NULL")
	_, err := Bun().NewDelete().Table("allocation_sessions").Where("allocation_id in (?)", subq).Exec(ctx)
	return err
}

// AddTask UPSERT's the existence of a task.
func AddTask(ctx context.Context, t *model.Task) error {
	return AddTaskTx(ctx, Bun(), t)
}

// AddTaskTx UPSERT's the existence of a task in a tx.
func AddTaskTx(ctx context.Context, idb bun.IDB, t *model.Task) error {
	_, err := idb.NewInsert().Model(t).
		Column("task_id", "task_type", "start_time", "job_id", "log_version",
			"config", "forked_from", "parent_id", "task_state", "no_pause").
		On("CONFLICT (task_id) DO UPDATE").
		Set("task_type=EXCLUDED.task_type").
		Set("start_time=EXCLUDED.start_time").
		Set("job_id=EXCLUDED.job_id").
		Set("log_version=EXCLUDED.log_version").
		Set("config=EXCLUDED.config").
		Set("forked_from=EXCLUDED.forked_from").
		Set("parent_id=EXCLUDED.parent_id").
		Set("task_state=EXCLUDED.task_state").
		Set("no_pause=EXCLUDED.no_pause").
		Exec(ctx)
	return MatchSentinelError(err)
}

// TaskByID returns a task by its ID.
func TaskByID(ctx context.Context, tID model.TaskID) (*model.Task, error) {
	var t model.Task
	if err := Bun().NewSelect().Model(&t).Where("task_id = ?", tID).Scan(ctx, &t); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = ErrNotFound
		}
		return nil, fmt.Errorf("querying task ID %s: %w", tID, err)
	}

	return &t, nil
}

// AddNonExperimentTasksContextDirectory adds a context directory for a non experiment task.
func AddNonExperimentTasksContextDirectory(ctx context.Context, tID model.TaskID, bytes []byte) error {
	if bytes == nil {
		bytes = []byte{}
	}

	if _, err := Bun().NewInsert().Model(&model.TaskContextDirectory{
		TaskID:           tID,
		ContextDirectory: bytes,
	}).Exec(ctx); err != nil {
		return fmt.Errorf("persisting context directory files for task %s: %w", tID, err)
	}

	return nil
}

// NonExperimentTasksContextDirectory returns a non experiment's context directory.
func NonExperimentTasksContextDirectory(ctx context.Context, tID model.TaskID) ([]byte, error) {
	res := &model.TaskContextDirectory{}
	if err := Bun().NewSelect().Model(res).Where("task_id = ?", tID).Scan(ctx, res); err != nil {
		return nil, fmt.Errorf("querying task ID %s context directory files: %w", tID, err)
	}

	return res.ContextDirectory, nil
}

// TaskCompleted checks if the end time exists for a task, if so, the task has completed.
func TaskCompleted(ctx context.Context, tID model.TaskID) (bool, error) {
	return Bun().NewSelect().Table("tasks").
		Where("task_id = ?", tID).Where("end_time IS NOT NULL").Exists(ctx)
}

// CompleteTask persists the completion of a task.
func CompleteTask(ctx context.Context, tID model.TaskID, endTime time.Time) error {
	if _, err := Bun().NewUpdate().Table("tasks").Set("end_time = ?", endTime).
		Where("task_id = ?", tID).Exec(ctx); err != nil {
		return fmt.Errorf("completing task: %w", err)
	}
	return nil
}

// CompleteGenericTask persists the completion of a task of type GENERIC.
func CompleteGenericTask(tID model.TaskID, endTime time.Time) error {
	err := CompleteTask(context.Background(), tID, endTime)
	if err != nil {
		return err
	}
	_, err = Bun().
		NewRaw(`UPDATE tasks
				SET task_state = (
	    		CASE WHEN task_state = ? THEN ?::task_state
	    		ELSE ?::task_state END)
				WHERE task_id = ?
	    `, model.TaskStateStoppingCanceled, model.TaskStateCanceled, model.TaskStateCompleted, tID).
		Exec(context.Background())
	if err != nil {
		return errors.Wrap(err, "completing task")
	}
	return nil
}

// KillGenericTask persists the termination of a task of type GENERIC.
func KillGenericTask(tID model.TaskID, endTime time.Time) error {
	err := CompleteTask(context.Background(), tID, endTime)
	if err != nil {
		return err
	}
	_, err = Bun().NewUpdate().
		Table("tasks").
		Set("task_state = ?", model.TaskStateCanceled).
		Where("task_id = ?", tID).
		Exec(context.Background())
	if err != nil {
		return errors.Wrap(err, "killing task")
	}
	return nil
}

// SetPausedState sets given task to a PAUSED state.
func SetPausedState(taskID model.TaskID, endTime time.Time) error {
	_, err := Bun().NewUpdate().
		Table("tasks").
		Set("task_state = ?", model.TaskStatePaused).
		Set("end_time = ?", endTime).
		Where("task_id = ?", taskID).
		Exec(context.Background())
	if err != nil {
		return errors.Wrap(err, "pausing task")
	}
	return nil
}

// IsPaused returns true if given task is in paused/pausing state.
func IsPaused(ctx context.Context, tID model.TaskID) (bool, error) {
	count, err := Bun().NewSelect().Table("tasks").
		Where("task_id = ?", tID).
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.Where("task_state = ?", model.TaskStateStoppingPaused).
				WhereOr("task_state = ?", model.TaskStatePaused)
		}).Count(context.Background())
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SetErrorState sets given task to a ERROR state.
func SetErrorState(taskID model.TaskID, endTime time.Time) error {
	_, err := Bun().NewUpdate().
		Table("tasks").
		Set("task_state = ?", model.TaskStateError).
		Set("end_time = ?", endTime).
		Where("task_id = ?", taskID).
		Exec(context.Background())
	if err != nil {
		return errors.Wrap(err, "setting error task state")
	}
	return nil
}

// AddAllocation upserts the existence of an allocation. Allocation IDs may conflict in the event
// the master restarts and the trial run ID increment is not persisted, but it is the same
// allocation so this is OK.
func AddAllocation(ctx context.Context, a *model.Allocation) error {
	_, err := Bun().NewInsert().Model(a).On("CONFLICT (allocation_id) DO UPDATE").
		Set("task_id=EXCLUDED.task_id, slots=EXCLUDED.slots").
		Set("resource_pool=EXCLUDED.resource_pool,start_time=EXCLUDED.start_time").
		Set("state=EXCLUDED.state, ports=EXCLUDED.ports").Exec(ctx)
	return err
}

// AddAllocationExitStatus adds the allocation exit status to the allocations table.
func AddAllocationExitStatus(ctx context.Context, a *model.Allocation) error {
	if _, err := Bun().NewUpdate().Model(a).
		Column("exit_reason", "exit_error", "status_code").
		Where("allocation_id = ?", a.AllocationID).Exec(ctx); err != nil {
		return fmt.Errorf("adding allocation exit status to db: %w", err)
	}
	return nil
}

// CompleteAllocation persists the end of an allocation lifetime.
func CompleteAllocation(ctx context.Context, a *model.Allocation) error {
	if a.StartTime == nil {
		a.StartTime = a.EndTime
	}

	_, err := Bun().NewUpdate().Model(a).Set("start_time = ?, end_time = ?", a.StartTime, a.EndTime).
		Where("allocation_id = ?", a.AllocationID).Exec(ctx)

	return err
}

// CompleteAllocationTelemetry returns the analytics of an allocation for the telemetry.
func CompleteAllocationTelemetry(ctx context.Context, aID model.AllocationID) ([]byte, error) {
	var res []byte
	err := Bun().NewRaw(`
	SELECT json_build_object(
		'allocation_id', a.allocation_id,
		'job_id', t.job_id,
		'task_type', t.task_type,
		'duration_sec', COALESCE(EXTRACT(EPOCH FROM (a.end_time - a.start_time)), 0)
	)
	FROM allocations as a JOIN tasks as t
	ON a.task_id = t.task_id
	WHERE a.allocation_id = ?`, aID).Scan(ctx, &res)
	return res, err
}

// AllocationByID retrieves an allocation by its ID.
func AllocationByID(ctx context.Context, aID model.AllocationID) (*model.Allocation, error) {
	var a model.Allocation
	if err := Bun().NewSelect().Table("allocations").
		Where("allocation_id = ?", aID).Scan(ctx, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// StartAllocationSession creates a row in the allocation_sessions table.
func StartAllocationSession(
	ctx context.Context,
	allocationID model.AllocationID,
	owner *model.User,
) (string, error) {
	if owner == nil {
		return "", errors.New("owner cannot be nil for allocation session")
	}

	taskSession := &model.AllocationSession{
		AllocationID: allocationID,
		OwnerID:      &owner.ID,
	}

	if _, err := Bun().NewInsert().Model(taskSession).
		Returning("id").Exec(ctx, &taskSession.ID); err != nil {
		return "", err
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(GetTokenKeys().PrivateKey, taskSession, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate task authentication token: %w", err)
	}
	return token, nil
}

// DeleteAllocationSession deletes the task session with the given AllocationID.
func DeleteAllocationSession(ctx context.Context, allocationID model.AllocationID) error {
	_, err := Bun().NewDelete().Table("allocation_sessions").Where("allocation_id = ?", allocationID).Exec(ctx)
	return err
}

// UpdateAllocationState stores the latest task state and readiness.
func UpdateAllocationState(ctx context.Context, a model.Allocation) error {
	_, err := Bun().NewUpdate().Table("allocations").
		Set("state = ?, is_ready = ?", a.State, a.IsReady).
		Where("allocation_id = ?", a.AllocationID).Exec(ctx)

	return err
}

// UpdateAllocationPorts stores the latest task state and readiness.
func UpdateAllocationPorts(ctx context.Context, a model.Allocation) error {
	_, err := Bun().NewUpdate().Table("allocations").
		Set("ports = ?", a.Ports).
		Where("allocation_id = ?", a.AllocationID).
		Exec(ctx)
	return err
}

// UpdateAllocationStartTime stores the latest start time.
func UpdateAllocationStartTime(ctx context.Context, a model.Allocation) error {
	_, err := Bun().NewUpdate().Table("allocations").
		Set("start_time = ?", a.StartTime).Where("allocation_id = ?", a.AllocationID).Exec(ctx)
	return err
}

// UpdateAllocationProxyAddress stores the proxy address.
func UpdateAllocationProxyAddress(ctx context.Context, a model.Allocation) error {
	_, err := Bun().NewUpdate().Table("allocations").Set("proxy_address = ?", a.ProxyAddress).
		Where("allocation_id = ?", a.AllocationID).Exec(ctx)
	return err
}

// CloseOpenAllocations finds all allocations that were open when the master crashed
// and adds an end time.
func CloseOpenAllocations(ctx context.Context, exclude []model.AllocationID) error {
	if _, err := Bun().NewRaw(`UPDATE allocations SET start_time = cluster_heartbeat FROM cluster_id
	WHERE start_time is NULL`).Exec(ctx); err != nil {
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

	if _, err := Bun().NewRaw(` UPDATE allocations 
	SET end_time = greatest(cluster_heartbeat, start_time), state = 'TERMINATED' FROM cluster_id
	WHERE end_time IS NULL AND (? = '' OR allocation_id NOT IN (SELECT unnest(string_to_array(?, ','))))`,
		excludedFilter, excludedFilter).Exec(ctx); err != nil {
		return errors.Wrap(err, "closing old allocations")
	}
	return nil
}

// RecordTaskStats record stats for tasks.
func RecordTaskStats(ctx context.Context, stats *model.TaskStats) error {
	return RecordTaskStatsBun(ctx, stats)
}

// RecordTaskStatsBun record stats for tasks with bun.
func RecordTaskStatsBun(ctx context.Context, stats *model.TaskStats) error {
	_, err := Bun().NewInsert().Model(stats).Exec(context.TODO())
	return err
}

// RecordTaskEndStats record end stats for tasks.
func RecordTaskEndStats(ctx context.Context, stats *model.TaskStats) error {
	return RecordTaskEndStatsBun(ctx, stats)
}

// RecordTaskEndStatsBun record end stats for tasks with bun.
func RecordTaskEndStatsBun(ctx context.Context, stats *model.TaskStats) error {
	query := Bun().NewUpdate().Model(stats).Column("end_time").
		Where("allocation_id = ?", stats.AllocationID).
		Where("event_type = ?", stats.EventType).
		Where("end_time IS NULL")
	if stats.ContainerID == nil {
		// Just doing Where("container_id = ?", stats.ContainerID) in the null case
		// generates WHERE container_id = NULL which doesn't seem to match on null rows.
		// We don't use this case anywhere currently but this feels like an easy bug to write
		// without this.
		query = query.Where("container_id IS NULL")
	} else {
		query = query.Where("container_id = ?", stats.ContainerID)
	}

	if _, err := query.Exec(ctx); err != nil {
		return fmt.Errorf("recording task end stats %+v: %w", stats, err)
	}

	return nil
}

// EndAllTaskStats called at master starts, in case master previously crashed.
func EndAllTaskStats(ctx context.Context) error {
	_, err := Bun().NewRaw(`UPDATE task_stats 
	SET end_time = greatest(cluster_heartbeat, task_stats.start_time) FROM cluster_id, allocations
	WHERE allocations.allocation_id = task_stats.allocation_id AND allocations.end_time IS NOT NULL
	AND task_stats.end_time IS NULL`).Exec(ctx)
	if err != nil {
		return fmt.Errorf("ending all task stats: %w", err)
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

// AddTaskLogs bulk-inserts a list of *model.TaskLog objects to the database with automatic IDs.
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

// TaskLogsFields returns the unique fields that can be filtered on for the given task.
func (db *PgDB) TaskLogsFields(taskID model.TaskID) (*apiv1.TaskLogsFieldsResponse, error) {
	var fields apiv1.TaskLogsFieldsResponse
	err := db.QueryProto("get_task_logs_fields", &fields, taskID)
	return &fields, err
}

// MaxTerminationDelay is the max delay before a consumer can be sure all logs have been recevied.
// For Postgres, we don't need to wait very long at all; this was a hypothetical cap on fluent
// to DB latency prior to fluent's deprecation.
func (db *PgDB) MaxTerminationDelay() time.Duration {
	// TODO: K8s logs can take a bit to get to us, so much so we should investigate.
	return 5 * time.Second
}
