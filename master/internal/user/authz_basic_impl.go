package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZBasic is basic.
type UserAuthZBasic struct{}

func (a *UserAuthZBasic) CanGetUser(curUser, targetUser model.User) bool {
	return true
}

func (a *UserAuthZBasic) FilterUserList(
	curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	return users, nil
}

func (a *UserAuthZBasic) CanCreateUser(
	curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can create users")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersPassword(curUser, targetUser model.User) error {
	if !curUser.Admin && curUser.ID != targetUser.ID {
		return fmt.Errorf("only admin privileged users can change other user's passwords")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersActive(curUser, targetUser model.User, toActiveVal bool) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersAdmin(curUser, targetUser model.User, toAdminVal bool) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersAgentUserGroup(
	curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersUsername(curUser, targetUser model.User) error {
	if !curUser.Admin {
		return fmt.Errorf("only admin privileged users can update users")
	}
	return nil
}

func (a *UserAuthZBasic) CanSetUsersDisplayName(curUser, targetUser model.User) error {
	if !curUser.Admin && curUser.ID != targetUser.ID {
		return fmt.Errorf("only admin privileged users can set another user's display name")
	}
	return nil
}

func (a *UserAuthZBasic) CanGetUsersImage(curUser, targetUser model.User) error {
	return nil
}

func (a *UserAuthZBasic) CanGetUsersOwnSettings(curUser model.User) error {
	return nil
}

func (a *UserAuthZBasic) CanCreateUsersOwnSetting(
	curUser model.User, setting model.UserWebSetting,
) error {
	return nil
}

func (a *UserAuthZBasic) CanResetUsersOwnSettings(curUser model.User) error {
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
