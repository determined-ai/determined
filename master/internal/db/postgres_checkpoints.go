package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
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

// GetModelIDsAssociatedWithCheckpoint returns the model ids associated with a checkpoint,
// returning nil if error.
func GetModelIDsAssociatedWithCheckpoint(ctx context.Context, ckptUUID uuid.UUID) ([]int32, error) {
	var modelIDs []int32
	if err := Bun().NewRaw(`
	SELECT DISTINCT(model_id) as ID FROM model_versions m INNER JOIN checkpoints_view c
	ON m.checkpoint_uuid = c.uuid WHERE c.uuid = ?`,
		ckptUUID.String()).Scan(ctx, &modelIDs); err != nil {
		return nil, fmt.Errorf("getting model ids associated with checkpoint uuid: %w", err)
	}

	return modelIDs, nil
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
func MarkCheckpointsDeleted(ctx context.Context, deleteCheckpoints []uuid.UUID) error {
	if len(deleteCheckpoints) == 0 {
		return nil
	}

	err := Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model(&model.CheckpointV2{}).
			Set("state = ?", model.DeletedState).
			Where("uuid IN (?)", bun.In(deleteCheckpoints)).
			Exec(ctx); err != nil {
			return fmt.Errorf("deleting checkpoints from checkpoints_v2: %w", err)
		}

		if err := UpdateCheckpointSizeTx(ctx, tx, deleteCheckpoints); err != nil {
			return fmt.Errorf("updating checkpoints size: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error adding checkpoint metadata: %w", err)
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
	defer rows.Close()

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

// UpdateCheckpointSizeTx updates checkpoint size and count to experiment and trial.
func UpdateCheckpointSizeTx(ctx context.Context, idb bun.IDB, checkpoints []uuid.UUID) error {
	if idb == nil {
		idb = Bun()
	}

	var experimentIDs []int
	err := idb.NewRaw(`
UPDATE trials SET checkpoint_size=sub.size, checkpoint_count=sub.count FROM (
	SELECT trial_id, sum(size) as size, sum(count) as count
	FROM (
		WITH trial_ids AS (
			SELECT t.id AS trial_id
			FROM checkpoints_v2 INNER JOIN trials t ON checkpoints_v2.task_id = t.task_id
			WHERE uuid IN (?)
		)
		SELECT t.id AS trial_id,
		COALESCE(SUM(size) FILTER (WHERE checkpoints_v2.state != 'DELETED'), 0) AS size,
		COUNT(*) FILTER (WHERE checkpoints_v2.state != 'DELETED') AS count
		FROM checkpoints_v2 INNER JOIN trials t on checkpoints_v2.task_id = t.task_id
		WHERE t.id IN (SELECT trial_id FROM trial_ids)
		GROUP BY t.id
	) ssub
	GROUP BY trial_id
) sub
WHERE trials.id = sub.trial_id
RETURNING experiment_id`, bun.In(checkpoints), bun.In(checkpoints)).Scan(ctx, &experimentIDs)
	if err != nil {
		return errors.Wrap(err, "errors updating trial checkpoint sizes and counts")
	}
	if len(experimentIDs) == 0 { // Checkpoint potentially to non experiment.
		return nil
	}

	uniqueExpIDs := maps.Keys(set.FromSlice(experimentIDs))
	var res bool // Need this since bun.NewRaw() doesn't have a Exec(ctx) method.
	err = idb.NewRaw(`
UPDATE experiments SET checkpoint_size=sub.size, checkpoint_count=sub.count FROM (
	SELECT experiment_id, SUM(checkpoint_size) AS size, SUM(checkpoint_count) as count FROM trials
	WHERE experiment_id IN (?)
	GROUP BY experiment_id
) sub
WHERE experiments.id = sub.experiment_id
RETURNING true`, bun.In(uniqueExpIDs)).Scan(ctx, &res)
	if err != nil {
		return errors.Wrap(err, "errors updating experiment checkpoint sizes and counts")
	}

	return nil
}
