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
