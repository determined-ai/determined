package checkpoints

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
)

// CheckpointByUUID looks up a checkpoint by UUID, returning nil if none exists.
func CheckpointByUUID(ctx context.Context, id uuid.UUID) (*model.Checkpoint, error) {
	var checkpoint model.Checkpoint

	if err := db.Bun().NewSelect().
		Model(&checkpoint).Where("uuid = ?", id.String()).Scan(ctx); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "error querying for checkpoint (%v)", id.String())
	}
	return &checkpoint, nil
}

// CheckpointByUUIDs looks up a checkpoint by list of UUIDS, returning nil if error.
func CheckpointByUUIDs(ctx context.Context, ckptUUIDs []uuid.UUID) ([]model.Checkpoint, error) {
	var checkpoints []model.Checkpoint

	if err := db.Bun().NewSelect().Model(&checkpoints).
		Where("checkpoint.uuid IN (SELECT UNNEST(?::uuid[]))", pgdialect.Array(ckptUUIDs)).Scan(ctx); err != nil {
		return nil, fmt.Errorf("getting the checkpoints with a uuid in the set of given uuids: %w", err)
	}
	return checkpoints, nil
}

// GetModelIDsAssociatedWithCheckpoint returns the model ids associated with a checkpoint,
// returning nil if error.
func GetModelIDsAssociatedWithCheckpoint(ctx context.Context, ckptUUID uuid.UUID) ([]int32, error) {
	var modelIDs []int32
	if err := db.Bun().NewRaw(`
	SELECT DISTINCT(model_id) as ID FROM model_versions m INNER JOIN checkpoints_view c
	ON m.checkpoint_uuid = c.uuid WHERE c.uuid = ?`,
		ckptUUID.String()).Scan(ctx, &modelIDs); err != nil {
		return nil, fmt.Errorf("getting model ids associated with checkpoint uuid: %w", err)
	}

	return modelIDs, nil
}

// GetRegisteredCheckpoints gets the checkpoints in
// the model registrys from the list of checkpoints provided.
func GetRegisteredCheckpoints(ctx context.Context, checkpoints []uuid.UUID) (map[uuid.UUID]bool, error) {
	var checkpointIDRows []struct {
		ID uuid.UUID
	}

	if err := db.Bun().NewRaw(`
	SELECT DISTINCT(mv.checkpoint_uuid) as ID FROM model_versions AS mv
	WHERE mv.checkpoint_uuid IN (SELECT UNNEST(?::uuid[]));`,
		pgdialect.Array(checkpoints)).Scan(ctx, &checkpointIDRows); err != nil {
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

	err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
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
	ExperimentID       int    `bun:"experimentid"`
	CheckpointUUIDSStr string `bun:"checkpointuuidsstr"`
}

// GroupCheckpointUUIDsByExperimentID creates the mapping of checkpoint uuids to experiment id.
// The checkpount uuids grouped together are comma separated.
func GroupCheckpointUUIDsByExperimentID(ctx context.Context, checkpoints []uuid.UUID) (
	[]*ExperimentCheckpointGrouping, error,
) {
	var groupeIDcUUIDS []*ExperimentCheckpointGrouping

	err := db.Bun().NewSelect().Model(&groupeIDcUUIDS).
		ModelTableExpr("checkpoints_view as c").
		ColumnExpr("c.experiment_id AS ExperimentID").
		ColumnExpr("string_agg(c.uuid::text, ',') AS CheckpointUUIDSStr").
		Where("c.uuid IN (SELECT UNNEST(?::uuid[]))", pgdialect.Array(checkpoints)).
		Group("c.experiment_id").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("grouping checkpoint UUIDs by experiment ids: %w", err)
	}

	return groupeIDcUUIDS, nil
}

// UpdateCheckpointSizeTx updates checkpoint size and count to experiment and trial.
func UpdateCheckpointSizeTx(ctx context.Context, idb bun.IDB, checkpoints []uuid.UUID) error {
	if idb == nil {
		idb = db.Bun()
	}

	var experimentIDs []int
	err := idb.NewRaw(`
UPDATE runs SET checkpoint_size=sub.size, checkpoint_count=sub.count FROM (
	SELECT
		run_id,
		COALESCE(SUM(size) FILTER (WHERE state != 'DELETED'), 0) AS size,
		COUNT(*) FILTER (WHERE state != 'DELETED') AS count
	FROM checkpoints_v2
	JOIN run_checkpoints rc ON rc.checkpoint_id = checkpoints_v2.uuid
	WHERE rc.run_id IN (
		SELECT run_id FROM run_checkpoints WHERE checkpoint_id IN (?)
	)
	GROUP BY run_id
) sub
WHERE runs.id = sub.run_id
RETURNING experiment_id`, bun.In(checkpoints)).Scan(ctx, &experimentIDs)
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
