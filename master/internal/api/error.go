package api

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
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
)

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

// APIErrToGRPC converts internal api error categories into grpc status.Errors.
func APIErrToGRPC(err error) error {
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
