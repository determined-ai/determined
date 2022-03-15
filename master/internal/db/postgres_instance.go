package db

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// AddAllocation persists the existence of an allocation.
func (db *PgDB) AddInstance(a *model.RawInstance) error {
	return db.namedExecOne(`
INSERT INTO raw_instance (resource_pool, instance_id, slots, start_time)
SELECT :resource_pool, :instance_id, :slots, :start_time
WHERE NOT EXISTS (
	SELECT * FROM raw_agent WHERE instance_id = :instance_id AND end_time IS NULL
)
`, a)
}

// CompleteAllocation persists the end of an allocation lifetime.
func (db *PgDB) RemoveInstance(a *model.RawInstance) error {
	return db.namedExecOne(`
UPDATE raw_instance
SET end_time = :end_time
WHERE instance_id = :instance_id AND end_time IS NULL
`, a)
}
