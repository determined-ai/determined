package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
)

const (
	cookieName          = "oauth2_state"
	cookieTTL           = 5 * 60
	defaultRedirectPath = "/det/login"
	// This must match the value at $PROJECT_ROOT/cli/determined_cli/sso.CLI_REDIRECT_PORT.
	cliRedirectPath         = "http://localhost:49176"
	deprecatedCliRelayState = "cli=true"
	cliRelayState           = "cli"
	envVarName              = "DETERMINED_OIDC_CLIENT_SECRET"
)

// Service handles OIDC interactions.
type Service struct {
	config       config.OIDCConfig
	db           *db.PgDB
	provider     *oidc.Provider
	oauth2Config oauth2.Config
}

// New initiates an OIDC Service.
func New(db *db.PgDB, config config.OIDCConfig) (*Service, error) {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, config.IDPSSOURL)
	if err != nil {
		return nil, err
	}

	ru, err := url.Parse(config.IDPRecipientURL)
	if err != nil {
		return nil, err
	}
	// join instead of replacing path in case we're behind a rewriting proxy
	ru.Path = path.Join(ru.Path, OidcRoot, CallbackPath)

	secret := config.ClientSecret
	if secret == "" {
		secret = os.Getenv(envVarName)
	}
	if secret == "" {
		return nil, fmt.Errorf("client secret has not been set")
	}

	return &Service{
		config:   config,
		db:       db,
		provider: provider,
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: secret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  ru.String(),
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
	}, nil
}

// callback validates the response from the OIDC provider, checking that the
// request matches the response, the oauth2 token is valid, and that the user
// is active.
func (s *Service) callback(c echo.Context) error {
	state, err := c.Cookie(cookieName)
	if err != nil {
		return errors.Wrap(err, "could not retrieve state cookie")
	}
	if c.QueryParam("state") != state.Value {
		return errors.New("oidc state did not match")
	}

	var oauth2token *oauth2.Token
	relayParam := c.QueryParam("relayState")
	// Tolerate older CLI versions (<=0.19.5)
	if relayParam == cliRelayState || relayParam == deprecatedCliRelayState {
		configCopy := s.oauth2Config
		configCopy.RedirectURL = fmt.Sprintf("%s?relayState=%s", configCopy.RedirectURL, relayParam)
		oauth2token, err = configCopy.Exchange(c.Request().Context(), c.QueryParam("code"))
	} else {
		oauth2token, err = s.oauth2Config.Exchange(c.Request().Context(), c.QueryParam("code"))
	}

	if err != nil {
		return errors.Wrap(err, "failed to exchange oauth2 token")
	}

	userInfo, err := s.provider.UserInfo(c.Request().Context(), oauth2.StaticTokenSource(oauth2token))
	if err != nil {
		return errors.Wrap(err, "failed to get user info from oidc provider")
	}

	var claims map[string]interface{}
	if err = userInfo.Claims(&claims); err != nil {
		return errors.Wrap(err, "failed to extract OIDC claims")
	}

	claimValueRaw, ok := claims[s.config.AuthenticationClaim]
	if !ok {
		return errors.New("user info did not contain expected claim value")
	}
	claimValue, ok := claimValueRaw.(string)
	if !ok {
		return errors.New("user info claim value was not a string")
	}

	log.
		WithField("auth-claim", s.config.AuthenticationClaim).
		WithField("auth-claim-value", claimValue).
		WithField("scim-attribute", s.config.SCIMAuthenticationAttribute).
		Debug("attempting to authenticate user via OIDC")

	u, err := s.db.UserBySCIMAttribute(s.config.SCIMAuthenticationAttribute, claimValue)
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
	switch relayState := c.QueryParam("relayState"); relayState {
	case cliRelayState:
		redirectPath = cliRedirectPath + fmt.Sprintf("?token=%s", url.QueryEscape(token))
	case "":
	default:
		redirectPath += fmt.Sprintf("?relayState=%s", url.QueryEscape(relayState))
	}

	return c.Redirect(http.StatusSeeOther, redirectPath)
}

// initiate saves a random string as a cookie and redirects the user to the
// configured OIDC provider. The OIDC provider return the random string in
// order to establish request/response correlation.
func (s *Service) initiate(c echo.Context) error {
	state, err := randString(16)
	if err != nil {
		return err
	}

	c.SetCookie(&http.Cookie{
		Name:     cookieName,
		Value:    state,
		MaxAge:   cookieTTL,
		Secure:   true,
		HttpOnly: true,
	})

	relayState := map[string]string{"relayState": c.QueryParam("relayState")}
	return c.Redirect(http.StatusFound, authCodeURLWithParams(s.oauth2Config, state, relayState))
}

// randString generates n randomized chars.
func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// authCodeURLWithParams attaches the specified key:value pairs as querystring
// parameters to the redirect URL used by AuthCodeURL.
func authCodeURLWithParams(conf oauth2.Config, state string, kv map[string]string) string {
	u, err := url.Parse(conf.RedirectURL)
	if err != nil {
		return conf.AuthCodeURL(state)
	}
	queries := u.Query()
	for k, v := range kv {
		if v != "" {
			queries.Add(k, v)
		}
	}

	u.RawQuery = queries.Encode()
	conf.RedirectURL = u.String()
	return conf.AuthCodeURL(state)
}
