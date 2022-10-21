package usergroup

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// UserGroupAuthZRBAC is the RBAC implementation.
type UserGroupAuthZRBAC struct{}

// CanGetGroup checks if a user can view a given group.
func (a *UserGroupAuthZRBAC) CanGetGroup(ctx context.Context, curUser model.User, gid int) (
	bool, error,
) {
	err := CanViewGroup(ctx, curUser.ID, gid)
	if err == nil {
		return true, nil
	} else if _, ok := err.(authz.PermissionDeniedError); ok {
		return false, nil
	}
	return false, err
}

// FilterGroupsList returns the list it was given and a nil error.
func (a *UserGroupAuthZRBAC) FilterGroupsList(ctx context.Context, curUser model.User,
	query *bun.SelectQuery,
) (*bun.SelectQuery, error) {
	err := db.DoPermissionsExist(ctx, curUser.ID,
		rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
	if err == nil {
		return query, nil
	} else if _, ok := err.(authz.PermissionDeniedError); !ok {
		return query, err
	}

	query = query.Where(
		`EXISTS(SELECT 1
			FROM user_group_membership AS m
			WHERE m.group_id=groups.id AND m.user_id = ?)`,
		curUser.ID)

	return query, nil
}

// CanUpdateGroups checks if a user can create, delete, or update groups.
func (a *UserGroupAuthZRBAC) CanUpdateGroups(ctx context.Context, curUser model.User) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_UPDATE_GROUP)
}

// CanViewGroup checks if a user has the ability to view the group by checking whether
// user has the assign roles permission or belongs to the group.
func CanViewGroup(ctx context.Context, userBelongsTo model.UserID, gid int) error {
	err := db.DoPermissionsExist(ctx, userBelongsTo,
		rbacv1.PermissionType_PERMISSION_TYPE_ASSIGN_ROLES)
	if err == nil {
		return nil
	} else if _, ok := err.(authz.PermissionDeniedError); !ok {
		return err
	}

	exists, err := db.Bun().NewSelect().Table("groups").
		Join("user_group_membership ugm ON ugm.group_id = groups.ID").
		Where("ugm.user_id = ?", userBelongsTo).
		Where("groups.ID = ?", gid).
		Exists(ctx)
	if err != nil {
		return err
	} else if !exists {
		return authz.PermissionDeniedError{RequiredPermissions: []rbacv1.PermissionType{}}
	}
	return nil
}

func init() {
	AuthZProvider.Register("rbac", &UserGroupAuthZRBAC{})
}
