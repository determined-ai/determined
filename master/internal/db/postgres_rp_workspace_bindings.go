package db

import (
	"context"

	"github.com/uptrace/bun"
)

// RPWorkspaceBinding is a struct reflecting the db table rp_workspace_bindings.
type RPWorkspaceBinding struct {
	bun.BaseModel `bun:"table:rp_workspace_bindings"`
	WorkspaceID   int    `bun:"workspace_id"`
	PoolName      string `bun:"pool_name"`
	Validity      bool   `bun:"validity"`
}

// AddRPWorkspaceBindings inserts new bindings between workspaceIds and poolName.
func (db *PgDB) AddRPWorkspaceBindings(ctx context.Context, workspaceIds []int32, poolName string,
) error {
	var bindings []RPWorkspaceBinding
	for _, workspaceID := range workspaceIds {
		bindings = append(bindings, RPWorkspaceBinding{
			WorkspaceID: int(workspaceID),
			PoolName:    poolName,
			Validity:    true,
		})
	}

	_, err := Bun().NewInsert().Model(&bindings).Exec(ctx)
	return err
}

// RemoveRPWorkspaceBindings removes the bindings between workspaceIds and poolName.
func (db *PgDB) RemoveRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string,
) error {
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").Where("workspace_id IN (?)",
		bun.In(workspaceIds)).Where("pool_name = ?", poolName).Exec(ctx)

	return err
}

// OverwriteRPWorkspaceBindings overwrites the bindings between workspaceIds and poolName.
func (db *PgDB) OverwriteRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string) error {
	// Remove existing ones with this pool name
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").
		Where("pool_name = ?", poolName).Exec(ctx)
	if err != nil {
		return err
	}

	err = db.AddRPWorkspaceBindings(ctx, workspaceIds, poolName)
	return err
}
