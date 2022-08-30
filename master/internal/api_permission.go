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

	resp := &apiv1.GetClusterPermissionsResponse{Roles: &apiv1.ClusterRoles{
		Editor: []string{},
		Viewer: []string{},
	}}
	err = a.m.db.QueryProto(
		"get_cluster_permissions",
		resp.Roles,
		int32(curUser.ID),
	)

	return resp, err
}

func (a *apiServer) GetPermissionsSummary(
	ctx context.Context, req *apiv1.GetPermissionsSummaryRequest,
) (*apiv1.GetPermissionsSummaryResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetPermissionsSummaryResponse{
		Roles: &apiv1.ClusterRoles{
			Editor: []string{},
			Viewer: []string{},
		},
		Assignments: &apiv1.ClusterAssignments{
			Editor: &apiv1.AssignmentCollection{},
			Viewer: &apiv1.AssignmentCollection{},
		},
	}

	// load cluster-scope roles first
	err = a.m.db.QueryProto(
		"get_cluster_permissions",
		resp.Roles,
		int32(curUser.ID),
	)
	if err != nil {
		return nil, err
	}

	// load remaining assigned roles
	err = a.m.db.QueryProto(
		"get_assigned_permissions",
		resp.Assignments,
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

	resp := &apiv1.GetWorkspacePermissionsResponse{Roles: &apiv1.ClusterRoles{
		Editor: []string{},
		Viewer: []string{},
	}}
	err = a.m.db.QueryProto(
		"get_workspace_permissions",
		resp.Roles,
		int32(curUser.ID),
		ws.Id,
	)

	return resp, err
}
