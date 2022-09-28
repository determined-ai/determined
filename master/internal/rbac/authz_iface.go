package rbac

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

type RBACAuthZ interface {
	// GET /api/v1/permissions/summary
	// GetPermissionsSummary()
	// should always be allowed

	// CanGetRoles checks if a user has access to certain roles.
	// Should usually be allowed.
	// POST /api/v1/roles/search/by-ids
	CanGetRoles(curUser model.User) error

	// FilterRolesQuery filters a role search to show only what a user
	// can access.
	// POST /api/v1/roles/search
	// ListRoles
	FilterRolesQuery(curUser model.User, query *bun.SelectQuery) (*bun.SelectQuery, error)

	// CanGetUserRoles checks
	// GET /api/v1/roles/search/by-user/{user_id}
	// GetRolesAssignedToUser()
	CanGetUserRoles(curUser model.User, userId int) error

	// GET /api/v1/roles/search/by-group/{group_id}
	// GetRolesAssignedToGroup()
	CanGetGroupRoles(curUser model.User, groupId int) error

	// POST /api/v1/roles/search/by-assignability
	// SearchRolesAssignableToScope()
	CanSearchScope(curUser model.User, workspaceId int) error

	// POST /api/v1/roles/add-assignments
	// AssignRoles()
	// AssignWorkspaceAdminToUserTx()
	CanAssignRoles(curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
		userRoleAssignments []*rbacv1.UserRoleAssignment) error

	// POST /api/v1/roles/remove-assignments"
	// RemoveAssignments
	CanRemoveRoles(curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
		userRoleAssignments []*rbacv1.UserRoleAssignment) error
}

// AuthZProvider is the authz registry for RBAC.
var AuthZProvider authz.AuthZProviderType[RBACAuthZ]
