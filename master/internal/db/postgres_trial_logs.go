package db

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialLogs takes a trial ID and log offset, limit and filters and returns matching trial logs.
func (db *PgDB) TrialLogs(
	trialID, offset, limit int, fs []api.Filter, _ interface{},
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
    l.timestamp,
    l.level,
    l.stdtype,
    l.source
FROM trial_logs l
WHERE l.trial_id = $1
%s
ORDER BY l.id ASC OFFSET $2 LIMIT $3
`, fragment)

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

// TrialLogCount returns the number of logs in postgres for the given trial. This shouldn't be called,
// instead the master's TrialLogBackend should be called which may call this.
func (db *PgDB) TrialLogCount(trialID int) (int, error) {
	trialStatus := struct {
		State   model.State
		NumLogs int
	}{}
	err := db.Query("trial_status", &trialStatus, trialID)
	if err != nil {
		return 0, err
	}
	return trialStatus.NumLogs, err
}
