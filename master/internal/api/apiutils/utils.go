package apiutils

import (
	"context"
	"fmt"
	"reflect"

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

// HaveAtLeastOneSharedDefinedField compares two different configurations and
// returns an error if both try to define the same field.
func HaveAtLeastOneSharedDefinedField(config1, config2 interface{}) error {
	v1 := reflect.ValueOf(config1)
	v2 := reflect.ValueOf(config2)

	// If the values are pointers, dereference them
	if v1.Kind() == reflect.Ptr {
		v1 = v1.Elem()
	}
	if v2.Kind() == reflect.Ptr {
		v2 = v2.Elem()
	}

	// Check if both values are valid structs
	if v1.Kind() != reflect.Struct || v2.Kind() != reflect.Struct {
		return fmt.Errorf("both inputs must be structs")
	}

	hasSharedField := false

	// Iterate over the fields in the struct
	for i := 0; i < v1.NumField(); i++ {
		field1 := v1.Field(i)
		field2 := v2.Field(i)

		// Check if the field is a pointer, map, or interface
		if field1.Kind() == reflect.Ptr || field1.Kind() == reflect.Map || field1.Kind() == reflect.Interface {
			if !field1.IsNil() && !field2.IsNil() {
				hasSharedField = true
				// Compare the dereferenced values
				if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
					return fmt.Errorf("shared non-null field has different values")
				}
			}
		} else if field1.IsValid() && field2.IsValid() && !field1.IsZero() && !field2.IsZero() {
			hasSharedField = true
			// For non-pointer fields, compare directly if both are non-zero
			if !reflect.DeepEqual(field1.Interface(), field2.Interface()) {
				return fmt.Errorf("shared non-null field has different values")
			}
		}
	}

	if !hasSharedField {
		return nil // No shared non-null fields to compare
	}

	return nil // Configs are equal in shared non-null fields
}
