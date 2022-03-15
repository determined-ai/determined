package db

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// AddAllocation persists the existence of an allocation.
func (db *PgDB) AddAgent(a *model.AgentStats) error {
	return db.namedExecOne(`
INSERT INTO agent_stats (resource_pool, agent_id, slots, start_time)
SELECT :resource_pool, :agent_id, :slots, :start_time
WHERE NOT EXISTS (
	SELECT * FROM agent_stats WHERE agent_id = :agent_id AND end_time IS NULL
)
`, a)
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) RemoveAgent(a *model.AgentStats) error {
	return db.namedExecOne(`
UPDATE agent_stats
SET end_time = :end_time
WHERE agent_id = :agent_id AND end_time IS NULL
`, a)
}
