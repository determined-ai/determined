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
	// No interface on login / logout and getting currently logged in user
	// since it doesn't make sense to control these routes.

	// GET /api/v1/users/:user_id
	// Denying a user shouldn't return an error. Only a server error that needs to be
	// reported to the user should return an errr.
	CanGetUser(curUser, targetUser model.User) (canGetUser bool, serverError error)

	// GET /users
	// GET /api/v1/users
	// FilterUserList normally shouldn't return an error. It should just remove
	// users that the requesting user shouldn't see. It returns an error directly without
	// indication it occurred during a filtering stage to bubble up a failure to the user.
	FilterUserList(curUser model.User, users []model.FullUser) ([]model.FullUser, error)

	// POST /user
	// POST /api/v1/users
	CanCreateUser(curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup) error

	// PATCH /users/:username
	// POST /api/v1/users/:user_id/password
	CanSetUsersPassword(curUser, targetUser model.User) error
	// PATCH /users/:username
	CanSetUsersActive(curUser, targetUser model.User, toActiveVal bool) error
	// PATCH /users/:username
	CanSetUsersAdmin(curUser, targetUser model.User, toAdminVal bool) error
	// PATCH /users/:username
	CanSetUsersAgentUserGroup(
		curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
	) error
	// PATCH /users/:username/username
	CanSetUsersUsername(curUser, targetUser model.User) error
	// PATCH /api/v1/users/:user_id
	CanSetUsersDisplayName(curUser, targetUser model.User) error

	// GET /users/:username/image
	CanGetUsersImage(curUser, targetUsername model.User) error

	// GET /api/v1/users/setting
	CanGetUsersOwnSettings(curUser model.User) error
	// POST /api/v1/users/setting/reset
	CanCreateUsersOwnSetting(curUser model.User, setting model.UserWebSetting) error
	// POST /api/v1/users/setting
	CanResetUsersOwnSettings(curUser model.User) error

	// GET /api/v1/tasks/count
	// TODO(nick) move this when we add an AuthZ for notebooks.
	CanGetActiveTasksCount(curUser model.User) error
	CanAccessNTSCTask(curUser model.User, ownerID model.UserID) (canView bool, serverError error)
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserAuthZ]
