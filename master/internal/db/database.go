package db

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
)

// ErrNotFound is returned if nothing is found.
var ErrNotFound = errors.New("not found")

// ErrDuplicateUser is returned when trying to create a user with a username
// that is already taken.
type ErrDuplicateUser struct {
	Username string
}

func (s ErrDuplicateUser) Error() string {
	return fmt.Sprintf("user with username %q already exists", s.Username)
}

// ErrUserSessionNotFound is returned when a user session was requested using an
// id for which there is no session.
type ErrUserSessionNotFound struct {
	SessionID model.SessionID
}

func (s ErrUserSessionNotFound) Error() string {
	return fmt.Sprintf("session with id %d not found", s.SessionID)
}

// ErrNoSuchUsername is returned when a user with the given name is requested
// but does not exist.
type ErrNoSuchUsername struct {
	Username string
}

func (s ErrNoSuchUsername) Error() string {
	return fmt.Sprintf("no user exists with username '%s'", s.Username)
}
