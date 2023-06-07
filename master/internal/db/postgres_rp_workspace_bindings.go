package db

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// RPWorkspaceBinding is a struct reflecting the db table rp_workspace_bindings.
type RPWorkspaceBinding struct {
	bun.BaseModel `bun:"table:rp_workspace_bindings"`
	WorkspaceID   int    `bun:"workspace_id"`
	PoolName      string `bun:"pool_name"`
	Validity      bool   `bun:"validity"`
}

// AddRPWorkspaceBindings inserts new bindings between workspaceIds and poolName.
func AddRPWorkspaceBindings(ctx context.Context, workspaceIds []int32, poolName string,
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
func RemoveRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string,
) error {
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").Where("workspace_id IN (?)",
		bun.In(workspaceIds)).Where("pool_name = ?", poolName).Exec(ctx)

	return err
}

// ReadWorkspacesBoundToRP get the bindings between workspaceIds and the requested resource pool.
func ReadWorkspacesBoundToRP(
	ctx context.Context, poolName string, offset, limit int32,
) ([]*RPWorkspaceBinding, *apiv1.Pagination, error) {
	var rpWorkspaceBindings []*RPWorkspaceBinding
	query := Bun().NewSelect().Model(&rpWorkspaceBindings).Where("pool_name = ?",
		poolName)

	pagination, err := runPagedBunQuery(ctx, query, int(offset), int(limit))
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return rpWorkspaceBindings, pagination, nil
		}

		return nil, nil, err
	}

	return rpWorkspaceBindings, pagination, nil
}

// TODO find a good house for this function.
func runPagedBunQuery(
	ctx context.Context, query *bun.SelectQuery, offset, limit int,
) (*apiv1.Pagination, error) {
	// Count number of items without any limits or offsets.
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}

	// Calculate end and start indexes.
	startIndex := offset
	if offset > total || offset < -total {
		startIndex = total
	} else if offset < 0 {
		startIndex = total + offset
	}

	endIndex := startIndex + limit
	switch {
	case limit == -2:
		endIndex = startIndex
	case limit == -1:
		endIndex = total
	case limit == 0:
		endIndex = 100 + startIndex
		if total < endIndex {
			endIndex = total
		}
	case startIndex+limit > total:
		endIndex = total
	}

	// Add start and end index to query.
	query.Offset(startIndex)
	query.Limit(endIndex - startIndex)

	// Bun bug treating limit=0 as no limit when it
	// should be the exact opposite of no records returned.
	if endIndex-startIndex != 0 {
		if err = query.Scan(ctx); err != nil {
			return nil, err
		}
	}

	return &apiv1.Pagination{
		Offset:     int32(offset),
		Limit:      int32(limit),
		Total:      int32(total),
		StartIndex: int32(startIndex),
		EndIndex:   int32(endIndex),
	}, nil
}

// OverwriteRPWorkspaceBindings overwrites the bindings between workspaceIds and poolName.
func (db *PgDB) OverwriteRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string,
) error {
	// Remove existing ones with this pool name
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").
		Where("pool_name = ?", poolName).Exec(ctx)
	if err != nil {
		return err
	}

	err = AddRPWorkspaceBindings(ctx, workspaceIds, poolName)
	return err
}
