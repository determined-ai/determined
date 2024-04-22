package oidc

import (
	"github.com/labstack/echo/v4"
)

// Default paths for handling OIDC requests.
const (
	OidcRoot     = "/oidc"
	CallbackPath = "/callback"
	InitiatePath = "/sso"
)

// RegisterAPIHandler enables OIDC-related endpoints.
func RegisterAPIHandler(e *echo.Echo, s *Service) {
	oidc := e.Group(OidcRoot)
	oidc.GET(CallbackPath, s.callback)
	oidc.GET(InitiatePath, s.initiate)
}
