package rmerrors

import "github.com/pkg/errors"

// ErrUnsupported is returned when an unsupported feature of a resource manager is used.
type ErrUnsupported string

func (e ErrUnsupported) Error() string {
	return string(e)
}

// ErrNotSupported is returned when an unsupported feature of a resource manager is used.
var ErrNotSupported = errors.New("operation not supported")

// ErrNoDefaultResourcePool is returned when a default resource pool is requested but no
// default resource pool is set.
var ErrNoDefaultResourcePool = errors.New("no default resource pool set")
