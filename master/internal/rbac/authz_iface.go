package rbac

import (
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// RBACAuthZ describes authz methods for RBAC.
type RBACAuthZ interface {
	// GET /api/v1/permissions/summary
	// GetPermissionsSummary()
	// should always be allowed

	// CanGetRoles checks if a user has access to certain roles.
	// Should usually be allowed.
	// POST /api/v1/roles/search/by-ids
	CanGetRoles(curUser model.User) error

	// FilterRolesQuery filters a role search to show only what a user can access.
	// POST /api/v1/roles/search
	// ListRoles
	FilterRolesQuery(curUser model.User, query *bun.SelectQuery) (*bun.SelectQuery, error)

	// CanGetUserRoles checks if a user can get another user's assigned roles.
	// GET /api/v1/roles/search/by-user/{user_id}
	// GetRolesAssignedToUser()
	CanGetUserRoles(curUser model.User, userID int) error

	// CanGetGroupRoles checks if a user can get the roles assigned to a group.
	// GET /api/v1/roles/search/by-group/{group_id}
	// GetRolesAssignedToGroup()
	CanGetGroupRoles(curUser model.User, groupID int) error

	// CanSearchScope checks if a user can search a scope for roles.
	// POST /api/v1/roles/search/by-assignability
	// SearchRolesAssignableToScope()
	CanSearchScope(curUser model.User, workspaceID int) error

	// CanAssignRoles checks if a user has the assign roles permission
	// POST /api/v1/roles/add-assignments
	// AssignRoles()
	// AssignWorkspaceAdminToUserTx()
	CanAssignRoles(curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
		userRoleAssignments []*rbacv1.UserRoleAssignment) error

	// CanRemoveRoles checks if a user has the assign roles permission
	// POST /api/v1/roles/remove-assignments
	// RemoveAssignments
	CanRemoveRoles(curUser model.User, groupRoleAssignments []*rbacv1.GroupRoleAssignment,
		userRoleAssignments []*rbacv1.UserRoleAssignment) error
}

// AuthZProvider is the authz registry for RBAC.
var AuthZProvider authz.AuthZProviderType[RBACAuthZ]
