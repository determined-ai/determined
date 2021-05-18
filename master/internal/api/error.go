package api

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

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
	// ErrBadRequest is the inner error for errors that convert to a 400. Currently
	// only apiServer.askAtDefaultSystem respects this.
	ErrBadRequest = errors.New("bad request")
	// ErrNotFound is the inner error for errors that convert to a 404.
	ErrNotFound = errors.New("not found")
)

// AsErrBadRequest returns an error that wraps ErrBadRequest, so that errors.Is can identify it.
func AsErrBadRequest(msg string, args ...interface{}) error {
	return errors.Wrapf(
		ErrBadRequest,
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
