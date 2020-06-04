package db

import (
	"fmt"

	"github.com/pkg/errors"
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
