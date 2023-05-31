package internal

import (
	"context"
<<<<<<< HEAD

	"github.com/pkg/errors"

=======
>>>>>>> a5a5431ac (unbind workspace RP handler + db helper)
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	workspaceauth "github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/pkg/errors"
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
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	if err = workspaceauth.AuthZProvider.Get().CanBindRPWorkspace(ctx, *curUser,
		req.WorkspaceIds); err != nil {
		return nil, authz.SubIfUnauthorized(
			err,
			errors.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings.`,
				curUser.Username))
	}

	err = a.m.db.AddRPWorkspaceBindings(ctx, req.WorkspaceIds, req.ResourcePoolName)
	if err != nil {
		return nil, err
	}
	return &apiv1.BindRPToWorkspaceResponse{}, nil
}

func (a *apiServer) OverwriteRPWorkspaceBindings(
	ctx context.Context, req *apiv1.OverwriteRPWorkspaceBindingsRequest,
) (*apiv1.OverwriteRPWorkspaceBindingsResponse, error) {
	return nil, nil
}

func (a *apiServer) UnbindRPFromWorkspace(
	ctx context.Context, req *apiv1.UnbindRPFromWorkspaceRequest,
) (*apiv1.UnbindRPFromWorkspaceResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	// Check permissions for all workspaces. Return err if any workspace doesn't have permissions.
	// No partial unbinding.
	if err = workspaceauth.AuthZProvider.Get().CanUnBindRPWorkspace(ctx, *curUser,
		req.WorkspaceIds); err != nil {
		return nil, authz.SubIfUnauthorized(err,
			errors.Errorf(
				`current user %q doesn't have permissions to modify resource pool bindings.`,
				curUser.Username))
	}

	err = a.m.db.RemoveRPWorkspaceBindings(ctx, req.WorkspaceIds, req.ResourcePoolName)
	if err != nil {
		return nil, err
	}
	return &apiv1.UnbindRPFromWorkspaceResponse{}, nil
}

func (a *apiServer) ListWorkspacesBoundToRP(
	ctx context.Context, req *apiv1.ListWorkspacesBoundToRPRequest,
) (*apiv1.ListWorkspacesBoundToRPResponse, error) {
	return nil, nil
}

func (a *apiServer) ListRPWorkspaceBindings(
	ctx context.Context, req *apiv1.ListRPWorkspaceBindingsRequest,
) (*apiv1.ListRPWorkspaceBindingsResponse, error) {
	return nil, nil
}
