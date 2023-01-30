package db

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
)

// GetCommandOwnerID gets a command's ownerID from a taskID. Uses persisted command state.
// Returns db.ErrNotFound if a command with given taskID does not exist.
func GetCommandOwnerID(ctx context.Context, taskID model.TaskID) (model.UserID, error) {
	ownerIDBun := &struct {
		bun.BaseModel `bun:"table:command_state"`
		OwnerID       model.UserID `bun:"owner_id"`
	}{}

	if err := Bun().NewSelect().Model(ownerIDBun).
		ColumnExpr("generic_command_spec->'Base'->'Owner'->'id' AS owner_id").
		Where("task_id = ?", taskID).
		Scan(ctx); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, ErrNotFound
		}
		return 0, err
	}

	return ownerIDBun.OwnerID, nil
}

// TaskMetadata captures minimal metadata about a task.
type TaskMetadata struct {
	bun.BaseModel `bun:"table:command_state"`
	WorkspaceID   model.AccessScopeID `bun:"workspace_id"`
	TaskType      model.TaskType      `bun:"task_type"`
	ExperimentIDs []int32             `bun:"experiment_ids"`
	TrialIDs      []int32             `bun:"trial_ids"`
}

// IdentifyTask returns the task metadata for a given task ID.
// Returns db.ErrNotFound if a command with given taskID does not exist.
func IdentifyTask(ctx context.Context, taskID model.TaskID) (TaskMetadata, error) {
	metadata := TaskMetadata{}
	if err := Bun().NewSelect().Model(&metadata).
		ColumnExpr("generic_command_spec->'Metadata'->'workspace_id' AS workspace_id").
		ColumnExpr("generic_command_spec->'TaskType' as task_type").
		ColumnExpr("generic_command_spec->`ExperimentIDs` as experiment_ids").
		ColumnExpr("generic_command_spec->`TrialIDs` as trial_ids").
		Where("task_id = ?", taskID).
		Scan(ctx); err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return metadata, ErrNotFound
		}
		return metadata, err
	}
	return metadata, nil
}
