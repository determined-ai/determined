package user

import (
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/pkg/model"
)

// UserAuthZ describes authz methods for `user` package.
type UserAuthZ interface {
	// POST /logout
	// POST /login
	// GET /users/me
	// GET /api/v1/auth/user
	// No interface on login / logout and get me since it doesn't make sense
	// to control authentication since we are controling authorization.

	// TODO still leaking not found information...
	// Need /get/users in order to deny leaking information

	// GET /api/v1/users/:user_id
	CanGetUser(currentUser model.User, targetUser model.User) error

	// GET /users
	// GET /api/v1/users
	FilterUserList(currentUser model.User, users []model.FullUser) ([]model.FullUser, error)

	// POST /user
	// POST /api/v1/users
	// TODO admin on it
	CanCreateUser(
		currentUser model.User, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
	) error

	// TODO admin / own password
	// PATCH /users/:username
	// POST /api/v1/users/:user_id/password
	CanSetUsersPassword(currentUser model.User, targetUser model.User) error

	// PATCH /users/:username
	CanSetUsersActive(currentUser model.User, targetUser model.User, toActiveVal bool) error

	// PATCH /users/:username
	CanSetUsersAdmin(currentUser model.User, targetUser model.User, toAdminVal bool) error

	// PATCH /users/:username
	CanSetUsersAgentUserGroup(
		currentUser model.User, targetUser model.User, agentUserGroup model.AgentUserGroup,
	) error

	// PATCH /users/:username/username
	// TODO admin right? Not own username I think? YEP JUST need admin.
	CanSetUsersUsername(currentUser model.User, targetUser model.User) error

	// PATCH /api/v1/users/:user_id
	// TODO admin + yourself
	CanSetUsersDisplayName(currentUser model.User, targetUser model.User) error

	// GET /users/:username/image
	// TODO should be username? I mean ideally not right but who knows.
	CanGetUsersImage(currentUser model.User, targetUsername string) error

	// GET /api/v1/users/setting
	CanGetUsersOwnSettings(currentUser model.User) error

	// POST /api/v1/users/setting/reset
	CanCreateUsersOwnSetting(currentUser model.User, setting model.UserWebSetting) error

	// POST /api/v1/users/setting
	CanResetUsersOwnSettings(currentUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserAuthZ]
