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

var dbSingleton *mocks.DB

func mustErrorFromAuthz(t *testing.T, expected any, actualError error) {
	require.Contains(t, actualError.Error(), functionName(expected))
}

func setup() (*Service, *mocks.DB, echo.Context) {
	// Make mock database (which for some reason can't be initialized each test?)
	if dbSingleton == nil {
		dbSingleton = &mocks.DB{}
	}

	// Set master config to use our mock AuthZ interface.
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "restricted"}

	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(model.User{})

	InitService(dbSingleton, nil, nil)
	return GetService(), dbSingleton, ctx
}

func TestAuthzGetMe(t *testing.T) {
	svc, _, ctx := setup()
	_, err := svc.getMe(ctx)
	mustErrorFromAuthz(t, AuthZProvider.Get().CanGetMe, err)
}

func TestAuthzUserList(t *testing.T) {
	svc, db, ctx := setup()
	db.On("UserList").Return(nil, nil)
	_, err := svc.getUsers(ctx)
	mustErrorFromAuthz(t, AuthZProvider.Get().FilterUserList, err)
}

func TestAuthzPatchUser(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	cases := []struct {
		expected any
		body     string
	}{
		{AuthZProvider.Get().CanSetUsersPassword, `{"password":"new"}`},
		{AuthZProvider.Get().CanSetUsersActive, `{"active":false}`},
		{AuthZProvider.Get().CanSetUsersAdmin, `{"admin":true}`},
		{
			AuthZProvider.Get().CanSetUsersAgentUserGroup,
			`{"agent_user_group":{"gid":3,"uid":3,"user":"uname","group":"gname"}}`,
		},
	}
	for _, testCase := range cases {
		ctx.SetRequest(httptest.NewRequest(http.MethodPatch, "/",
			strings.NewReader(testCase.body)))

		db.On("UserByUsername", "admin").Return(&model.User{}, nil)
		_, err := svc.patchUser(ctx)

		fmt.Println(ctx.ParamValues(), ctx.ParamNames())

		mustErrorFromAuthz(t, testCase.expected, err)
	}
}

func TestAuthzPatchUsername(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	db.On("UserByUsername", "admin").Return(&model.User{}, nil)
	_, err := svc.patchUsername(ctx)
	mustErrorFromAuthz(t, AuthZProvider.Get().CanSetUsersUsername, err)
}

func TestAuthzPostUsername(t *testing.T) {
	// TODO test for leaking information here!
	// Specifically we can leak the existance of a user here.
	svc, db, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	ctx.SetRequest(httptest.NewRequest("", "/", strings.NewReader(`{"username":"x"}`)))
	db.On("UserByUsername", "admin").Return(&model.User{}, nil)
	_, err := svc.postUser(ctx)
	mustErrorFromAuthz(t, AuthZProvider.Get().CanCreateUser, err)
}

func TestAuthzGetUserImage(t *testing.T) {
	svc, _, ctx := setup()
	ctx.SetParamNames("username")
	ctx.SetParamValues("admin")

	_, err := svc.getUserImage(ctx)
	mustErrorFromAuthz(t, AuthZProvider.Get().CanGetUsersImage, err)
}
