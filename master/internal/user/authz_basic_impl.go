package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZBasic is basic.
type UserAuthZBasic struct{}

// CanSetUserPassword for basic authz.
func (a *UserAuthZBasic) CanSetUserPassword(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin && currentUser.ID != targetUser.ID {
		return fmt.Errorf("non-admin users can only change their own password")
	}
	return nil
}

// CanSetUserActive for basic authz.
func (a *UserAuthZBasic) CanSetUserActive(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return fmt.Errorf("only admin can activate/deactivate user")
	}
	return nil
}

// CanSetUserAdmin for basic authz.
func (a *UserAuthZBasic) CanSetUserAdmin(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return fmt.Errorf("only admin can change user from/to admin")
	}
	return nil
}

// CanSetUserAgentGroup for basic authz.
func (a *UserAuthZBasic) CanSetUserAgentGroup(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin {
		return fmt.Errorf("only admin can set user agent group")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
