package user

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZBasic is basic OSS controls.
type UserAuthZBasic struct{}

// CanGetUser always returns nil.
func (a *UserAuthZBasic) CanGetUser(
	ctx context.Context, curUser, targetUser model.User,
) error {
	return nil
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

// CanSetUsersRemote returns an error if the user is not an admin.
func (a *UserAuthZBasic) CanSetUsersRemote(ctx context.Context, curUser model.User) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update other users")
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
	ctx context.Context, curUser model.User, settings []model.UserWebSetting,
) error {
	return nil
}

// CanResetUsersOwnSettings always returns nil.
func (a *UserAuthZBasic) CanResetUsersOwnSettings(ctx context.Context, curUser model.User) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
