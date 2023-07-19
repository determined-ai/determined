package user

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/stretchr/testify/require"
)

func TestStandardAuth(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	c.SetPath("/agents/test")
	c.SetRequest(httptest.NewRequest(http.MethodPatch, "/standardTest", nil))
	service := Service{}
	require.Equal(t, authStandard, service.getAuthLevel(c))

	c.SetPath("/random/unlisted/endpoint")
	require.Equal(t, authStandard, service.getAuthLevel(c))
}

func TestNoAuth(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	c.SetPath("/")
	c.SetRequest(httptest.NewRequest(http.MethodPatch, "/standardTest", nil))
	service := Service{}
	require.Equal(t, authNone, service.getAuthLevel(c))

	c.SetPath("/agents")
	require.Equal(t, authNone, service.getAuthLevel(c))

	c.SetPath("/agentss")
	require.Equal(t, authStandard, service.getAuthLevel(c))
	c.SetRequest(httptest.NewRequest(http.MethodPatch, "/agents?id=1", nil))
	require.Equal(t, authNone, service.getAuthLevel(c))
}
