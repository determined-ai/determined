package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetPermissionsSummary(
	ctx context.Context, req *apiv1.GetPermissionsSummaryRequest,
) (*apiv1.GetPermissionsSummaryResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx, a.m.db, &a.m.config.InternalConfig.ExternalSessions)
	if err != nil {
		return nil, err
	}

	// load permissions for cluster and other scopes
	roles := []*apiv1.Role{}
	err = a.m.db.QueryProto(
		"get_permission_roles",
		&roles,
		int32(curUser.ID),
	)
	if err != nil {
		return nil, err
	}

	// load list of workspace IDs on assignments
	assignments := []*apiv1.AssignmentGroup{}
	err = a.m.db.QueryProto(
		"get_permission_assignments",
		&assignments,
		int32(curUser.ID),
	)
	if err != nil {
		return nil, err
	}

	resp := &apiv1.GetPermissionsSummaryResponse{Assignments: assignments, Roles: roles}
	return resp, err
}
