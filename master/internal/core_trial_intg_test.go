//go:build integration
// +build integration

package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func trialNotFoundErrEcho(id int) error {
	return echo.NewHTTPError(http.StatusNotFound, "trial not found: %d", id)
}

func TestTrialAuthZEcho(t *testing.T) {
	api, authZExp, _, curUser, _ := setupExpAuthTestEcho(t)
	trial := createTestTrial(t, api, curUser)

	funcCalls := []func(id int) error{
		func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("trial_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			_, err := api.m.getTrial(ctx)
			return err
		},
		func(id int) error {
			_, _, _, _, ctx := setupExpAuthTestEcho(t)
			ctx.SetParamNames("trial_id")
			ctx.SetParamValues(fmt.Sprintf("%d", id))
			ctx.SetRequest(httptest.NewRequest(http.MethodGet, "/", nil))
			_, err := api.m.getTrialMetrics(ctx)
			return err
		},
	}

	for i, funcCall := range funcCalls {
		require.Equal(t, trialNotFoundErrEcho(-999), funcCall(-999))

		// Can't view trials experiment gives same error.
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(false, nil).Once()
		require.Equal(t, trialNotFoundErrEcho(trial.ID), funcCall(trial.ID))

		// Experiment view error returns error unmodified.
		expectedErr := fmt.Errorf("canGetTrialError")
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(false, expectedErr).Once()
		require.Equal(t, expectedErr, funcCall(trial.ID))

		// Action func error returns error in forbidden.
		expectedErr = echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("%dError", i))
		authZExp.On("CanGetExperiment", mock.Anything, curUser, mock.Anything).Return(true, nil).Once()
		authZExp.On("CanGetExperimentArtifacts", mock.Anything, curUser, mock.Anything).
			Return(fmt.Errorf("%dError", i)).Once()
		require.Equal(t, expectedErr, funcCall(trial.ID))
	}
}
