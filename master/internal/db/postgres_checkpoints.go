package db

import (
	"fmt"

	"github.com/google/uuid"
)

// FilterForRegisteredCheckpoints gets the deleted checkpoints provided in the model registry.
func (db *PgDB) FilterForRegisteredCheckpoints(deleteCheckpoints []uuid.UUID) ([]uuid.UUID, error) {
	var checkpointIDRows []struct {
		ID uuid.UUID
	}

	if err := db.queryRows(`
	SELECT DISTINCT(c.uuid) AS ID FROM checkpoints_view AS c 
	JOIN model_versions AS mv ON mv.checkpoint_uuid = c.uuid
	WHERE c.uuid IN (SELECT UNNEST($1::uuid[])); 
`, &checkpointIDRows, deleteCheckpoints); err != nil {
		return nil, fmt.Errorf(
			"querying for all requested delete checkpoints registered in model registry: %w", err)
	}

	var checkpointIDs []uuid.UUID

	for _, cRow := range checkpointIDRows {
		checkpointIDs = append(checkpointIDs, cRow.ID)
	}

	return checkpointIDs, nil
}

// MarkCheckpointsDeleted updates the provided delete checkpoints to DELETED state.
func (db *PgDB) MarkCheckpointsDeleted(deleteCheckpoints []uuid.UUID) error {
	_, err := db.sql.Exec(`UPDATE raw_checkpoints c
    SET state = 'DELETED'
    WHERE c.uuid IN (SELECT UNNEST($1::uuid[]))`, deleteCheckpoints)
	if err != nil {
		return fmt.Errorf("deleting checkpoints from raw_checkpoints: %w", err)
	}

	_, err = db.sql.Exec(`UPDATE checkpoints_v2 c
    SET state = 'DELETED'
    WHERE c.uuid IN (SELECT UNNEST($1::uuid[]))`, deleteCheckpoints)
	if err != nil {
		return fmt.Errorf("deleting checkpoints from checkpoints_v2: %w", err)
	}

	return nil
}

// ExperimentCheckpointGrouping represents a mapping of checkpoint uuids to experiment id.
type ExperimentCheckpointGrouping struct {
	ExperimentID       int
	CheckpointUUIDSStr string
}

// GroupCheckpointUUIDsByExperimentID creates the mapping of checkpoint uuids to experiment id.
// The checkpount uuids grouped together are comma separated.
func (db *PgDB) GroupCheckpointUUIDsByExperimentID(checkpoints []uuid.UUID) (
	[]*ExperimentCheckpointGrouping, error) {
	var groupeIDcUUIDS []*ExperimentCheckpointGrouping

	rows, err := db.sql.Queryx(
		`SELECT e.id AS ExperimentID, array_agg(c.uuid::text, ',') AS CheckpointUUIDSStr
	FROM experiments e
	JOIN checkpoints_view c ON c.experiment_id = e.id
	WHERE c.uuid IN (SELECT UNNEST($1::uuid[]))
	GROUP BY e.id`, checkpoints)
	if err != nil {
		return nil, fmt.Errorf("grouping checkpoint UUIDs by experiment ids: %w", err)
	}

	for rows.Next() {
		var eIDcUUIDs ExperimentCheckpointGrouping
		err = rows.StructScan(&eIDcUUIDs)
		if err != nil {
			return nil,
				fmt.Errorf(
					"reading rows into a slice of struct that stores checkpoint ids grouped by exp ID:  %w", err)
		}
		groupeIDcUUIDS = append(groupeIDcUUIDS, &eIDcUUIDs)
	}

	return groupeIDcUUIDS, nil
}
