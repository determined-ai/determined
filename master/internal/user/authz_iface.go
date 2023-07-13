package user

import (
	"context"

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
	CanGetUser(ctx context.Context, curUser, targetUser model.User) error

	// GET /users
	// GET /api/v1/users
	// FilterUserList normally shouldn't return an error. It should just remove
	// users that the requesting user shouldn't see. It returns an error directly without
	// indication it occurred during a filtering stage to bubble up a failure to the user.
	FilterUserList(ctx context.Context, curUser model.User, users []model.FullUser) (
		[]model.FullUser, error)

	// POST /user
	// POST /api/v1/users
	CanCreateUser(
		ctx context.Context, curUser, userToAdd model.User, agentUserGroup *model.AgentUserGroup,
	) error

	// PATCH /users/:username
	// POST /api/v1/users/:user_id/password
	CanSetUsersPassword(ctx context.Context, curUser, targetUser model.User) error
	// PATCH /users/:username
	CanSetUsersActive(ctx context.Context, curUser, targetUser model.User, toActiveVal bool) error
	// PATCH /users/:username
	CanSetUsersAdmin(ctx context.Context, curUser, targetUser model.User, toAdminVal bool) error
	// PATCH /users/:username
	CanSetUsersRemote(ctx context.Context, curUser model.User) error
	// PATCH /users/:username
	CanSetUsersAgentUserGroup(
		ctx context.Context, curUser, targetUser model.User, agentUserGroup model.AgentUserGroup,
	) error
	// PATCH /users/:username/username
	CanSetUsersUsername(ctx context.Context, curUser, targetUser model.User) error
	// PATCH /api/v1/users/:user_id
	CanSetUsersDisplayName(ctx context.Context, curUser, targetUser model.User) error

	// GET /users/:username/image
	CanGetUsersImage(ctx context.Context, curUser, targetUsername model.User) error

	// GET /api/v1/users/setting
	CanGetUsersOwnSettings(ctx context.Context, curUser model.User) error
	// POST /api/v1/users/setting
	CanCreateUsersOwnSetting(
		ctx context.Context, curUser model.User, settings []model.UserWebSetting,
	) error
	// POST /api/v1/users/setting/reset
	CanResetUsersOwnSettings(ctx context.Context, curUser model.User) error
}

// AuthZProvider is the authz registry for `user` package.
var AuthZProvider authz.AuthZProviderType[UserAuthZ]
