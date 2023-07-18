package internal

import (
	"context"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/set"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	workspaceauth "github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetResourcePools(
	_ context.Context, req *apiv1.GetResourcePoolsRequest,
) (*apiv1.GetResourcePoolsResponse, error) {
	resp, err := a.m.rm.GetResourcePools(a.m.system, req)
	if err != nil {
		return nil, err
	}
	return resp, a.paginate(&resp.Pagination, &resp.ResourcePools, req.Offset, req.Limit)
}

func (a *apiServer) BindRPToWorkspace(
	ctx context.Context, req *apiv1.BindRPToWorkspaceRequest,
) (*apiv1.BindRPToWorkspaceResponse, error) {
	defaultComputePool, defaultAuxPool, err := a.m.config.DefaultResourcePools()
	if err != nil {
		return nil, err
	}
	if req.ResourcePoolName == defaultComputePool || req.ResourcePoolName == defaultAuxPool {
		return nil, errors.Errorf("default resource pool %s cannot be bound to any workspace",
			req.ResourcePoolName)
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	if err = workspaceauth.AuthZProvider.Get().CanModifyRPWorkspaceBindings(ctx, *curUser,
		allWorkspaceIDs); err != nil {
		return nil, authz.SubIfUnauthorized(
			err,
			errors.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings.`,
				curUser.Username))
	}

	err = db.AddRPWorkspaceBindings(ctx, allWorkspaceIDs, req.ResourcePoolName,
		config.GetMasterConfig().ResourceConfig.ResourcePools)
	if err != nil {
		return nil, err
	}
	return &apiv1.BindRPToWorkspaceResponse{}, nil
}

func (a *apiServer) OverwriteRPWorkspaceBindings(
	ctx context.Context, req *apiv1.OverwriteRPWorkspaceBindingsRequest,
) (*apiv1.OverwriteRPWorkspaceBindingsResponse, error) {
	defaultComputePool, defaultAuxPool, err := a.m.config.DefaultResourcePools()
	if err != nil {
		return nil, err
	}
	if req.ResourcePoolName == defaultComputePool || req.ResourcePoolName == defaultAuxPool {
		return nil, errors.Errorf("default resource pool %s cannot be bound to any workspace",
			req.ResourcePoolName)
	}

	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	if err = workspaceauth.AuthZProvider.Get().CanModifyRPWorkspaceBindings(ctx, *curUser,
		allWorkspaceIDs); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings.`,
				curUser.Username))
	}

	masterConfig := config.GetMasterConfig()
	err = db.OverwriteRPWorkspaceBindings(ctx, allWorkspaceIDs, req.ResourcePoolName,
		masterConfig.ResourcePools)
	if err != nil {
		return nil, err
	}

	return &apiv1.OverwriteRPWorkspaceBindingsResponse{}, nil
}

func (a *apiServer) UnbindRPFromWorkspace(
	ctx context.Context, req *apiv1.UnbindRPFromWorkspaceRequest,
) (*apiv1.UnbindRPFromWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	allWorkspaceIDs, err := combineWorkspaceIDsAndNames(ctx, req.WorkspaceIds, req.WorkspaceNames)
	if err != nil {
		return nil, err
	}

	// Check permissions for all workspaces. Return err if any workspace doesn't have permissions.
	// No partial unbinding.
	if err = workspaceauth.AuthZProvider.Get().CanModifyRPWorkspaceBindings(ctx, *curUser,
		allWorkspaceIDs); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings.`,
				curUser.Username))
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
	rpWorkspaceBindings, pagination, err := db.ReadWorkspacesBoundToRP(
		ctx, req.ResourcePoolName, req.Offset, req.Limit,
		config.GetMasterConfig().ResourcePools,
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
	workspaceIDs, err = workspaceauth.AuthZProvider.Get().FilterWorkspaceIDs(
		ctx, *curUser, workspaceIDs,
	)
	if err != nil {
		return nil, err
	}

	return &apiv1.ListWorkspacesBoundToRPResponse{
		WorkspaceIds: workspaceIDs, Pagination: pagination,
	}, nil
}

func combineWorkspaceIDsAndNames(ctx context.Context, ids []int32, names []string,
) ([]int32, error) {
	workspaceIDs, err := workspaceauth.WorkspaceIDsFromNames(ctx, names)
	if err != nil {
		return nil, err
	}

	idSet := set.New[int32]()
	for _, id := range ids {
		idSet.Insert(id)
	}
	for _, id := range workspaceIDs {
		idSet.Insert(id)
	}

	return idSet.ToSlice(), nil
}
