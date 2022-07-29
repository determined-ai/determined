//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var (
	pgDB      *db.PgDB
	authzUser *mocks.UserAuthZ = &mocks.UserAuthZ{}
)

func SetupUserAuthzTest(t *testing.T) (*apiServer, *mocks.UserAuthZ, model.User, context.Context) {
	if pgDB == nil {
		pgDB = db.MustResolveTestPostgres(t)
		db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")
		require.NoError(t, etc.SetRootPath("../static/srv"))

		user.AuthZProvider.Register("mock", authzUser)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}

	api := &apiServer{m: &Master{
		db: pgDB,
		config: &config.Config{
			InternalConfig: config.InternalConfig{},
		},
	}}

	user, err := pgDB.UserByUsername("determined")
	require.NoError(t, err, "Couldn't get determined user")
	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "determined"})
	require.NoError(t, err, "Couldn't login")
	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	return api, authzUser, *user, ctx
}

func TestAuthzGetUsers(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	// TODO error here is fine to not wrap.
	// Since we just want it to appear as a regular db error.
	// Don't error here possibly?!??!?!
	// Maybe just filter only? Its tough since we want the possibility of bubbling up errors.
	// While not exposing that filtering is occuring.

	// Error just passes error through.
	expectedErr := fmt.Errorf("filterUseList")
	authzUsers.On("FilterUserList", curUser, mock.Anything).Return(nil, expectedErr).Once()
	_, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever FilterUserList returns.
	users := []model.FullUser{
		{Username: "a"},
		{Username: "b"},
	}
	authzUsers.On("FilterUserList", curUser, mock.Anything).Return(users, nil).Once()
	actual, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.NoError(t, err)

	var expected apiv1.GetUsersResponse
	for _, u := range users {
		expected.Users = append(expected.Users, toProtoUserFromFullUser(u))
	}
	require.Equal(t, actual.Users, expected.Users)
}

func TestAuthzGetUser(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	// TODO information leakage here
	// Permission denied should give 404 or exact error that getFullModelUser returns.
	// Add a test to cover that case.

	expectedErr := fmt.Errorf("canGetUserError")
	authzUsers.On("CanGetUser", curUser, mock.Anything).Return(expectedErr).Once()
	_, err := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.Equal(t, expectedErr, err)
}

func TestAuthzPostUser(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canCreateUserError")
	authzUsers.On("CanCreateUser", curUser,
		model.User{Username: "u", Admin: true},
		&model.AgentUserGroup{UID: 5, GID: 6}).Return(fmt.Errorf("canCreateUserError")).Once()

	_, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username:       "u",
			Admin:          true,
			AgentUserGroup: &userv1.AgentUserGroup{AgentUid: 5, AgentGid: 6},
		},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzSetUserPassword(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	// TODO first check that A we can view the user.
	// If we can't we always return a 404. Right?
	// Then check if we can update the user.

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canSetUsersPassword")
	authzUsers.On("CanSetUsersPassword", curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()

	_, err := api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{
		UserId: int32(curUser.ID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzPatchUser(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	// TODO first check that A we can view the user.
	// If we can't we always return a 404. Right?
	// Then check if we can update the user.

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canSetUsersDisplayName")
	authzUsers.On("CanSetUsersDisplayName", curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()

	_, err := api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: int32(curUser.ID),
		User: &userv1.PatchUser{
			DisplayName: wrapperspb.String("u"),
		},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzGetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canGetUsersOwnSettings")
	authzUsers.On("CanGetUsersOwnSettings", curUser).
		Return(fmt.Errorf("canGetUsersOwnSettings")).Once()

	_, err := api.GetUserSetting(ctx, &apiv1.GetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzPostUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canCreateUsersOwnSetting")
	authzUsers.On("CanCreateUsersOwnSetting", curUser,
		model.UserWebSetting{UserID: curUser.ID, Key: "k", Value: "v"}).
		Return(fmt.Errorf("canCreateUsersOwnSetting")).Once()

	_, err := api.PostUserSetting(ctx, &apiv1.PostUserSettingRequest{
		Setting: &userv1.UserWebSetting{Key: "k", Value: "v"},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzResetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := SetupUserAuthzTest(t)

	expectedErr := errors.Wrap(grpcutil.ErrPermissionDenied, "canResetUsersOwnSettings")
	authzUsers.On("CanResetUsersOwnSettings", curUser).
		Return(fmt.Errorf("canResetUsersOwnSettings")).Once()

	_, err := api.ResetUserSetting(ctx, &apiv1.ResetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}
