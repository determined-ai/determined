package user

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZBasic is basic OSS controls.
type UserAuthZBasic struct{}

// CanGetUser always returns true.
func (a *UserAuthZBasic) CanGetUser(
	ctx context.Context, curUser, targetUser model.User,
) (canGetUser bool, serverError error) {
	return true, nil
}

// FilterUserList always returns the input user list and does not filtering.
func (a *UserAuthZBasic) FilterUserList(
	ctx context.Context, curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	return users, nil
}

// CanCreateUser returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanCreateUser(
	ctx context.Context, curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can create users")
	}
	return nil
}

// CanSetUsersPassword returns an error if the user is not an admin
// when trying to set another user's password.
func (a *UserAuthZBasic) CanSetUsersPassword(
	ctx context.Context, curUser, targetUser model.User,
) error {
	if !curUser.Admin && curUser.ID != targetUser.ID {
		return fmt.Errorf("only admin privileged users can change other user's passwords")
	}
	return nil
}

// CanSetUsersActive returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanSetUsersActive(
	ctx context.Context, curUser, targetUser model.User, toActiveVal bool,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

// CanSetUsersAdmin returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanSetUsersAdmin(
	ctx context.Context, curUser, targetUser model.User, toAdminVal bool,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

// CanSetUsersAgentUserGroup returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanSetUsersAgentUserGroup(
	ctx context.Context, curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

// CanSetUsersUsername returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanSetUsersUsername(
	ctx context.Context, curUser, targetUser model.User,
) error {
	if !curUser.Admin && curUser.ID != targetUser.ID {
		return fmt.Errorf("only admin privileged users can update other users")
	}
	return nil
}

// CanSetUsersDisplayName returns an error if the user is not an admin
// when trying to set another user's display name.
func (a *UserAuthZBasic) CanSetUsersDisplayName(
	ctx context.Context, curUser, targetUser model.User,
) error {
	if !curUser.Admin && curUser.ID != targetUser.ID {
		return fmt.Errorf("only admin privileged users can set another user's display name")
	}
	return nil
}

// CanGetUsersImage always returns nil.
func (a *UserAuthZBasic) CanGetUsersImage(
	ctx context.Context, curUser, targetUser model.User,
) error {
	return nil
}

// CanGetUsersOwnSettings always returns nil.
func (a *UserAuthZBasic) CanGetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	return nil
}

// CanCreateUsersOwnSetting always returns nil.
func (a *UserAuthZBasic) CanCreateUsersOwnSetting(
	ctx context.Context, curUser model.User, setting model.UserWebSetting,
) error {
	return nil
}

// CanResetUsersOwnSettings always returns nil.
func (a *UserAuthZBasic) CanResetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	return nil
}

// CanGetActiveTasksCount always returns a nil error.
func (a *UserAuthZBasic) CanGetActiveTasksCount(ctx context.Context, curUser model.User) error {
	return nil
}

// CanAccessNTSCTask returns true and nil error unless the developer master config option
// security.authz._strict_ntsc_enabled is true then it returns a boolean if the user is
// an admin or if the user owns the task and a nil error.
func (a *UserAuthZBasic) CanAccessNTSCTask(
	ctx context.Context, curUser model.User, ownerID model.UserID,
) (bool, error) {
	if !config.GetMasterConfig().Security.AuthZ.StrictNTSCEnabled {
		return true, nil
	}
	return curUser.Admin || curUser.ID == ownerID, nil
}

// CanSetUsersOwnActivity always returns nil.
func (a *UserAuthZBasic) CanSetUsersOwnActivity(ctx context.Context, curUser model.User) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
