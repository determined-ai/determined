package user

import (
	"context"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
)

// UserAuthZRBAC is the RBAC implementation of user authorization.
type UserAuthZRBAC struct{}

func canAdministrateUser(ctx context.Context, curUserID model.UserID) error {
	return db.DoesPermissionMatch(ctx, curUserID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
}

// CanGetUser always returns true.
func (a *UserAuthZRBAC) CanGetUser(
	ctx context.Context, curUser, targetUser model.User,
) (canGetUser bool, serverError error) {
	return true, nil
}

// FilterUserList always returns the input user list and does not filtering.
func (a *UserAuthZRBAC) FilterUserList(
	ctx context.Context, curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	return users, nil
}

// CanCreateUser returns an error if the user does not have admin permissions or
// does not have permission to update groups.
func (a *UserAuthZRBAC) CanCreateUser(
	ctx context.Context, curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
}

// CanSetUsersPassword returns an error if the user is not the target user and does not have admin
// permissions when trying to set another user's password.
func (a *UserAuthZRBAC) CanSetUsersPassword(
	ctx context.Context, curUser, targetUser model.User,
) error {
	err := db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("only admin privileged users can change other user's passwords")
	}
	return nil
}

// CanSetUsersActive returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersActive(
	ctx context.Context, curUser, targetUser model.User, toActiveVal bool,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersAdmin returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersAdmin(
	ctx context.Context, curUser, targetUser model.User, toAdminVal bool,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersAgentUserGroup returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersAgentUserGroup(
	ctx context.Context, curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersUsername returns an error if the user does not have admin permissions.
func (a *UserAuthZRBAC) CanSetUsersUsername(
	ctx context.Context, curUser, targetUser model.User,
) error {
	return canAdministrateUser(ctx, curUser.ID)
}

// CanSetUsersDisplayName returns an error if the user is not an admin and does not have admin
// permissions when trying to set another user's display name.
func (a *UserAuthZRBAC) CanSetUsersDisplayName(
	ctx context.Context, curUser, targetUser model.User,
) error {
	if curUser == targetUser {
		return nil
	}

	err := db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
	if err != nil && curUser.ID != targetUser.ID {
		return errors.New("only admin privileged users can set another user's display name")
	}
	return nil
}

// CanGetUsersImage always returns nil.
func (a *UserAuthZRBAC) CanGetUsersImage(
	ctx context.Context, curUser, targetUser model.User,
) error {
	return nil
}

// CanGetUsersOwnSettings always returns nil.
func (a *UserAuthZRBAC) CanGetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	return nil
}

// CanCreateUsersOwnSetting always returns nil.
func (a *UserAuthZRBAC) CanCreateUsersOwnSetting(
	ctx context.Context, curUser model.User, setting model.UserWebSetting,
) error {
	return nil
}

// CanResetUsersOwnSettings always returns nil.
func (a *UserAuthZRBAC) CanResetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	return nil
}

// CanGetActiveTasksCount returns an error if a user can't administrate users.
func (a *UserAuthZRBAC) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	return db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER)
}

// CanAccessNTSCTask returns false if a user can't administrate users and it is not their task.
func (a *UserAuthZRBAC) CanAccessNTSCTask(
	ctx context.Context, curUser model.User, ownerID model.UserID,
) (bool, error) {
	if curUser.ID == ownerID {
		return true, nil
	}
	if err := db.DoesPermissionMatch(ctx, curUser.ID, nil,
		rbacv1.PermissionType_PERMISSION_TYPE_ADMINISTRATE_USER); err != nil {
		if _, ok := err.(authz.PermissionDeniedError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func init() {
	AuthZProvider.Register("rbac", &UserAuthZRBAC{})
}
