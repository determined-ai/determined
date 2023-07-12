package api

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/config"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/labstack/echo/v4"
)

// JSONErrorHandler sends a JSON response with a single "message" key containing the error message.
func JSONErrorHandler(err error, c echo.Context) {
	// Default to a 500 internal server error unless the endpoint explicitly returns otherwise.
	var (
		code             = http.StatusInternalServerError
		msg  interface{} = err
	)
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		msg = he.Message
	}
	if code >= 500 {
		c.Logger().Error(err)
	}
	if !c.Response().Committed {
		// For the HEAD method, the server MUST NOT return a message-body in the response.
		if c.Request().Method == echo.HEAD {
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, map[string]interface{}{"message": fmt.Sprint(msg)})
		}
		// Log the error returned from formatting the error response.
		if err != nil {
			c.Logger().Error(err)
		}
	}
}

var (
	// ErrInvalid is the inner error for errors that convert to a 400. Currently
	// only apiServer.askAtDefaultSystem respects this.
	ErrInvalid = errors.New("invalid arguments")
	// ErrNotFound is the inner error for errors that convert to a 404.
	ErrNotFound = errors.New("not found")
	// ErrNotImplemented is the inner error for errors that convert to a 501.
	ErrNotImplemented = errors.New("not implemented")

	// ErrAPIRemoved is an error to inform the client they are calling an old, removed API.
	ErrAPIRemoved = errors.New(`the API being called was removed,
please ensure the client consuming the API is up to date and report a bug if the problem persists`)
)

// NotFoundErrMsg creates a formatted message about a resource not being found.
func NotFoundErrMsg(name string, id string) string {
	msg := fmt.Sprintf(`%s '%s' not found%s`, name, id, AddRBACSuffix())
	if id == "" {
		msg = fmt.Sprintf("%s not found%s", name, AddRBACSuffix())
	}
	return msg
}

// NotFoundErrs is a wrapper function to create status.Errors with an informative message as to
// what category of error (NotFound), the name (trial/task/workspace etc) & the specific ID is.
// The statusErr bool returns a status.Error if true, or a NewHTTPError if false..
func NotFoundErrs(name string, id string, statusErr bool) error {
	msg := NotFoundErrMsg(name, id)
	if statusErr {
		return status.Error(codes.NotFound, msg)
	}
	return echo.NewHTTPError(http.StatusNotFound, msg)
}

// AddRBACSuffix adds a "check your permission" string to errors if RBAC is enabled.
// This suffix is applied to any endpoint that can be limited by RBAC, specifically 404 errors
// and should not expose that the entity exists.
func AddRBACSuffix() string {
	if config.GetAuthZConfig().IsRBACUIEnabled() {
		return ", please check your permissions."
	}
	return ""
}

// AsValidationError returns an error that wraps ErrInvalid, so that errors.Is can identify it.
func AsValidationError(msg string, args ...interface{}) error {
	return errors.Wrapf(
		ErrInvalid,
		msg,
		args...,
	)
}

// AsErrNotFound returns an error that wraps ErrNotFound, so that errors.Is can identify it.
func AsErrNotFound(msg string, args ...interface{}) error {
	return errors.Wrapf(
		ErrNotFound,
		msg,
		args...,
	)
}

// WrapWithFallbackCode prepares errors for returning to the client by providing a fallback code
// and more context.
func WrapWithFallbackCode(err error, code codes.Code, msg string) error {
	err = APIErrToGRPC(err)
	if s, ok := status.FromError(err); ok {
		return status.New(s.Code(), msg+": "+s.Message()).Err()
	}
	return status.New(code, msg+": "+err.Error()).Err()
}

// APIErrToGRPC converts internal api error categories into grpc status.Errors.
func APIErrToGRPC(err error) error {
	// If the error is already a grpc status.Error, return it as is.
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch {
	case errors.Is(err, ErrInvalid):
		return status.Errorf(
			codes.InvalidArgument,
			err.Error(),
		)
	case errors.Is(err, ErrNotFound):
		return status.Errorf(
			codes.NotFound,
			err.Error(),
		)
	default:
		return err
	}
}

// EchoErrToGRPC converts internal api error categories into grpc status.Errors.
func EchoErrToGRPC(err error) (bool, error) {
	if err, ok := err.(*echo.HTTPError); ok {
		return true, status.Error(
			codeFromHTTPStatus(err.Code),
			err.Error(),
		)
	}
	return false, err
}

// GrpcErrToEcho converts grpc status.Errors into internal api error categories.
func GrpcErrToEcho(err error) (bool, error) {
	status, ok := status.FromError(err)
	if !ok {
		return false, err
	}
	switch status.Code() {
	case codes.NotFound:
		return true, echo.NewHTTPError(http.StatusNotFound, status.Message())
	case codes.Unauthenticated:
		return true, echo.NewHTTPError(http.StatusUnauthorized, status.Message())
	case codes.InvalidArgument:
		return true, echo.NewHTTPError(http.StatusBadRequest, status.Message())
	case codes.OK:
		return true, nil
	case codes.PermissionDenied:
		return true, echo.NewHTTPError(http.StatusForbidden, status.Message())
	default:
		return false, err
	}
}

func codeFromHTTPStatus(code int) codes.Code {
	switch {
	case code == 404:
		return codes.NotFound
	case code == 403, code == 401:
		return codes.Unauthenticated
	case code == 400:
		return codes.InvalidArgument
	case 200 <= code && code < 300:
		return codes.OK
	}
	return codes.Internal
}
