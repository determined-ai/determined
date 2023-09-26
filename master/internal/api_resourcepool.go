package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	workspaceauth "github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

func (a *apiServer) getUnboundResourcePools(ctx context.Context,
	resourcePools []*resourcepoolv1.ResourcePool,
) ([]*resourcepoolv1.ResourcePool, error) {
	var poolNames []string
	for _, pool := range resourcePools {
		poolNames = append(poolNames, pool.Name)
	}

	unboundPoolNames, err := db.GetUnboundRPs(ctx, poolNames)
	if err != nil {
		return nil, err
	}
	unboundPoolNameSet := set.FromSlice[string](unboundPoolNames)

	var unboundPools []*resourcepoolv1.ResourcePool

	for _, pool := range resourcePools {
		if unboundPoolNameSet.Contains(pool.Name) {
			unboundPools = append(unboundPools, pool)
		}
	}

	return unboundPools, nil
}

func (a *apiServer) GetResourcePools(
	ctx context.Context, req *apiv1.GetResourcePoolsRequest,
) (*apiv1.GetResourcePoolsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := a.m.rm.GetResourcePools(a.m.system, req)
	if err != nil {
		return nil, err
	}

	workspaces, err := workspaceauth.AllWorkspaces(ctx)
	if err != nil {
		return nil, err
	}
	var workspaceIDs []int32
	for _, w := range workspaces {
		workspaceIDs = append(workspaceIDs, int32(w.ID))
	}
	ids, err := workspaceauth.AuthZProvider.Get().FilterWorkspaceIDs(ctx, *curUser, workspaceIDs)
	if err != nil {
		return nil, err
	}

	filteredPools, err := rm.AuthZProvider.Get().FilterResourcePools(ctx, *curUser,
		resp.ResourcePools, ids)
	if err != nil {
		return nil, err
	}
	resp.ResourcePools = filteredPools

	if req.Unbound {
		unboundPools, err := a.getUnboundResourcePools(ctx, filteredPools)
		if err != nil {
			return nil, err
		}
		resp.ResourcePools = unboundPools
	}

	return resp, api.Paginate(&resp.Pagination, &resp.ResourcePools, req.Offset, req.Limit)
}

func (a *apiServer) BindRPToWorkspace(
	ctx context.Context, req *apiv1.BindRPToWorkspaceRequest,
) (*apiv1.BindRPToWorkspaceResponse, error) {
	err := a.checkIfPoolIsDefault(req.ResourcePoolName)
	if err != nil {
		return nil, err
	}

	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	err = a.canUserModifyWorkspaces(ctx, allWorkspaceIDs)
	if err != nil {
		return nil, err
	}

	rpConfigs, err := a.resourcePoolsAsConfigs()
	if err != nil {
		return nil, err
	}

	err = db.AddRPWorkspaceBindings(ctx, allWorkspaceIDs, req.ResourcePoolName,
		rpConfigs)
	if err != nil {
		return nil, err
	}
	return &apiv1.BindRPToWorkspaceResponse{}, nil
}

func (a *apiServer) OverwriteRPWorkspaceBindings(
	ctx context.Context, req *apiv1.OverwriteRPWorkspaceBindingsRequest,
) (*apiv1.OverwriteRPWorkspaceBindingsResponse, error) {
	err := a.checkIfPoolIsDefault(req.ResourcePoolName)
	if err != nil {
		return nil, err
	}

	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	err = a.canUserModifyWorkspaces(ctx, allWorkspaceIDs)
	if err != nil {
		return nil, err
	}

	rpConfigs, err := a.resourcePoolsAsConfigs()
	if err != nil {
		return nil, err
	}
	err = db.OverwriteRPWorkspaceBindings(ctx, allWorkspaceIDs, req.ResourcePoolName,
		rpConfigs)
	if err != nil {
		return nil, err
	}

	return &apiv1.OverwriteRPWorkspaceBindingsResponse{}, nil
}

func (a *apiServer) UnbindRPFromWorkspace(
	ctx context.Context, req *apiv1.UnbindRPFromWorkspaceRequest,
) (*apiv1.UnbindRPFromWorkspaceResponse, error) {
	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	err = a.canUserModifyWorkspaces(ctx, allWorkspaceIDs)
	if err != nil {
		return nil, err
	}

	err = db.RemoveRPWorkspaceBindings(ctx, allWorkspaceIDs, req.ResourcePoolName)
	if err != nil {
		return nil, err
	}
	return &apiv1.UnbindRPFromWorkspaceResponse{}, nil
}

func (a *apiServer) ListWorkspacesBoundToRP(
	ctx context.Context, req *apiv1.ListWorkspacesBoundToRPRequest,
) (*apiv1.ListWorkspacesBoundToRPResponse, error) {
	rpConfigs, err := a.resourcePoolsAsConfigs()
	if err != nil {
		return nil, err
	}
	rpWorkspaceBindings, pagination, err := db.ReadWorkspacesBoundToRP(
		ctx, req.ResourcePoolName, req.Offset, req.Limit,
		rpConfigs,
	)
	if err != nil {
		return nil, err
	}

	var workspaceIDs []int32
	for _, rpWorkspaceBinding := range rpWorkspaceBindings {
		workspaceIDs = append(workspaceIDs, int32(rpWorkspaceBinding.WorkspaceID))
	}

	// Show the workspaces the user has access to.
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	allIDsLen := len(workspaceIDs)
	workspaceIDs, err = workspaceauth.AuthZProvider.Get().FilterWorkspaceIDs(
		ctx, *curUser, workspaceIDs,
	)
	if err != nil {
		return nil, err
	}

	if len(workspaceIDs) == 0 && allIDsLen != 0 {
		return nil, fmt.Errorf("resource pool %s does not exist or is not available to view",
			req.ResourcePoolName)
	}

	return &apiv1.ListWorkspacesBoundToRPResponse{
		WorkspaceIds: workspaceIDs, Pagination: pagination,
	}, nil
}

func (a *apiServer) checkIfPoolIsDefault(poolName string) error {
	defaultComputePool, err := a.m.rm.GetDefaultComputeResourcePool(
		a.m.system,
		sproto.GetDefaultComputeResourcePoolRequest{})
	if err != nil {
		return err
	}

	defaultAuxPool, err := a.m.rm.GetDefaultAuxResourcePool(
		a.m.system,
		sproto.GetDefaultAuxResourcePoolRequest{},
	)
	if err != nil {
		return err
	}

	isDefaultCompute := poolName == defaultComputePool.PoolName
	isDefaultAux := poolName == defaultAuxPool.PoolName
	if isDefaultCompute || isDefaultAux {
		return fmt.Errorf(
			"default resource pool %s cannot be bound to any workspace",
			poolName,
		)
	}
	return nil
}

func (a *apiServer) canUserModifyWorkspaces(ctx context.Context, ids []int32) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	if err = workspaceauth.AuthZProvider.Get().CanModifyRPWorkspaceBindings(ctx, *curUser,
		ids); err != nil {
		return authz.SubIfUnauthorized(err,
			fmt.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings`,
				curUser.Username))
	}
	return nil
}

func (a *apiServer) resourcePoolsAsConfigs() ([]config.ResourcePoolConfig, error) {
	resp, err := a.m.rm.GetResourcePools(a.m.system, &apiv1.GetResourcePoolsRequest{})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return []config.ResourcePoolConfig{}, nil
	}

	var rpConfigs []config.ResourcePoolConfig
	for _, rp := range resp.ResourcePools {
		rpConfigs = append(rpConfigs, config.ResourcePoolConfig{
			PoolName: rp.Name,
		})
	}

	return rpConfigs, nil
}

func combineWorkspaceIDsAndNames(ctx context.Context, ids []int32, names []string,
) ([]int32, error) {
	workspaceIDs, err := workspaceauth.WorkspaceIDsFromNames(ctx, names)
	if err != nil {
		return nil, err
	}

	idSet := set.FromSlice(ids)
	for _, id := range workspaceIDs {
		idSet.Insert(id)
	}

	return idSet.ToSlice(), nil
}
