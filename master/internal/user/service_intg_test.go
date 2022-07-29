//go:build integration
// +build integration

package user

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pkg/errors"
	//"github.com/pkg/errors"

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

	InitService(pgDB, nil, nil)
	return GetService(), userAuthzSingleton, ctx
}

func TestAuthzUserList(t *testing.T) {
	svc, authzUser, ctx := setup(t)

	// Error passes through.
	expectedErr := fmt.Errorf("filterUserListError")
	authzUser.On("FilterUserList", model.User{}, mock.Anything).Return(nil, expectedErr).Once()
	_, err := svc.getUsers(ctx)
	require.Equal(t, err, expectedErr)

	// Nil error returns whatever FilterUserList returns.
	users := []model.FullUser{
		{Username: "a"},
		{Username: "b"},
	}
	authzUser.On("FilterUserList", model.User{}, mock.Anything).Return(users, nil).Once()
	actualUsers, err := svc.getUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, users, actualUsers)
}

func TestAuthzPatchUser(t *testing.T) {
	svc, authzUser, ctx := setup(t)
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	cases := []struct {
		expectedCall string
		args         []any
		body         string
	}{
		{"CanSetUsersPassword", []any{model.User{}, mock.Anything}, `{"password":"new"}`},
		{"CanSetUsersActive", []any{model.User{}, mock.Anything, false}, `{"active":false}`},
		{"CanSetUsersAdmin", []any{model.User{}, mock.Anything, true}, `{"admin":true}`},
		{
			"CanSetUsersAgentUserGroup",
			[]any{
				model.User{},
				mock.Anything,
				model.AgentUserGroup{GID: 3, UID: 3, User: "uname", Group: "gname"},
			},
			`{"agent_user_group":{"gid":3,"uid":3,"user":"uname","group":"gname"}}`,
		},
	}
	for _, testCase := range cases {
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))

		expectedErr := errors.Wrap(forbiddenError, testCase.expectedCall+"Error")
		authzUser.On(testCase.expectedCall, testCase.args...).
			Return(fmt.Errorf(testCase.expectedCall + "Error")).Once()

		_, err := svc.patchUser(ctx)
		require.Equal(t, expectedErr.Error(), err.Error())

		// TODO test for leaking information here!
		// Specifically we can leak the existance of a user here.
		// We need to ensure we get the same error as not found as found.

	}
}

func TestAuthzPatchUsername(t *testing.T) {
	svc, authzUser, ctx := setup(t)
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	expectedErr := errors.Wrap(forbiddenError, "canSetUsersUsernameError")
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	authzUser.On("CanSetUsersUsername", model.User{}, mock.Anything).
		Return(fmt.Errorf("canSetUsersUsernameError")).Once()

	_, err := svc.patchUsername(ctx)
	require.Equal(t, expectedErr.Error(), err.Error())

	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	// We can also leak if another username is taken but we are stuck here / / /.
	// Like changing your own username is often something people can do.
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
	authzUser.On("CanCreateUser", model.User{}, model.User{Username: "x"}, agentGroup).
		Return(fmt.Errorf("canCreateUserError")).Once()

	_, err := svc.postUser(ctx)
	authzUser.AssertCalled(t, "CanCreateUser", model.User{}, model.User{Username: "x"}, agentGroup)
	require.Contains(t, expectedErr.Error(), err.Error())

	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	// Don't think we can get around this!? -- Like username is taken is a fact of life.
}

func TestAuthzGetUserImage(t *testing.T) {
	svc, authzUser, ctx := setup(t)
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	expectedErr := errors.Wrap(forbiddenError, "canGetUsersImageError")
	authzUser.On("CanGetUsersImage", model.User{}, "admin").
		Return(fmt.Errorf("canGetUsersImageError"))

	_, err := svc.getUserImage(ctx)
	authzUser.AssertCalled(t, "CanGetUsersImage", model.User{}, "admin")
	require.Contains(t, expectedErr.Error(), err.Error())

	// TODO test for the existance of a user here.
	// Should return same for non existant user.
}
