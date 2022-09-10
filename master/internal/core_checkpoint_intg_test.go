//go:build integration
// +build integration

package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
)

func SetupCheckpointAuthTestEcho(t *testing.T) (
	*apiServer, model.User, echo.Context,
) {
	api, _, _, user, _ := SetupProjectAuthZTest(t)

	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(user)

	return api, user, ctx
}

func TestAuthZGetCheckpointEcho(t *testing.T) {

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id string) error
		Params       []any
	}{
		{"CanGetCheckpointTgz", func(id string) error {
			api, _, ctx := SetupCheckpointAuthTestEcho(t)
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/tgz", nil))
			return api.m.getCheckpointTgz(ctx)
		}, []any{mock.Anything, mock.Anything}},
		{"CanGetCheckpointZip", func(id string) error {
			api, _, ctx := SetupCheckpointAuthTestEcho(t)
			ctx.SetParamNames("checkpoint_uuid")
			ctx.SetParamValues(id)
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/zip", nil))
			return api.m.getCheckpointZip(ctx)
		}, []any{mock.Anything, mock.Anything}},
	}

	for _, curCase := range cases {
		// Checkpoint not found
		require.Equal(t,
			echo.NewHTTPError(http.StatusNotFound, "checkpoint 7e0bad2c-b3f6-4988-916c-eb5581b19db0 does not exist"),
			curCase.IDToReqCall("7e0bad2c-b3f6-4988-916c-eb5581b19db0"))

		// Invalid checkpoint UUID
		require.Equal(t,
			echo.NewHTTPError(http.StatusBadRequest,
				"unable to parse checkpoint UUID badbad-b3f6-4988-916c-eb5581b19db0: invalid UUID length: 34"),
			curCase.IDToReqCall("badbad-b3f6-4988-916c-eb5581b19db0"))

		// authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).Return(false, nil).Once()
		// require.Equal(t, expNotFoundErrEcho(exp.ID), curCase.IDToReqCall(exp.ID))

		// // CanGetExperiment error returns unmodified.
		// expectedErr := fmt.Errorf("canGetExperimentError")
		// authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).
		// 	Return(false, expectedErr).Once()
		// require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// // Deny returns error with Forbidden.
		// expectedErr = echo.NewHTTPError(http.StatusForbidden, curCase.DenyFuncName+"Error")
		// authZExp.On("CanGetExperiment", mock.Anything, mock.Anything).Return(true, nil).Once()
		// authZExp.On(curCase.DenyFuncName, curCase.Params...).
		// 	Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		// require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}
}
