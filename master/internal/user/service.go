package user

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo"
	"github.com/o1egl/paseto"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// sessionDuration is how long a newly created session is valid.
const sessionDuration = 7 * 24 * time.Hour

type agentUserGroup struct {
	UID   *int   `json:"uid,omitempty"`
	GID   *int   `json:"gid,omitempty"`
	User  string `json:"user"`
	Group string `json:"group"`
}

func (h *agentUserGroup) Validate() (*model.AgentUserGroup, error) {
	switch {
	case h.UID == nil:
		return nil, errors.New("uid must be set")
	case h.GID == nil:
		return nil, errors.New("gid must be set")
	case len(h.User) == 0:
		return nil, errors.New("user must be set")
	case len(h.Group) == 0:
		return nil, errors.New("group must be set")
	}

	return &model.AgentUserGroup{
		UID:   *h.UID,
		GID:   *h.GID,
		User:  h.User,
		Group: h.Group,
	}, nil
}

// Service describes a user manager.
type Service struct {
	tokenKeys *model.AuthTokenKeypair
	db        *db.PgDB
	system    *actor.System
}

// New creates a new user service.
func New(db *db.PgDB, system *actor.System) (*Service, error) {
	tokenKeys, err := getOrCreateKeys(db)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get auth token keypair")
	}

	return &Service{tokenKeys, db, system}, nil
}

func (s *Service) getUserFromSession(session model.UserSession) (*model.User, error) {
	user, err := s.db.UserBySessionID(session.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get user from session: %d", session.ID)
	}
	return user, nil
}

// ProcessAuthentication is a middleware processing function that attempts
// to authenticate incoming HTTP requests.  Note that the middleware looks
// for an authentication in three places (in the following order):
// 1. The HTTP Authorization header.
// 2. A cookie named "auth".
// 3. A Query parameter named "_auth".
func (s *Service) ProcessAuthentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authRaw := c.Request().Header.Get("Authorization")
		var token string
		if authRaw != "" {
			// We attempt to parse out the token, which should be
			// transmitted as a Bearer authentication token.
			if !strings.HasPrefix(authRaw, "Bearer ") {
				return echo.ErrUnauthorized
			}
			token = strings.TrimPrefix(authRaw, "Bearer ")
		} else if cookie, err := c.Cookie("auth"); err == nil {
			token = cookie.Value
		} else {
			// If we found no token, then abort the request with an HTTP 401.
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		userSession, err := s.ValidateToken(token)
		if err != nil {
			switch e := err.(type) {
			case db.ErrUserSessionNotFound:
				return echo.NewHTTPError(http.StatusForbidden)
			case model.ErrUserSessionExpired:
				return echo.NewHTTPError(http.StatusForbidden)
			default:
				return e
			}
		}

		// We have a valid session, so we should be able to use it
		// to pull a user from the database.
		user, err := s.getUserFromSession(*userSession)
		if err != nil {
			return err
		}

		if !user.Active {
			return echo.NewHTTPError(http.StatusUnauthorized)
		}

		// Set data on the request context that might be useful to
		// event handlers.
		c.(*context.DetContext).SetUser(*user)
		c.(*context.DetContext).SetUserSession(*userSession)
		return next(c)
	}
}

func (s *Service) generateToken(user model.User) (string, error) {
	userSession := model.UserSession{
		UserID: user.ID,
		Expiry: time.Now().Add(sessionDuration),
	}

	if err := s.db.AddUserSession(&userSession); err != nil {
		return "", err
	}

	v2 := paseto.NewV2()
	token, err := v2.Sign(s.tokenKeys.PrivateKey, userSession, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate authentication token")
	}

	return token, nil
}

// ValidateToken returns a user session given an authentication token.
func (s *Service) ValidateToken(token string) (*model.UserSession, error) {
	v2 := paseto.NewV2()

	var userSession model.UserSession

	err := v2.Verify(token, s.tokenKeys.PublicKey, &userSession, nil)

	if err != nil {
		return nil, err
	}

	// Ensure the token that we fetched is still valid.  This means:
	// 1. It is still in the database
	// 2. It is not expired
	session, err := s.db.SessionBySessionID(userSession.ID)
	if err != nil {
		return nil, err
	}

	if session.Expiry.Before(time.Now()) {
		return nil, model.ErrUserSessionExpired{SessionID: userSession.ID}
	}

	return session, nil
}

func (s *Service) postLogout(c echo.Context) (interface{}, error) {
	// Delete the cookie if one is set.
	if cookie, err := c.Cookie("auth"); err == nil {
		cookie.Value = ""
		cookie.Expires = time.Unix(0, 0)
		c.SetCookie(cookie)
	}

	// Delete the user session information from the database.
	sess := c.(*context.DetContext).MustGetUserSession()

	if err := s.db.DeleteSessionByID(sess.ID); err != nil {
		return nil, err
	}

	return "", nil
}

func (s *Service) postLogin(c echo.Context) (interface{}, error) {
	type (
		request struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		response struct {
			Token string `json:"token"`
		}
	)

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	malformedRequestError := echo.NewHTTPError(http.StatusBadRequest)
	badCredentialsError := echo.NewHTTPError(http.StatusForbidden, "invalid credentials")

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		return nil, malformedRequestError
	}

	params.Username = strings.ToLower(params.Username)

	// Get the user from the database.
	user, err := s.db.UserByUsername(params.Username)
	if err != nil {
		return nil, badCredentialsError
	}

	// The user must be active.
	if !user.Active {
		return nil, badCredentialsError
	}

	var token string
	if !user.ValidatePassword(params.Password) {
		return nil, badCredentialsError
	}

	token, err = s.generateToken(*user)
	if err != nil {
		return nil, err
	}

	// The caller of this REST endpoint can request that the master set a cookie.
	// This is used by the WebUI for persistence of sessions.
	if c.QueryParam("cookie") == "true" {
		cookie := new(http.Cookie)
		cookie.Name = "auth"
		cookie.Value = token
		cookie.Expires = time.Now().Add(sessionDuration)
		c.SetCookie(cookie)
	}

	return response{
		Token: token,
	}, nil
}

// getMe returns information about the current authenticated user.
func (s *Service) getMe(c echo.Context) (interface{}, error) {
	me := c.(*context.DetContext).MustGetUser()

	return s.db.UserByID(me.ID)
}

func (s *Service) getUsers(c echo.Context) (interface{}, error) {
	return s.db.UserList()
}

func (s *Service) patchUser(c echo.Context) (interface{}, error) {
	type (
		request struct {
			Password *string `json:"password,omitempty"`
			Active   *bool   `json:"active,omitempty"`
			Admin    *bool   `json:"admin,omitempty"`

			AgentUserGroup *agentUserGroup `json:"agent_user_group,omitempty"`
		}
		response struct {
			message string
		}
	)

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	args := struct {
		Username string `path:"username"`
	}{}
	if err = api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	args.Username = strings.ToLower(args.Username)

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "bad request")
		return nil, malformedRequestError
	}

	forbiddenError := echo.NewHTTPError(http.StatusForbidden)
	authenticatedUser := c.(*context.DetContext).MustGetUser()
	user, err := s.db.UserByUsername(args.Username)

	if err != nil {
		switch err.(type) {
		case db.ErrNoSuchUsername:
			if authenticatedUser.Admin {
				return nil, echo.NewHTTPError(
					http.StatusBadRequest,
					fmt.Sprintf("failed to get user '%s'", args.Username))
			}
			return nil, forbiddenError
		default:
			return nil, err
		}
	}

	var toUpdate []string

	if params.Password != nil {
		if !user.PasswordCanBeModifiedBy(authenticatedUser) {
			return nil, forbiddenError
		}
		if err = user.UpdatePasswordHash(*params.Password); err != nil {
			return nil, err
		}
		toUpdate = append(toUpdate, "password_hash")
	}

	if params.Active != nil {
		if !user.ActiveCanBeModifiedBy(authenticatedUser) {
			return nil, forbiddenError
		}
		user.Active = *params.Active
		toUpdate = append(toUpdate, "active")
	}

	if params.Admin != nil {
		if !user.AdminCanBeModifiedBy(authenticatedUser) {
			return nil, forbiddenError
		}
		user.Admin = *params.Admin
		toUpdate = append(toUpdate, "admin")
	}

	var ug *model.AgentUserGroup
	if pug := params.AgentUserGroup; pug != nil {
		if !user.AdminCanBeModifiedBy(authenticatedUser) {
			return nil, forbiddenError
		}

		u, pErr := pug.Validate()
		if pErr != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, pErr.Error())
		}
		ug = u
	}

	if err = s.db.UpdateUser(user, toUpdate, ug); err != nil {
		return nil, err
	}

	return response{
		message: fmt.Sprintf("successfully updated %v", args.Username),
	}, nil
}

func (s *Service) postUser(c echo.Context) (interface{}, error) {
	type (
		request struct {
			Username string `json:"username"`
			Admin    bool   `json:"admin"`
			Active   bool   `json:"active"`

			AgentUserGroup *agentUserGroup `json:"agent_user_group,omitempty"`
		}
		response struct {
			message string
		}
	)

	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "bad request")
		return nil, malformedRequestError
	}

	currUser := c.(*context.DetContext).MustGetUser()
	if !currUser.CanCreateUser() {
		insufficientPermissionsError := echo.NewHTTPError(
			http.StatusForbidden,
			"insufficient permissions")
		return nil, insufficientPermissionsError
	}

	var ug *model.AgentUserGroup
	if pug := params.AgentUserGroup; pug != nil {
		u, pErr := pug.Validate()
		if pErr != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, pErr.Error())
		}
		ug = u
	}

	params.Username = strings.ToLower(params.Username)
	err = s.db.AddUser(&model.User{
		Username: params.Username,
		Admin:    params.Admin,
		Active:   params.Active,
	}, ug)

	if err != nil {
		switch err.(type) {
		case db.ErrDuplicateUser:
			return nil, echo.NewHTTPError(http.StatusBadRequest, "user already exists")
		default:
			return nil, err
		}
	}

	telemetry.ReportUserCreated(s.system, params.Admin, params.Active)

	return response{
		message: fmt.Sprintf("successfully created user: %s", params.Username),
	}, nil
}
