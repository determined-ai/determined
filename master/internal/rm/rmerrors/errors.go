package rmerrors

import "github.com/pkg/errors"

// ErrUnsupported is returned when an unsupported feature of a resource manager is used.
type ErrUnsupported string

func (e ErrUnsupported) Error() string {
	return string(e)
}

// ErrNotSupported is returned when an unsupported feature of a resource manager is used.
var ErrNotSupported = errors.New("operation not supported")
