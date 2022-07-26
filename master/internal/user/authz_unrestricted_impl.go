package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZUnrestricted is unrestricted.
// TODO XXX: this is for demo purposes only. Remove before merging to master branch.
type UserAuthZUnrestricted struct {
	AlwaysAllow bool
}

// FilterUseList for unresticted authz.
func (a *UserAuthZUnrestricted) FilterUserList(
	curUser model.User, users []model.FullUser,
) ([]model.FullUser, error) {
	if !a.AlwaysAllow {
		return nil, fmt.Errorf("FilterUserList DENY")
	}
	return users, nil
}

// CanCreateUser for unresticted authz.
func (a *UserAuthZUnrestricted) CanCreateUser(
	curUser model.User, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanCreateUser DENY")
	}
	return nil
}

// CanGetMe for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetMe(curUser model.User) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanGetMe DENY")
	}
	return nil
}

// CanSetUserPassword for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersPassword(
	currentUser model.User, targetUser model.User,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanSetUserPassword DENY")
	}
	return nil
}

// CanSetUsersActive for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersActive(
	curUser, userToAdd model.User, toActiveVal bool,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanSetUsersActive DENY")
	}
	return nil
}

// CanSetUsersAdmin for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersAdmin(
	curUser, userToAdd model.User, toAdminVal bool,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanSetUsersAdmin DENY")
	}
	return nil
}

// CanSetUsersAgentUserGroup for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersAgentUserGroup(
	curUser, userToAdd model.User, agentUserGroup model.AgentUserGroup,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanSetUsersAgentUserGroup DENY")
	}
	return nil
}

// CanSetUsersUsername for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUsersUsername(
	curUser, targetUser model.User,
) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanSetUsersUsername DENY")
	}
	return nil
}

// CanGetUsersImage for unresticted authz.
func (a *UserAuthZUnrestricted) CanGetUsersImage(curUser model.User, targetUsername string) error {
	if !a.AlwaysAllow {
		return fmt.Errorf("CanGetUsersImage DENY")
	}
	return nil
}

func init() {
	AuthZProvider.Register("unrestricted", &UserAuthZUnrestricted{AlwaysAllow: true})
	AuthZProvider.Register("restricted", &UserAuthZUnrestricted{AlwaysAllow: false})
}
