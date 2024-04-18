//go:build integration
// +build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

var (
	iters   = 5
	maxWsID = float64(-1)
)

var (
	userModelViewer           model.User
	userModelEditor           model.User
	userModelEditorRestricted model.User
)

var roles = map[string]int{
	"ClusterAdmin":        1,
	"WorkspaceAdmin":      2,
	"WorkspaceCreator":    3,
	"Viewer":              4,
	"Editor":              5,
	"ModelRegistryViewer": 6,
	"EditorRestricted":    7,
}

var wsIDs,
	viewerGroupIDs,
	editorGroupIDs,
	editorRestrictedGroupIDs []int

func setup(t *testing.T, pgDB *PgDB) {
	ctx := context.TODO()
	raData := map[string]interface{}{}

	// Create workspaces and groups with viewer, editor, and editor-restricted privileges in
	// each workspace.
	for i := 0; i < iters; i++ {
		nameExt, err := uuid.NewRandom()
		require.NoError(t, err)

		viewerGroupName := fmt.Sprintf("test_group_viewer_%s", nameExt)
		editorGroupName := fmt.Sprintf("test_group_editor_%s", nameExt)
		editorRestrictedGroupName := fmt.Sprintf("test_group_editor_restricted_%s",
			nameExt)

		wsName := fmt.Sprintf("test_workspace_permissions_%s", nameExt)
		wsID, _ := RequireMockWorkspaceID(t, pgDB, wsName)
		wsIDs = append(wsIDs, wsID)
		maxWsID = math.Max(float64(wsID), maxWsID)

		wsIDSql := sql.NullInt32{Int32: int32(wsID), Valid: true}
		ras := &model.RoleAssignmentScope{WorkspaceID: wsIDSql}
		_, err = Bun().NewInsert().Model(ras).Returning("id").Exec(ctx)
		require.NoError(t, err, "error inserting role assignment scopes")

		scopeID := ras.ID

		grp := &model.Group{Name: viewerGroupName}
		_, err = Bun().NewInsert().Model(grp).Returning("id").Exec(ctx)
		require.NoError(t, err, "error inserting viewer group")
		viewerGroupIDs = append(viewerGroupIDs, grp.ID)

		raData["group_id"] = grp.ID
		raData["role_id"] = roles["Viewer"]
		raData["scope_id"] = scopeID
		_, err = Bun().NewInsert().Model(&raData).Table("role_assignments").Exec(ctx)
		require.NoError(t, err, "error inserting viewer role assignment")

		grp = &model.Group{Name: editorGroupName}
		_, err = Bun().NewInsert().Model(grp).Returning("id").Exec(ctx)
		require.NoError(t, err, "error inserting editor group")
		editorGroupIDs = append(editorGroupIDs, grp.ID)

		raData["group_id"] = grp.ID
		raData["role_id"] = roles["Editor"]
		raData["scope_id"] = scopeID
		_, err = Bun().NewInsert().Model(&raData).Table("role_assignments").Exec(ctx)
		require.NoError(t, err, "error inserting editor role assignment")

		grp = &model.Group{Name: editorRestrictedGroupName}
		_, err = Bun().NewInsert().Model(grp).Returning("id").Exec(ctx)
		require.NoError(t, err, "error inserting editor-restricted group")
		editorRestrictedGroupIDs = append(editorRestrictedGroupIDs, grp.ID)

		raData["group_id"] = grp.ID
		raData["role_id"] = roles["EditorRestricted"]
		raData["scope_id"] = scopeID
		_, err = Bun().NewInsert().Model(&raData).Table("role_assignments").Exec(ctx)
		require.NoError(t, err, "error inserting editor-restricted role assignment")
	}

	// Create 3 users and add each user to a group with viewer, editor, or editor-restricted
	// privileges within their respective workspaces.
	userModelViewer = model.User{Username: uuid.New().String(), Active: true}
	_, err := HackAddUser(context.TODO(), &userModelViewer)
	require.NoError(t, err)

	userModelEditor = model.User{Username: uuid.New().String(), Active: true}
	_, err = HackAddUser(context.TODO(), &userModelEditor)
	require.NoError(t, err)

	userModelEditorRestricted = model.User{Username: uuid.New().String(), Active: true}
	_, err = HackAddUser(context.TODO(), &userModelEditorRestricted)
	require.NoError(t, err)

	for i := range []int{0, 1, 2} {
		viewerGID := viewerGroupIDs[i]
		groupMembership := map[string]interface{}{
			"user_id":  userModelViewer.ID,
			"group_id": viewerGID,
		}
		_, err = Bun().NewInsert().Model(&groupMembership).Table("user_group_membership").
			Exec(ctx)
		require.NoError(t, err, "error inserting user group membership "+
			strconv.Itoa(viewerGID))

		editorGID := editorGroupIDs[i]
		groupMembership["user_id"] = userModelEditor.ID
		groupMembership["group_id"] = editorGID

		_, err = Bun().NewInsert().Model(&groupMembership).Table("user_group_membership").
			Exec(ctx)
		require.NoError(t, err, "error inserting user group membership "+
			strconv.Itoa(editorGID))

		editorRestrictedGID := editorRestrictedGroupIDs[i]
		groupMembership["user_id"] = userModelEditorRestricted.ID
		groupMembership["group_id"] = editorRestrictedGID

		_, err = Bun().NewInsert().Model(&groupMembership).Table("user_group_membership").
			Exec(ctx)
		require.NoError(t, err, "error inserting user group membership "+
			strconv.Itoa(editorRestrictedGID))
	}
}

func cleanUp(t *testing.T) {
	ctx := context.TODO()

	// Remove users.
	_, err := Bun().NewDelete().Table("users").Where("id = ?", userModelViewer.ID).Exec(ctx)
	require.NoError(t, err)

	_, err = Bun().NewDelete().Table("users").Where("id = ?", userModelEditor.ID).Exec(ctx)
	require.NoError(t, err)

	_, err = Bun().NewDelete().Table("users").Where("id = ?", userModelEditorRestricted.ID).
		Exec(ctx)
	require.NoError(t, err)

	// Remove workspaces.
	_, err = Bun().NewDelete().Table("workspaces").Where("id IN (?)", bun.In(wsIDs)).Exec(ctx)
	require.NoError(t, err, "error cleaning up workspace")

	// Remove groups.
	_, err = Bun().NewDelete().Table("groups").Where("id IN (?)", bun.In(viewerGroupIDs)).Exec(ctx)
	require.NoError(t, err, "error deleting viewer groups")

	_, err = Bun().NewDelete().Table("groups").Where("id IN (?)", bun.In(editorGroupIDs)).Exec(ctx)
	require.NoError(t, err, "error deleting editor groups")

	_, err = Bun().NewDelete().Table("groups").Where("id IN (?)", bun.In(editorRestrictedGroupIDs)).
		Exec(ctx)
	require.NoError(t, err, "error deleting editor-restricted groups")
}

func TestPermissionMatch(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	t.Cleanup(func() { cleanUp(t) })
	setup(t, pgDB)
	userIDViewer := userModelViewer.ID
	userIDEditor := userModelEditor.ID
	userIDEditorRestricted := userModelEditorRestricted.ID

	badWorkspaceID := int32(maxWsID) + 10

	t.Run("test DoesPermissionMatch", func(t *testing.T) {
		workspaceID := int32(wsIDs[0])
		err := DoesPermissionMatch(ctx, userIDViewer, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		require.NoError(t, err, "error when searching for permissions")

		err = DoesPermissionMatch(ctx, userIDViewer, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update experiments")

		err = DoesPermissionMatch(ctx, userIDEditor, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.NoError(t, err)

		// Verify that the user assigned to the groups with EditorRestricted privileges within the
		//  given workspace cannot create or update NSC tasks, while the user assigned to groups
		// with Editor privileges within the given workspace can create and update NSC tasks.
		err = DoesPermissionMatch(ctx, userIDEditorRestricted, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update NSC tasks")

		err = DoesPermissionMatch(ctx, userIDEditorRestricted, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to create NSC tasks")

		err = DoesPermissionMatch(ctx, userIDEditor, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
		require.NoError(t, err)

		err = DoesPermissionMatch(ctx, userIDEditor, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
		require.NoError(t, err)

		err = DoesPermissionMatch(ctx, userIDViewer, &badWorkspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		require.IsType(t, authz.PermissionDeniedError{}, err, "workspace should not exist")
	})

	t.Run("test DoesPermissionMatchAll single input", func(t *testing.T) {
		workspaceID := int32(wsIDs[0])
		err := DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceID)
		require.NoError(t, err, "error when searching for permissions")

		err = DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceID)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update experiments")

		err = DoesPermissionMatchAll(ctx, userIDEditor,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC, workspaceID)
		require.NoError(t, err)

		err = DoesPermissionMatchAll(ctx, userIDEditorRestricted,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC, workspaceID)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update experiments")
	})

	t.Run("test DoesPermissionMatchAll multiple inputs no failure", func(t *testing.T) {
		workspaceIDs := []int32{int32(wsIDs[0]), int32(wsIDs[1]), int32(wsIDs[2])}
		err := DoesPermissionMatchAll(ctx, userIDEditor,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC, workspaceIDs...)
		require.NoError(t, err, "error when searching for permissions")

		err = DoesPermissionMatchAll(ctx, userIDEditor,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC, workspaceIDs...)
		require.NoError(t, err, "error when searching for permissions")
	})

	t.Run("test DoesPermissionMatchAll multiple inputs single failure", func(t *testing.T) {
		workspaceIDs := []int32{int32(wsIDs[0]), int32(wsIDs[1]), int32(wsIDs[2])}

		err := DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceIDs...)
		require.NoError(t, err, "error when searching for permissions")

		workspaceIDs = []int32{int32(wsIDs[0]), badWorkspaceID}
		err = DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		workspaceIDs = []int32{int32(wsIDs[0]), badWorkspaceID}
		err = DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		workspaceIDs = []int32{int32(wsIDs[0]), int32(wsIDs[1])}
		err = DoesPermissionMatchAll(ctx, userIDEditorRestricted,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})

	t.Run("test DoesPermissionMatchAll multiple failures", func(t *testing.T) {
		workspaceIDs := []int32{badWorkspaceID, int32(wsIDs[1]), int32(wsIDs[2])}
		err := DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		workspaceIDs = []int32{int32(wsIDs[0]), int32(wsIDs[1]), int32(wsIDs[3])}
		err = DoesPermissionMatchAll(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})

	t.Run("test DoesPermissionExist", func(t *testing.T) {
		err := DoPermissionsExist(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		require.NoError(t, err, "error when checking if permission exists in any workspace")

		err = DoPermissionsExist(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		err = DoPermissionsExist(ctx, userIDEditor,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.NoError(t, err)

		err = DoPermissionsExist(ctx, userIDEditorRestricted,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_NSC)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})

	t.Run("test GetWorkspacesWithPermission", func(t *testing.T) {
		workspaceIDs := []int{wsIDs[0], wsIDs[1], wsIDs[2]}
		var noWorkspaces []int
		workspaces, err := GetNonGlobalWorkspacesWithPermission(ctx, userIDViewer,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_NSC)
		require.NoError(t, err, "error when searching for permissions")
		require.Equal(t, workspaceIDs, workspaces)

		workspaces, err = GetNonGlobalWorkspacesWithPermission(ctx, userIDEditorRestricted,
			rbacv1.PermissionType_PERMISSION_TYPE_CREATE_NSC)
		require.NoError(t, err, "error when searching for permissions")
		require.Equal(t, noWorkspaces, workspaces)
	})
}

func TestEditorVSEditorRestricted(t *testing.T) {
	// Verify that the EditorRestricted role only has two less permissions than Editor and that
	// it does not have the create or update notebooks/shells/commands permissions.
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	numEditorRestrictedPermissions, err := Bun().NewSelect().Table("permission_assignments").
		Where("role_id = ?", roles["EditorRestricted"]).Count(ctx)
	require.NoError(t, err)

	numEditorPermissions, err := Bun().NewSelect().Table("permission_assignments").
		Where("role_id = ?", roles["Editor"]).Count(ctx)
	require.NoError(t, err)

	require.Equal(t, numEditorPermissions-2, numEditorRestrictedPermissions)

	// Verify that EditorRestricted role does not have create/update nsc permissions
	num, err := Bun().NewSelect().Table("permission_assignments").
		Where("role_id = ?", roles["EditorRestricted"]).
		Where("permission_id = 3001 OR permission_id = 3003").
		Count(ctx)
	require.NoError(t, err)

	require.Equal(t, 0, num)
}
