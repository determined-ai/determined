//go:build integration
// +build integration

package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

var (
	tickInterval = 100 * time.Millisecond
	proxyAuth    = func(c echo.Context) (done bool, err error) {
		return true, nil
	}
	serviceIDs = []string{"a", "b", "c"}
	u          = url.URL{Path: "localhost:8082"}
)

func register(t *testing.T, prTCP bool, unauth bool) {
	for _, id := range serviceIDs {
		DefaultProxy.Register(id, &u, prTCP, unauth)
		if DefaultProxy.GetService(id) == nil {
			t.Errorf("failed to find registered service %s", id)
		}
	}
	if len(DefaultProxy.Summaries()) != len(serviceIDs) {
		t.Errorf("failed to register all services")
	}
}

func unregister(t *testing.T) {
	for _, id := range serviceIDs {
		if DefaultProxy.GetService(id) == nil {
			t.Errorf("failed to find registered service %s", id)
		}
		DefaultProxy.Unregister(id)
		if DefaultProxy.GetService(id) != nil {
			t.Errorf("failed to unregister service %s", id)
		}
	}
}

// TODO carolina/bradley: add to utils.
func waitForCondition(timeout time.Duration, condition func() bool) bool {
	for i := 0; i < int(timeout/tickInterval); i++ {
		if condition() {
			return true
		}
		time.Sleep(tickInterval)
	}
	return false
}

func conditionServerUp() bool {
	resp, err := http.Get("http://" + u.Path + "/proxy")
	if err == nil {
		resp.Body.Close() //nolint:errcheck
	}
	return err == nil
}

func TestProxyLifecycle(t *testing.T) {
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../../static/migrations")
	require.NoError(t, etc.SetRootPath("../../static/srv"))

	cases := []struct {
		name                 string
		proxyTCP             bool
		allowUnauthenticated bool
	}{
		// ProxyTCP = AllowUnauthenticated = True
		{"tcp & unauthenticated true", true, true},
		// ProxyTCP = AllowUnauthenticated = False
		{"tcp & unauthenticated false", false, false},
		// ProxyTCP = true, AllowUnauthenticated = False
		{"tcp true & unauthenticated false", true, false},
		// ProxyTCP = False, AllowUnauthenticated = True
		{"tcp false & unauthenticated true", false, true},
	}

	// First init the new Proxy
	InitProxy(proxyAuth)
	// And check that the Proxy struct is set up correctly
	require.NotNil(t, DefaultProxy.HTTPAuth)
	require.Equal(t, map[string]*Service{}, DefaultProxy.services)
	require.Equal(t, "", DefaultProxy.syslog.Message)

	// Then create the new proxy handler for the services
	handler := DefaultProxy.NewProxyHandler("service")
	require.NotNil(t, handler)
	if handler == nil {
		t.Errorf("handler not created for cluster")
	}

	// Then follow the lifecycle for each case
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// First register the services
			register(t, tt.proxyTCP, tt.allowUnauthenticated)
			require.Equal(t, len(serviceIDs), len(DefaultProxy.Summaries()))
			// Check that service fields are set correctly
			for id, service := range DefaultProxy.Summaries() {
				require.Equal(t, service.URL, &u)
				require.Equal(t, service.ProxyTCP, tt.proxyTCP)
				require.Equal(t, service.AllowUnauthenticated, tt.allowUnauthenticated)

				service, ok := DefaultProxy.Summary(id)
				require.True(t, ok)
				require.Equal(t, service.URL, &u)
				require.Equal(t, service.ProxyTCP, tt.proxyTCP)
				require.Equal(t, service.AllowUnauthenticated, tt.allowUnauthenticated)
			}
			// Then unregister
			unregister(t)
			require.Equal(t, map[string]Service{}, DefaultProxy.Summaries())
		})
	}

	// Now at the very end, to test clear proxy ...
	register(t, true, true)
	require.Equal(t, len(serviceIDs), len(DefaultProxy.Summaries()))
	// Clear the services by ClearProxy
	DefaultProxy.ClearProxy()
	if len(DefaultProxy.Summaries()) != 0 {
		t.Errorf("failed to clear all proxy services.")
	}
	require.Equal(t, 0, len(DefaultProxy.Summaries()))
}

func TestNewProxyHandler(t *testing.T) {
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, "file://../../static/migrations")
	require.NoError(t, etc.SetRootPath("../../static/srv"))
	// First init the new Proxy
	InitProxy(proxyAuth)
	// And check that the Proxy struct is set up correctly
	require.NotNil(t, DefaultProxy.HTTPAuth)
	require.Equal(t, map[string]*Service{}, DefaultProxy.services)
	require.Equal(t, "", DefaultProxy.syslog.Message)

	e := echo.New()
	// Create http test recorder
	req := httptest.NewRequest(http.MethodGet, u.Path+"/proxy", nil)
	// Create new echo context
	c := e.NewContext(req, httptest.NewRecorder())

	go func() {
		if err := e.Start(u.Path); err != nil && err != http.ErrServerClosed {
			t.Logf("failed to start server: %s", err)
		}
	}()

	ok := waitForCondition(5*time.Second, conditionServerUp)
	if !ok {
		t.FailNow()
	}
	// Case 1: handler returns OK because service name is registered/found
	register(t, true, true)
	c.SetPath("/:service")
	c.SetParamNames("service")
	c.SetParamValues("a")

	handler := DefaultProxy.NewProxyHandler("service")
	assert.NoError(t, handler(c))

	// Case 2: handler returns error because service name not found
	handler = DefaultProxy.NewProxyHandler("wrong")
	assert.Error(t, handler(c))
}
