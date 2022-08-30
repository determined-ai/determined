package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetClusterPermissions(
	ctx context.Context, req *apiv1.GetClusterPermissionsRequest,
) (*apiv1.GetClusterPermissionsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetClusterPermissionsResponse{}
	err = a.m.db.QueryProto(
		"get_cluster_permissions",
		resp,
		int32(curUser.ID),
	)

	return resp, err
}

func (a *apiServer) GetWorkspacePermissions(
	ctx context.Context, req *apiv1.GetWorkspacePermissionsRequest,
) (*apiv1.GetWorkspacePermissionsResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	// confirm workspace exists / confirm view permission
	ws, err := a.GetWorkspaceByID(req.WorkspaceId, *curUser, false)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetWorkspacePermissionsResponse{}
	err = a.m.db.QueryProto(
		"get_workspace_permissions",
		resp,
		int32(curUser.ID),
		ws.Id,
	)

	return resp, err
}
