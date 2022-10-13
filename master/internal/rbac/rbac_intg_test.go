//go:build integration
// +build integration

package rbac

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

const (
	pathToMigrations = "file://../../static/migrations"
)

var (
	testGroupStatic = usergroup.Group{
		ID:   10001,
		Name: "testGroupStatic",
	}
	testGroupOwnedByUser = usergroup.Group{} // Auto created upon user creation.
	testUser             = model.User{
		ID:       1217651234,
		Username: fmt.Sprintf("IntegrationTest%d", 1217651234),
		Admin:    false,
		Active:   false,
	}

	testRole = Role{
		ID:   10002,
		Name: "test role 1",
	}
	testRole2 = Role{
		ID:   10003,
		Name: "test role 2",
	}
	testRole3 = Role{
		ID:   10004,
		Name: "test role 3",
	}
	testRole4 = Role{
		ID:   10005,
		Name: "test role 4",
	}
	testRoles = []Role{testRole, testRole2, testRole3, testRole4}

	testPermission = Permission{
		ID:     10006,
		Name:   "test permission 1",
		Global: false,
	}
	testPermission2 = Permission{
		ID:     10007,
		Name:   "test permission 2",
		Global: false,
	}
	testPermission3 = Permission{
		ID:     10008,
		Name:   "test permission 3",
		Global: false,
	}
	globalTestPermission = Permission{
		ID:     10009,
		Name:   "test permission global",
		Global: true,
	}
	testPermissions = []Permission{
		testPermission, testPermission2, testPermission3,
		globalTestPermission,
	}

	testWorkspace = Workspace{
		ID:   10011,
		Name: "test workspace",
	}

	testRoleAssignment = RoleAssignmentScope{
		ID: 10010,
		WorkspaceID: sql.NullInt32{
			Int32: int32(testWorkspace.ID),
			Valid: true,
		},
	}
)

type Workspace struct {
	bun.BaseModel `bun:"table:workspaces"`

	ID       int    `bun:"id,notnull"`
	Name     string `bun:"name,notnull"`
	Archived bool   `bun:"archived,notnull"`
}

func TestRbac(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	t.Cleanup(func() { cleanUp(ctx, t, pgDB) })
	setUp(ctx, t, pgDB)

	rbacRole := &rbacv1.Role{
		RoleId: int32(testRole.ID),
		Name:   testRole.Name,
		Permissions: []*rbacv1.Permission{
			{
				Id:       rbacv1.PermissionType(testPermission.ID),
				Name:     testPermission.Name,
				IsGlobal: testPermission.Global,
			},
		},
	}
	rbacRole2 := &rbacv1.Role{
		RoleId: int32(testRole2.ID),
		Name:   testRole2.Name,
		Permissions: []*rbacv1.Permission{
			{
				Id:       rbacv1.PermissionType(testPermission.ID),
				Name:     testPermission.Name,
				IsGlobal: testPermission.Global,
			},
		},
	}
	rbacRole3 := &rbacv1.Role{
		RoleId: int32(testRole3.ID),
		Name:   testRole3.Name,
		Permissions: []*rbacv1.Permission{
			{
				Id:       rbacv1.PermissionType(testPermission.ID),
				Name:     testPermission.Name,
				IsGlobal: testPermission.Global,
			},
		},
	}

	workspaceID := testRoleAssignment.WorkspaceID.Int32
	userRoleAssignment := rbacv1.UserRoleAssignment{
		UserId: int32(testUser.ID),
		RoleAssignment: &rbacv1.RoleAssignment{
			Role:             rbacRole,
			ScopeWorkspaceId: wrapperspb.Int32(workspaceID),
		},
	}

	groupRoleAssignment := rbacv1.GroupRoleAssignment{
		GroupId: int32(testGroupStatic.ID),
		RoleAssignment: &rbacv1.RoleAssignment{
			Role:             rbacRole,
			ScopeWorkspaceId: wrapperspb.Int32(workspaceID),
		},
	}
	assignmentScope := RoleAssignmentScope{}
	assignment := RoleAssignment{}

	t.Run("test user role assignment", func(t *testing.T) {
		// TODO: populate the permission assignments table in the future
		err := AddRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{}, []*rbacv1.UserRoleAssignment{&userRoleAssignment})
		require.NoError(t, err, "error adding role assignment")

		err = db.Bun().NewSelect().Model(&assignmentScope).Where(
			"scope_workspace_id=?", testRoleAssignment.WorkspaceID.Int32).Scan(ctx)
		require.NoError(t, err, "error getting created assignment scope")

		err = db.Bun().NewSelect().Model(&assignment).Where("group_id=?", testGroupOwnedByUser.ID).
			Scan(ctx)
		require.NoError(t, err, "error getting created role assignment")
		require.Equal(t, testGroupOwnedByUser.ID, assignment.GroupID, "incorrect group ID was assigned")
		require.Equal(t, testRole.ID, assignment.RoleID, "incorrect role ID was assigned")
		require.Equal(t, assignmentScope.ID, assignment.ScopeID, "incorrect scope ID was assigned")
	})

	t.Run("test delete user role assignment", func(t *testing.T) {
		err := RemoveRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{}, []*rbacv1.UserRoleAssignment{&userRoleAssignment})
		require.NoError(t, err, "error removing role assignment")

		err = db.Bun().NewSelect().Model(&assignmentScope).Where(
			"scope_workspace_id=?", testRoleAssignment.WorkspaceID.Int32).Scan(ctx)
		require.NoError(t, err, "assignment scope should still exist after removal")

		err = db.Bun().NewSelect().Model(&assignment).Where("group_id=?", testGroupOwnedByUser.ID).
			Scan(ctx)
		require.Errorf(t, err, "assignment should not exist after removal")
		require.True(t, errors.Is(db.MatchSentinelError(err), db.ErrNotFound), "incorrect error returned")
	})

	t.Run("test group role assignment", func(t *testing.T) {
		// TODO: populate the permission assignments table in the future
		err := AddRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{&groupRoleAssignment}, []*rbacv1.UserRoleAssignment{})
		require.NoError(t, err, "error adding role assignment")

		err = db.Bun().NewSelect().Model(&assignmentScope).Where(
			"scope_workspace_id=?", testRoleAssignment.WorkspaceID.Int32).Scan(ctx)
		require.NoError(t, err, "error getting created assignment scope")

		err = db.Bun().NewSelect().Model(&assignment).Where("group_id=?", testGroupStatic.ID).Scan(ctx)
		require.NoError(t, err, "error getting created role assignment")
		require.Equal(t, testGroupStatic.ID, assignment.GroupID, "incorrect group ID was assigned")
		require.Equal(t, testRole.ID, assignment.RoleID, "incorrect role ID was assigned")
		require.Equal(t, assignmentScope.ID, assignment.ScopeID, "incorrect scope ID was assigned")
	})

	t.Run("test delete group role assignment", func(t *testing.T) {
		err := RemoveRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{&groupRoleAssignment}, []*rbacv1.UserRoleAssignment{})
		require.NoError(t, err, "error removing role assignment")

		err = db.Bun().NewSelect().Model(&assignmentScope).Where(
			"scope_workspace_id=?", testRoleAssignment.WorkspaceID.Int32).Scan(ctx)
		require.NoError(t, err, "assignment scope should still exist after removal")

		err = db.Bun().NewSelect().Model(&assignment).Where("group_id=?", testGroupStatic.ID).Scan(ctx)
		require.Errorf(t, err, "assignment should not exist after removal")
		require.True(t, errors.Is(db.MatchSentinelError(err), db.ErrNotFound), "incorrect error returned")
	})

	t.Run("test add role twice", func(t *testing.T) {
		err := AddRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{&groupRoleAssignment}, []*rbacv1.UserRoleAssignment{})
		require.NoError(t, err, "error adding role assignment")
		err = AddRoleAssignments(
			ctx, []*rbacv1.GroupRoleAssignment{&groupRoleAssignment}, []*rbacv1.UserRoleAssignment{})
		require.Error(t, err, "adding the same role assignment should error")
		require.True(t, errors.Is(err, db.ErrDuplicateRecord), "error should be a duplicate record error")
	})

	t.Run("test insert multiple scopes", func(t *testing.T) {
		nilAssignment := &rbacv1.RoleAssignment{ScopeWorkspaceId: nil}
		_, err := getOrCreateRoleAssignmentScopeTx(ctx, nil, nilAssignment)
		require.NoError(t, err, "error with inserting a nil ")

		_, err = getOrCreateRoleAssignmentScopeTx(ctx, nil, nilAssignment)
		require.NoError(t, err, "inserting the same role assignment scope should not fail")

		rows, err := db.Bun().NewSelect().Table("role_assignment_scopes").
			Where("scope_workspace_id IS NULL").Count(ctx)
		require.Equal(t, 1, rows, "there should only have been one null scope created")
	})

	t.Run("test get all roles with pagination", func(t *testing.T) {
		permissionsToAdd := []map[string]interface{}{
			{
				"permission_id": globalTestPermission.ID,
				"role_id":       testRole4.ID,
			},
			{
				"permission_id": testPermission.ID,
				"role_id":       testRole3.ID,
			},
		}

		for _, p := range permissionsToAdd {
			perm := p
			_, err := db.Bun().NewInsert().Model(&perm).TableExpr("permission_assignments").Exec(ctx)
			require.NoError(t, err, "failure inserting permission assignments in local setup")
		}

		allRoles, _, err := GetAllRoles(ctx, false, 0, 10)
		roles := filterToTestRoles(allRoles)

		require.NoError(t, err, "error getting all roles")
		require.Equal(t, 4, len(roles), "incorrect number of roles retrieved")
		require.True(t, compareRoles(testRole, roles[0]),
			"test role 1 is not equivalent to the retrieved role")
		require.True(t, compareRoles(testRole2, roles[1]),
			"test role 2 is not equivalent to the retrieved role")
		require.True(t, compareRoles(testRole3, roles[2]),
			"test role 3 is not equivalent to the retrieved role")
		require.True(t, compareRoles(testRole4, roles[3]),
			"test role 4 is not equivalent to the retrieved role")

		globalRoles, _, err := GetAllRoles(ctx, true, 0, 10)
		roles = filterToTestRoles(globalRoles)
		require.NoError(t, err, "error getting non-global roles")
		require.Equal(t, 3, len(roles), "incorrect number of non-global roles retrieved")
		require.True(t, compareRoles(testRole, roles[0]),
			"test role 1 is not equivalent to the retrieved role")
		require.True(t, compareRoles(testRole2, roles[1]),
			"test role 2 is not equivalent to the retrieved role")
		require.True(t, compareRoles(testRole3, roles[2]),
			"test role 3 is not equivalent to the retrieved role")

		roles, _, err = GetAllRoles(ctx, false, 0, len(allRoles)+1)
		require.NoError(t, err, "error getting roles with limit")
		require.Len(t, roles, len(allRoles))

		roles, _, err = GetAllRoles(ctx, false, 0, len(allRoles)-1)
		require.NoError(t, err, "error getting roles with limit")
		require.Len(t, roles, len(allRoles)-1)

		roles, _, err = GetAllRoles(ctx, false, 2, len(allRoles))
		require.NoError(t, err, "error getting roles with limit")
		require.Len(t, roles, len(allRoles)-2)
		require.True(t, compareRoles(allRoles[2], roles[0]), "offset returned wrong first role")

		roles, _, err = GetAllRoles(ctx, false, len(allRoles), len(allRoles))
		require.NoError(t, err, "error getting roles with limit")
		require.Len(t, roles, 0)
	})

	t.Run("test getting roles by id", func(t *testing.T) {
		rolesWithAssignment, err := GetRolesByIDs(ctx, int32(testRole2.ID), int32(testRole4.ID))
		require.NoError(t, err, "error getting roles 2 and 4 by ID")
		require.Equal(t, testRole2.ID, int(rolesWithAssignment[0].Role.RoleId),
			"test role 2 is not equivalent to the retrieved role")
		require.Equal(t, testRole2.Name, rolesWithAssignment[0].Role.Name,
			"test role 2 is not equivalent to the retrieved role")
		require.Equal(t, testRole4.ID, int(rolesWithAssignment[1].Role.RoleId),
			"test role 4 is not equivalent to the retrieved role")
		require.Equal(t, testRole4.Name, rolesWithAssignment[1].Role.Name,
			"test role 4 is not equivalent to the retrieved role")
	})

	t.Run("test getting roles assigned to group", func(t *testing.T) {
		groupRoleAssignments := []*rbacv1.GroupRoleAssignment{
			{
				GroupId: int32(testGroupStatic.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             rbacRole2,
					ScopeWorkspaceId: wrapperspb.Int32(workspaceID),
				},
			},
			{
				GroupId: int32(testGroupStatic.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             rbacRole3,
					ScopeWorkspaceId: wrapperspb.Int32(workspaceID),
				},
			},
		}

		err := AddRoleAssignments(
			ctx, groupRoleAssignments, []*rbacv1.UserRoleAssignment{})
		require.NoError(t, err, "error adding role assignments")

		roles, err := GetRolesAssignedToGroupsTx(ctx, nil, int32(testGroupStatic.ID))
		require.NoError(t, err, "error getting roles assigned to group")
		require.Len(t, roles, 3, "incorrect number of roles retrieved")
		require.True(t, compareRoles(testRole, roles[0]),
			"testRole is not the first role retrieved by group id")
		require.True(t, compareRoles(testRole2, roles[1]),
			"testRole2 is not the second role retrieved by group id")
		require.True(t, compareRoles(testRole3, roles[2]),
			"testRole3 is not the first role retrieved by group id")

		err = RemoveRoleAssignments(ctx, groupRoleAssignments, nil)
		require.NoError(t, err, "error removing assignments from group")
		roles, err = GetRolesAssignedToGroupsTx(ctx, nil, int32(testGroupStatic.ID))
		require.Equal(t, 1, len(roles), "incorrect number of roles retrieved")
	})

	t.Run("test UserPermissionsForScope", func(t *testing.T) {
		groupRoleAssignments := []*rbacv1.GroupRoleAssignment{
			{
				GroupId: int32(testGroupStatic.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role: rbacRole,
				},
			},
			{
				GroupId: int32(testGroupOwnedByUser.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role:             rbacRole2,
					ScopeWorkspaceId: wrapperspb.Int32(workspaceID),
				},
			},
		}

		permissionAssignments := []PermissionAssignment{
			{
				PermissionID: globalTestPermission.ID,
				RoleID:       testRole.ID,
			},
			{
				PermissionID: testPermission.ID,
				RoleID:       testRole.ID,
			},
			{
				PermissionID: testPermission2.ID,
				RoleID:       testRole2.ID,
			},
			{
				PermissionID: testPermission3.ID,
				RoleID:       testRole2.ID,
			},
		}

		t.Cleanup(func() {
			// clean out role assignments
			err := RemoveRoleAssignments(ctx, groupRoleAssignments, nil)
			require.NoError(t, err, "error removing group role assignments during cleanup")

			// clean out permission assignments
			_, err = db.Bun().NewDelete().Model(&permissionAssignments).WherePK().Exec(ctx)
			require.NoError(t, err, "error removing permission assignments during cleanup")
		})

		err := AddRoleAssignments(ctx, groupRoleAssignments, nil)
		require.NoError(t, err, "error adding role assignments")

		_, err = db.Bun().NewInsert().Model(&permissionAssignments).Exec(ctx)
		require.NoError(t, err, "error adding permission assignments during setup")

		// Test for non-existent users
		permissions, err := UserPermissionsForScope(ctx, -9999, 0)
		require.NoError(t, err,
			"unexpected error from UserPermissionsForScope when non-existent user")
		require.Empty(t, permissions, "Expected empty permissions for non-existent user")

		// Test for scope-assigned role
		permissions, err = UserPermissionsForScope(ctx, testUser.ID, testWorkspace.ID)
		require.Len(t, permissions, 4, "Expected four permissions from %v", permissions)
		require.True(t, permissionsContainsAll(permissions,
			globalTestPermission.ID, testPermission.ID, testPermission2.ID, testPermission3.ID),
			"failed to find expected permissions for scope-assigned role in %v", permissions)

		// Test for globally assigned role
		permissions, err = UserPermissionsForScope(ctx, testUser.ID, 0)
		require.Len(t, permissions, 2, "Expected two permissions from %v", permissions)
		require.True(t, permissionsContainsAll(permissions, globalTestPermission.ID,
			testPermission.ID), "failed to find expected permissions in %v", permissions)
		require.False(t, permissionsContainsAll(permissions, testPermission2.ID),
			"Unexpectedly found permission %v for user in %v", testPermission2.ID, permissions)
		require.False(t, permissionsContainsAll(permissions, testPermission3.ID),
			"Unexpectedly found permission %v for user in %v", testPermission3.ID, permissions)
	})

	t.Run("test GetPermissionSummary", func(t *testing.T) {
		permissionsToAdd := []map[string]interface{}{
			{
				"permission_id": globalTestPermission.ID,
				"role_id":       testRole.ID,
			},
			{
				"permission_id": testPermission.ID,
				"role_id":       testRole.ID,
			},
		}
		for _, p := range permissionsToAdd {
			perm := p
			_, err := db.Bun().NewInsert().Model(&perm).TableExpr("permission_assignments").Exec(ctx)
			require.NoError(t, err, "failure inserting permission assignments in local setup")
		}

		summary, err := GetPermissionSummary(ctx, testUser.ID)
		require.NoError(t, err)
		require.Len(t, summary, 1)
		for k, v := range summary {
			// Ignore checking IDs since they are generated.
			for _, e := range v {
				e.ScopeID = 0
				e.Scope.ID = 0
			}
			sort.Slice(k.Permissions, func(i, j int) bool {
				return k.Permissions[i].ID < k.Permissions[j].ID
			})

			expectedRole := testRole
			expectedRole.Permissions = []Permission{
				testPermission,
				globalTestPermission,
			}
			require.Len(t, v, 1)
			require.Equal(t, *k, expectedRole)
			require.Equal(t, v[0], &RoleAssignment{
				GroupID: testGroupStatic.ID,
				RoleID:  testRole.ID,
				ScopeID: 0,
				Scope: &RoleAssignmentScope{
					ID: 0,
					WorkspaceID: sql.NullInt32{
						Valid: true,
						Int32: int32(testWorkspace.ID),
					},
				},
			})
		}
	})

	t.Run("testOnWorkspace", func(t *testing.T) {
		testOnWorkspace(ctx, t, pgDB)
	})
}

func setUp(ctx context.Context, t *testing.T, pgDB *db.PgDB) {
	_, err := pgDB.AddUser(&testUser, nil)
	require.NoError(t, err, "failure creating user in setup")

	_, _, err = usergroup.AddGroupWithMembers(ctx, testGroupStatic, testUser.ID)
	require.NoError(t, err, "failure creating static test group")

	err = db.Bun().NewSelect().Model(&testGroupOwnedByUser).
		Where("user_id = ?", testUser.ID).Scan(ctx)
	require.NoError(t, err, "failure getting test user personal group")

	_, err = db.Bun().NewInsert().Model(&testPermissions).Exec(ctx)
	require.NoError(t, err, "failure creating permission in setup")

	_, err = db.Bun().NewInsert().Model(&testRoles).Exec(ctx)
	require.NoError(t, err, "failure creating role in setup")

	workspace := map[string]interface{}{
		"name": testWorkspace.Name,
		"id":   testWorkspace.ID,
	}
	_, err = db.Bun().NewInsert().Model(&workspace).TableExpr("workspaces").Exec(ctx)
	require.NoError(t, err, "failure creating workspace in setup")
}

func cleanUp(ctx context.Context, t *testing.T, pgDB *db.PgDB) {
	_, err := db.Bun().NewDelete().Table("workspaces").Where(
		"name=?", "test workspace").Exec(ctx)
	if err != nil {
		t.Logf("Error cleaning up workspace")
	}

	_, err = db.Bun().NewDelete().Model(&testPermissions).WherePK().Exec(ctx)
	if err != nil {
		t.Logf("Error cleaning up permissions")
	}

	_, err = db.Bun().NewDelete().Table("roles").Where("id IN (?)",
		bun.In([]int32{
			int32(testRole.ID), int32(testRole2.ID),
			int32(testRole3.ID), int32(testRole4.ID),
		})).Exec(ctx)
	if err != nil {
		t.Logf("Error cleaning up role")
	}

	err = usergroup.RemoveUsersFromGroupTx(ctx, nil, testGroupStatic.ID, testUser.ID)
	if err != nil {
		t.Logf("Error cleaning up group membership on (%v, %v): %v", testGroupStatic.ID, testUser.ID, err)
	}

	_, err = db.Bun().NewDelete().Table("role_assignments").Where(
		"group_id=?", testGroupStatic.ID).Exec(ctx)
	if err != nil {
		t.Log("Error cleaning up static group from role assignment")
	}

	err = usergroup.DeleteGroup(ctx, testGroupStatic.ID)
	if err != nil {
		t.Logf("Error cleaning up static group: %v", err)
	}

	_, err = db.Bun().NewDelete().Table("users").Where("id = ?", testUser.ID).Exec(ctx)
	if err != nil {
		t.Logf("Error cleaning up user: %v\n", err)
	}
}

func compareRoles(expected, actual Role) bool {
	switch {
	case expected.ID != actual.ID:
		return false
	case expected.Name != actual.Name:
		return false
	case !expected.Created.Equal(actual.Created):
		return false
	}
	return true
}

func permissionsContainsAll(permissions []Permission, ids ...int) bool {
	foundIDs := make(map[int]bool)

	for _, id := range ids {
		for _, p := range permissions {
			if p.ID == id {
				foundIDs[id] = true
			}
		}
	}

	for _, id := range ids {
		if !foundIDs[id] {
			return false
		}
	}

	return true
}

func filterToTestRoles(rolesGotten []Role) []Role {
	var roles []Role
	for _, r := range rolesGotten {
		for _, n := range []Role{testRole, testRole2, testRole3, testRole4} {
			if r.Name == n.Name {
				r := r
				roles = append(roles, r)
			}
		}
	}
	return roles
}

func testOnWorkspace(ctx context.Context, t *testing.T, pgDB db.DB) {
	// Don't error if we pass a non-existent workspaceID.
	roles, err := GetRolesWithAssignmentsOnWorkspace(ctx, -999)
	require.NoError(t, err)
	require.Len(t, roles, 0)
	users, membership, err := GetUsersAndGroupMembershipOnWorkspace(ctx, -999)
	require.NoError(t, err)
	require.Len(t, users, 0)
	require.Len(t, membership, 0)

	// Create empty workspace.
	ws := struct {
		bun.BaseModel `bun:"table:workspaces"`
		ID            int `bun:"id,pk,autoincrement"`
		Name          string
	}{Name: uuid.New().String()}
	_, err = db.Bun().NewInsert().Model(&ws).Exec(ctx)
	require.NoError(t, err)

	// Don't error with workspace with no assignmnets.
	roles, err = GetRolesWithAssignmentsOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, roles, 0)
	users, membership, err = GetUsersAndGroupMembershipOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, users, 0)
	require.Len(t, membership, 0)

	// Add users and assignments.
	user0 := model.User{Username: uuid.New().String()}
	_, err = pgDB.AddUser(&user0, nil)
	user1 := model.User{Username: uuid.New().String()}
	_, err = pgDB.AddUser(&user1, nil)
	user2 := model.User{Username: uuid.New().String()}
	_, err = pgDB.AddUser(&user2, nil)
	require.NoError(t, err)
	require.NoError(t, AddRoleAssignments(ctx, nil,
		[]*rbacv1.UserRoleAssignment{
			{
				UserId: int32(user0.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role: &rbacv1.Role{
						RoleId: 2,
					},
					ScopeWorkspaceId: wrapperspb.Int32(int32(ws.ID)),
				},
			},
			{
				UserId: int32(user0.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role: &rbacv1.Role{
						RoleId: 5,
					},
					ScopeWorkspaceId: nil, // Global shouldn't show up.
				},
			},
			{
				UserId: int32(user0.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role: &rbacv1.Role{
						RoleId: 5,
					},
					ScopeWorkspaceId: wrapperspb.Int32(1), // Different workspace shouldn't show up.
				},
			},
			{
				UserId: int32(user1.ID),
				RoleAssignment: &rbacv1.RoleAssignment{
					Role: &rbacv1.Role{
						RoleId: 5,
					},
					ScopeWorkspaceId: wrapperspb.Int32(1), // Different workspace shouldn't show up.
				},
			},
		},
	))

	// Verify personal assignments work correctly.
	roles, err = GetRolesWithAssignmentsOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, roles, 1)
	require.Equal(t, 2, roles[0].ID)
	require.Len(t, roles[0].RoleAssignments, 1)
	require.Equal(t, user0.ID, roles[0].RoleAssignments[0].Group.OwnerID)
	require.Equal(t, int32(ws.ID), roles[0].RoleAssignments[0].Scope.WorkspaceID.Int32)

	users, membership, err = GetUsersAndGroupMembershipOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, user0.ID, users[0].ID)
	require.Len(t, membership, 0) // Personal groups don't show.

	// Add groups and group assignments.
	group0, _, err := usergroup.AddGroupWithMembers(ctx, usergroup.Group{Name: uuid.New().String()},
		user0.ID)
	require.NoError(t, err)
	group1, _, err := usergroup.AddGroupWithMembers(ctx, usergroup.Group{Name: uuid.New().String()},
		user0.ID, user1.ID)
	require.NoError(t, err)
	group2, _, err := usergroup.AddGroupWithMembers(ctx, usergroup.Group{Name: uuid.New().String()})
	require.NoError(t, err)
	group3, _, err := usergroup.AddGroupWithMembers(ctx, usergroup.Group{Name: uuid.New().String()},
		user2.ID)
	require.NoError(t, err)
	require.NoError(t, AddRoleAssignments(ctx, []*rbacv1.GroupRoleAssignment{
		{
			GroupId: int32(group0.ID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: 2,
				},
				ScopeWorkspaceId: wrapperspb.Int32(int32(ws.ID)),
			},
		},
		{
			GroupId: int32(group1.ID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: 5,
				},
				ScopeWorkspaceId: wrapperspb.Int32(int32(ws.ID)),
			},
		},
		{
			GroupId: int32(group2.ID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: 4,
				},
				ScopeWorkspaceId: wrapperspb.Int32(int32(ws.ID)),
			},
		},
		{
			GroupId: int32(group3.ID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: 2,
				},
				ScopeWorkspaceId: wrapperspb.Int32(1), // Shouldn't show up since different workspace.
			},
		},
		{
			GroupId: int32(group3.ID),
			RoleAssignment: &rbacv1.RoleAssignment{
				Role: &rbacv1.Role{
					RoleId: 2,
				},
				ScopeWorkspaceId: nil, // Global shouldn't be returned.
			},
		},
	}, nil))

	// Verify personal and group assignments work correctly.
	roles, err = GetRolesWithAssignmentsOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, roles, 3)
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })

	// Role 2 assignment should have an assignment to group0 and user0.
	require.Equal(t, 2, roles[0].ID)
	require.Len(t, roles[0].RoleAssignments, 2)
	sort.Slice(roles[0].RoleAssignments, func(i, j int) bool {
		return roles[0].RoleAssignments[i].GroupID < roles[0].RoleAssignments[j].GroupID
	})
	require.Equal(t, roles[0].RoleAssignments[0].Group.OwnerID, user0.ID)
	require.Equal(t, group0.ID, roles[0].RoleAssignments[1].Group.ID)
	require.Equal(t, roles[0].ID, roles[0].RoleAssignments[0].Role.ID)

	// Role 4 assignment should have an assignment to group2.
	require.Equal(t, 4, roles[1].ID)
	require.Len(t, roles[1].RoleAssignments, 1)
	require.Equal(t, group2.ID, roles[1].RoleAssignments[0].Group.ID)

	// Role 5 assignment should have an assignment to group1.
	require.Equal(t, 5, roles[2].ID)
	require.Len(t, roles[2].RoleAssignments, 1)
	require.Equal(t, group1.ID, roles[2].RoleAssignments[0].Group.ID)

	users, membership, err = GetUsersAndGroupMembershipOnWorkspace(ctx, ws.ID)
	require.NoError(t, err)
	require.Len(t, users, 2)
	sort.Slice(users, func(i, j int) bool { return users[i].ID < users[j].ID })
	require.Equal(t, user0.ID, users[0].ID)
	require.Equal(t, user1.ID, users[1].ID)
	require.ElementsMatch(t, membership, []usergroup.GroupMembership{
		{UserID: user0.ID, GroupID: group0.ID},
		{UserID: user0.ID, GroupID: group1.ID},
		{UserID: user1.ID, GroupID: group1.ID},
	})
}
