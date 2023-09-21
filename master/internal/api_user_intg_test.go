//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/determined-ai/determined/master/internal/rm/actorrm"
	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
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
	thePgDB   *db.PgDB
	authzUser *mocks.UserAuthZ
	system    *actor.System
	mockRM    *actorrm.ResourceManager
)

// pgdb can be nil to use the singleton database for testing.
func setupAPITest(t *testing.T, pgdb *db.PgDB) (*apiServer, model.User, context.Context) {
	if pgdb == nil {
		if thePgDB == nil {
			thePgDB = db.MustResolveTestPostgres(t)
			db.MustMigrateTestPostgres(t, thePgDB, "file://../static/migrations")
			require.NoError(t, etc.SetRootPath("../static/srv"))

			system = actor.NewSystem("mock")
			ref, _ := system.ActorOf(sproto.K8sRMAddr, actor.ActorFunc(
				func(context *actor.Context) error {
					switch context.Message().(type) {
					case sproto.DeleteJob:
						context.Respond(sproto.EmptyDeleteJobResponse())
					}
					return nil
				}))
			mockRM = actorrm.Wrap(ref)
		}
		pgdb = thePgDB
	} else {
		// After a custom db is provided, we need to reinitialize the pgdb singleton.
		thePgDB = nil
	}

	api := &apiServer{
		m: &Master{
			trialLogBackend: pgdb,
			system:          system,
			db:              pgdb,
			taskLogBackend:  pgdb,
			rm:              mockRM,
			config: &config.Config{
				InternalConfig: config.InternalConfig{
					ExternalSessions: model.ExternalSessions{},
				},
				TaskContainerDefaults: model.TaskContainerDefaultsConfig{},
				ResourceConfig: config.ResourceConfig{
					ResourceManager: &config.ResourceManagerConfig{},
				},
			},
			taskSpec: &tasks.TaskSpec{SSHRsaSize: 1024},
		},
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "basic"}

	userModel, err := user.ByUsername(context.TODO(), "admin")
	require.NoError(t, err, "Couldn't get admin user")
	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: "admin"})
	require.NoError(t, err, "Couldn't login")
	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))

	return api, *userModel, ctx
}

func fetchUserIds(t *testing.T, api *apiServer, ctx context.Context, req *apiv1.GetUsersRequest) []model.UserID {
	resp, err := api.GetUsers(ctx, req)
	require.NoError(t, err)
	var ids []model.UserID
	for _, u := range resp.Users {
		ids = append(ids, model.UserID(u.Id))
	}
	return ids
}

func TestGetUsersRemote(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	remoteUser, err := user.Add(
		context.TODO(),
		&model.User{
			Username: uuid.New().String(),
			Remote:   true,
		},
		nil,
	)
	require.NoError(t, err)

	nonRemoteUser, err := user.Add(
		context.TODO(),
		&model.User{
			Username: uuid.New().String(),
			Remote:   false,
		},
		nil,
	)
	require.NoError(t, err)

	resp, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.NoError(t, err)
	for _, u := range resp.Users {
		if model.UserID(u.Id) == remoteUser {
			require.True(t, u.Remote)
		} else if model.UserID(u.Id) == nonRemoteUser {
			require.False(t, u.Remote)
		}
	}
}

func TestFilterUser(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	userID1, _ := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   false,
			Admin:    false,
		},
		nil,
	)
	userID2, _ := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   true,
			Admin:    false,
		},
		nil,
	)
	userID3, _ := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   true,
			Admin:    true,
		},
		nil,
	)
	userID4, _ := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   false,
			Admin:    true,
		},
		nil,
	)

	userIds := fetchUserIds(t, api, ctx, &apiv1.GetUsersRequest{})
	for _, u := range []model.UserID{userID1, userID2, userID3, userID4} {
		require.True(t, slices.Contains(userIds, u), fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(t, api, ctx, &apiv1.GetUsersRequest{Admin: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID3, userID4} {
		require.True(t, slices.Contains(userIds, u), fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(t, api, ctx, &apiv1.GetUsersRequest{Active: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID2, userID3} {
		require.True(t, slices.Contains(userIds, u), fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(t, api, ctx, &apiv1.GetUsersRequest{Active: ptrs.Ptr(true), Admin: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID3} {
		require.True(t, slices.Contains(userIds, u), fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
}

func TestPatchUser(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	userID, err := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   false,
		},
		nil,
	)
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
	_, err = user.Add(ctx,
		&model.User{
			Username:    similiarName + "uPPER",
			DisplayName: null.StringFrom(similiarDisplay + "lOwEr"),
		},
		nil,
	)
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

func TestRenameUserThenReuseName(t *testing.T) {
	username := uuid.New().String()
	api, _, ctx := setupAPITest(t, nil)
	resp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username: username,
		},
	})
	require.NoError(t, err)

	// Can't create user with same username.
	_, err = api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username: username,
		},
	})
	require.Error(t, err)

	_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
		UserId: resp.User.Id,
		User: &userv1.PatchUser{
			Username: ptrs.Ptr(uuid.New().String()),
		},
	})
	require.NoError(t, err)

	// Should be able to make a new user with original username.
	_, err = api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username: username,
		},
	})
	require.NoError(t, err)
}

// pgdb can be nil to use the singleton database for testing.
func setupUserAuthzTest(
	t *testing.T, pgdb *db.PgDB,
) (*apiServer, *mocks.UserAuthZ, model.User, context.Context) {
	api, curUser, ctx := setupAPITest(t, pgdb)

	if authzUser == nil {
		authzUser = &mocks.UserAuthZ{}
		user.AuthZProvider.Register("mock", authzUser)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}

	return api, authzUser, curUser, ctx
}

func TestAuthzGetUsers(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

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
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	// Error passes through when CanGetUser returns non nil error.
	expectedErr := fmt.Errorf("canGetUserError")
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).
		Return(expectedErr).Once()
	_, err := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.Equal(t, expectedErr, err)

	// Ensure when CanGetUser returns false we get the same error as the user not being found.
	_, notFoundError := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: -999})
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), notFoundError.Error())

	authzUsers.On("CanGetUser", mock.Anything, curUser,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), err.Error())

	// As a spot check just make sure we can still get users with no error.
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(nil).Once()
	user, err := api.GetUser(ctx, &apiv1.GetUserRequest{UserId: 1})
	require.NoError(t, err)
	require.NotNil(t, user)
}

func TestAuthzPostUser(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

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
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	// If we can view the user we can get the error message from CanSetUsersPassword.
	expectedErr := status.Error(codes.PermissionDenied, "canSetUsersPassword")
	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(nil).Once()

	_, err := api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, expectedErr.Error(), err.Error())

	// If we can't view the user we just get the same as passing in a not found user.
	_, notFoundError := api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: -9999})
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), notFoundError.Error())

	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), err.Error())

	// If CanGetUser returns an error we also return that error.
	cantViewUserError := fmt.Errorf("cantViewUserError")
	authzUsers.On("CanSetUsersPassword", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersPassword")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).
		Return(cantViewUserError).Once()
	_, err = api.SetUserPassword(ctx, &apiv1.SetUserPasswordRequest{UserId: int32(curUser.ID)})
	require.Equal(t, err, cantViewUserError)
}

func TestAuthzPatchUser(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	// If we can view the user we get the error from canSetUsersDisplayName.
	expectedErr := status.Error(codes.PermissionDenied, "canSetUsersDisplayName")
	authzUsers.On("CanSetUsersDisplayName", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser, mock.Anything).Return(nil).Once()

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
		Return(cantViewUserError).Once()
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, cantViewUserError.Error(), err.Error())

	// If we can't view the user get the same as passing in user not found.
	authzUsers.On("CanSetUsersDisplayName", mock.Anything, curUser, mock.Anything).
		Return(fmt.Errorf("canSetUsersDisplayName")).Once()
	authzUsers.On("CanGetUser", mock.Anything, curUser,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), err.Error())

	req.UserId = -9999
	_, err = api.PatchUser(ctx, req)
	require.Equal(t, apiPkg.NotFoundErrs("user", "", true).Error(), err.Error())
}

func TestAuthzGetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	expectedErr := status.Error(codes.PermissionDenied, "canGetUsersOwnSettings")
	authzUsers.On("CanGetUsersOwnSettings", mock.Anything, curUser).
		Return(fmt.Errorf("canGetUsersOwnSettings")).Once()

	_, err := api.GetUserSetting(ctx, &apiv1.GetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzPostUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	expectedErr := status.Error(codes.PermissionDenied, "canCreateUsersOwnSetting")
	authzUsers.On("CanCreateUsersOwnSetting", mock.Anything, curUser,
		[]*model.UserWebSetting{{UserID: curUser.ID, Key: "k", Value: "v"}}).
		Return(fmt.Errorf("canCreateUsersOwnSetting")).Once()

	_, err := api.PostUserSetting(ctx, &apiv1.PostUserSettingRequest{
		Settings: []*userv1.UserWebSetting{{Key: "k", Value: "v"}},
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzResetUserSetting(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	expectedErr := status.Error(codes.PermissionDenied, "canResetUsersOwnSettings")
	authzUsers.On("CanResetUsersOwnSettings", mock.Anything, curUser).
		Return(fmt.Errorf("canResetUsersOwnSettings")).Once()

	_, err := api.ResetUserSetting(ctx, &apiv1.ResetUserSettingRequest{})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestPostUserActivity(t *testing.T) {
	api, _, curUser, ctx := setupUserAuthzTest(t, nil)

	_, err := api.PostUserActivity(ctx, &apiv1.PostUserActivityRequest{
		ActivityType: userv1.ActivityType_ACTIVITY_TYPE_GET,
		EntityType:   userv1.EntityType_ENTITY_TYPE_PROJECT,
		EntityId:     1,
	})

	require.NoError(t, err)

	activityCount, err := getActivityEntry(ctx, curUser.ID, 1)
	require.NoError(t, err)
	require.Equal(t, activityCount, 1, ctx)

	_, err = api.PostUserActivity(ctx, &apiv1.PostUserActivityRequest{
		ActivityType: userv1.ActivityType_ACTIVITY_TYPE_GET,
		EntityType:   userv1.EntityType_ENTITY_TYPE_PROJECT,
		EntityId:     1,
	})

	require.NoError(t, err)

	activityCount, err = getActivityEntry(ctx, curUser.ID, 1)
	require.NoError(t, err)
	require.Equal(t, activityCount, 1, ctx)
}

func getActivityEntry(ctx context.Context, userID model.UserID, entityID int32) (int, error) {
	return db.Bun().NewSelect().Model((*model.UserActivity)(nil)).Where("user_id = ?",
		int32(userID)).Where("entity_id = ?", entityID).Count(ctx)
}
