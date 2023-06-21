//go:build integration
// +build integration

package user

import (
	stdContext "context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/etc"

	"github.com/determined-ai/determined/master/pkg/model"
)

// Mocks don't like being initialized more than once?
var (
	pgDB               *db.PgDB
	userAuthzSingleton *mocks.UserAuthZ = &mocks.UserAuthZ{}
	notFoundUsername                    = "usernotfound99999"
)

func init() {
	AuthZProvider.Register("mock", userAuthzSingleton)
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
}

func setup(t *testing.T) (*Service, *mocks.UserAuthZ, echo.Context) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(model.User{})

	pgDB = db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../../static/migrations")
	require.NoError(t, etc.SetRootPath("../../static/srv"))

	externalSessions := &model.ExternalSessions{}
	InitService(pgDB, nil, externalSessions)
	return GetService(), userAuthzSingleton, ctx
}

func TestAddUserExec(t *testing.T) {
	setup(t)
	ctx := stdContext.TODO()
	u := &model.User{Username: uuid.New().String()}
	err := AddUserExec(u)
	require.NoError(t, err)

	actual := &model.User{}
	require.NoError(t, db.Bun().NewSelect().Model(actual).Where("id = ?", u.ID).Scan(ctx))
	require.Equal(t, u.Username, actual.Username)

	actualGroup := &struct {
		bun.BaseModel `bun:"table:groups,alias:groups"`
		Name          string `bun:"group_name,notnull"  json:"name"`
	}{}
	require.NoError(t, db.Bun().NewSelect().Model(actualGroup).Where("user_id = ?", u.ID).Scan(ctx))
	require.Equal(t, fmt.Sprintf("%d%s", u.ID, db.PersonalGroupPostfix), actualGroup.Name)

	groupMember := &struct {
		bun.BaseModel `bun:"table:user_group_membership"`
		UserID        model.UserID `bun:"user_id,notnull"`
	}{
		UserID: u.ID,
	}
	require.NoError(t, db.Bun().NewSelect().Model(groupMember).Where("user_id = ?", u.ID).Scan(ctx))
	require.Equal(t, u.ID, groupMember.UserID)
}

func TestAuthzUserList(t *testing.T) {
	svc, authzUser, ctx := setup(t)

	// Error passes through.
	expectedErr := fmt.Errorf("filterUserListError")
	authzUser.On("FilterUserList", mock.Anything, model.User{}, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := svc.getUsers(ctx)
	require.Equal(t, err, expectedErr)

	// Nil error returns whatever FilterUserList returns.
	users := []model.FullUser{
		{Username: "a"},
		{Username: "b"},
	}
	authzUser.On("FilterUserList", mock.Anything, model.User{}, mock.Anything).
		Return(users, nil).Once()
	actualUsers, err := svc.getUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, users, actualUsers)
}

func TestAuthzPatchUser(t *testing.T) {
	svc, authzUser, ctx := setup(t)
	cases := []struct {
		expectedCall string
		args         []any
		body         string
	}{
		{
			"CanSetUsersPassword",
			[]any{mock.Anything, model.User{}, mock.Anything},
			`{"password":"new"}`,
		},
		{
			"CanSetUsersActive",
			[]any{mock.Anything, model.User{}, mock.Anything, false},
			`{"active":false}`,
		},
		{
			"CanSetUsersAdmin",
			[]any{mock.Anything, model.User{}, mock.Anything, true},
			`{"admin":true}`,
		},
		{
			"CanSetUsersAgentUserGroup",
			[]any{
				mock.Anything,
				model.User{},
				mock.Anything,
				model.AgentUserGroup{GID: 3, UID: 3, User: "uname", Group: "gname"},
			},
			`{"agent_user_group":{"gid":3,"uid":3,"user":"uname","group":"gname"}}`,
		},
	}
	for _, testCase := range cases {
		// If we can view the user we get the can set function error.
		ctx.SetParamNames("username")
		ctx.SetParamValues("admin")
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))
		expectedErr := errors.Wrap(forbiddenError, testCase.expectedCall+"Error")
		authzUser.On(testCase.expectedCall, testCase.args...).
			Return(fmt.Errorf(testCase.expectedCall + "Error")).Once()
		authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
			Return(nil).Once()

		_, err := svc.patchUser(ctx)
		require.Equal(t, expectedErr.Error(), err.Error())

		// If CanGetUser returns an error we get that error.
		ctx.SetParamNames("username")
		ctx.SetParamValues("admin")
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))
		authzUser.On(testCase.expectedCall, testCase.args...).
			Return(fmt.Errorf(testCase.expectedCall + "Error")).Once()
		cantGetUserError := fmt.Errorf("cantGetUserError")
		authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
			Return(cantGetUserError).Once()

		_, err = svc.patchUser(ctx)
		require.Equal(t, cantGetUserError, err)

		// If we can't view the user we get the same error as user not being found.
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))
		authzUser.On(testCase.expectedCall, testCase.args...).
			Return(fmt.Errorf(testCase.expectedCall + "Error")).Once()
		authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()

		_, err = svc.patchUser(ctx)
		require.Equal(t, api.NotFoundErrs("user", "admin", false).Error(), err.Error())

		ctx.SetParamNames("username")
		ctx.SetParamValues(notFoundUsername)
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))
		_, err = svc.patchUser(ctx)
		require.Equal(t, api.NotFoundErrs("user", notFoundUsername, false).Error(), err.Error())
	}
}

func TestAuthzPatchUsername(t *testing.T) {
	svc, authzUser, ctx := setup(t)

	// If we can view the user we get canSetUsersUsername error.
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")
	expectedErr := errors.Wrap(forbiddenError, "canSetUsersUsernameError")
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	authzUser.On("CanSetUsersUsername", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canSetUsersUsernameError")).Once()
	authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).Return(nil).Once()

	_, err := svc.patchUsername(ctx)
	require.Equal(t, expectedErr.Error(), err.Error())

	// If we get an error from canGetUser we return that error.
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	authzUser.On("CanSetUsersUsername", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canSetUsersUsernameError")).Once()
	cantGetUserError := fmt.Errorf("cantGetUserError")
	authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
		Return(cantGetUserError).Once()

	_, err = svc.patchUsername(ctx)
	require.Equal(t, cantGetUserError, err)

	// If we can't view the user we get the same error as the user not existing.
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	authzUser.On("CanSetUsersUsername", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canSetUsersUsernameError")).Once()
	authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()

	_, err = svc.patchUsername(ctx)
	require.Equal(t, db.ErrNotFound.Error(), err.Error())

	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	ctx.SetParamNames("username")
	ctx.SetParamValues(notFoundUsername)
	_, err = svc.patchUsername(ctx)
	require.Equal(t, db.ErrNotFound.Error(), err.Error())
}

func TestAuthzPostUser(t *testing.T) {
	svc, authzUser, ctx := setup(t)
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(
		`{"username":"x","agent_user_group":{"uid":1,"gid":2,"user":"u","group":"g"}}`)))

	agentGroup := &model.AgentUserGroup{
		UID:   1,
		GID:   2,
		User:  "u",
		Group: "g",
	}
	expectedErr := errors.Wrap(forbiddenError, "canCreateUserError")
	authzUser.On("CanCreateUser", mock.Anything, model.User{}, model.User{Username: "x"}, agentGroup).
		Return(fmt.Errorf("canCreateUserError")).Once()

	_, err := svc.postUser(ctx)
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzGetUserImage(t *testing.T) {
	svc, authzUser, ctx := setup(t)

	// If we can get the user return the error from canGetUsersImageError.
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")
	expectedErr := errors.Wrap(forbiddenError, "canGetUsersImageError")
	authzUser.On("CanGetUsersImage", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canGetUsersImageError")).Once()
	authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).Return(nil).Once()

	_, err := svc.getUserImage(ctx)
	require.Equal(t, expectedErr.Error(), err.Error())

	// If we get an error from canGetUser we return that error.
	authzUser.On("CanGetUsersImage", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canGetUsersImageError")).Once()
	cantGetUserError := fmt.Errorf("cantGetUserError")
	authzUser.On("CanGetUser", mock.Anything, model.User{}, mock.Anything).
		Return(cantGetUserError).Once()

	_, err = svc.getUserImage(ctx)
	require.Equal(t, cantGetUserError, err)

	// If we can't view the user return the same error as the user not existing.
	authzUser.On("CanGetUsersImage", mock.Anything, model.User{}, mock.Anything).
		Return(fmt.Errorf("canGetUsersImageError"))
	authzUser.On("CanGetUser", mock.Anything, model.User{},
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()

	_, err = svc.getUserImage(ctx)
	require.Equal(t, db.ErrNotFound.Error(), err.Error())

	ctx.SetParamNames("username")
	ctx.SetParamValues(notFoundUsername)
	_, err = svc.getUserImage(ctx)
	require.Equal(t, db.ErrNotFound.Error(), err.Error())
}
