package api

import (
	"fmt"
	"net/http"

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
