package db

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// RecordInstanceStats insert a record of instance start time if instance has not been
// started or already ended.
func (db *PgDB) RecordInstanceStats(a *model.InstanceStats) error {
	err := db.namedExecOne(`
INSERT INTO provisioner_instance_stats (resource_pool, instance_id, slots, start_time)
SELECT :resource_pool, :instance_id, :slots, CURRENT_TIMESTAMP
WHERE NOT EXISTS (
	SELECT * FROM provisioner_instance_stats WHERE instance_id = :instance_id AND end_time IS NULL
)
`, a)
	if err != nil {
		return fmt.Errorf("error recording provisioner instance stats: %w", err)
	}
	return nil
}

// EndInstanceStats updates the end time of an instance.
func (db *PgDB) EndInstanceStats(a *model.InstanceStats) error {
	err := db.namedExecOne(`
UPDATE provisioner_instance_stats
SET end_time = (SELECT CURRENT_TIMESTAMP)
WHERE instance_id = :instance_id AND end_time IS NULL
`, a)
	if err != nil {
		return fmt.Errorf("error ending provisioner instance stats: %w", err)
	}
	return nil
}

// EndAllInstanceStats called at master starts, in case master previously crushed
// If master stops, statistics would treat “live” instances as live until master restarts.
func (db *PgDB) EndAllInstanceStats() error {
	_, err := db.sql.Exec(`
UPDATE provisioner_instance_stats SET end_time = greatest(cluster_heartbeat, start_time) 
FROM cluster_id
WHERE end_time IS NULL`)
	if err != nil {
		return fmt.Errorf("error ending all provisioner instance stats: %w", err)
	}

	return nil
}
