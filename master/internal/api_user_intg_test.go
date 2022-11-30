//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/rm/actorrm"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var (
	pgDB      *db.PgDB
	authzUser *mocks.UserAuthZ
	system    *actor.System
	mockRM    *actorrm.ResourceManager
)

func setupAPITest(t *testing.T) (*apiServer, model.User, context.Context) {
	if pgDB == nil {
		pgDB = db.MustResolveTestPostgres(t)
		db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")
		require.NoError(t, etc.SetRootPath("../static/srv"))

		system = actor.NewSystem("mock")
		ref, _ := system.ActorOf(sproto.K8sRMAddr, actor.ActorFunc(
			func(context *actor.Context) error {
				return nil
			}))
		mockRM = actorrm.Wrap(ref)
	}

	api := &apiServer{
		m: &Master{
			system:         system,
			db:             pgDB,
			taskLogBackend: pgDB,
			rm:             mockRM,
			config: &config.Config{
				InternalConfig:        config.InternalConfig{},
				TaskContainerDefaults: model.TaskContainerDefaultsConfig{},
				ResourceConfig: &config.ResourceConfig{
					ResourceManager: &config.ResourceManagerConfig{},
				},
			},
			taskSpec: &tasks.TaskSpec{},
		},
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "basic"}

	userModel, err := user.UserByUsername("admin")
	require.NoError(t, err, "Couldn't get admin user")
	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "admin"})
	require.NoError(t, err, "Couldn't login")
	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	return api, *userModel, ctx
}

func TestPatchUser(t *testing.T) {
	api, _, ctx := setupAPITest(t)
	userID, err := api.m.db.AddUser(&model.User{
		Username: uuid.New().String(),
		Active:   false,
	}, nil)
	require.NoError(t, err)

	username := uuid.New().String()
	displayName := uuid.New().String()
	password := uuid.New().String()
	resp, err := api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: int32(userID),
		User: &userv1.PatchUser{
			Admin:  wrapperspb.Bool(true),
			Active: wrapperspb.Bool(true),
			AgentUserGroup: &userv1.AgentUserGroup{
				AgentUid:   ptrs.Ptr(int32(5)),
				AgentUser:  ptrs.Ptr("agentuser"),
				AgentGid:   ptrs.Ptr(int32(6)),
				AgentGroup: ptrs.Ptr("agentgroup"),
			},
			Username:    ptrs.Ptr(username),
			DisplayName: ptrs.Ptr(displayName),
			Password:    ptrs.Ptr(password),
			IsHashed:    false,
		},
	})
	require.NoError(t, err)
	require.Equal(t, username, resp.User.Username)
	require.True(t, resp.User.Admin)
	require.True(t, resp.User.Active)
	require.Equal(t, ptrs.Ptr(int32(5)), resp.User.AgentUserGroup.AgentUid)
	require.Equal(t, ptrs.Ptr("agentuser"), resp.User.AgentUserGroup.AgentUser)
	require.Equal(t, ptrs.Ptr(int32(6)), resp.User.AgentUserGroup.AgentGid)
	require.Equal(t, ptrs.Ptr("agentgroup"), resp.User.AgentUserGroup.AgentGroup)
	require.Equal(t, displayName, resp.User.DisplayName)

	// Can we login with new password?
	_, err = api.Login(ctx, &apiv1.LoginRequest{
		Username: username,
		Password: password,
		IsHashed: false,
	})
	require.NoError(t, err)

	// Null out display name and set a client side hashed password.
	password = uuid.New().String()
	displayName = ""
	resp, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: int32(userID),
		User: &userv1.PatchUser{
			DisplayName: ptrs.Ptr(displayName),
			Password:    ptrs.Ptr(replicateClientSideSaltAndHash(password)),
			IsHashed:    true,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "", resp.User.DisplayName)

	_, err = api.Login(ctx, &apiv1.LoginRequest{
		Username: username,
		Password: password,
		IsHashed: false,
	})
	require.NoError(t, err)

	// Verify we can't set a display name similar to another username or display name.
	similiarName := uuid.New().String()
	similiarDisplay := uuid.New().String()
	_, err = api.m.db.AddUser(&model.User{
		Username:    similiarName + "uPPER",
		DisplayName: null.StringFrom(similiarDisplay + "lOwEr"),
	}, nil)
	require.NoError(t, err)

	_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: int32(userID),
		User: &userv1.PatchUser{
			DisplayName: ptrs.Ptr(similiarName + "uppEr"),
		},
	})
	require.Error(t, err)

	_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: int32(userID),
		User: &userv1.PatchUser{
			DisplayName: ptrs.Ptr(similiarDisplay + "LOWer"),
		},
	})
	require.Error(t, err)
}

func setupUserAuthzTest(t *testing.T) (*apiServer, *mocks.UserAuthZ, model.User, context.Context) {
	api, curUser, ctx := setupAPITest(t)

	if authzUser == nil {
		authzUser = &mocks.UserAuthZ{}
		user.AuthZProvider.Register("mock", authzUser)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}

	return api, authzUser, curUser, ctx
}

func TestAuthzGetUsers(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	// Error just passes error through.
	expectedErr := fmt.Errorf("filterUseList")
	authzUsers.On("FilterUserList", mock.Anything, curUser, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever FilterUserList returns.
	users := []model.FullUser{
		{Username: "a"},
		{Username: "b"},
	}
	authzUsers.On("FilterUserList", mock.Anything, curUser, mock.Anything).Return(users, nil).Once()
	actual, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.NoError(t, err)

	var expected apiv1.GetUsersResponse
	for _, u := range users {
		expected.Users = append(expected.Users, toProtoUserFromFullUser(u))
	}
	require.Equal(t, expected.Users, actual.Users)
}

func TestAuthzGetUser(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	// Error passes through when CanGetUser returns non nil error.
	expectedErr := fmt.Errorf("canGetUserError")
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).
		Return(false, expectedErr).Once()
	_, err := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.Equal(t, expectedErr, err)

	// Ensure when CanGetUser returns false we get the same error as the user not being found.
	_, notFoundError := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: -999})
	require.Equal(t, errUserNotFound.Error(), notFoundError.Error())

	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.Equal(t, notFoundError.Error(), err.Error())

	// As a spot check just make sure we can still get users with no error.
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	user, err := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.NoError(t, err)
	require.NotNil(t, user)
}

func TestAuthzPostUser(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	expectedErr := status.Error(codes.PermissionDenied, "canCreateUserError")
	authzUsers.On("CanCreateUser", mock.Anything, curUser,
		model.User{Username: "admin", Admin: true},
		&model.AgentUserGroup{
			UID:   5,
			GID:   6,
			User:  "five",
			Group: "six",
		}).Return(fmt.Errorf("canCreateUserError")).Once()

	var five int32 = 5
	var six int32 = 6
	_, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username: "admin",
			Admin:    true,
			AgentUserGroup: &userv1.AgentUserGroup{
				AgentUid:   &five,
				AgentGid:   &six,
				AgentUser:  ptrs.Ptr("five"),
				AgentGroup: ptrs.Ptr("six"),
			},
		},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzSetUserPassword(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	// If we can view the user we can get the error message from CanSetUsersPassword.
	expectedErr := status.Error(codes.PermissionDenied, "canSetUsersPassword")
	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()

	_, err := api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, expectedErr.Error(), err.Error())

	// If we can't view the user we just get the same as passing in a not found user.
	_, notFoundError := api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: -9999})
	require.Equal(t, errUserNotFound.Error(), notFoundError.Error())

	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, errUserNotFound.Error(), err.Error())

	// If CanGetUser returns an error we also return that error.
	cantViewUserError := fmt.Errorf("cantViewUserError")
	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).
		Return(false, cantViewUserError).Once()
	_, err = api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, err, cantViewUserError)
}

func TestAuthzPatchUser(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	// If we can view the user we get the error from canSetUsersDisplayName.
	expectedErr := status.Error(codes.PermissionDenied, "canSetUsersDisplayName")
	authzUsers.On("CanSetUsersDisplayName", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()

	req := &apiv1.PatchUserRequest{
		UserId: int32(curUser.ID),
		User: &userv1.PatchUser{
			DisplayName: ptrs.Ptr("u"),
		},
	}
	_, err := api.PatchUser(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())

	// If CanGetUser returns an error we also return the error.
	cantViewUserError := fmt.Errorf("cantViewUserError")
	authzUsers.On("CanSetUsersDisplayName", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).
		Return(false, cantViewUserError).Once()
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, cantViewUserError.Error(), err.Error())

	// If we can't view the user get the same as passing in user not found.
	authzUsers.On("CanSetUsersDisplayName", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, errUserNotFound.Error(), err.Error())

	req.UserId = -9999
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, errUserNotFound.Error(), err.Error())
}

func TestAuthzGetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	expectedErr := status.Error(codes.PermissionDenied, "canGetUsersOwnSettings")
	authzUsers.On("CanGetUsersOwnSettings", mock.Anything, curUser).
		Return(fmt.Errorf("canGetUsersOwnSettings")).Once()

	_, err := api.GetUserSetting(ctx, &apiv1.GetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzPostUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	expectedErr := status.Error(codes.PermissionDenied, "canCreateUsersOwnSetting")
	authzUsers.On("CanCreateUsersOwnSetting", mock.Anything, curUser,
		model.UserWebSetting{UserID: curUser.ID, Key: "k", Value: "v"}).
		Return(fmt.Errorf("canCreateUsersOwnSetting")).Once()

	_, err := api.PostUserSetting(ctx, &apiv1.PostUserSettingRequest{
		Setting: &userv1.UserWebSetting{Key: "k", Value: "v"},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzResetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	expectedErr := status.Error(codes.PermissionDenied, "canResetUsersOwnSettings")
	authzUsers.On("CanResetUsersOwnSettings", mock.Anything, curUser).
		Return(fmt.Errorf("canResetUsersOwnSettings")).Once()

	_, err := api.ResetUserSetting(ctx, &apiv1.ResetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TesAuthzPostUserActivity(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t)

	expectedErr := status.Error(codes.PermissionDenied, "canSetUsersOwnActivity")
	authzUsers.On("canSetUsersOwnActivity", mock.Anything, curUser).
		Return(fmt.Errorf("canSetUsersOwnActivity")).Once()

	_, err := api.PostUserActivity(ctx, &apiv1.ResetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())

	resp, err = api.PostUserActivity(ctx, &apiv1.PostUserActivityRequest{
			ActivityType:  ActivityType_ACTIVITY_TYPE_GET,
			ActivityTime:    time.Now(),
			EntityType:  EntityType_ENTITY_TYPE_PROJECT,
			EntityId: 1
			UserId: int32(userID)
	})
	require.NoError(t, err)
}
