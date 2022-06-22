package db

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"
)

// RecordAgentStats insert a record of instance start time if instance has not been
// started or already ended.
func (db *PgDB) RecordAgentStats(a *model.AgentStats) error {
	return db.namedExecOne(`
INSERT INTO agent_stats (resource_pool, agent_id, slots, start_time)
SELECT :resource_pool, :agent_id, :slots, CURRENT_TIMESTAMP
WHERE NOT EXISTS (
	SELECT * FROM agent_stats WHERE agent_id = :agent_id AND end_time IS NULL
)
`, a)
}

// EndAgentStats updates the end time of an instance.
func EndAgentStats(a *model.AgentStats) error {
	res, err := Bun().NewUpdate().Table("agent_stats").Set(
		"end_time = (SELECT CURRENT_TIMESTAMP)").Where(
		"agent_id = ?", a.AgentID).Where("end_time IS NULL").Exec(context.TODO())
	if err != nil {
		return err
	}
	count, err := res.RowsAffected()
	if count != 1 {
		log.Debugf("End agent stats call affects %d row(s), this number is suppose to be 1.", count)
	}
	return err
}

// EndAllAgentStats called at master starts, in case master previously crushed
// If master stops, statistics would treat “live” agents as live until master restarts.
func (db *PgDB) EndAllAgentStats() error {
	_, err := db.sql.Exec(`
UPDATE agent_stats SET end_time = greatest(cluster_heartbeat, start_time) FROM cluster_id
WHERE end_time IS NULL`)
	return err
}
