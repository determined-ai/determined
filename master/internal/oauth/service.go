package oauth

import (
	"context"
	"strconv"

	"github.com/pkg/errors"

	"net/http"
	"net/url"

	"github.com/labstack/echo"
	log "github.com/sirupsen/logrus"
	oauth2Errors "gopkg.in/oauth2.v3/errors"
	"gopkg.in/oauth2.v3/manage"
	"gopkg.in/oauth2.v3/server"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/user"
)

const loginPath = "/det/login"

// Service is an OAuth service capable of handling the OAuth 2.0 authorization code flow and
// managing clients and tokens.
type Service struct {
	users       *user.Service
	server      *server.Server
	tokenStore  *db.TokenStore
	clientStore *db.ClientStore
}

// New constructs a new OAuth service.
func New(users *user.Service, db *db.PgDB) (*Service, error) {
	manager := manage.NewDefaultManager()
	s := &Service{
		users:       users,
		server:      server.NewDefaultServer(manager),
		tokenStore:  db.TokenStore(),
		clientStore: db.ClientStore(),
	}

	manager.MapTokenStorage(s.tokenStore)
	manager.MapClientStorage(s.clientStore)

	s.server.SetAllowGetAccessRequest(true)
	s.server.SetUserAuthorizationHandler(s.userAuthorizationHandler)
	s.server.SetClientInfoHandler(clientFormHandler)

	s.server.SetInternalErrorHandler(func(err error) (re *oauth2Errors.Response) {
		log.WithError(err).Error("OAuth internal error occurred")
		return nil
	})
	s.server.SetResponseErrorHandler(func(re *oauth2Errors.Response) {
		log.WithError(re.Error).WithField("response", re).Errorf("OAuth response error occurred")
	})

	return s, nil
}

type contextKey struct{}

// userAuthorizationHandler is the callback used by the OAuth library to allow us to determine
// whether a user is logged in and chooses to authorize the application.
func (s *Service) userAuthorizationHandler(w http.ResponseWriter, r *http.Request) (string, error) {
	// Ignore the error, since we just want to know whether we can get a session at all.
	user, session, _ := s.users.UserAndSessionFromRequest(r)
	if session == nil {
		c := r.Context().Value(contextKey{}).(echo.Context)
		return "", c.Redirect(http.StatusFound, loginPath+"?redirect="+url.QueryEscape(r.URL.String()))
	}

	log.WithFields(log.Fields{
		"username":    user.Username,
		"request_url": r.URL,
	}).Infof("user authorizing an OAuth application")

	if !user.Admin {
		return "", errors.Errorf("non-admin user %s cannot authorize OAuth applications", user.Username)
	}

	// For now, automatically authorize the application for simplicity.
	return strconv.Itoa(int(session.UserID)), nil
}

// authorize handles requests for new client authorizations.
func (s *Service) authorize(c echo.Context) error {
	// Smuggle the Echo context into the request so the real handler (userAuthorizationHandler) can use
	// it. (Note that c.Request().Context() is distinct from c.)
	ctx := context.WithValue(c.Request().Context(), contextKey{}, c)
	c.SetRequest(c.Request().WithContext(ctx))

	return s.server.HandleAuthorizeRequest(c.Response().Writer, c.Request())
}

// clientFormHandler verifies a token request by hashing the client ID and secret instead of using
// the secret from the form directly.
func clientFormHandler(r *http.Request) (string, string, error) {
	clientID := r.Form.Get("client_id")
	clientSecret := r.Form.Get("client_secret")
	if clientID == "" || clientSecret == "" {
		return "", "", oauth2Errors.ErrInvalidClient
	}

	log.WithField("client_id", clientID).Infof("OAuth token requested")

	hash, err := hashSecret(clientID, clientSecret)
	if err != nil {
		return "", "", err
	}

	return clientID, hash, nil
}

func (s *Service) token(c echo.Context) error {
	return s.server.HandleTokenRequest(c.Response().Writer, c.Request())
}
