package apiutils

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
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
	ErrInternal = status.Error(codes.Internal, "internal server error")

	// ErrorPassthroughSet is the default set of errors that will be passed through by
	// MapAndFilterErrors without blocking or mapping.
	ErrorPassthroughSet = map[error]bool{
		ErrBadRequest:                true,
		ErrInvalidLimit:              true,
		ErrNotFound:                  true,
		ErrDuplicateRecord:           true,
		ErrInternal:                  true,
		grpcutil.ErrPermissionDenied: true,
	}

	// ErrorMapping is the default mapping of errors used by MapAndFilterErrors to, for example,
	// map errors from other application layers to what an API package will want to return.
	ErrorMapping = map[error]error{
		db.ErrNotFound:        ErrNotFound,
		db.ErrDuplicateRecord: ErrDuplicateRecord,
		db.ErrInvalidInput:    ErrBadRequest,
	}
)

// MapAndFilterErrors takes in an error at the db level and translates it into a standard error.
func MapAndFilterErrors(err error, passthrough map[error]bool, mapping map[error]error) error {
	if err == nil {
		return nil
	}

	if _, ok := err.(authz.PermissionDeniedError); ok {
		return status.Error(codes.PermissionDenied, err.Error())
	}

	if passthrough == nil {
		passthrough = ErrorPassthroughSet
	}
	if mapping == nil {
		mapping = ErrorMapping
	}

	if errors.Is(err, context.Canceled) {
		return status.Error(codes.Canceled, err.Error())
	}

	// Filter
	if allowed := passthrough[err]; allowed {
		return err
	}

	// Map
	if mappedErr := mapping[err]; mappedErr != nil {
		return mappedErr
	}
	for inputErr, outputErr := range mapping {
		if errors.Is(err, inputErr) {
			return status.Error(status.Code(outputErr), err.Error())
		}
	}

	logrus.WithError(err).Warn("suppressing error at API boundary")
	return ErrInternal
}
