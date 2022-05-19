package user

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/model"
)

type UserAuthZBasic struct {
}

func (a *UserAuthZBasic) CanSetUserPassword(currentUser model.User, targetUser model.User) error {
	if !currentUser.Admin && currentUser.ID != targetUser.ID {
		return fmt.Errorf("non-admin users can only change their own password")
	}
	return nil
}

func init() {
	AuthZProvider.Register("basic", &UserAuthZBasic{})
}
