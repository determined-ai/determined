package apiutils

import (
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/db"
)

const (
	MaxLimit = 500
)

var (
	ErrBadRequest   = status.Error(codes.InvalidArgument, "bad request")
	ErrInvalidLimit = status.Errorf(codes.InvalidArgument,
		"Bad request: limit is required and must be <= %d", MaxLimit)
	ErrNotFound        = status.Error(codes.NotFound, "not found")
	ErrDuplicateRecord = status.Error(codes.AlreadyExists, "duplicate record")
	ErrInternal        = status.Error(codes.Internal, "internal server error")
	errPassthroughMap  = map[error]bool{
		nil:                true,
		ErrBadRequest:      true,
		ErrInvalidLimit:    true,
		ErrNotFound:        true,
		ErrDuplicateRecord: true,
		ErrInternal:        true,
	}
)

func MapAndFilterErrors(err error) error {
	if allowed := errPassthroughMap[err]; allowed {
		return err
	}

	switch {
	case errors.Is(err, db.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, db.ErrDuplicateRecord):
		return status.Error(codes.AlreadyExists, err.Error())
	}

	logrus.WithError(err).Debug("suppressing error at API boundary")

	return ErrInternal
}
