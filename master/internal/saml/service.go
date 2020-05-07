package saml

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/RobotsAndPencils/go-saml"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
)

const defaultRedirectPath = "/det/login"

// New constructs a new SAML service that is capable of sending SAML requests and consuming
// responses.
func New(db *db.PgDB, c config.SAMLConfig) (*Service, error) {
	sp := saml.ServiceProviderSettings{
		IDPSSOURL:                   c.IDPSSOURL,
		IDPSSODescriptorURL:         c.IDPSSODescriptorURL,
		IDPPublicCertPath:           c.IDPCertPath,
		AssertionConsumerServiceURL: c.IDPRecipientURL,
	}
	err := sp.Init()
	if err != nil {
		return nil, errors.Wrap(err, "error creating SAML service")
	}

	return &Service{
		db:         db,
		samlConfig: sp,
	}, nil
}

// Service is a SAML service capable of sending SAML requests and consuming responses.
type Service struct {
	db         *db.PgDB
	samlConfig saml.ServiceProviderSettings
}

// MakeRedirectBinding makes a SAML redirect binding as described at
// https://en.wikipedia.org/wiki/SAML_2.0#HTTP_Redirect_Binding.
func (s *Service) MakeRedirectBinding(relayState string) (string, error) {
	authnRequest := s.samlConfig.GetAuthnRequest()
	b64XML, err := authnRequest.EncodedString()
	if err != nil {
		return "", errors.Wrap(err, "error encoding redirect binding")
	}

	url, err := saml.GetAuthnRequestURL(s.samlConfig.IDPSSOURL, b64XML, relayState)
	if err != nil {
		return "", errors.Wrap(err, "error generating redirect request")
	}

	return url, nil
}

func (s *Service) redirectWithSAMLRequest(c echo.Context) error {
	url, err := s.MakeRedirectBinding(c.QueryParam("relayState"))
	if err != nil {
		return errors.Wrap(err, "error creating redirect binding")
	}
	return c.Redirect(http.StatusSeeOther, url)
}

func (s *Service) consumeAssertion(c echo.Context) error {
	encodedXML := c.FormValue("SAMLResponse")
	if encodedXML == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "SAMLResponse form value missing")
	}

	response, err := saml.ParseEncodedResponse(encodedXML)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error parsing SAMLResponse")
	}

	err = response.Validate(&s.samlConfig)
	if err != nil {
		logrus.Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, "error validating SAMLResponse")
	}

	uid := response.GetAttribute("userName")
	if uid == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "SAML attribute identifier userName missing")
	}

	u, err := user.UserByUsername(uid)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "user has not been provisioned")
	}

	if !u.Active {
		return echo.NewHTTPError(http.StatusBadRequest, "user is inactive")
	}

	token, err := s.db.StartUserSession(u)
	if err != nil {
		return err
	}

	c.SetCookie(user.NewCookieFromToken(token))

	redirectPath := defaultRedirectPath
	relayState := c.FormValue("RelayState")
	if relayState != "" {
		redirectPath += fmt.Sprintf("?relayState=%s", url.QueryEscape(relayState))
	}

	return c.Redirect(http.StatusSeeOther, redirectPath)
}
