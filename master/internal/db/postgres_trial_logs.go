package db

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"

	"github.com/determined-ai/determined/master/pkg/model"
)

// TrialLogs takes a trial ID and log offset, limit and filters and returns matching trial logs.
func (db *PgDB) TrialLogs(
	trialID, offset, limit int, fs []api.Filter,
) ([]*model.TrialLog, error) {
	params := []interface{}{trialID, offset, limit}
	fragment, params := filtersToSQL(fs, params)
	query := fmt.Sprintf(`
SELECT
    l.id,
    l.trial_id,
    encode(l.message, 'escape') as message,
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
	return b, db.queryRows(query, &b, params...)
}
