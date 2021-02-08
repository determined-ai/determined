package db

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialLogs takes a trial ID and log offset, limit and filters and returns matching trial logs.
func (db *PgDB) TrialLogs(
	trialID, offset, limit int, fs []api.Filter, order apiv1.OrderBy, _ interface{},
) ([]*model.TrialLog, interface{}, error) {
	params := []interface{}{trialID, offset, limit}
	fragment, params := filtersToSQL(fs, params)
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
ORDER BY timestamp %s OFFSET $2 LIMIT $3
`, fragment, orderByToSQL(order))

	var b []*model.TrialLog
	return b, nil, db.queryRows(query, &b, params...)
}

// AddTrialLogs adds a list of *model.TrialLog objects to the database with automatic IDs.
func (db *PgDB) AddTrialLogs(logs []*model.TrialLog) error {
	if len(logs) == 0 {
		return nil
	}

	var text strings.Builder
	text.WriteString(`
INSERT INTO trial_logs
  (trial_id, message, log, agent_id, container_id, rank_id, timestamp, level, stdtype, source)
 VALUES
`)

	args := make([]interface{}, 0, len(logs)*10)

	for i, log := range logs {
		if i > 0 {
			text.WriteString(",")
		}
		fmt.Fprintf(&text, " ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*10+1, i*10+2, i*10+3, i*10+4, i*10+5, i*10+6, i*10+7, i*10+8, i*10+9, i*10+10)

		var l *model.RawString
		if log.Log != nil {
			r := model.RawString(*log.Log)
			l = &r
		}

		args = append(args, log.TrialID, log.Message, l, log.AgentID, log.ContainerID, log.RankID,
			log.Timestamp, log.Level, log.StdType, log.Source)
	}

	if _, err := db.sql.Exec(text.String(), args...); err != nil {
		return errors.Wrapf(err, "error inserting %d trial logs", len(logs))
	}

	return nil
}

// TrialLogCount returns the number of logs in postgres for the given trial.
func (db *PgDB) TrialLogCount(trialID int, fs []api.Filter) (int, error) {
	params := []interface{}{trialID}
	fragment, params := filtersToSQL(fs, params)
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

// TrialLogFields returns the unique fields that can be filtered on for the given trial.
func (db *PgDB) TrialLogFields(trialID int) (*apiv1.TrialLogsFieldsResponse, error) {
	var fields apiv1.TrialLogsFieldsResponse
	err := db.QueryProto("get_trial_log_fields", &fields, trialID)
	return &fields, err
}
