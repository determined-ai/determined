package db

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// AddInstance insert a record of instance start time if instance has not been
// started or already ended.
func (db *PgDB) AddInstance(a *model.InstanceStats) error {
	return db.namedExecOne(`
INSERT INTO instance_stats (resource_pool, instance_id, slots, start_time)
SELECT :resource_pool, :instance_id, :slots, :start_time
WHERE NOT EXISTS (
	SELECT * FROM agent_stats WHERE instance_id = :instance_id AND end_time IS NULL
)
`, a)
}

// RemoveInstance updates the end time of an instance.
func (db *PgDB) RemoveInstance(a *model.InstanceStats) error {
	return db.namedExecOne(`
UPDATE instance_stats
SET end_time = :end_time
WHERE instance_id = :instance_id AND end_time IS NULL
`, a)
}
