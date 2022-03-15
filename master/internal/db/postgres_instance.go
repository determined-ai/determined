package db

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// AddAllocation persists the existence of an allocation.
func (db *PgDB) AddInstance(a *model.InstanceStats) error {
	return db.namedExecOne(`
INSERT INTO instance_stats (resource_pool, instance_id, slots, start_time)
SELECT :resource_pool, :instance_id, :slots, :start_time
WHERE NOT EXISTS (
	SELECT * FROM agent_stats WHERE instance_id = :instance_id AND end_time IS NULL
)
`, a)
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) RemoveInstance(a *model.InstanceStats) error {
	return db.namedExecOne(`
UPDATE instance_stats
SET end_time = :end_time
WHERE instance_id = :instance_id AND end_time IS NULL
`, a)
}
