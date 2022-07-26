package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZ describes authz methods for `user` package.
type UserAuthZ interface {
	// TODO still leaking not found information...
	// Need /get/users in order to deny leaking information

	// POST /logout
	// POST /login
	// No interface on login / logout since it doesn't make sense to control authentication
	// since we are controling authorization.

	// GET /users
	FilterUserList(currentUser model.User, users []model.FullUser) ([]model.FullUser, error)

	// POST /user
	// TODO admin on it
	CanCreateUser(
		currentUser model.User, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
	) error

	// GET /users/me
	CanGetMe(currentUser model.User) error

	// PATCH /users/:username
	// TODO admin / own password
	CanSetUsersPassword(currentUser model.User, targetUser model.User) error
	CanSetUsersActive(currentUser model.User, targetUser model.User, toActiveVal bool) error
	CanSetUsersAdmin(currentUser model.User, targetUser model.User, toAdminVal bool) error
	CanSetUsersAgentUserGroup(
		currentUser model.User, targetUser model.User, agentUserGroup model.AgentUserGroup,
	) error

	// PATCH /users/:username/username
	// TODO admin right? Not own username I think? YEP JUST need admin.
	CanSetUsersUsername(currentUser model.User, targetUser model.User) error

	// GET /users/:username/image
	// TODO should be username? I mean ideally not right but who knows.
	CanGetUsersImage(currentUser model.User, targetUsername string) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserAuthZ]
