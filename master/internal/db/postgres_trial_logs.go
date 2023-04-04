package db

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/pkg/model"
)

var trialLogsFieldMap = map[string]string{
	// Map timestamp to an expression that provides backwards compatibility when timestamp is missing.
	"timestamp": `coalesce(timestamp,
       to_timestamp(
         substring(convert_from(message, 'UTF-8') from
           '\[([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\]'),
         'YYYY-MM-DD hh24:mi:ss'
       )
     )`,
}

type trialLogsFollowState struct {
	// The last ID returned by the query. Historically the trial logs API when streaming
	// repeatedly made a request like SELECT ... FROM trial_logs ... ORDER BY k OFFSET N LIMIT M.
	// Since offset is less than optimal (no filtering is done during the initial
	// index scan), we at least pass Postgres the ID and let it begin after a certain ID rather
	// than offset N into the query.
	id int64
}

// TrialLogs takes a trial ID and log offset, limit and filters and returns matching trial logs.
func (db *PgDB) TrialLogs(
	trialID, limit int, fs []api.Filter, order apiv1.OrderBy, followState interface{},
) ([]*model.TrialLog, interface{}, error) {
	if followState != nil {
		fs = append(fs, api.Filter{
			Field:     "id",
			Operation: api.FilterOperationGreaterThan,
			Values:    []int64{followState.(*trialLogsFollowState).id},
		})
	}

	params := []interface{}{trialID, limit}
	fragment, params := filtersToSQL(fs, params, trialLogsFieldMap)

	query := fmt.Sprintf(`
SELECT
    l.id,
    l.trial_id,
    CASE
      WHEN log IS NOT NULL THEN
        coalesce(to_char(timestamp, '[YYYY-MM-DD"T"HH24:MI:SS"Z"]' ), '[UNKNOWN TIME]')
        || ' '
        || coalesce(substring(container_id, 1, 8), '[UNKNOWN CONTAINER]')
        || coalesce(' [rank=' || (rank_id::text) || ']', '')
        || ' || '
        || coalesce(level || ': ', '')
        || encode(log, 'escape')
      ELSE encode(message, 'escape')
    END AS message,
    l.agent_id,
    l.container_id,
    coalesce(l.timestamp,
      to_timestamp(
        substring(convert_from(message, 'UTF-8') from
          '\[([0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z)\]'),
        'YYYY-MM-DD hh24:mi:ss'
      )
    ) AS timestamp,
    l.level,
    l.stdtype,
    l.source
FROM trial_logs l
WHERE l.trial_id = $1
%s
ORDER BY l.id %s LIMIT $2
`, fragment, OrderByToSQL(order))

	var b []*model.TrialLog
	if err := db.queryRows(query, &b, params...); err != nil {
		return nil, nil, err
	}

	if len(b) > 0 {
		lastLog := b[len(b)-1]
		followState = &trialLogsFollowState{id: int64(*lastLog.ID)}
	}

	return b, followState, nil
}

// DeleteTrialLogs deletes the logs for the given trial IDs.
func (db *PgDB) DeleteTrialLogs(ids []int) error {
	if _, err := db.sql.Exec(`
DELETE FROM trial_logs
WHERE trial_id IN (SELECT unnest($1::int [])::int);
`, ids); err != nil {
		return errors.Wrapf(err, "error deleting trial logs for trials %v", ids)
	}
	return nil
}

// TrialLogsCount returns the number of logs in postgres for the given trial.
func (db *PgDB) TrialLogsCount(trialID int, fs []api.Filter) (int, error) {
	params := []interface{}{trialID}
	fragment, params := filtersToSQL(fs, params, trialLogsFieldMap)
	query := fmt.Sprintf(`
SELECT count(*)
FROM trial_logs
WHERE trial_id = $1
%s
`, fragment)
	var count int
	if err := db.sql.QueryRow(query, params...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// TrialLogsFields returns the unique fields that can be filtered on for the given trial.
func (db *PgDB) TrialLogsFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error) {
	var fields apiv1.TrialLogsFieldsResponse
	err := db.QueryProto("get_trial_log_fields", &fields, trialID)
	return &fields, err
}
