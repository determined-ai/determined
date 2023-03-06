package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

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
	if len(deleteCheckpoints) > 0 {
		if err := UpdateCheckpointSize(deleteCheckpoints); err != nil {
			return fmt.Errorf("updating checkpoints size: %w", err)
		}
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

// UpdateCheckpointSize updates checkpoint size and count to experiment and trial.
func UpdateCheckpointSize(checkpoints []uuid.UUID) error {
	trialID := Bun().NewSelect().Table("checkpoints_view").
		Column("trial_id").
		Where("uuid IN (?)", bun.In(checkpoints)).
		Distinct()

	sizeTuple := Bun().NewSelect().TableExpr("checkpoints_view AS c").
		ColumnExpr("jsonb_each(c.resources) AS size_tuple").
		Column("experiment_id").
		Column("uuid").
		Column("trial_id").
		Where("state != ?", "DELETED").
		Where("c.resources != 'null'::jsonb").
		Where("trial_id IN (?)", trialID)

	sizeAndCount := Bun().NewSelect().With("cp_size_tuple", sizeTuple).With("trial_ids", trialID).
		Table("cp_size_tuple").
		ColumnExpr("coalesce(sum((size_tuple).value::text::bigint), 0) AS size").
		ColumnExpr("count(distinct(uuid)) AS count").
		ColumnExpr("trial_ids.trial_id").
		GroupExpr("trial_ids.trial_id").
		Join("RIGHT JOIN trial_ids ON trial_ids.trial_id = cp_size_tuple.trial_id")

	_, err := Bun().NewUpdate().With("size_and_count", sizeAndCount).
		Table("trials", "size_and_count").
		Set("checkpoint_size = size").
		Set("checkpoint_count = count").
		Where("id IN (?)", trialID).
		Where("trials.id = size_and_count.trial_id").
		Exec(context.Background())
	if err != nil {
		return err
	}

	experimentID := Bun().NewSelect().Table("checkpoints_view").
		Column("experiment_id").
		Where("uuid IN (?)", bun.In(checkpoints)).Distinct()

	sizeAndCount = Bun().NewSelect().Table("trials").
		ColumnExpr("coalesce(sum(checkpoint_size), 0) AS size").
		ColumnExpr("coalesce(sum(checkpoint_count), 0) AS count").
		Column("experiment_id").
		Group("experiment_id").
		Where("experiment_id IN (?)", experimentID)

	_, err = Bun().NewUpdate().With("size_and_count", sizeAndCount).
		Table("experiments", "size_and_count").
		Set("checkpoint_size = size").
		Set("checkpoint_count = count").
		Where("id IN (?)", experimentID).
		Where("experiments.id = experiment_id").
		Exec(context.Background())

	return err
}
