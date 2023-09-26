package rbac

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// RBACAPIServerWrapper is a struct that implements RBACAPIServer.
type RBACAPIServerWrapper struct{}

// GetPermissionsSummary is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetPermissionsSummary(
	ctx context.Context, req *apiv1.GetPermissionsSummaryRequest,
) (*apiv1.GetPermissionsSummaryResponse, error) {
	return rbacAPIServer.GetPermissionsSummary(ctx, req)
}

// GetGroupsAndUsersAssignedToWorkspace is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetGroupsAndUsersAssignedToWorkspace(
	ctx context.Context, req *apiv1.GetGroupsAndUsersAssignedToWorkspaceRequest,
) (*apiv1.GetGroupsAndUsersAssignedToWorkspaceResponse, error) {
	return rbacAPIServer.GetGroupsAndUsersAssignedToWorkspace(ctx, req)
}

// GetRolesByID is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetRolesByID(ctx context.Context, req *apiv1.GetRolesByIDRequest) (
	resp *apiv1.GetRolesByIDResponse, err error,
) {
	return rbacAPIServer.GetRolesByID(ctx, req)
}

// GetRolesAssignedToUser is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetRolesAssignedToUser(ctx context.Context,
	req *apiv1.GetRolesAssignedToUserRequest,
) (*apiv1.GetRolesAssignedToUserResponse, error) {
	return rbacAPIServer.GetRolesAssignedToUser(ctx, req)
}

// GetRolesAssignedToGroup is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetRolesAssignedToGroup(ctx context.Context,
	req *apiv1.GetRolesAssignedToGroupRequest,
) (*apiv1.GetRolesAssignedToGroupResponse, error) {
	return rbacAPIServer.GetRolesAssignedToGroup(ctx, req)
}

// SearchRolesAssignableToScope is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) SearchRolesAssignableToScope(
	ctx context.Context,
	req *apiv1.SearchRolesAssignableToScopeRequest,
) (*apiv1.SearchRolesAssignableToScopeResponse, error) {
	return rbacAPIServer.SearchRolesAssignableToScope(ctx, req)
}

// ListRoles is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) ListRoles(ctx context.Context, req *apiv1.ListRolesRequest) (
	*apiv1.ListRolesResponse, error,
) {
	return rbacAPIServer.ListRoles(ctx, req)
}

// AssignRoles is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) AssignRoles(ctx context.Context, req *apiv1.AssignRolesRequest) (
	*apiv1.AssignRolesResponse, error,
) {
	return rbacAPIServer.AssignRoles(ctx, req)
}

// RemoveAssignments is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) RemoveAssignments(ctx context.Context,
	req *apiv1.RemoveAssignmentsRequest,
) (*apiv1.RemoveAssignmentsResponse, error) {
	return rbacAPIServer.RemoveAssignments(ctx, req)
}

// AssignWorkspaceAdminToUserTx is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) AssignWorkspaceAdminToUserTx(
	ctx context.Context, idb bun.IDB, workspaceID int, userID model.UserID,
) error {
	return rbacAPIServer.AssignWorkspaceAdminToUserTx(ctx, idb, workspaceID, userID)
}

// GetUsersEE is a wrapper the same function the RBACAPIServer interface.
func (s *RBACAPIServerWrapper) GetUsersEE(
	ctx context.Context, req *apiv1.GetUsersEERequest,
) (*apiv1.GetUsersEEResponse, error) {
	return rbacAPIServer.GetUsersEE(ctx, req)
}
