package proxy

import (
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

var (
	proxyAuth = func(c echo.Context) (done bool, err error) {
		return true, nil
	}
	serviceIDs = []string{"a", "b", "c"}
	u          = url.URL{}
)

func register(t *testing.T, prTCP bool, unauth bool) {
	for _, id := range serviceIDs {
		DefaultProxy.Register(id, &u, prTCP, unauth)
		if DefaultProxy.GetService(id) == nil {
			t.Errorf("failed to find registered service %s", id)
		}
	}
	if len(DefaultProxy.Summary()) != len(serviceIDs) {
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
			require.Equal(t, len(serviceIDs), len(DefaultProxy.Summary()))
			// Check that service fields are set correctly
			for _, service := range DefaultProxy.Summary() {
				require.Equal(t, service.URL, &u)
				require.Equal(t, service.ProxyTCP, tt.proxyTCP)
				require.Equal(t, service.AllowUnauthenticated, tt.allowUnauthenticated)
			}
			// Then unregister
			unregister(t)
			require.Equal(t, map[string]Service{}, DefaultProxy.Summary())
		})
	}

	// Now at the very end, to test clear proxy ...
	register(t, true, true)
	require.Equal(t, len(serviceIDs), len(DefaultProxy.Summary()))
	// Clear the services by ClearProxy
	DefaultProxy.ClearProxy()
	if len(DefaultProxy.Summary()) != 0 {
		t.Errorf("failed to clear all proxy services.")
	}
	require.Equal(t, 0, len(DefaultProxy.Summary()))
}
