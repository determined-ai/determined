package db

import (
	"context"

	"github.com/uptrace/bun"
)

// RemoveRPWorkspaceBindings removes the bindings between workspaceIds and poolName.
func (db *PgDB) RemoveRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string) error {
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").Where("workspace_id IN (?)",
		bun.In(workspaceIds)).Where("pool_name = ?", poolName).Exec(ctx)

	return err
}
