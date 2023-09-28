package user

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	once        sync.Once
	userService *Service
)

var externalSessionsError = echo.NewHTTPError(
	http.StatusForbidden,
	"not enabled with external sessions")

var forbiddenError = echo.NewHTTPError(
	http.StatusForbidden,
	"user not authorized")

const (
	// authNone indicates a request needs no authentication.
	authNone int = 0
	// authStandard indicates a request needs authentication.
	authStandard = 1
	// authAdmin indicates a request needs admin authentication.
	authAdmin = 2
)

// unauthenticatedPointsList contains URIs and paths that are exempted from authentication.
var unauthenticatedPointsList = []string{
	"/",
	"/docs/.*",
	"/info",
	"/task-logs",
	"/agents",
	"/det",
	"/det/.*",
	"/login",
	"/api/v1/.*",
	"/proxy/:service/.*",
	"/agents\\?id=.*",
}

// adminAuthPointsList contains the paths that require admin authentication.
var adminAuthPointsList = []string{
	"/agents/.*/slots/.*",
}

var unauthenticatedPointsPattern = regexp.MustCompile("^" +
	strings.Join(unauthenticatedPointsList, "$|^") + "$")

var adminAuthPointsPattern = regexp.MustCompile("^" +
	strings.Join(adminAuthPointsList, "$|^") + "$")

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
	db        *db.PgDB
	system    *actor.System
	extConfig *model.ExternalSessions
}

// InitService creates the user service singleton.
func InitService(db *db.PgDB, system *actor.System, extConfig *model.ExternalSessions) {
	once.Do(func() {
		userService = &Service{db, system, extConfig}
	})
}

// GetService returns a reference to the user service singleton.
func GetService() *Service {
	if userService == nil {
		panic("Singleton UserService is not yet initialized.")
	}
	return userService
}

// The middleware looks for a token in two places (in this order):
// 1. The HTTP Authorization header.
// 2. A cookie named "auth".
func (s *Service) extractToken(r *http.Request) (string, error) {
	authRaw := r.Header.Get("Authorization")
	if authRaw != "" {
		// We attempt to parse out the token, which should be
		// transmitted as a Bearer authentication token.
		if !strings.HasPrefix(authRaw, "Bearer ") {
			return "", echo.ErrUnauthorized
		}
		return strings.TrimPrefix(authRaw, "Bearer "), nil
	} else if cookie, err := r.Cookie("det_jwt"); err == nil {
		return cookie.Value, nil
	} else if cookie, err := r.Cookie("auth"); err == nil {
		return cookie.Value, nil
	}
	// If we found no token, then abort the request with an HTTP 401.
	return "", echo.NewHTTPError(http.StatusUnauthorized)
}

// UserAndSessionFromRequest gets the user and session corresponding to the given request.
func (s *Service) UserAndSessionFromRequest(
	r *http.Request,
) (*model.User, *model.UserSession, error) {
	token, err := s.extractToken(r)
	if err != nil {
		return nil, nil, err
	}
	return ByToken(context.TODO(), token, s.extConfig)
}

// getAuthLevel returns what level of authentication a request needs.
func (s *Service) getAuthLevel(c echo.Context) int {
	switch {
	case adminAuthPointsPattern.MatchString(c.Request().RequestURI):
		return authAdmin
	case unauthenticatedPointsPattern.MatchString(c.Path()):
		return authNone
	case unauthenticatedPointsPattern.MatchString(c.Request().RequestURI):
		return authNone
	default:
		return authStandard
	}
}

// ProcessAuthentication is a middleware processing function that attempts
// to authenticate incoming HTTP requests.
func (s *Service) ProcessAuthentication(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var adminOnly bool
		switch s.getAuthLevel(c) {
		case authNone:
			return next(c)
		case authStandard:
			adminOnly = false
		case authAdmin:
			adminOnly = true
		}

		user, session, err := s.UserAndSessionFromRequest(c.Request())
		switch err {
		case nil:
			if !user.Active {
				return echo.NewHTTPError(http.StatusForbidden, "user not active")
			}
			if adminOnly && !user.Admin && !config.GetAuthZConfig().IsRBACEnabled() {
				return echo.NewHTTPError(http.StatusForbidden, "user not admin")
			}

			// Set data on the request context that might be useful to
			// event handlers.
			c.(*detContext.DetContext).SetUser(*user)
			c.(*detContext.DetContext).SetUserSession(*session)
			return next(c)
		case db.ErrNotFound:
			return echo.NewHTTPError(http.StatusUnauthorized)
		default:
			return err
		}
	}
}

func (s *Service) postLogout(c echo.Context) (interface{}, error) {
	// Delete the cookie if one is set.
	if cookie, err := c.Cookie("auth"); err == nil {
		cookie.Value = ""
		cookie.Expires = time.Unix(0, 0)
		c.SetCookie(cookie)
	}

	// Delete the user session information from the database.
	sess := c.(*detContext.DetContext).MustGetUserSession()

	if err := DeleteSessionByID(context.TODO(), sess.ID); err != nil {
		return nil, err
	}

	return "", nil
}

func (s *Service) postLogin(c echo.Context) (interface{}, error) {
	if s.extConfig.JwtKey != "" {
		return nil, echo.NewHTTPError(http.StatusMisdirectedRequest,
			"authentication is configured to be external")
	}

	type (
		request struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		response struct {
			Token string `json:"token"`
		}
	)

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest)
	}

	// Get the user from the database.
	user, err := ByUsername(context.TODO(), params.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		return nil, echo.NewHTTPError(http.StatusForbidden, "user not found")
	default:
		return nil, err
	}

	// The user must be active.
	if !user.Active {
		return nil, echo.NewHTTPError(http.StatusForbidden, "user not active")
	}

	var token string
	if !user.ValidatePassword(params.Password) {
		return nil, echo.NewHTTPError(http.StatusForbidden, "invalid credentials")
	}

	token, err = StartSession(context.TODO(), user)
	if err != nil {
		return nil, err
	}

	// The caller of this REST endpoint can request that the master set a cookie.
	// This is used by the WebUI for persistence of sessions.
	if c.QueryParam("cookie") == "true" {
		c.SetCookie(NewCookieFromToken(token))
	}

	return response{
		Token: token,
	}, nil
}

// NewCookieFromToken creates a new cookie from the given token.
func NewCookieFromToken(token string) *http.Cookie {
	cookie := new(http.Cookie)
	cookie.Name = "auth"
	cookie.Value = token
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(SessionDuration)
	return cookie
}

// getMe returns information about the current authenticated user.
func (s *Service) getMe(c echo.Context) (interface{}, error) {
	me := c.(*detContext.DetContext).MustGetUser()
	return ByID(context.TODO(), me.ID)
}

func (s *Service) getUsers(c echo.Context) (interface{}, error) {
	userList, err := List(context.TODO())
	if err != nil {
		return nil, err
	}

	var ctx context.Context
	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}

	return AuthZProvider.Get().FilterUserList(ctx,
		c.(*detContext.DetContext).MustGetUser(), userList)
}

func canViewUserErrorHandle(currUser, user model.User, actionErr, notFoundErr error) error {
	ctx := context.TODO()
	if err := AuthZProvider.Get().CanGetUser(ctx, currUser, user); err != nil {
		return authz.SubIfUnauthorized(err, notFoundErr)
	}
	return actionErr
}

func (s *Service) patchUser(c echo.Context) (interface{}, error) {
	if s.extConfig.Enabled() {
		return nil, externalSessionsError
	}
	var ctx context.Context
	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}

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

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	args := struct {
		Username string `path:"username"`
	}{}
	if err = api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "bad request")
		return nil, malformedRequestError
	}

	userNotFoundErr := api.NotFoundErrs("user", args.Username, false)

	currUser := c.(*detContext.DetContext).MustGetUser()
	user, err := ByUsername(ctx, args.Username)
	switch err {
	case nil:
	case db.ErrNotFound:
		return nil, userNotFoundErr
	default:
		return nil, err
	}

	var toUpdate []string
	if params.Password != nil {
		if err = AuthZProvider.Get().CanSetUsersPassword(ctx, currUser, *user); err != nil {
			return nil, canViewUserErrorHandle(currUser, *user,
				errors.Wrap(forbiddenError, err.Error()), userNotFoundErr)
		}

		if err = user.UpdatePasswordHash(*params.Password); err != nil {
			return nil, err
		}
		toUpdate = append(toUpdate, "password_hash")
	}

	if params.Active != nil {
		if err = AuthZProvider.Get().CanSetUsersActive(ctx, currUser, *user, *params.Active); err != nil {
			return nil, canViewUserErrorHandle(currUser, *user,
				errors.Wrap(forbiddenError, err.Error()), userNotFoundErr)
		}

		user.Active = *params.Active
		toUpdate = append(toUpdate, "active")
	}

	if params.Admin != nil {
		if err = AuthZProvider.Get().CanSetUsersAdmin(ctx, currUser, *user, *params.Admin); err != nil {
			return nil, canViewUserErrorHandle(currUser, *user,
				errors.Wrap(forbiddenError, err.Error()), userNotFoundErr)
		}

		user.Admin = *params.Admin
		toUpdate = append(toUpdate, "admin")
	}

	var ug *model.AgentUserGroup
	if pug := params.AgentUserGroup; pug != nil {
		u, pErr := pug.Validate()
		if pErr != nil {
			return nil, echo.NewHTTPError(http.StatusBadRequest, pErr.Error())
		}
		ug = u

		if err := AuthZProvider.Get().CanSetUsersAgentUserGroup(ctx, currUser, *user, *ug); err != nil {
			return nil, canViewUserErrorHandle(currUser, *user,
				errors.Wrap(forbiddenError, err.Error()), userNotFoundErr)
		}
	}

	if err := Update(ctx, user, toUpdate, ug); err != nil {
		return nil, err
	}

	return response{
		message: fmt.Sprintf("successfully updated %v", args.Username),
	}, nil
}

func (s *Service) patchUsername(c echo.Context) (interface{}, error) {
	if s.extConfig.Enabled() {
		return nil, externalSessionsError
	}
	type (
		request struct {
			NewUsername *string `json:"username,omitempty"`
		}
		response struct {
			message string
		}
	)

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	args := struct {
		Username string `path:"username"`
	}{}
	if err = api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "bad request")
		return nil, malformedRequestError
	}

	user, err := ByUsername(context.TODO(), args.Username)
	if err != nil {
		return nil, err
	}

	currUser := c.(*detContext.DetContext).MustGetUser()

	var ctx context.Context

	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}
	if err = AuthZProvider.Get().CanSetUsersUsername(ctx, currUser,
		*user); err != nil {
		return nil, canViewUserErrorHandle(currUser, *user,
			errors.Wrap(forbiddenError, err.Error()), db.ErrNotFound)
	}

	if params.NewUsername == nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "username is required")
		return nil, malformedRequestError
	}

	switch u, uErr := ByUsername(ctx, *params.NewUsername); {
	case uErr == db.ErrNotFound:
	case uErr != nil:
		return nil, uErr
	case u != nil:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "username is taken")
	}

	if err = UpdateUsername(ctx, &user.ID, *params.NewUsername); err != nil {
		return nil, err
	}

	return response{
		message: fmt.Sprintf("successfully updated %v", args.Username),
	}, nil
}

func (s *Service) postUser(c echo.Context) (interface{}, error) {
	if s.extConfig.Enabled() {
		return nil, externalSessionsError
	}
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

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return nil, err
	}

	var params request
	if err = json.Unmarshal(body, &params); err != nil {
		malformedRequestError := echo.NewHTTPError(http.StatusBadRequest, "bad request")
		return nil, malformedRequestError
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

	userToAdd := model.User{
		Username: params.Username,
		Admin:    params.Admin,
		Active:   params.Active,
	}
	currUser := c.(*detContext.DetContext).MustGetUser()

	var ctx context.Context
	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}
	if err = AuthZProvider.Get().CanCreateUser(ctx, currUser, userToAdd,
		ug); err != nil {
		return nil, errors.Wrap(forbiddenError, err.Error())
	}

	_, err = Add(ctx, &userToAdd, ug)
	switch {
	case err == db.ErrDuplicateRecord:
		return nil, echo.NewHTTPError(http.StatusBadRequest, "user already exists")
	case err != nil:
		return nil, err
	}

	telemetry.ReportUserCreated(params.Admin, params.Active)

	return response{
		message: fmt.Sprintf("successfully created user: %s", params.Username),
	}, nil
}

func (s *Service) getUserImage(c echo.Context) (interface{}, error) {
	args := struct {
		Username string `path:"username"`
	}{}
	if err := api.BindArgs(&args, c); err != nil {
		return nil, err
	}

	user, err := ByUsername(context.TODO(), args.Username)
	if err != nil {
		return nil, err
	}
	currUser := c.(*detContext.DetContext).MustGetUser()

	var ctx context.Context

	if c.Request() == nil || c.Request().Context() == nil {
		ctx = context.TODO()
	} else {
		ctx = c.Request().Context()
	}
	if err := AuthZProvider.Get().CanGetUsersImage(ctx, currUser,
		*user); err != nil {
		return nil, canViewUserErrorHandle(currUser, *user,
			errors.Wrap(forbiddenError, err.Error()), db.ErrNotFound)
	}

	c.Response().Header().Set("cache-control", "public, max-age=3600")

	return ProfileImage(ctx, args.Username)
}
