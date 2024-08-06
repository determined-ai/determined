package saml

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/samlsp"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"gopkg.in/guregu/null.v3"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/proxy"
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
	db           *db.PgDB
	samlProvider *samlsp.Middleware
	userConfig   userConfig
}

// userConfig represents the user defined configurations for SAML integration.
type userConfig struct {
	autoProvisionUsers          bool
	groupsAttributeName         string
	displayNameAttributeName    string
	agentUIDAttributeName       string
	agentGIDAttributeName       string
	agentUserNameAttributeName  string
	agentGroupNameAttributeName string
}

// New constructs a new SAML service that is capable of sending SAML requests and consuming
// responses.
func New(db *db.PgDB, c config.SAMLConfig) (*Service, error) {
	uc := userConfig{
		autoProvisionUsers:       c.AutoProvisionUsers,
		groupsAttributeName:      c.GroupsAttributeName,
		displayNameAttributeName: c.DisplayNameAttributeName,
	}

	key, cert, err := proxy.GenSignedCert()
	if err != nil {
		return nil, err
	}
	keyPair, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	keyPair.Leaf, err = x509.ParseCertificate(keyPair.Certificate[0])
	if err != nil {
		return nil, err
	}

	idpMetadataURL, err := url.Parse(c.IDPMetadataURL)
	if err != nil {
		return nil, err
	}
	idpMetadata, err := samlsp.FetchMetadata(context.Background(), http.DefaultClient,
		*idpMetadataURL)
	if err != nil {
		return nil, err
	}

	recipientURL, err := url.Parse(c.IDPRecipientURL)
	if err != nil {
		return nil, err
	}

	rootURL, err := url.Parse(recipientURL.Scheme + "://" + recipientURL.Host)
	if err != nil {
		return nil, err
	}

	middleWare, _ := samlsp.New(samlsp.Options{
		EntityID:    rootURL.String(),
		URL:         *rootURL,
		Key:         keyPair.PrivateKey.(*rsa.PrivateKey),
		Certificate: keyPair.Leaf,
		IDPMetadata: idpMetadata,
		SignRequest: true,
	})

	middleWare.ServiceProvider.AcsURL.Path = recipientURL.Path

	return &Service{
		db:           db,
		samlProvider: middleWare,
		userConfig:   uc,
	}, nil
}

// MakeRedirectBinding makes a SAML redirect binding as described at
// https://en.wikipedia.org/wiki/SAML_2.0#HTTP_Redirect_Binding.
func (s *Service) MakeRedirectBinding(relayState string) (string, error) {
	authenticationRequest, err := s.samlProvider.ServiceProvider.MakeRedirectAuthenticationRequest(relayState)
	if err != nil {
		return "", err
	}

	return authenticationRequest.String(), nil
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

	response := saml.Response{}
	bytesXML, err := base64.StdEncoding.DecodeString(encodedXML)
	if err != nil {
		return err
	}
	err = xml.Unmarshal(bytesXML, &response)
	if err != nil {
		return err
	}
	xmlResponse, err := s.samlProvider.ServiceProvider.ParseXMLResponse(bytesXML, []string{response.InResponseTo})
	if err != nil {
		return err
	}

	userAttr := s.toUserAttributes(xmlResponse)
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

	ug, err := user.GetAgentUserGroup(ctx, u.ID, 0)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "unable to look up user group")
	}
	u, err = s.syncUser(ctx, u, userAttr, ug)
	if err != nil {
		logrus.WithError(err).WithField("user", userAttr.userName).Error("error syncing user")
		return echo.NewHTTPError(http.StatusInternalServerError, "error syncing user")
	}

	logrus.Error(" ****** printing user posix info")
	logrus.Errorf("uid: %s, %d\n", s.userConfig.agentUIDAttributeName, userAttr.agentUID)
	logrus.Errorf("gid: %s, %d\n", s.userConfig.agentGIDAttributeName, userAttr.agentGID)
	logrus.Errorf("user: %s, %s\n", s.userConfig.agentUserNameAttributeName, userAttr.agentUserName)
	logrus.Errorf("group: %s, %s\n", s.userConfig.agentGroupNameAttributeName, userAttr.agentGroupName)

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
	userName       string
	displayName    string
	agentUID       int
	agentGID       int
	agentUserName  string
	agentGroupName string
	groups         []string
}

func (s *Service) toUserAttributes(response *saml.Assertion) *userAttributes {
	uName := getSAMLAttribute(response, "userName")
	if uName == "" {
		return nil
	}

	strAgentUID := getSAMLAttribute(response, s.userConfig.agentUIDAttributeName)
	tempAgentUID, err := strconv.Atoi(strAgentUID)
	if err != nil {
		logrus.WithError(err).WithField("agentUID", strAgentUID).Error("unable to convert to integer")
	}
	strAgentGID := getSAMLAttribute(response, s.userConfig.agentGIDAttributeName)
	tempAgentGID, err := strconv.Atoi(strAgentGID)
	if err != nil {
		logrus.WithError(err).WithField("agentGID", strAgentGID).Error("unable to convert to integer")
	}

	return &userAttributes{
		userName:       uName,
		displayName:    getSAMLAttribute(response, s.userConfig.displayNameAttributeName),
		agentUID:       tempAgentUID,
		agentGID:       tempAgentGID,
		agentUserName:  getSAMLAttribute(response, s.userConfig.agentUserNameAttributeName),
		agentGroupName: getSAMLAttribute(response, s.userConfig.agentGroupNameAttributeName),
		groups:         getAttributeValues(response, s.userConfig.groupsAttributeName),
	}
}

// getSAMLAttribute is similar to a function provided by the previously used saml library.
func getSAMLAttribute(r *saml.Assertion, name string) string {
	for _, statement := range r.AttributeStatements {
		for _, attr := range statement.Attributes {
			if attr.Name == name || attr.FriendlyName == name {
				return attr.Values[0].Value
			}
		}
	}
	return ""
}

// getAttributeValues is similar to a function provided by the previously used saml library.
func getAttributeValues(r *saml.Assertion, name string) []string {
	var values []string
	for _, statement := range r.AttributeStatements {
		for _, attr := range statement.Attributes {
			if attr.Name == name || attr.FriendlyName == name {
				for _, v := range attr.Values {
					values = append(values, v.Value)
				}
			}
		}
	}
	return values
}

func mergeUserGroups(sessionData *userAttributes, dbData *model.AgentUserGroup) *model.AgentUserGroup {
	result := model.AgentUserGroup{
		UID:   dbData.UID,
		GID:   dbData.GID,
		User:  dbData.User,
		Group: dbData.Group,
	}

	if sessionData.agentUID != 0 {
		result.UID = sessionData.agentUID
	}
	if sessionData.agentGID != 0 {
		result.GID = sessionData.agentGID
	}
	if sessionData.agentUserName != "" {
		result.User = sessionData.agentUserName
	}
	if sessionData.agentGroupName != "" {
		result.Group = sessionData.agentGroupName
	}

	return &result
}

// syncUser syncs the mutable user fields parsed from the claim, only if there are non-null changes.
func (s *Service) syncUser(ctx context.Context, u *model.User, uAttr *userAttributes,
	ug *model.AgentUserGroup,
) (*model.User, error) {
	ugUpdate := mergeUserGroups(uAttr, ug)
	if ugUpdate.UID == ug.UID && ugUpdate.GID == ug.GID && ugUpdate.User == ug.User && ugUpdate.Group == ug.Group {
		// nothing in user group uto update
		ugUpdate = nil
	}
	err := db.Bun().RunInTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable},
		func(ctx context.Context, tx bun.Tx) error {
			// If the config is set to auto-provision users, sync the display name.
			if s.userConfig.autoProvisionUsers {
				updateDisplayName := uAttr.displayName != "" && uAttr.displayName != u.DisplayName.String
				if updateDisplayName || ugUpdate != nil {
					err := user.Update(ctx,
						&model.User{
							ID:          u.ID,
							Username:    uAttr.userName,
							DisplayName: null.NewString(uAttr.displayName, true),
						}, []string{"display_name"}, ugUpdate)
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
