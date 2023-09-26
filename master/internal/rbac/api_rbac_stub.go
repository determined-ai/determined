package rbac

import (
	"context"

	"github.com/uptrace/bun"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type rbacAPIServerStub struct{}

const stubUnimplementedMessage = "Determined Enterprise Edition is required for this functionality"

// UnimplementedError is the error returned for unimplemented functions.
var UnimplementedError = status.Error(codes.Unimplemented, stubUnimplementedMessage)

func (s *rbacAPIServerStub) GetPermissionsSummary(
	ctx context.Context, req *apiv1.GetPermissionsSummaryRequest,
) (*apiv1.GetPermissionsSummaryResponse, error) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) GetGroupsAndUsersAssignedToWorkspace(
	context.Context, *apiv1.GetGroupsAndUsersAssignedToWorkspaceRequest,
) (*apiv1.GetGroupsAndUsersAssignedToWorkspaceResponse, error) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) GetRolesByID(ctx context.Context, req *apiv1.GetRolesByIDRequest) (
	resp *apiv1.GetRolesByIDResponse, err error,
) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) GetRolesAssignedToUser(ctx context.Context,
	req *apiv1.GetRolesAssignedToUserRequest,
) (*apiv1.GetRolesAssignedToUserResponse, error) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) GetRolesAssignedToGroup(ctx context.Context,
	req *apiv1.GetRolesAssignedToGroupRequest,
) (*apiv1.GetRolesAssignedToGroupResponse, error) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) SearchRolesAssignableToScope(ctx context.Context,
	req *apiv1.SearchRolesAssignableToScopeRequest) (*apiv1.SearchRolesAssignableToScopeResponse,
	error,
) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) ListRoles(ctx context.Context, req *apiv1.ListRolesRequest) (
	*apiv1.ListRolesResponse, error,
) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) AssignRoles(ctx context.Context, req *apiv1.AssignRolesRequest) (
	*apiv1.AssignRolesResponse, error,
) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) RemoveAssignments(ctx context.Context,
	req *apiv1.RemoveAssignmentsRequest,
) (*apiv1.RemoveAssignmentsResponse, error) {
	return nil, UnimplementedError
}

func (s *rbacAPIServerStub) AssignWorkspaceAdminToUserTx(
	ctx context.Context, idb bun.IDB, workspaceID int, userID model.UserID,
) error {
	return nil
}

func (s *rbacAPIServerStub) GetUsersEE(
	ctx context.Context, req *apiv1.GetUsersEERequest,
) (*apiv1.GetUsersEEResponse, error) {
	return nil, UnimplementedError
}
