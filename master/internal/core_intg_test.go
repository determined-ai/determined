//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	api, _, _ := setupAPITest(t, nil)

	assertHealthCheck := func(t *testing.T, expectedCode int, expectedHealth model.HealthCheck) {
		e := echo.New()
		rec := httptest.NewRecorder()
		c := e.NewContext(httptest.NewRequest(http.MethodGet, "/health", nil), rec)

		require.NoError(t, api.m.healthCheckEndpoint(c))
		require.Equal(t, expectedCode, rec.Code)

		var healthCheck model.HealthCheck
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&healthCheck))
		require.Equal(t, expectedHealth, healthCheck)
	}

	t.Run("healthy", func(t *testing.T) {
		mockRM := &mocks.ResourceManager{}
		api.m.rm = mockRM
		mockRM.On("HealthCheck").Return([]model.ResourceManagerHealth{
			{
				Name:   "TEST",
				Status: model.Healthy,
			},
		}).Once()

		assertHealthCheck(t, http.StatusOK, model.HealthCheck{
			Status:   model.Healthy,
			Database: model.Healthy,
			ResourceManagers: []model.ResourceManagerHealth{
				{
					Name:   "TEST",
					Status: model.Healthy,
				},
			},
		})
	})

	t.Run("unhealthy", func(t *testing.T) {
		mockRM := &mocks.ResourceManager{}
		api.m.rm = mockRM
		mockRM.On("HealthCheck").Return([]model.ResourceManagerHealth{
			{
				Name:   "TEST",
				Status: model.Unhealthy,
			},
		}).Once()

		require.NoError(t, db.Bun().Close())
		thePgDB = nil // Invalidate our db cache now that we closed the database.

		assertHealthCheck(t, http.StatusServiceUnavailable, model.HealthCheck{
			Status:   model.Unhealthy,
			Database: model.Unhealthy,
			ResourceManagers: []model.ResourceManagerHealth{
				{
					Name:   "TEST",
					Status: model.Unhealthy,
				},
			},
		})
	})
}
