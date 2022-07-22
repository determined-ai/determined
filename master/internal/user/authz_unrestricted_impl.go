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

func init() {
	AuthZProvider.Register("unrestricted", &UserAuthZUnrestricted{})
}
