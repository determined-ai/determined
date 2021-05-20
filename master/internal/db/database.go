package db

import (
	"github.com/pkg/errors"
)

// ErrNotFound is returned if nothing is found.
var ErrNotFound = errors.New("not found")

// ErrTooManyRowsAffected is returned if too many rows are affected.
var ErrTooManyRowsAffected = errors.New("too many rows are affected")

// ErrDuplicateRecord is returned when trying to create a row that already exists.
var ErrDuplicateRecord = errors.New("row already exists")
