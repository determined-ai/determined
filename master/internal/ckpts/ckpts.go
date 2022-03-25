package ckpts

import (
	"context"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

func ByUUID(ctx context.Context, uuid string) (model.Checkpoint, error) {
	var out model.Checkpoint
	err := db.Bun().NewSelect().Model(&out).
		Where("uuid = ?", uuid).
		Scan(ctx)
	if err != nil {
		return model.Checkpoint{}, errors.Wrapf(err, "error selecting checkpoint(%v)", uuid)
	}
	return out, err
}

func ByUUIDExpanded(ctx context.Context, uuid string) (model.CheckpointExpanded, error) {
	var out model.CheckpointExpanded
	err := db.Bun().NewSelect().Model(&out).
		Where("uuid = ?", uuid).
		Scan(ctx)
	if err != nil {
		return model.CheckpointExpanded{}, errors.Wrapf(err, "error selecting checkpoint(%v)", uuid)
	}
	return out, err
}

func ByIDExpanded(ctx context.Context, id int) (model.CheckpointExpanded, error) {
	var out model.CheckpointExpanded
	err := db.Bun().NewSelect().Model(&out).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return model.CheckpointExpanded{}, errors.Wrapf(err, "error selecting checkpoint(%v)", id)
	}
	return out, err
}

// if model.State is nil, that indicates the checkpoint does not exist
// returns sql.ErrNoRows when no suitable checkpoint state is found
func State(ctx context.Context, uuid string) (model.State, error) {
	// Use ByUUID and not ByUUIDExpanded to avoid scanning many different tables.
	ckpt, err := ByUUID(ctx, uuid)
	return ckpt.State, err
}

func GetTrialCheckpointsExpanded(
	ctx context.Context,
	trial int,
	validationStates []string,
	orderBy string,
	asc bool,
	limit int,
	offset int,
) ([]model.CheckpointExpanded, int, error) {
	var ckpts []model.CheckpointExpanded
	q := db.Bun().NewSelect().Model(&ckpts)

	// Apply filters
	q = q.Where("trial_id = ?", trial)
	if len(validationStates) > 0 {
		// XXX: rb, were there supposed to be two filters here?  this looks like the wrong filter.
		q = q.Where("checkpoint_state IN (?)", bun.In(validationStates))
	}

	// choose an ordering
	direction := " ASC"
	if !asc {
		direction = " DESC"
	}

	if orderBy != "" {
		q = q.Order(orderBy + direction)
	}

	// secondary/default ordering is based on latest_batch
	// XXX: the get_checkpoints_for_trial also had an ORDER BY clause for report time
	q = q.Order("(metadata->>'latest_batch')::int8" + direction)

	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(
			err, "error counting checkpoints for trial %d from database", trial,
		)
	}

	// apply pagination and get the actual list of checkpoints
	err = q.Limit(limit).Offset(offset).Scan(ctx)
	if err != nil {
		return nil, 0, errors.Wrapf(
			err, "error fetching checkpoints for trial %d from database", trial,
		)
	}

	return ckpts, total, nil
}
