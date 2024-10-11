//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gopkg.in/guregu/null.v3"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/logpattern"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
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
)

const (
	lifespan = "5s"
	desc     = "test desc"
)

// MockRM returns a mock resource manager that basically returns OK on every call. We should update this to an
// RM that makes sure callers uphold expected invariants (release, kill not called before allocate, release not
// called twice for the same resource, etc).
func MockRM() *mocks.ResourceManager {
	var mockRM mocks.ResourceManager
	mockRM.On("DeleteJob", mock.Anything).Return(func(sproto.DeleteJob) sproto.DeleteJobResponse {
		return sproto.EmptyDeleteJobResponse()
	}, nil)
	mockRM.On("ResolveResourcePool", mock.Anything, mock.Anything, mock.Anything).Return(
		func(name rm.ResourcePoolName, _, _ int) rm.ResourcePoolName {
			return name
		},
		nil,
	)
	mockRM.On("ValidateResources", mock.Anything).Return(nil, nil)
	mockRM.On("TaskContainerDefaults", mock.Anything, mock.Anything).Return(
		func(name rm.ResourcePoolName, def model.TaskContainerDefaultsConfig) model.TaskContainerDefaultsConfig {
			return def
		},
		nil,
	)
	mockRM.On("SetGroupMaxSlots", mock.Anything).Return()
	mockRM.On("SetGroupWeight", mock.Anything).Return(nil)
	mockRM.On("Allocate", mock.Anything).Return(func(msg sproto.AllocateRequest) *sproto.ResourcesSubscription {
		return rmevents.Subscribe(msg.AllocationID)
	}, nil)

	mockRM.On("SmallerValueIsHigherPriority").Return(true, nil)

	return &mockRM
}

// pgdb can be nil to use the singleton database for testing.
func setupAPITest(t *testing.T, pgdb *db.PgDB,
	altMockRM ...*mocks.ResourceManager,
) (*apiServer, model.User, context.Context) {
	mockRM := MockRM()
	if len(altMockRM) == 1 {
		mockRM = altMockRM[0]
	}

	if pgdb == nil {
		if thePgDB == nil {
			thePgDB, _ = db.MustResolveTestPostgres(t)
			db.MustMigrateTestPostgres(t, thePgDB, "file://../static/migrations")
			require.NoError(t, etc.SetRootPath("../static/srv"))

			l, err := logpattern.New(context.TODO())
			require.NoError(t, err)
			logpattern.SetDefault(l)
		}
		pgdb = thePgDB
	} else {
		// After a custom db is provided, we need to reinitialize the pgdb singleton.
		thePgDB = nil
	}
	jobservice.SetDefaultService(mockRM)

	api := &apiServer{
		m: &Master{
			trialLogBackend: pgdb,
			db:              pgdb,
			taskLogBackend:  pgdb,
			rm:              mockRM,
			config: &config.Config{
				InternalConfig: config.InternalConfig{
					ExternalSessions: model.ExternalSessions{},
				},
				TaskContainerDefaults: model.TaskContainerDefaultsConfig{},
				ResourceConfig:        *config.DefaultResourceConfig(),
			},
			taskSpec: &tasks.TaskSpec{SSHRsaSize: 1024},
			allRms:   map[string]rm.ResourceManager{config.DefaultClusterName: mockRM},
		},
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "basic"}

	username := uuid.New().String()
	newUserModel := &model.User{
		Username:     username,
		PasswordHash: null.NewString("", false),
		Active:       true,
		Admin:        true,
	}
	_, err := user.Add(context.TODO(), newUserModel, nil)
	require.NoError(t, err, "Couldn't create admin user")
	resp, err := api.Login(context.TODO(), &apiv1.LoginRequest{Username: username})
	require.NoError(t, err, "Couldn't login")
	userModel, err := user.ByUsername(context.TODO(), username)
	require.NoError(t, err, "Couldn't get admin user")
	ctx := metadata.NewIncomingContext(context.TODO(),
		metadata.Pairs("x-user-token", fmt.Sprintf("Bearer %s", resp.Token)))
	return api, *userModel, ctx
}

func fetchUserIds(ctx context.Context, t *testing.T, api *apiServer, req *apiv1.GetUsersRequest) []model.UserID {
	resp, err := api.GetUsers(ctx, req)
	require.NoError(t, err)
	var ids []model.UserID
	for _, u := range resp.Users {
		ids = append(ids, model.UserID(u.Id))
	}
	return ids
}

func TestProcessAuth(t *testing.T) {
	api, _, _ := setupAPITest(t, nil)
	extConfig := model.ExternalSessions{}
	user.InitService(api.m.db, &extConfig)

	e := echo.New()
	handler := user.GetService().ProcessAuthentication(
		func(c echo.Context) error {
			require.Fail(t, "Should not have reached this point")
			return nil
		},
	)
	req := httptest.NewRequest(http.MethodGet, "/authed-route", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := handler(c)
	require.Error(t, err)
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	require.Equal(t, http.StatusUnauthorized, httpError.Code)
}

func setupNewAllocation(t *testing.T, dbPtr *db.PgDB) *model.Allocation {
	ctx := context.TODO()

	tIn := db.RequireMockTask(t, dbPtr, nil)
	a := model.Allocation{
		AllocationID: model.AllocationID(fmt.Sprintf("%s-1", tIn.TaskID)),
		TaskID:       tIn.TaskID,
		StartTime:    ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond)),
		State:        ptrs.Ptr(model.AllocationStateTerminated),
	}

	err := db.AddAllocation(ctx, &a)
	require.NoError(t, err, "failed to add allocation")

	res, err := db.AllocationByID(ctx, a.AllocationID)
	require.NoError(t, err)
	require.Equal(t, a, *res)
	return res
}

func TestAuthMiddleware(t *testing.T) {
	proxies := []string{"/proxied-path-a"}
	api, _, ctx := setupAPITest(t, nil)
	extConfig := model.ExternalSessions{}
	user.InitService(api.m.db, &extConfig)

	username := uuid.New().String()
	resp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
		User: &userv1.User{
			Username: username,
			Active:   true,
		},
		Password: "testpassword",
	})
	require.NoError(t, err)

	user := model.User{Username: username, ID: model.UserID(resp.User.Id)}

	allocation := setupNewAllocation(t, api.m.db)
	allocationToken, err := db.StartAllocationSession(ctx, allocation.AllocationID, &user)
	require.NoError(t, err)
	require.NotEmpty(t, allocationToken)

	allocationHeader := grpcutil.GrpcMetadataPrefix + grpcutil.AllocationTokenHeader

	proxiedSubRoute := "/proxied-path-a/anysubroute"
	redirectedSubRoute := "/det/login?redirect=/proxied-path-a/anysubroute"

	tests := []struct {
		path         string
		expectedCode int
		expectedLoc  string // Expected location, empty if no redirect expected
		headers      map[string]string
	}{
		{proxiedSubRoute, http.StatusSeeOther, redirectedSubRoute, map[string]string{}},
		{"/proxied-path-a", http.StatusUnauthorized, "", map[string]string{
			"Accept": "application/json",
		}},
		{proxiedSubRoute, http.StatusOK, "", map[string]string{
			allocationHeader: fmt.Sprintf("Bearer %s", allocationToken),
		}},
		{proxiedSubRoute, http.StatusSeeOther, redirectedSubRoute, map[string]string{
			allocationHeader: "Bearer invalid-token",
		}},
		{proxiedSubRoute, http.StatusUnauthorized, "", map[string]string{
			"Accept":         "application/json",
			allocationHeader: "Bearer invalid-token",
		}},
		{"/non-proxied-path", http.StatusUnauthorized, "", map[string]string{}},
		{"/non-proxied-path", http.StatusUnauthorized, "", map[string]string{
			"Accept": "application/json",
		}},
	}

	e := echo.New()
	for _, tc := range tests {
		t.Run(fmt.Sprintf("Path: %s, Accept: %s", tc.path, tc.headers), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if len(tc.headers) > 0 {
				for k, v := range tc.headers {
					req.Header.Set(k, v)
				}
			}

			middleware := processAuthWithRedirect(proxies)
			fn := middleware(func(ctx echo.Context) error { return ctx.NoContent(http.StatusOK) })

			err := fn(c)

			switch tc.expectedCode {
			case http.StatusUnauthorized:
				require.Error(t, err, "Expected an error but got none")
				httpError, ok := err.(*echo.HTTPError) // Cast error to *echo.HTTPError to check code
				if ok && httpError != nil {
					require.Equal(t, tc.expectedCode, httpError.Code, "HTTP status code does not match expected")
				} else {
					require.Fail(t, "Error is not an HTTPError as expected")
				}
			case http.StatusSeeOther:
				require.Equal(t, tc.expectedCode, rec.Code, "HTTP status code does not match expected")
				require.NoError(t, err, "Did not expect an error but got one")
				require.Contains(t, rec.Header().Get("Location"), tc.expectedLoc,
					"Location header does not match expected redirect")
			case http.StatusOK:
				require.Equal(t, tc.expectedCode, rec.Code, "HTTP status code does not match expected")
				require.NoError(t, err, "Did not expect an error but got one")

			default:
				require.Fail(t, "Unsupported branch")
			}
		})
	}
}

func TestLoginRemote(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	t.Run("created with remote", func(t *testing.T) {
		username := uuid.New().String()
		resp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
			User: &userv1.User{
				Username: username,
				Remote:   true,
				Active:   true,
			},
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
		})
		require.ErrorIs(t, err, grpcutil.ErrInvalidCredentials)

		// Can't change password while they are remote.
		_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
			UserId: resp.User.Id,
			User: &userv1.PatchUser{
				Password: ptrs.Ptr("pass"),
			},
		})
		require.ErrorContains(t, err, "Cannot set password")

		// Changing back to unremote means we can login with blank password.
		_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
			UserId: resp.User.Id,
			User: &userv1.PatchUser{
				Remote: ptrs.Ptr(false),
			},
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
		})
		require.NoError(t, err)
	})

	t.Run("created with remote changed with password", func(t *testing.T) {
		username := uuid.New().String()
		resp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
			User: &userv1.User{
				Username: username,
				Remote:   true,
				Active:   true,
			},
		})
		require.NoError(t, err)

		_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
			UserId: resp.User.Id,
			User: &userv1.PatchUser{
				Remote:   ptrs.Ptr(false),
				Password: ptrs.Ptr("testpassword"),
			},
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
			Password: "testpassword",
		})
		require.NoError(t, err)
	})

	t.Run("created without remote", func(t *testing.T) {
		username := uuid.New().String()
		resp, err := api.PostUser(ctx, &apiv1.PostUserRequest{
			User: &userv1.User{
				Username: username,
				Active:   true,
			},
			Password: "testpassword",
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
			Password: "testpassword",
		})
		require.NoError(t, err)

		// Cannot login when we switch to remote.
		_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
			UserId: resp.User.Id,
			User: &userv1.PatchUser{
				Remote: ptrs.Ptr(true),
			},
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
		})
		require.ErrorIs(t, err, grpcutil.ErrInvalidCredentials)

		// We set the password to the unloginable hash.
		var expectedUser model.User
		err = db.Bun().NewSelect().Model(&expectedUser).
			Where("username = ?", username).
			Scan(ctx, &expectedUser)
		require.NoError(t, err)
		require.Equal(t, expectedUser.PasswordHash, model.NoPasswordLogin)

		// Changing back to unremote unsets password to blank.
		_, err = api.PatchUser(ctx, &apiv1.PatchUserRequest{
			UserId: resp.User.Id,
			User: &userv1.PatchUser{
				Remote: ptrs.Ptr(true),
			},
		})
		require.NoError(t, err)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
			Password: "testpassword",
		})
		require.ErrorIs(t, err, grpcutil.ErrInvalidCredentials)

		_, err = api.Login(ctx, &apiv1.LoginRequest{
			Username: username,
		})
		require.ErrorIs(t, err, grpcutil.ErrInvalidCredentials)
	})
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

	userIds := fetchUserIds(ctx, t, api, &apiv1.GetUsersRequest{})
	for _, u := range []model.UserID{userID1, userID2, userID3, userID4} {
		require.Contains(t, userIds, u, fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(ctx, t, api, &apiv1.GetUsersRequest{Admin: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID3, userID4} {
		require.Contains(t, userIds, u, fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(ctx, t, api, &apiv1.GetUsersRequest{Active: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID2, userID3} {
		require.Contains(t, userIds, u, fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
	}
	userIds = fetchUserIds(ctx, t, api, &apiv1.GetUsersRequest{Active: ptrs.Ptr(true), Admin: ptrs.Ptr(true)})
	for _, u := range []model.UserID{userID3} {
		require.Contains(t, userIds, u, fmt.Sprintf("userIds: %v, expected user id: %d", userIds, u))
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
			Password:    ptrs.Ptr(user.ReplicateClientSideSaltAndHash(password)),
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

func TestPatchUsers(t *testing.T) {
	// currently activate/deactivate only
	api, _, ctx := setupAPITest(t, nil)
	userID, err := user.Add(ctx,
		&model.User{
			Username: uuid.New().String(),
			Active:   false,
		},
		nil,
	)
	require.NoError(t, err)

	resp, err := api.PatchUsers(ctx, &apiv1.PatchUsersRequest{
		Activate: true,
		UserIds:  []int32{int32(userID)},
	})
	require.NoError(t, err)
	require.Equal(t, "", resp.Results[0].Error)

	resp2, err := api.GetUser(ctx, &apiv1.GetUserRequest{
		UserId: int32(userID),
	})
	require.NoError(t, err)
	require.True(t, resp2.User.Active)
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
	altMockRM ...*mocks.ResourceManager,
) (*apiServer, *mocks.UserAuthZ, model.User, context.Context) {
	api, curUser, ctx := setupAPITest(t, pgdb, altMockRM...)

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

func TestAuthzPostUserDuplicate(t *testing.T) {
	api, authzUsers, _, ctx := setupUserAuthzTest(t, nil)
	user := &userv1.User{
		Username:       uuid.New().String(),
		Admin:          true,
		AgentUserGroup: nil,
	}

	authzUsers.On("CanCreateUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Successfully post user once.
	_, err := api.PostUser(ctx, &apiv1.PostUserRequest{User: user})
	require.NoError(t, err)

	// Post duplicate user & receive expected error.
	expectedErr := apiPkg.ErrUserExists
	_, err = api.PostUser(ctx, &apiv1.PostUserRequest{User: user})
	require.Contains(t, expectedErr.Error(), err.Error())
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
	require.Equal(t, 1, activityCount, ctx)

	_, err = api.PostUserActivity(ctx, &apiv1.PostUserActivityRequest{
		ActivityType: userv1.ActivityType_ACTIVITY_TYPE_GET,
		EntityType:   userv1.EntityType_ENTITY_TYPE_PROJECT,
		EntityId:     1,
	})

	require.NoError(t, err)

	activityCount, err = getActivityEntry(ctx, curUser.ID, 1)
	require.NoError(t, err)
	require.Equal(t, 1, activityCount, ctx)
}

func getActivityEntry(ctx context.Context, userID model.UserID, entityID int32) (int, error) {
	return db.Bun().NewSelect().Model((*model.UserActivity)(nil)).Where("user_id = ?",
		int32(userID)).Where("entity_id = ?", entityID).Count(ctx)
}

// TestPostAccessToken tests given user's WITHOUT lifespan input
// POST /api/v1/users/{user_Id}/token - Create and get a user's access token.
func TestPostAccessToken(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	userID, err := getTestUser(ctx)
	require.NoError(t, err)

	// Without lifespan input
	resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId: int32(userID),
	})
	token, tokenID := resp.Token, resp.TokenId
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	err = checkOutput(ctx, t, api, userID, "", "")
	require.NoError(t, err)

	// cleaning test data
	err = user.DeleteSessionByID(context.TODO(), model.SessionID(userID))
	require.NoError(t, err)
}

// TestPostAccessTokenWithLifespan tests given user's  WITH lifespan, description input
// POST /api/v1/users/{user_Id}/token - Create and get a user's access token
// Input body contains lifespan = "5s or "2h".
func TestPostAccessTokenWithLifespan(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	userID, err := getTestUser(ctx)
	require.NoError(t, err)

	// With lifespan input
	resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId:      int32(userID),
		Lifespan:    lifespan,
		Description: desc,
	})
	token, tokenID := resp.Token, resp.TokenId
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, tokenID)

	err = checkOutput(ctx, t, api, userID, lifespan, desc)
	require.NoError(t, err)

	// cleaning test data
	err = user.DeleteSessionByID(context.TODO(), model.SessionID(userID))
	require.NoError(t, err)
}

// TestGetAccessTokens tests all access token info
// GET /api/v1/users/tokens - Get all access token info
// from user_sessions db for admin.
func TestGetAccessTokens(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	// Create test user 1 and do not revoke or set description
	userID1, err := getTestUser(ctx)
	require.NoError(t, err)

	createTestToken(ctx, t, api, userID1)

	usernameForGivenUserID, err := getUsernameForGivenUserID(ctx, userID1)
	require.NoError(t, err)
	filter := fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)

	tokenInfo1, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfo1)
	// Loop through the returned access token records
	userIDFound := false
	tokenID1 := 0
	for _, tokenInfo := range tokenInfo1.TokenInfo {
		// Check if user ID matches
		if tokenInfo.UserId == int32(userID1) {
			userIDFound = true
			tokenID1 = int(tokenInfo.Id)
		}
	}
	require.True(t, userIDFound, "User ID should be present in tokenInfo1")

	// Create test user 2 and revoke and set description
	userID2, err := getTestUser(ctx)
	require.NoError(t, err)

	createTestToken(ctx, t, api, userID2)

	usernameForGivenUserID, err = getUsernameForGivenUserID(ctx, userID2)
	require.NoError(t, err)
	filter = fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)

	// Tests TestGetAccessToken info for giver userID
	tokenInfo2, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfo2)
	// Loop through the returned access token records
	userIDFound = false
	tokenID2 := 0
	for _, tokenInfo := range tokenInfo2.TokenInfo {
		// Check if user ID matches
		if tokenInfo.UserId == int32(userID2) {
			userIDFound = true
			tokenID2 = int(tokenInfo.Id)
		}
	}
	require.True(t, userIDFound, "User ID should be present in tokenInfo2")

	description := "test desc"
	// Tests TestPatchAccessToken info for giver tokenID
	_, err = api.PatchAccessToken(ctx, &apiv1.PatchAccessTokenRequest{
		TokenId:     int32(tokenID2),
		Description: &description,
		SetRevoked:  true,
	})
	require.NoError(t, err)

	resp, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{})
	require.NoError(t, err)

	for _, u := range resp.TokenInfo {
		if model.UserID(u.Id) == model.UserID(tokenID1) {
			require.False(t, u.Revoked)
			require.NotEqual(t, description, u.Description)
		} else if model.UserID(u.Id) == model.UserID(tokenID2) {
			require.True(t, u.Revoked)
			require.Equal(t, description, u.Description)
		}
	}

	// Clean up of test users
	for _, u := range resp.TokenInfo {
		err = user.DeleteSessionByID(context.TODO(), model.SessionID(u.Id))
		require.NoError(t, err)
	}
}

// TestAuthzOtherAccessToken tests authorization of user creating/viewing/patching
// given user's token.
func TestAuthzOtherAccessToken(t *testing.T) {
	api, authzUsers, curUser, ctx := setupUserAuthzTest(t, nil)

	// POST API Auth check
	expectedErr := status.Error(codes.PermissionDenied, "canCreateAccessToken")
	authzUsers.On("CanCreateAccessToken", mock.Anything, curUser, curUser).
		Return(fmt.Errorf("canCreateAccessToken")).Once()

	_, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
		UserId: int32(curUser.ID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())

	// GET API Auth check
	var query bun.SelectQuery
	expectedErr = status.Error(codes.PermissionDenied, "canGetAccessTokens")
	authzUsers.On("CanGetAccessTokens", mock.Anything, curUser, mock.Anything, curUser.ID).
		Return(&query, fmt.Errorf("canGetAccessTokens")).Once()

	filter := fmt.Sprintf(`{"username":"%s"}`, curUser.Username)
	_, err = api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func checkOutput(ctx context.Context, t *testing.T, api *apiServer, userID model.UserID,
	lifespan string, desc string,
) error {
	usernameForGivenUserID, err := getUsernameForGivenUserID(ctx, userID)
	require.NoError(t, err)

	filter := fmt.Sprintf(`{"username":"%s"}`, usernameForGivenUserID)
	tokenInfos, err := api.GetAccessTokens(ctx, &apiv1.GetAccessTokensRequest{
		Filter: filter,
	})
	require.NoError(t, err)
	require.NotNil(t, tokenInfos)

	tokenID := model.TokenID(0)
	if desc != "" {
		descFound := false
		for _, tokenInfo := range tokenInfos.TokenInfo {
			// Check if user ID matches
			if tokenInfo.Description == desc {
				descFound = true
				tokenID = model.TokenID(tokenInfo.Id)
			}
		}
		require.True(t, descFound, "Desc should be present in tokenInfo")
	}

	if lifespan != "" {
		err = testSetLifespan(ctx, t, userID, lifespan, tokenID)
		require.NoError(t, err)
	}

	return nil
}

func createTestToken(ctx context.Context, t *testing.T, api *apiServer, userID model.UserID) {
	if userID == 0 {
		// Create a test token for current user without lifespan input
		resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{})
		require.NoError(t, err)
		require.NotNil(t, resp.Token)
		require.NotNil(t, resp.TokenId)
	} else {
		// Create a test token for user_id without lifespan input
		resp, err := api.PostAccessToken(ctx, &apiv1.PostAccessTokenRequest{
			UserId: int32(userID),
		})
		require.NoError(t, err)
		require.NotNil(t, resp.Token)
		require.NotNil(t, resp.TokenId)
	}
}

func getTestUser(ctx context.Context) (model.UserID, error) {
	return user.Add(
		ctx,
		&model.User{
			Username: uuid.New().String(),
			Remote:   false,
		},
		nil,
	)
}

func testSetLifespan(ctx context.Context, t *testing.T, userID model.UserID, lifespan string,
	tokenID model.TokenID,
) error {
	expLifespan := user.DefaultTokenLifespan
	var err error
	if lifespan != "" {
		expLifespan, err = time.ParseDuration(lifespan)
		if err != nil {
			return fmt.Errorf("Invalid duration format")
		}
	}
	var expiry, createdAt time.Time
	err = db.Bun().NewSelect().
		Table("user_sessions").
		Column("expiry", "created_at").
		Where("user_id = ?", userID).
		Where("token_type = ?", model.TokenTypeAccessToken).
		Where("id = ?", tokenID).
		Scan(ctx, &expiry, &createdAt)
	if err != nil {
		return fmt.Errorf("Error getting the set lifespan, creation time")
	}

	actLifespan := expiry.Sub(createdAt)
	require.Equal(t, expLifespan, actLifespan)

	return nil
}

func getUsernameForGivenUserID(ctx context.Context, userID model.UserID) (string, error) {
	var usernameForGivenUserID string
	err := db.Bun().NewSelect().
		Table("users").
		Column("username").
		Where("id = ?", userID).
		Scan(ctx, &usernameForGivenUserID)
	if err != nil {
		return "", fmt.Errorf("Error getting username for the user")
	}
	return usernameForGivenUserID, nil
}
