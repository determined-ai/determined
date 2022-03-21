package db

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// RecordAgentStats insert a record of instance start time if instance has not been
// started or already ended.
func (db *PgDB) RecordAgentStats(a *model.AgentStats) error {
	return db.namedExecOne(`
INSERT INTO agent_stats (resource_pool, agent_id, slots, start_time)
SELECT :resource_pool, :agent_id, :slots, :start_time
WHERE NOT EXISTS (
	SELECT * FROM agent_stats WHERE agent_id = :agent_id AND end_time IS NULL
)
`, a)
}

// EndAgentStats updates the end time of an instance.
func (db *PgDB) EndAgentStats(a *model.AgentStats) error {
	return db.namedExecOne(`
UPDATE agent_stats
SET end_time = :end_time
WHERE agent_id = :agent_id AND end_time IS NULL
`, a)
}
