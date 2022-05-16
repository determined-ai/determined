package db

import (
	"fmt"
)

// GetDeleteCheckpointsInModelRegistry gets the deleted checkpoints provided in the model registry.
func (db *PgDB) FilterForRegisteredCheckpoints(deleteCheckpoints []string) ([]string, error) {
	var checkpointIDRows []struct {
		ID string
	}

	if err := db.queryRows(`
	SELECT DISTINCT(c.uuid::text) AS ID FROM checkpoints_view AS c 
	JOIN model_versions AS mv ON mv.checkpoint_uuid = c.uuid
	WHERE c.uuid::text IN (SELECT UNNEST($1::text[])); 
`, &checkpointIDRows, deleteCheckpoints); err != nil {
		return nil, fmt.Errorf(
			"querying for all requested delete checkpoints registered in model registry")
	}

	var checkpointIDs []string

	for _, cRow := range checkpointIDRows {
		checkpointIDs = append(checkpointIDs, cRow.ID)
	}

	return checkpointIDs, nil
}

// UpdateCheckpointsStateToDelete updates the provided delete checkpoints to DELETED state.
func (db *PgDB) MarkCheckpointsDeleted(deleteCheckpoints []string) error {
	_, err := db.sql.Exec(`UPDATE raw_checkpoints c
    SET state = 'DELETED'
    WHERE c.uuid::text IN (SELECT UNNEST($1::text[]))`, deleteCheckpoints)
	if err != nil {
		return fmt.Errorf("deleting checkpoints from raw_checkpoints: %w", err)
	}

	_, err = db.sql.Exec(`UPDATE checkpoints_v2 c
    SET state = 'DELETED'
    WHERE c.uuid::text IN (SELECT UNNEST($1::text[]))`, deleteCheckpoints)
	if err != nil {
		fmt.Errorf("deleting checkpoints from checkpoints_v2: %w", err)
	}

	return nil
}

type ExperimentCheckpointGrouping struct {
	EID    int
	CUUIDS []string
}

func (db *PgDB) GetExpIDsUsingCheckpointUUIDs(checkpoints []string) ([]*ExperimentCheckpointGrouping, error) {

	var groupeIDcUUIDS []*ExperimentCheckpointGrouping

	rows, err := db.sql.Queryx(
		`SELECT e.id AS eID, array_agg(c.uuid::text) AS cUUIDs
	FROM experiments e
	JOIN checkpoints_view c ON c.experiment_id = e.id
	WHERE c.uuid::text IN (SELECT UNNEST($1::text[]))
	GROUP BY e.id`, checkpoints)
	if err != nil {
		return nil, fmt.Errorf("grouping checkpoint UUIDs by experiment ids")
	}

	for rows.Next() {
		var eIDcUUIDs ExperimentCheckpointGrouping
		err = rows.StructScan(eIDcUUIDs)
		if err != nil {
			return nil, fmt.Errorf("reading rows into a slice of struct that stores checkpoint ids grouped by exp ID")
		}
		groupeIDcUUIDS = append(groupeIDcUUIDS, &eIDcUUIDs)
	}

	return groupeIDcUUIDS, nil
}
