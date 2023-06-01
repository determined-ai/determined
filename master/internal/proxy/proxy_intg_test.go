package proxy

import (
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

var TestProxy *Proxy

var (
	proxyAuth = func(c echo.Context) (done bool, err error) {
		return true, nil
	}
	serviceIDs = []string{"a", "b", "c"}
	u          = url.URL{} // "localhost:8081" TODO CAROLINA
)

func register(t *testing.T, prTCP bool, unauth bool) {
	for _, id := range serviceIDs {
		TestProxy.Register(id, &u, prTCP, unauth)
		if TestProxy.GetService(id) == nil {
			t.Logf("failed to find registered service %s", id)
		}
	}
	if len(TestProxy.Summary()) != len(serviceIDs) {
		t.Logf("failed to register all services")
	}
}

func unregister(t *testing.T) {
	for _, id := range serviceIDs {
		if TestProxy.GetService(id) == nil {
			t.Logf("failed to find registered service %s", id)
		}
		TestProxy.Unregister(id)
		if TestProxy.GetService(id) != nil {
			t.Logf("failed to unregister service %s", id)
		}
	}
	if len(TestProxy.Summary()) != 0 {
		t.Logf("failed to unregister all services.")
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
	TestProxy.InitProxy(proxyAuth)
	// And check that the Proxy struct is set up correctly
	require.NotNil(t, TestProxy.HTTPAuth)
	require.Equal(t, map[string]*Service{}, TestProxy.services)
	require.Equal(t, "", TestProxy.syslog.Message)

	// Then create the new proxy handler for the services
	handler := TestProxy.NewProxyHandler("service")
	require.NotNil(t, handler)
	if handler == nil {
		t.Logf("handler not created for cluster")
	}

	// Then follow the lifecycle for each case
	for _, testCase := range cases {
		// First register the services
		register(t, testCase.proxyTCP, testCase.allowUnauthenticated)
		require.Equal(t, len(serviceIDs), len(TestProxy.Summary()))
		// Check that service fields are set correctly
		for _, service := range TestProxy.Summary() {
			require.Equal(t, service.URL, &u)
			require.Equal(t, service.ProxyTCP, testCase.proxyTCP)
			require.Equal(t, service.AllowUnauthenticated, testCase.allowUnauthenticated)
		}
		// Then unregister
		unregister(t)
		require.Equal(t, map[string]Service{}, TestProxy.Summary())
	}

	// Now at the very end, to test clear proxy ...
	register(t, true, true)
	require.Equal(t, len(serviceIDs), len(TestProxy.Summary()))
	// Clear the services by ClearProxy
	TestProxy.ClearProxy()
	if len(TestProxy.Summary()) != 0 {
		t.Logf("failed to clear all proxy services.")
	}
	require.Equal(t, 0, len(TestProxy.Summary()))
}
