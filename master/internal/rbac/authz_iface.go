package rbac

import "github.com/uptrace/bun"

type RBACAuthZ interface {
	// GET /api/v1/permissions/summary
	// GetPermissionsSummary()
	// should always be allowed

	// POST /api/v1/roles/search/by-ids
	// GetRolesByID()
	CanGetRoles() // should always be allowed

	// POST /api/v1/roles/search
	// ListRoles
	FilterRolesQuery(query *bun.SelectQuery) (*bun.SelectQuery, error)

	// GET /api/v1/roles/search/by-user/{user_id}
	// GetRolesAssignedToUser()
	CanGetUserRoles() // filters by users

	// GET /api/v1/roles/search/by-group/{group_id}
	// GetRolesAssignedToGroup()
	CanGetGroupRoles()

	// POST /api/v1/roles/search/by-assignability
	// SearchRolesAssignableToScope()
	CanSearchScope()


	// POST /api/v1/roles/add-assignments
	// AssignRoles()
	// AssignWorkspaceAdminToUserTx()
	CanAssignRoles()

	// POST /api/v1/roles/remove-assignments"
	// RemoveAssignments
	CanRemoveRoles()


}