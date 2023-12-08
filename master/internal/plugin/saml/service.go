package saml

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/url"

	"github.com/RobotsAndPencils/go-saml"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	defaultRedirectPath = "/det/login"
	// This must match the value at $PROJECT_ROOT/cli/determined_cli/sso.CLI_REDIRECT_PORT.
	cliRedirectPath         = "http://localhost:49176"
	deprecatedCliRelayState = "cli=true"
	cliRelayState           = "cli"
)

// Service is a SAML service capable of sending SAML requests and consuming responses.
type Service struct {
	db         *db.PgDB
	samlConfig saml.ServiceProviderSettings
	userConfig userConfig
}

// userConfig represents the user defined configurations for SAML integration.
type userConfig struct {
	autoProvisionUsers       bool
	groupsAttributeName      string
	displayNameAttributeName string
}

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

	uc := userConfig{
		autoProvisionUsers:       c.AutoProvisionUsers,
		groupsAttributeName:      c.GroupsAttributeName,
		displayNameAttributeName: c.DisplayNameAttributeName,
	}

	return &Service{
		db:         db,
		samlConfig: sp,
		userConfig: uc,
	}, nil
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

	userAttr := s.toUserAttributes(response)
	if userAttr == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "SAML attribute identifier userName missing")
	}

	ctx := c.Request().Context()
	u, err := user.ByUsername(ctx, userAttr.userName)
	switch {
	case errors.Is(err, db.ErrNotFound) && s.userConfig.autoProvisionUsers:
		newUser, err := s.provisionUser(ctx, userAttr.userName, userAttr.groups)
		if err != nil {
			logrus.WithError(err).WithField("user", userAttr.userName).Error("error provisioning user")
			return echo.NewHTTPError(http.StatusInternalServerError, "error provisioning user")
		}
		u = newUser
	case errors.Is(err, db.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "user has not been provisioned")
	case err != nil:
		return echo.NewHTTPError(http.StatusInternalServerError, "unable to look up user")
	}

	u, err = s.syncUser(ctx, u, userAttr)
	if err != nil {
		logrus.WithError(err).WithField("user", userAttr.userName).Error("error syncing user")
		return echo.NewHTTPError(http.StatusInternalServerError, "error syncing user")
	}

	logrus.WithFields(logrus.Fields{
		"userName": userAttr.userName,
		"userId":   u.ID,
	}).Info("provisioned & synced user given claims")

	if !u.Active {
		return echo.NewHTTPError(http.StatusBadRequest, "user is inactive")
	}

	token, err := user.StartSession(ctx, u)
	if err != nil {
		return err
	}

	c.SetCookie(user.NewCookieFromToken(token))
	redirectPath := defaultRedirectPath
	switch relayState := c.FormValue("RelayState"); relayState {
	case deprecatedCliRelayState:
		fallthrough
	case cliRelayState:
		redirectPath = cliRedirectPath + fmt.Sprintf("?token=%s", url.QueryEscape(token))
	case "":
		// do nothing to the default redirectPath
	default:
		redirectPath += fmt.Sprintf("?relayState=%s", url.QueryEscape(relayState))
	}

	return c.Redirect(http.StatusSeeOther, redirectPath)
}

// userAttributes represents the set of user attributes from SAML authentication that we're concerned with.
type userAttributes struct {
	userName    string
	displayName string
	groups      []string
}

func (s *Service) toUserAttributes(response *saml.Response) *userAttributes {
	uName := response.GetAttribute("userName")
	if uName == "" {
		return nil
	}

	return &userAttributes{
		userName:    uName,
		displayName: response.GetAttribute(s.userConfig.displayNameAttributeName),
		groups:      response.GetAttributeValues(s.userConfig.groupsAttributeName),
	}
}

// syncUser syncs the mutable user fields parsed from the claim, only if there are non-null changes.
func (s *Service) syncUser(ctx context.Context, u *model.User, uAttr *userAttributes) (*model.User, error) {
	err := db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable},
		func(ctx context.Context, tx bun.Tx) error {
			// If the config is set to auto-provision users, sync the display name.
			if s.userConfig.autoProvisionUsers {
				if uAttr.displayName != "" && uAttr.displayName != u.DisplayName.String {
					err := user.Update(ctx,
						&model.User{
							ID:          u.ID,
							Username:    uAttr.userName,
							DisplayName: null.NewString(uAttr.displayName, true),
						}, []string{"display_name"}, nil)
					if err != nil {
						return fmt.Errorf("error setting display name of %q: %s", u.Username, err)
					}
				}
			}
			if s.userConfig.groupsAttributeName != "" {
				if err := usergroup.UpdateUserGroupMembershipTx(ctx, tx, u, uAttr.groups); err != nil {
					return fmt.Errorf("could not update user group membership: %s", err)
				}
			}
			return nil
		})
	if err != nil {
		return nil, err
	}
	return user.ByUsername(ctx, u.Username)
}

// provisionUser: If we get forwarded an identity for an unknown user from the IdP,
// create a remote user with no password in the user table.
func (s *Service) provisionUser(
	ctx context.Context,
	username string,
	groups []string,
) (*model.User, error) {
	u := model.User{
		Username:     username,
		PasswordHash: model.NoPasswordLogin,
		Active:       true,
		Remote:       true,
	}

	if err := db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable},
		func(ctx context.Context, tx bun.Tx) error {
			if _, err := user.AddUserTx(ctx, tx, &u); err != nil {
				return err
			}
			if s.userConfig.groupsAttributeName != "" {
				if err := usergroup.UpdateUserGroupMembershipTx(ctx, tx, &u, groups); err != nil {
					return fmt.Errorf("could not update user group membership: %w", err)
				}
			}
			return nil
		}); err != nil {
		return nil, err
	}
	return user.ByUsername(ctx, username)
}
