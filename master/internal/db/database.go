package db

import (
	"github.com/pkg/errors"
)

// ErrNotFound is returned if nothing is found.
var ErrNotFound = errors.New("not found")

// ErrDuplicateRecord is returned when trying to create a row that already exists.
var ErrDuplicateRecord = errors.New("duplicate")
