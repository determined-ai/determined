package user

import (
	"github.com/labstack/echo/v4"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStandardAuth(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	c.SetPath("/agents")
	service := Service{}
	require.Equal(t, authStandard, service.getAuthLevel(c))

	c.SetPath("/api/v1/master/info")
	require.Equal(t, authStandard, service.getAuthLevel(c))
}

func TestAdminAuth(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	c.SetPath("/config")
	service := Service{}
	require.Equal(t, authAdmin, service.getAuthLevel(c))

	c.SetPath("/agents/id/slots")
	require.Equal(t, authStandard, service.getAuthLevel(c))
	c.SetPath("/agents/id/slots/1")
	require.Equal(t, authAdmin, service.getAuthLevel(c))

	c.SetPath("/api/v1/agents")
	require.Equal(t, authStandard, service.getAuthLevel(c))
	c.SetPath("/api/v1/agents/id")
	require.Equal(t, authAdmin, service.getAuthLevel(c))
	c.SetPath("/api/v1/agents/id/enable")
	require.Equal(t, authAdmin, service.getAuthLevel(c))
}

func TestNoAuth(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)
	c.SetPath("/")
	service := Service{}
	require.Equal(t, authNone, service.getAuthLevel(c))

	c.SetPath("/agents")
	require.Equal(t, authNone, service.getAuthLevel(c))
	c.SetPath("/agentss")
	require.Equal(t, authStandard, service.getAuthLevel(c))

	c.SetPath("/det/something")
	require.Equal(t, authAdmin, service.getAuthLevel(c))
	c.SetPath("/agents?id=1")
	require.Equal(t, authNone, service.getAuthLevel(c))
	c.SetPath("/proxy/:service/serviceHash")
	require.Equal(t, authAdmin, service.getAuthLevel(c))
}
