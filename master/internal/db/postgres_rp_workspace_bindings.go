package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db/bunutils"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// RPWorkspaceBinding is a struct reflecting the db table rp_workspace_bindings.
type RPWorkspaceBinding struct {
	bun.BaseModel `bun:"table:rp_workspace_bindings"`

	WorkspaceID int    `bun:"workspace_id"`
	PoolName    string `bun:"pool_name"`
	Valid       bool   `bun:"valid"`
}

// AddRPWorkspaceBindings inserts new bindings between workspaceIds and poolName.
func AddRPWorkspaceBindings(ctx context.Context, workspaceIds []int32, poolName string,
	resourcePools []config.ResourcePoolConfig,
) error {
	if len(workspaceIds) == 0 {
		return nil
	}
	// Check if pool exists
	poolExists := false
	for _, pool := range resourcePools {
		if poolName == pool.PoolName {
			poolExists = true
			break
		}
	}

	if !poolExists {
		return errors.Errorf("pool with name %v doesn't exist",
			poolName)
	}

	var bindings []RPWorkspaceBinding
	for _, workspaceID := range workspaceIds {
		bindings = append(bindings, RPWorkspaceBinding{
			WorkspaceID: int(workspaceID),
			PoolName:    poolName,
			Valid:       true,
		})
	}

	_, err := Bun().NewInsert().Model(&bindings).Exec(ctx)
	return err
}

// RemoveRPWorkspaceBindings removes the bindings between workspaceIds and poolName.
func RemoveRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string,
) error {
	if len(workspaceIds) == 0 {
		return nil
	}

	var rpWorkspaceBindings []*RPWorkspaceBinding
	err := Bun().NewSelect().Model(&rpWorkspaceBindings).
		Where("workspace_id IN (?)", bun.In(workspaceIds)).
		Scan(ctx)
	if err != nil {
		return err
	}

	foundWorkspaces := set.New[int]()
	for _, binding := range rpWorkspaceBindings {
		foundWorkspaces.Insert(binding.WorkspaceID)
	}

	for _, workspaceID := range workspaceIds {
		if !foundWorkspaces.Contains(int(workspaceID)) {
			return errors.Errorf(" workspace with id %v and pool with name  %v binding doesn't exist",
				workspaceID, poolName)
		}
	}

	_, err = Bun().NewDelete().Table("rp_workspace_bindings").Where("workspace_id IN (?)",
		bun.In(workspaceIds)).Where("pool_name = ?", poolName).Exec(ctx)
	return err
}

// ReadWorkspacesBoundToRP get the bindings between workspaceIds and the requested resource pool.
func ReadWorkspacesBoundToRP(
	ctx context.Context, poolName string, offset, limit int32,
	resourcePools []config.ResourcePoolConfig,
) ([]*RPWorkspaceBinding, *apiv1.Pagination, error) {
	poolExists := false
	for _, validPool := range resourcePools {
		if validPool.PoolName == poolName {
			poolExists = true
			break
		}
	}
	if !poolExists {
		return nil, nil, errors.Errorf("pool with name %v doesn't exist or is not available", poolName)
	}
	var rpWorkspaceBindings []*RPWorkspaceBinding
	query := Bun().NewSelect().Model(&rpWorkspaceBindings).Where("pool_name = ?",
		poolName)

	query, pagination, err := bunutils.Paginate(ctx, query, int(offset), int(limit))
	if err != nil {
		return nil, nil, err
	}
	// Bun bug treating limit=0 as no limit when it
	// should be the exact opposite of no records returned.
	// TODO: revisit and check this for commonality.
	// We may put pagination.StartIndex-pagination.EndIndex != 0
	// back to the function and return a nil query if StartIndex = EndIndex. This is for
	// limit = -2, we don't run the query, return pagination only.
	if pagination.StartIndex-pagination.EndIndex != 0 {
		if err = query.Scan(ctx); err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return rpWorkspaceBindings, pagination, nil
			}
			return nil, nil, err
		}
	}

	return rpWorkspaceBindings, pagination, nil
}

// OverwriteRPWorkspaceBindings overwrites the bindings between workspaceIds and poolName.
func OverwriteRPWorkspaceBindings(ctx context.Context,
	workspaceIds []int32, poolName string, resourcePools []config.ResourcePoolConfig,
) error {
	// Check if pool exists
	poolExists := false
	for _, pool := range resourcePools {
		if poolName == pool.PoolName {
			poolExists = true
		}
	}

	if !poolExists {
		return errors.Errorf("pool with name %v doesn't exist",
			poolName)
	}
	// Remove existing ones with this pool name
	_, err := Bun().NewDelete().Table("rp_workspace_bindings").
		Where("pool_name = ?", poolName).Exec(ctx)
	if err != nil {
		return err
	}

	err = AddRPWorkspaceBindings(ctx, workspaceIds, poolName, resourcePools)
	return err
}

// GetAllBindings gets all valid rp-workspace bindings.
func GetAllBindings(
	ctx context.Context,
) ([]*RPWorkspaceBinding, error) {
	var rpWorkspaceBindings []*RPWorkspaceBinding
	err := Bun().NewSelect().Model(&rpWorkspaceBindings).Where("valid = ?", true).Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, err
	}
	return rpWorkspaceBindings, nil
}

// GetUnboundRPs get unbound resource pools.
func GetUnboundRPs(
	ctx context.Context, resourcePools []string,
) ([]string, error) {
	var boundResourcePools []string
	_, err := Bun().NewSelect().
		Column("pool_name").
		Table("rp_workspace_bindings").
		Distinct().
		Exec(ctx, &boundResourcePools)
	if err != nil {
		return nil, err
	}

	boundRPsMap := map[string]bool{}
	for _, boundRP := range boundResourcePools {
		boundRPsMap[boundRP] = true
	}

	var unboundRPs []string
	for _, resourcePool := range resourcePools {
		if !boundRPsMap[resourcePool] {
			unboundRPs = append(unboundRPs, resourcePool)
		}
	}

	return unboundRPs, nil
}

// RP is a helper strct for Bun query.
type RP struct {
	Name string
}

// ReadRPsAvailableToWorkspace returns the names of resource pools available to a
// workspace.
func ReadRPsAvailableToWorkspace(
	ctx context.Context,
	workspaceID int32,
	offset int32,
	limit int32,
	resourcePoolConfig []config.ResourcePoolConfig,
) ([]string, *apiv1.Pagination, error) {
	var resourcePoolNames []string
	for _, rp := range resourcePoolConfig {
		resourcePoolNames = append(resourcePoolNames, rp.PoolName)
	}
	unboundRPNames, err := GetUnboundRPs(ctx, resourcePoolNames)
	if err != nil {
		return nil, nil, err
	}
	unboundRPs := []*RP{}
	for _, unboundRPName := range unboundRPNames {
		unboundRPs = append(unboundRPs, &RP{unboundRPName})
	}

	var rpNames []string
	var query *bun.SelectQuery
	if len(unboundRPs) > 0 {
		// TODO: The elements in unboundRPs are structs with a string field.
		// Is there a better way to do this?
		values := Bun().NewValues(&unboundRPs)
		boundAndUnboundRPSubTable := Bun().NewSelect().
			ColumnExpr("pool_name AS Name").
			Table("rp_workspace_bindings").
			Where("workspace_id = ?", workspaceID).
			UnionAll(Bun().NewSelect().With("unboundRP", values).Table("unboundRP"))
		query = Bun().NewSelect().
			TableExpr("(?) AS rp", boundAndUnboundRPSubTable)
	} else {
		query = Bun().NewSelect().
			ColumnExpr("pool_name AS Name").
			Table("rp_workspace_bindings").
			Where("workspace_id = ?", workspaceID)
	}

	query, pagination, err := bunutils.Paginate(ctx, query, int(offset), int(limit))
	if err != nil {
		return nil, nil, err
	}
	// Bun bug treating limit=0 as no limit when it
	// should be the exact opposite of no records returned.
	// TODO: revisit and check this for commonality.
	// We may put pagination.StartIndex-pagination.EndIndex != 0
	// back to the function and return a nil query if StartIndex = EndIndex. This is for
	// limit = -2, we don't run the query, return pagination only.
	if pagination.StartIndex-pagination.EndIndex != 0 {
		if err = query.Scan(ctx, &rpNames); err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return rpNames, pagination, nil
			}
			return nil, nil, err
		}
	}

	return rpNames, pagination, nil
}

// GetDefaultPoolsForWorkspace returns the default compute and aux pools for a workspace.
func GetDefaultPoolsForWorkspace(ctx context.Context, workspaceID int,
) (computePool, auxPool string, err error) {
	var target model.Workspace
	err = Bun().NewSelect().Model(&target).Where("id = ?", workspaceID).Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return "", "", err
	}

	return target.DefaultComputePool, target.DefaultAuxPool, nil
}

// CheckIfRPUnbound checks to make sure the specified resource pools is not bound to any workspace
// and returns an error if it is.
func CheckIfRPUnbound(poolName string) error {
	exists, err := Bun().NewSelect().Table("rp_workspace_bindings").
		Where("pool_name = ?", poolName).
		Exists(context.TODO())
	if err != nil {
		return err
	} else if exists {
		return fmt.Errorf("default resource pool %s can not be bound to any workspaces", poolName)
	}
	return nil
}
