//go:build integration
// +build integration

package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/olddata"
)

func newTestEchoContext(user model.User) echo.Context {
	e := echo.New()
	c := e.NewContext(nil, nil)
	ctx := &context.DetContext{Context: c}
	ctx.SetUser(user)
	return ctx
}

func TestLegacyExperimentsEcho(t *testing.T) {
	err := etc.SetRootPath("../static/srv")
	require.NoError(t, err)

	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()

	prse := olddata.PreRemoveStepsExperiments()
	prse.MustMigrate(t, pgDB, "file://../static/migrations")

	api, user, _ := setupAPITest(t, pgDB)

	setExperimentIDParam := func(ctx echo.Context, id int32) {
		ctx.SetParamNames("experiment_id")
		ctx.SetParamValues(strconv.FormatInt(int64(id), 10))
	}

	t.Run("GetExperimentCheckpoints", func(t *testing.T) {
		ctx := newTestEchoContext(user)
		path := "/?save_experiment_best=0&save_trial_best=0&save_trial_latest=0"
		req := httptest.NewRequest(http.MethodGet, path, nil)
		ctx.SetRequest(req)
		setExperimentIDParam(ctx, prse.CompletedPBTExpID)
		_, err = api.m.getExperimentCheckpointsToGC(ctx)
		require.NoError(t, err)
	})
}

func TestAuthZGetExperimentAndCanDoActionsEcho(t *testing.T) {
	api, authZExp, _, curUser, _ := setupExpAuthTest(t, nil)
	exp := createTestExp(t, api, curUser)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
		Params       []any
	}{
		{"CanGetExperimentArtifacts", func(id int) error {
			ctx := newTestEchoContext(curUser)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost, "/", nil))
			return api.m.getExperimentModelDefinition(ctx)
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetExperimentArtifacts", func(id int) error {
			ctx := newTestEchoContext(curUser)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost, "/?path=rootPath", nil))
			return api.m.getExperimentModelFile(ctx)
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
		{"CanGetExperimentArtifacts", func(id int) error {
			ctx := newTestEchoContext(curUser)
			ctx.SetParamNames("experiment_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodPost,
				"/?save_experiment_best=10&save_trial_best=2&save_trial_latest=3", nil))

			_, err := api.m.getExperimentCheckpointsToGC(ctx)
			return err
		}, []any{mock.Anything, mock.Anything, mock.Anything}},
	}

	for _, curCase := range cases {
		// Not found returns same as permission denied.
		require.Equal(t, apiPkg.NotFoundErrs("experiment", "-999", false), curCase.IDToReqCall(-999))

		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.Equal(t, apiPkg.NotFoundErrs("experiment", fmt.Sprint(exp.ID), false),
			curCase.IDToReqCall(exp.ID))

		// CanGetExperiment error returns unmodified.
		expectedErr := fmt.Errorf("canGetExperimentError")
		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(expectedErr).Once()
		require.Equal(t, expectedErr, curCase.IDToReqCall(exp.ID))

		// Deny returns error with Forbidden.
		expectedErr = echo.NewHTTPError(http.StatusForbidden, curCase.DenyFuncName+"Error")
		authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		authZExp.On(curCase.DenyFuncName, curCase.Params...).
			Return(fmt.Errorf(curCase.DenyFuncName + "Error")).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(exp.ID).Error())
	}
}
