package context

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"
)

// DetContext is a wrapper around echo.Context so that some convenience functions that depend on
// context can be made accessible to handlers.
type DetContext struct {
	echo.Context
}

// SetUser sets the user for an echo request context.
func (c *DetContext) SetUser(user model.User) {
	c.Set("user", user)
}

// SetUserSession sets session information for an echo request context.
func (c *DetContext) SetUserSession(session model.UserSession) {
	c.Set("user-session", session)
}

// MustGetUser returns the user for the relevant echo request context. Panics if the user has not
// been set, so this method should only be used inside handlers that _require_ authentication.
func (c *DetContext) MustGetUser() model.User {
	user := c.Get("user")
	if user == nil {
		panic("Failed to get authenticated user from request context!")
	}
	return user.(model.User)
}

// MustGetUserSession returns the user session for the relevant echo request context. Panics if
// the user has not been set, so this method should only be used inside handlers that
// _require_ authentication.
func (c *DetContext) MustGetUserSession() model.UserSession {
	session := c.Get("user-session")
	if session == nil {
		panic("Failed to get user session from request context!")
	}
	return session.(model.UserSession)
}
