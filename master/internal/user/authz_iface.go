package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZ describes authz methods for `user` package.
type UserAuthZ interface {
	CanSetUserPassword(currentUser model.User, targetUser model.User) error
	CanSetUserDisplayName(currentUser model.User, targetUser model.User) error
	CanSetUserActive(currentUser model.User, targetUser model.User) error
	CanSetUserAdmin(currentUser model.User, targetUser model.User) error
	CanSetUserAgentGroup(currentUser model.User, targetUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserAuthZ]
