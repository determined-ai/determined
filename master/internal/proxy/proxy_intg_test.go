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
)

var (
	proxyAuth = func(c echo.Context) (done bool, err error) {
		return true, nil
	}
	serviceIDs = []string{"a", "b", "c"}
	u          = url.URL{Path: "localhost:8081"}
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

func TestProxyLifecycle(t *testing.T) {
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

	// Ensure server is up, before testing it.
	i := 0
	tk := time.NewTicker(time.Second)
	defer tk.Stop()
	for range tk.C {
		resp, err := http.Get("http://" + u.Path + "/proxy")
		if err == nil {
			resp.Body.Close() //nolint:errcheck
			break
		}
		i++
		if i > 5 {
			t.FailNow()
		}
	}

	// Case 1: handler returns OK because service name is registered/found
	t.Run("a", func(t *testing.T) { register(t, true, true) })
	c.SetPath("/:service")
	c.SetParamNames("service")
	c.SetParamValues("a")

	handler := DefaultProxy.NewProxyHandler("service")
	assert.NoError(t, handler(c))

	// Case 2: handler returns error because service name not found
	handler = DefaultProxy.NewProxyHandler("wrong")
	assert.Error(t, handler(c))
}
