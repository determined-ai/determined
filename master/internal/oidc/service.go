package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
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
	cliRedirectPath = "http://localhost:49176"
	cliRelayState   = "cli=true"
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

	return &Service{
		config:   config,
		db:       db,
		provider: provider,
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
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

	oauth2token, err := s.oauth2Config.Exchange(c.Request().Context(), c.QueryParam("code"))
	if err != nil {
		return errors.Wrap(err, "failed to exchange oauth2 token")
	}

	userInfo, err := s.provider.UserInfo(c.Request().Context(), oauth2.StaticTokenSource(oauth2token))
	if err != nil {
		return errors.Wrap(err, "failed to get user info from oidc provider")
	}

	u, err := s.db.UserByUsername(userInfo.Email)
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
	switch relayState := c.FormValue("RelayState"); relayState {
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

	return c.Redirect(http.StatusFound, s.oauth2Config.AuthCodeURL(state))
}

// randString generates n randomized chars.
func randString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
