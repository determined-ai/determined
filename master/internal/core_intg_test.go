//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-pg/pg/v10"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"

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
				ClusterName: "TEST",
				Status:      model.Healthy,
			},
		}).Once()

		assertHealthCheck(t, http.StatusOK, model.HealthCheck{
			Status:   model.Healthy,
			Database: model.Healthy,
			ResourceManagers: []model.ResourceManagerHealth{
				{
					ClusterName: "TEST",
					Status:      model.Healthy,
				},
			},
		})
	})

	t.Run("unhealthy", func(t *testing.T) {
		mockRM := &mocks.ResourceManager{}
		api.m.rm = mockRM
		mockRM.On("HealthCheck").Return([]model.ResourceManagerHealth{
			{
				ClusterName: "TEST",
				Status:      model.Unhealthy,
			},
		}).Once()

		require.NoError(t, db.Bun().Close())
		thePgDB = nil // Invalidate our db cache now that we closed the database.

		assertHealthCheck(t, http.StatusServiceUnavailable, model.HealthCheck{
			Status:   model.Unhealthy,
			Database: model.Unhealthy,
			ResourceManagers: []model.ResourceManagerHealth{
				{
					ClusterName: "TEST",
					Status:      model.Unhealthy,
				},
			},
		})
	})
}

func TestRun(t *testing.T) {
	type testScenario struct {
		name            string
		initialPassword string
		repeats         int
		checkRunErr     func(require.TestingT, error, ...interface{})
	}

	test := func(t *testing.T, scenario testScenario) {
		pgdb, teardown := db.MustResolveNewPostgresDatabase(t)
		t.Cleanup(teardown)
		mockRM := MockRM()

		pgOpts, err := pg.ParseURL(pgdb.URL)
		require.NoError(t, err)

		addr := strings.SplitN(pgOpts.Addr, ":", 2)

		for i := 0; i < scenario.repeats; i++ {
			m := &Master{
				rm: mockRM,
				config: &config.Config{
					Security: config.SecurityConfig{
						InitialUserPassword: scenario.initialPassword,
					},
					InternalConfig: config.InternalConfig{
						ExternalSessions: model.ExternalSessions{},
					},
					TaskContainerDefaults: model.TaskContainerDefaultsConfig{},
					ResourceConfig:        *config.DefaultResourceConfig(),
					Logging: model.LoggingConfig{
						DefaultLoggingConfig: &model.DefaultLoggingConfig{},
					},
				},
				taskSpec: &tasks.TaskSpec{SSHRsaSize: 1024},
			}
			require.NoError(t, m.config.Resolve())
			m.config.DB = config.DBConfig{
				User:             pgOpts.User,
				Password:         pgOpts.Password,
				Migrations:       "file://../static/migrations",
				ViewsAndTriggers: "../static/views_and_triggers",
				Host:             addr[0],
				Port:             addr[1],
				Name:             pgOpts.Database,
				SSLMode:          "disable",
			}
			// listen on any available port, we don't care
			m.config.Port = 0
			m.config.FeatureSwitches = []string{
				"prevent_blank_password",
			}

			ctx, cancel := context.WithCancel(context.Background())
			gRPCLogInitDone := make(chan struct{})
			var runErr error
			go func() {
				defer cancel()
				runErr = m.Run(ctx, gRPCLogInitDone)
			}()

			select {
			case <-gRPCLogInitDone:
				cancel()
			case <-ctx.Done():
				require.ErrorIs(t, ctx.Err(), context.Canceled)
			}
			scenario.checkRunErr(t, runErr)
		}
	}

	scenarios := []testScenario{
		{
			name:            "blank password",
			initialPassword: "",
			repeats:         5,
			checkRunErr:     require.Error,
		},
		// TODO: DET-10314 - the "happy path" is much harder to test than errors,
		// because once Run() gets all the way to actually serving endpoints etc.
		// there's a delicate shutdown ordering needed to avoid nil derefs on
		// logging, db connections, etc.
		// Running bcrypt is also ~20 seconds per password, so initialization is
		// inherently slow.
		// {
		// 	name:            "strong enough password",
		// 	initialPassword: "testPassword1",
		// 	repeats:         1,
		// 	checkRunErr:     require.NoError,
		// },
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			test(t, scenario)
		})
	}
}
