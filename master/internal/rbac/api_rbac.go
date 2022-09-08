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

type RBACAPIServerWrapper struct{}

func (s *RBACAPIServerWrapper) GetRolesByID(ctx context.Context, req *apiv1.GetRolesByIDRequest) (resp *apiv1.GetRolesByIDResponse, err error) {
	return rbacAPIServer.GetRolesByID(ctx, req)
}

func (s *RBACAPIServerWrapper) GetRolesAssignedToUser(ctx context.Context, req *apiv1.GetRolesAssignedToUserRequest) (*apiv1.GetRolesAssignedToUserResponse, error) {
	return rbacAPIServer.GetRolesAssignedToUser(ctx, req)
}

func (s *RBACAPIServerWrapper) GetRolesAssignedToGroup(ctx context.Context, req *apiv1.GetRolesAssignedToGroupRequest) (*apiv1.GetRolesAssignedToGroupResponse, error) {
	return rbacAPIServer.GetRolesAssignedToGroup(ctx, req)
}

func (s *RBACAPIServerWrapper) SearchRolesAssignableToScope(ctx context.Context, req *apiv1.SearchRolesAssignableToScopeRequest) (*apiv1.SearchRolesAssignableToScopeResponse, error) {
	return rbacAPIServer.SearchRolesAssignableToScope(ctx, req)
}

func (s *RBACAPIServerWrapper) ListRoles(ctx context.Context, req *apiv1.ListRolesRequest) (*apiv1.ListRolesResponse, error) {
	return rbacAPIServer.ListRoles(ctx, req)
}

func (s *RBACAPIServerWrapper) AssignRoles(ctx context.Context, req *apiv1.AssignRolesRequest) (*apiv1.AssignRolesResponse, error) {
	return rbacAPIServer.AssignRoles(ctx, req)
}

func (s *RBACAPIServerWrapper) RemoveAssignments(ctx context.Context, req *apiv1.RemoveAssignmentsRequest) (*apiv1.RemoveAssignmentsResponse, error) {
	return rbacAPIServer.RemoveAssignments(ctx, req)
}
