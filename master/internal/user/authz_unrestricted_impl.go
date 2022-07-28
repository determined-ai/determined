package user

import (
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZUnrestricted is unrestricted.
// TODO XXX: this is for demo purposes only. Remove before merging to master branch.
type UserAuthZUnrestricted struct{}

// CanSetUserPassword for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUserPassword(
	currentUser model.User, targetUser model.User,
) error {
	return nil
}

// CanSetUserDisplayName for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUserDisplayName(
	currentUser model.User, targetUser model.User,
) error {
	return nil
}

// CanSetUserActive for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUserActive(
	currentUser model.User, targetUser model.User,
) error {
	return nil
}

// CanSetUserAdmin for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUserAdmin(
	currentUser model.User, targetUser model.User,
) error {
	return nil
}

// CanSetUserAgentGroup for unresticted authz.
func (a *UserAuthZUnrestricted) CanSetUserAgentGroup(
	currentUser model.User, targetUser model.User,
) error {
	return nil
}

func init() {
	AuthZProvider.Register("unrestricted", &UserAuthZUnrestricted{})
}
