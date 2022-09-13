package rbac

import (
	"context"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var rbacAPIServer RBACAPIServer = &rbacAPIServerStub{}

type RBACAPIServer interface {
	GetRolesByID(context.Context, *apiv1.GetRolesByIDRequest) (
		resp *apiv1.GetRolesByIDResponse, err error)
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
}
