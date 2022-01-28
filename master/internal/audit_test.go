package internal

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/pkg/model"
)

type logStore struct {
	inner []*logrus.Entry
}

func (l *logStore) Fire(e *logrus.Entry) error {
	l.inner = append(l.inner, e)
	return nil
}

func (l *logStore) Levels() []logrus.Level {
	return logrus.AllLevels
}

func TestAuditLogMiddleware(t *testing.T) {
	// Given an echo server with our middleware, where we can introspect logs, and with a valid
	// DetContext.
	const url = "localhost:8081"

	logs := logStore{}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.AddHook(&logs)

	e := echo.New()
	e.Use(func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &detContext.DetContext{Context: c}
			cc.SetUser(model.User{
				Username: "brad",
			})
			return h(cc)
		}
	})
	e.Use(auditLogMiddleware())
	e.Any("/ok", echo.HandlerFunc(func(c echo.Context) error {
		return nil
	}))
	e.Any("/notok", echo.HandlerFunc(func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusUnauthorized)
	}))
	go func() {
		if err := e.Start(url); err != nil && err != http.ErrServerClosed {
			t.Logf("failed to start server: %s", err)
		}
	}()
	defer e.Shutdown(context.TODO()) //nolint:errcheck

	// Ensure server is up, before testing it.
	i := 0
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for range tk.C {
		resp, err := http.Get("http://" + url + "/proxy")
		if err == nil {
			resp.Body.Close() //nolint:errcheck
			// Clear any logs this may have produced, even though that should be none.
			logs.inner = nil
			break
		}
		i++
		if i > 5 {
			t.FailNow()
		}
	}

	// When making a GET request to some resource, proxies.
	resp, err := http.Get("http://" + url + "/proxy")
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	for path := range staticWebDirectoryPaths {
		resp, err = http.Get("http://" + url + path)
		require.NoError(t, err)
		defer resp.Body.Close() //nolint:errcheck
	}
	// Then webui static files are ignored.
	require.Len(t, logs.inner, 0)

	// When making a GET request to some resource.
	resp, err = http.Get("http://" + url + "/ok")
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	// Then the most recent log should indicate this request was logged at level DEBUG.
	require.Len(t, logs.inner, 1)
	require.Equal(t, logrus.DebugLevel, logs.inner[0].Level)
	require.Contains(t, logs.inner[0].Message, "GET /ok")

	// When making a GET request to some resource.
	resp, err = http.Post("http://"+url+"/ok", "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	// Then the most recent log should indicate this request was logged at level INFO.
	require.Len(t, logs.inner, 2)
	require.Equal(t, logrus.InfoLevel, logs.inner[1].Level)
	require.Contains(t, logs.inner[1].Message, "POST /ok")

	// When making a GET request to some resource.
	resp, err = http.Post("http://"+url+"/notok", "application/json", strings.NewReader(`
	{
		"username": "brad", "password": "adminorsomething",
	}`))
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	// Then the most recent log should look right, and the middleware should _not_ cleverly
	// log the body because it would leak passwords.
	require.Len(t, logs.inner, 3)
	require.Equal(t, logrus.InfoLevel, logs.inner[2].Level)
	require.Contains(t, logs.inner[2].Message, "/notok")
	require.Equal(t, logs.inner[2].Data["unauthorized"], true)
}
