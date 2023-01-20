//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/model"
)

func expNotFoundErrEcho(id int) error {
	return echo.NewHTTPError(http.StatusNotFound, "experiment not found: %d", id)
}

func setupExpAuthTestEcho(t *testing.T) (
	*apiServer, *mocks.ExperimentAuthZ, *mocks.ProjectAuthZ, model.User, echo.Context,
) {
	api, authZExp, projectAuthZ, user, _ := setupExpAuthTest(t)

	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(user)

	return api, authZExp, projectAuthZ, user, ctx
}

func echoPostExperiment(
	ctx echo.Context, api *apiServer, t *testing.T, params CreateExperimentParams,
) error {
	bytes, err := json.Marshal(params)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPatch, "/",
		strings.NewReader(string(bytes)))
	ctx.SetRequest(req)
	_, err = api.m.postExperiment(ctx)
	return err
}

func TestAuthZPostExperimentEcho(t *testing.T) {
	api, authZExp, _, curUser, ctx := setupExpAuthTestEcho(t)

	_, _, _, _, grpcCtx := setupExpAuthTest(t) //nolint: dogsled
	_, projectID := createProjectAndWorkspace(grpcCtx, t, api)

	// Can't view project passed in.
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	err := echoPostExperiment(ctx, api, t, CreateExperimentParams{
		ConfigBytes: minExpConfToYaml(t),
		ProjectID:   &projectID,
	})
	require.Equal(t, echo.NewHTTPError(http.StatusNotFound,
		fmt.Sprintf("project (%d) not found", projectID)).Error(), err.Error())

	// Can't view project passed in from config.
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
	err = echoPostExperiment(ctx, api, t, CreateExperimentParams{
		ConfigBytes: minExpConfToYaml(t) + "project: Uncategorized\nworkspace: Uncategorized",
	})
	require.Equal(t, echo.NewHTTPError(http.StatusNotFound,
		"workspace 'Uncategorized' or project 'Uncategorized' not found").Error(), err.Error())

	// Same as passing in a non existent project.
	err = echoPostExperiment(ctx, api, t, CreateExperimentParams{
		ConfigBytes: minExpConfToYaml(t) + "project: doesnotexist\nworkspace: doesnotexist",
	})
	require.Equal(t, echo.NewHTTPError(http.StatusNotFound,
		"workspace 'doesnotexist' or project 'doesnotexist' not found").Error(), err.Error())

	// Can't create experiment deny.
	expectedErr := echo.NewHTTPError(http.StatusForbidden, "canCreateExperimentError")
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	authZExp.On("CanCreateExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canCreateExperimentError")).Once()
	err = echoPostExperiment(ctx, api, t, CreateExperimentParams{
		ConfigBytes: minExpConfToYaml(t),
	})
	require.Equal(t, expectedErr, err)

	// Can't activate experiment deny.
	expectedErr = echo.NewHTTPError(http.StatusForbidden, "canActivateExperimentError")
	pAuthZ.On("CanGetProject", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
	authZExp.On("CanCreateExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(nil).Once()
	authZExp.On("CanEditExperiment", mock.Anything, curUser, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canActivateExperimentError")).Once()
	err = echoPostExperiment(ctx, api, t, CreateExperimentParams{
		Activate:    true,
		ConfigBytes: minExpConfToYaml(t),
	})
	require.Equal(t, expectedErr, err)
}

func TestAuthZGetExperimentAndCanDoActionsEcho(t *testing.T) {
	api, authZExp, _, curUser, _ := setupExpAuthTestEcho(t)
	exp := createTestExp(t, api, curUser)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
		Params       []any
	}{
		{"CanGetExperimentArtifacts", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost, "/", nil))
			return api.m.getExperimentModelDefinition(ctx)
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost, "/?path=rootPath", nil))
			return api.m.getExperimentModelFile(ctx)
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetExperimentArtifacts", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost,
				"/?save_experiment_best=10&save_trial_best=2&save_trial_latest=3", nil))

			_, err := api.m.getExperimentCheckpointsToGC(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanSetExperimentsMaxSlots", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			req := httptest.NewRequest(http.MethodPatch, "/",
				strings.NewReader(`{"resources":{"max_slots":5}}`))
			req.Header.Set(echo.HeaderContentType, "application/merge-patch+json")
			ctx.SetRequest(req)
			_, err := api.m.patchExperiment(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything, 5}},
		{"CanSetExperimentsWeight", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			req := httptest.NewRequest(http.MethodPatch, "/",
				strings.NewReader(`{"resources":{"weight":2.5}}`))
			req.Header.Set(echo.HeaderContentType, "application/merge-patch+json")
			ctx.SetRequest(req)
			_, err := api.m.patchExperiment(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything, 2.5}},
		{"CanSetExperimentsPriority", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			req := httptest.NewRequest(http.MethodPatch, "/",
				strings.NewReader(`{"resources":{"priority":3}}`))
			req.Header.Set(echo.HeaderContentType, "application/merge-patch+json")
			ctx.SetRequest(req)
			_, err := api.m.patchExperiment(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything, 3}},
		{"CanSetExperimentsCheckpointGCPolicy", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			req := httptest.NewRequest(http.MethodPatch, "/",
				strings.NewReader(`{"checkpoint_storage":{`+
					`"save_experiment_best":3,"save_trial_best":4,"save_trial_latest":5}}`))
			req.Header.Set(echo.HeaderContentType, "application/merge-patch+json")
			ctx.SetRequest(req)
			_, err := api.m.patchExperiment(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanForkFromExperiment", func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			return echoPostExperiment(ctx, api, t, CreateExperimentParams{
				ConfigBytes: minExpConfToYaml(t),
				ParentID:    &id,
			})
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
	}

	for _, curCase := range cases {
		// Not found returns same as permission denied.
		require.Equal(t, expNotFoundErrEcho(-999), curCase.IDToReqCall(-999))

		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(false, nil).Once()
		require.Equal(t, expNotFoundErrEcho(exp.ID), curCase.IDToReqCall(exp.ID))

		// CanGetExperiment error returns unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(false, expectedErr).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// Deny returns error with Forbidden.
		expectedErr = echo.NewHTTPError(http.StatusForbidden, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(true, nil).Once()
		authZExp.On(curCase.DenyFuncName, curCase.Params...).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}
}
