package saml

import (
	"github.com/labstack/echo/v4"
)

// Default paths for handling SAML requests.
const (
	SAMLRoot     = "/saml"
	SSOPath      = "/sso"
	InitiatePath = "/initiate"
)

// RegisterAPIHandler registers endpoints used by SAML.
func RegisterAPIHandler(e *echo.Echo, s *Service) {
	saml := e.Group(SAMLRoot)
	saml.POST(SSOPath, s.consumeAssertion)
	saml.GET(InitiatePath, s.redirectWithSAMLRequest)
}
