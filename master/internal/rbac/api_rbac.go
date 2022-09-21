package rbac

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var rbacAPIServer RBACAPIServer = &rbacAPIServerStub{}

// RBACAPIServer is the interface for all functions in RBAC.
type RBACAPIServer interface {
	GetPermissionsSummary(context.Context, *apiv1.GetPermissionsSummaryRequest) (
		*apiv1.GetPermissionsSummaryResponse, error)
	GetGroupsAndUsersAssignedToWorkspace(
		context.Context, *apiv1.GetGroupsAndUsersAssignedToWorkspaceRequest,
	) (*apiv1.GetGroupsAndUsersAssignedToWorkspaceResponse, error)
	GetRolesByID(context.Context, *apiv1.GetRolesByIDRequest) (
		*apiv1.GetRolesByIDResponse, error)
	GetRolesAssignedToUser(context.Context, *apiv1.GetRolesAssignedToUserRequest) (
		*apiv1.GetRolesAssignedToUserResponse, error)
	GetRolesAssignedToGroup(context.Context, *apiv1.GetRolesAssignedToGroupRequest) (
		*apiv1.GetRolesAssignedToGroupResponse, error)
	SearchRolesAssignableToScope(context.Context, *apiv1.SearchRolesAssignableToScopeRequest) (
		*apiv1.SearchRolesAssignableToScopeResponse, error)
	ListRoles(context.Context, *apiv1.ListRolesRequest) (
		*apiv1.ListRolesResponse, error)
	AssignRoles(context.Context, *apiv1.AssignRolesRequest) (
		*apiv1.AssignRolesResponse, error)
	RemoveAssignments(context.Context, *apiv1.RemoveAssignmentsRequest) (
		*apiv1.RemoveAssignmentsResponse, error)
	AssignWorkspaceAdminToUserTx(
		ctx context.Context, idb bun.IDB, workspaceID int, userID model.UserID,
	) error
}
