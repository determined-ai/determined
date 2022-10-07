package apiutils

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/grpcutil"

	"github.com/determined-ai/determined/master/internal/db"
)

const (
	// MaxLimit is the maximum limit value for pagination.
	MaxLimit = 500
)

var (
	// ErrBadRequest is the returned standard error for bad requests.
	ErrBadRequest = status.Error(codes.InvalidArgument, "bad request")
	// ErrInvalidLimit is the returned standard error for invalid limit for pagination.
	ErrInvalidLimit = status.Errorf(codes.InvalidArgument,
		"Bad request: limit is required and must be <= %d", MaxLimit)
	// ErrNotFound is the returned standard error for value(s) not found.
	ErrNotFound = status.Error(codes.NotFound, "not found")
	// ErrDuplicateRecord is the returned standard error for finding duplicates.
	ErrDuplicateRecord = status.Error(codes.AlreadyExists, "duplicate record")
	// ErrInternal is the returned standard error for an internal error.
	ErrInternal       = status.Error(codes.Internal, "internal server error")
	errPassthroughMap = map[error]bool{
		nil:                          true,
		ErrBadRequest:                true,
		ErrInvalidLimit:              true,
		ErrNotFound:                  true,
		ErrDuplicateRecord:           true,
		ErrInternal:                  true,
		grpcutil.ErrPermissionDenied: true,
	}
)

// MapAndFilterErrors takes in an error at the db level and translates it into a standard error.
func MapAndFilterErrors(err error) error {
	if allowed := errPassthroughMap[err]; allowed {
		return err
	}

	if _, ok := err.(authz.PermissionDeniedError); ok {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	switch {
	case errors.Is(err, db.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, db.ErrDuplicateRecord):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, db.ErrInvalidInput):
		return ErrBadRequest
	}

	logrus.WithError(err).Debug("suppressing error at API boundary")

	return ErrInternal
}
