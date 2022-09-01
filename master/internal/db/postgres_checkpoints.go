package db

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// CheckpointByUUID looks up a checkpoint by UUID, returning nil if none exists.
func (db *PgDB) CheckpointByUUID(id uuid.UUID) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint
	if err := db.query(`
	SELECT * FROM checkpoints_view c
	WHERE c.uuid = $1`, &checkpoint, id.String()); errors.Cause(err) == ErrNotFound {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v)", id.String())
	}
	return &checkpoint, nil
}

// CheckpointByUUIDs looks up a checkpoint by list of UUIDS, returning nil if error.
func (db *PgDB) CheckpointByUUIDs(ckptUUIDs []uuid.UUID) ([]model.Checkpoint, error) {
	var checkpoints []model.Checkpoint
	if err := db.queryRows(`
	SELECT * FROM checkpoints_view c WHERE c.uuid 
	IN (SELECT UNNEST($1::uuid[]));`, &checkpoints, ckptUUIDs); err != nil {
		return nil, fmt.Errorf("getting the checkpoints with a uuid in the set of given uuids: %w", err)
	}
	return checkpoints, nil
}

// GetRegisteredCheckpoints gets the checkpoints in
// the model registrys from the list of checkpoints provided.
func (db *PgDB) GetRegisteredCheckpoints(checkpoints []uuid.UUID) (map[uuid.UUID]bool, error) {
	var checkpointIDRows []struct {
		ID uuid.UUID
	}

	if err := db.queryRows(`
	SELECT DISTINCT(mv.checkpoint_uuid) as ID FROM model_versions AS mv
	WHERE mv.checkpoint_uuid IN (SELECT UNNEST($1::uuid[]));
`, &checkpointIDRows, checkpoints); err != nil {
		return nil, fmt.Errorf(
			"filtering checkpoint uuids by those registered in the model registry: %w", err)
	}

	checkpointIDs := make(map[uuid.UUID]bool, len(checkpointIDRows))

	for _, cRow := range checkpointIDRows {
		checkpointIDs[cRow.ID] = true
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
	[]*ExperimentCheckpointGrouping, error,
) {
	var groupeIDcUUIDS []*ExperimentCheckpointGrouping

	rows, err := db.sql.Queryx(
		`SELECT c.experiment_id AS ExperimentID, string_agg(c.uuid::text, ',') AS CheckpointUUIDSStr
	FROM checkpoints_view c
	WHERE c.uuid IN (SELECT UNNEST($1::uuid[]))
	GROUP BY c.experiment_id`, checkpoints)
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
