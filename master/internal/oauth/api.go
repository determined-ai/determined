package oauth

import (
	"github.com/labstack/echo"
)

// Root is the root of all OAuth-related routes.
const Root = "/oauth2"

// RegisterAPIHandler registers endpoints used by OAuth.
func RegisterAPIHandler(e *echo.Echo, s *Service) {
	oauth := e.Group(Root)

	// OAuth flow.
	oauth.GET("/authorize", s.authorize)
	oauth.Any("/token", s.token)

	// Client management.
	oauth.POST("/clients", s.addClient, s.users.ProcessAdminAuthentication)
	oauth.GET("/clients", s.clients, s.users.ProcessAdminAuthentication)
	oauth.DELETE("/clients/:id", s.deleteClient, s.users.ProcessAdminAuthentication)
}
