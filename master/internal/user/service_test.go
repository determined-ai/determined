package user

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/require"
	//"github.com/pkg/errors"
	//"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/mocks"

	"github.com/determined-ai/determined/master/pkg/model"
)

// Mocks don't like being initialized more than once?
var (
	dbSingleton        *mocks.DB        = &mocks.DB{}
	userAuthzSingleton *mocks.UserAuthZ = &mocks.UserAuthZ{}
)

func init() {
	AuthZProvider.Register("mock", userAuthzSingleton)
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
}

func setup() (*Service, *mocks.DB, *mocks.UserAuthZ, echo.Context) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(model.User{})

	InitService(dbSingleton, nil, nil)
	return GetService(), dbSingleton, userAuthzSingleton, ctx
}

func TestAuthzGetMe(t *testing.T) {
	svc, _, authzUser, ctx := setup()

	authzUser.On("CanGetMe", model.User{}).Return(fmt.Errorf("canGetMeError"))

	_, err := svc.getMe(ctx)
	authzUser.AssertCalled(t, "CanGetMe", model.User{})
	require.Contains(t, err.Error(), "canGetMeError")
}

func TestAuthzUserList(t *testing.T) {
	svc, db, authzUser, ctx := setup()

	db.On("UserList").Return([]model.FullUser{}, nil)
	authzUser.On("FilterUserList", model.User{}, []model.FullUser{}).
		Return(nil, fmt.Errorf("filterUserListError"))

	_, err := svc.getUsers(ctx)
	authzUser.AssertCalled(t, "FilterUserList", model.User{}, []model.FullUser{})
	require.Contains(t, err.Error(), "filterUserListError")
}

func TestAuthzPatchUser(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, authzUser, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	cases := []struct {
		expectedCall string
		args         []any
		body         string
	}{
		{"CanSetUsersPassword", []any{model.User{}, model.User{}}, `{"password":"new"}`},
		{"CanSetUsersActive", []any{model.User{}, model.User{}, false}, `{"active":false}`},
		{"CanSetUsersAdmin", []any{model.User{}, model.User{}, true}, `{"admin":true}`},
		{
			"CanSetUsersAgentUserGroup",
			[]any{
				model.User{},
				model.User{},
				model.AgentUserGroup{GID: 3, UID: 3, User: "uname", Group: "gname"},
			},
			`{"agent_user_group":{"gid":3,"uid":3,"user":"uname","group":"gname"}}`,
		},
	}
	for _, testCase := range cases {
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))

		db.On("UserByUsername", "admin").Return(&model.User{}, nil)
		authzUser.On(testCase.expectedCall, testCase.args...).
			Return(fmt.Errorf(testCase.expectedCall + "Error"))

		_, err := svc.patchUser(ctx)
		authzUser.AssertCalled(t, testCase.expectedCall, testCase.args...)
		require.Contains(t, err.Error(), testCase.expectedCall+"Error")
	}
}

func TestAuthzPatchUsername(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, authzUser, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	db.On("UserByUsername", "admin").Return(&model.User{}, nil)
	authzUser.On("CanSetUsersUsername", model.User{}, model.User{}).
		Return(fmt.Errorf("canSetUsersUsernameError"))

	_, err := svc.patchUsername(ctx)
	authzUser.AssertCalled(t, "CanSetUsersUsername", model.User{}, model.User{})
	require.Contains(t, err.Error(), "canSetUsersUsernameError")
}

func TestAuthzPostUsername(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, authzUser, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")
	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))

	var agentGroup *model.AgentUserGroup
	db.On("UserByUsername", "admin").Return(&model.User{}, nil)
	authzUser.On("CanCreateUser", model.User{}, model.User{Username: "x"}, agentGroup).
		Return(fmt.Errorf("canCreateUserError"))

	_, err := svc.postUser(ctx)
	authzUser.AssertCalled(t, "CanCreateUser", model.User{}, model.User{Username: "x"}, agentGroup)
	require.Contains(t, err.Error(), "canCreateUserError")
}

func TestAuthzGetUserImage(t *testing.T) {
	svc, _, authzUser, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	authzUser.On("CanGetUsersImage", model.User{}, "admin").
		Return(fmt.Errorf("canGetUsersImageError"))

	_, err := svc.getUserImage(ctx)
	authzUser.AssertCalled(t, "CanGetUsersImage", model.User{}, "admin")
	require.Contains(t, err.Error(), "canGetUsersImageError")
}
