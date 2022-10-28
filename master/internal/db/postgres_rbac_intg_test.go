//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

var (
	offset       = 1000000
	editorOffset = 1100000
	iters        = 10
)

var userModel model.User

func setup(t *testing.T, pgDB *PgDB) {
	userModel = model.User{Username: uuid.New().String(), Active: true}
	_, err := pgDB.AddUser(&userModel, nil)
	require.NoError(t, err)

	ctx := context.TODO()
	workspaceData := map[string]interface{}{}
	rasData := map[string]interface{}{}
	groupData := map[string]interface{}{}
	raData := map[string]interface{}{}

	for i := 0; i < iters; i++ {
		constantID := fmt.Sprintf("%d", offset+i)
		editorID := fmt.Sprintf("%d", editorOffset+i)
		viewerGroupName := fmt.Sprintf("test_group_viewer_%s", constantID)
		editorGroupName := fmt.Sprintf("test_group_editor_%s", editorID)

		workspaceData["id"] = constantID
		workspaceData["name"] = fmt.Sprintf("test_workspace_permissions_%s", constantID)
		_, err = Bun().NewInsert().Model(&workspaceData).TableExpr("workspaces").Exec(ctx)
		require.NoError(t, err, "error inserting workspace")

		rasData["id"] = constantID
		rasData["scope_workspace_id"] = constantID
		_, err = Bun().NewInsert().Model(&rasData).TableExpr("role_assignment_scopes").Exec(ctx)
		require.NoError(t, err, "error inserting role assignment scopes")

		groupData["id"] = constantID
		groupData["group_name"] = viewerGroupName
		_, err = Bun().NewInsert().Model(&groupData).TableExpr("groups").Exec(ctx)
		require.NoError(t, err, "error inserting viewer group")

		groupData["id"] = editorID
		groupData["group_name"] = editorGroupName
		_, err = Bun().NewInsert().Model(&groupData).TableExpr("groups").Exec(ctx)
		require.NoError(t, err, "error inserting editor group")

		raData["group_id"] = constantID
		raData["role_id"] = "4"
		raData["scope_id"] = constantID
		_, err = Bun().NewInsert().Model(&raData).TableExpr("role_assignments").Exec(ctx)
		require.NoError(t, err, "error inserting viewer role assignment")

		raData["group_id"] = editorID
		raData["role_id"] = "5"
		raData["scope_id"] = constantID
		_, err = Bun().NewInsert().Model(&raData).TableExpr("role_assignments").Exec(ctx)
		require.NoError(t, err, "serror inserting editor role assignment")
	}

	groupMembership := map[string]interface{}{"user_id": userModel.ID, "group_id": "1000000"}
	_, err = Bun().NewInsert().Model(&groupMembership).TableExpr("user_group_membership").
		Exec(ctx)
	require.NoError(t, err, "error inserting user group membership 1000000")

	groupMembership = map[string]interface{}{"user_id": userModel.ID, "group_id": "1000001"}
	_, err = Bun().NewInsert().Model(&groupMembership).TableExpr("user_group_membership").Exec(ctx)
	require.NoError(t, err, "error inserting user group membership 1000001")

	groupMembership = map[string]interface{}{"user_id": userModel.ID, "group_id": "1000002"}
	_, err = Bun().NewInsert().Model(&groupMembership).TableExpr("user_group_membership").Exec(ctx)
	require.NoError(t, err, "error inserting user group membership 1000002")
}

func cleanUp(t *testing.T) {
	ctx := context.TODO()
	constantIDs := []int{}
	editorIDs := []int{}

	for i := 0; i < iters; i++ {
		constantIDs = append(constantIDs, offset+i)
		editorIDs = append(editorIDs, editorOffset+i)
	}

	_, err := Bun().NewDelete().Table("users").Where("id = ?", userModel.ID).Exec(ctx)
	require.NoError(t, err)

	_, err = Bun().NewDelete().Table("workspaces").Where("id IN (?)",
		bun.In(constantIDs)).Exec(ctx)
	require.NoError(t, err, "error cleaning up workspace")

	_, err = Bun().NewDelete().Table("groups").Where("id IN (?)", bun.In(constantIDs)).Exec(ctx)
	require.NoError(t, err, "error deleting viewer groups")

	_, err = Bun().NewDelete().Table("groups").Where("id IN (?)", bun.In(editorIDs)).Exec(ctx)
	require.NoError(t, err, "error deleting editor groups")
}

func TestPermissionMatch(t *testing.T) {
	ctx := context.Background()
	pgDB := MustResolveTestPostgres(t)
	MustMigrateTestPostgres(t, pgDB, MigrationsFromDB)

	t.Cleanup(func() { cleanUp(t) })
	setup(t, pgDB)
	userID := userModel.ID

	t.Run("test DoesPermissionMatch", func(t *testing.T) {
		workspaceID := int32(1000000)
		err := DoesPermissionMatch(ctx, userID, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		require.NoError(t, err, "error when searching for permissions")

		err = DoesPermissionMatch(ctx, userID, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update experiments")

		workspaceID = int32(99999)
		err = DoesPermissionMatch(ctx, userID, &workspaceID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)

		require.IsType(t, authz.PermissionDeniedError{}, err, "workspace should not exist")
	})

	t.Run("test DoesPermissionMatchAll", func(t *testing.T) {
		var workspaceID int32 = 1000000
		err := DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceID)
		require.NoError(t, err, "error when searching for permissions")

		err = DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceID)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"user should not have permission to update experiments")
	})

	t.Run("test DoesPermissionMatchAll multiple inputs single failure", func(t *testing.T) {
		workspaceIDs := []int32{1000000, 1000001, 1000002}
		err := DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceIDs...)
		require.NoError(t, err, "error when searching for permissions")

		workspaceIDs = []int32{1000000, 999999}
		err = DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		workspaceIDs = []int32{1000000, 1000011}
		err = DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})

	t.Run("test DoesPermissionMatchAll multiple failures", func(t *testing.T) {
		workspaceIDs := []int32{99999, 1000001, 1000002}
		err := DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")

		workspaceIDs = []int32{1000000, 1000001, 1000003}
		err = DoesPermissionMatchAll(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT, workspaceIDs...)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})

	t.Run("test DoesPermissionExist", func(t *testing.T) {
		err := DoPermissionsExist(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA)
		require.NoError(t, err, "error when checking if permission exists in any workspace")

		err = DoPermissionsExist(ctx, userID,
			rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_EXPERIMENT)
		require.IsType(t, authz.PermissionDeniedError{}, err,
			"error should have been returned when searching for permissions")
	})
}
