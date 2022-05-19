package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

type UserAuthZ interface {
	CanSetUserPassword(currentUser model.User, targetUser model.User) error
}

var AuthZProvider authz.AuthZProviderType[UserAuthZ]
