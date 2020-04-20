package scim

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)

// route wraps a handler returning Go objects into something that responds in a
// more HTTP native way by serializing responses, setting status codes and
// headers.
//
// This differs from api.Route() because the SCIM standard requires specific
// formatting of errors that differs from the standard endpoints.
func route(handler func(c echo.Context) (interface{}, error)) echo.HandlerFunc {
	return func(c echo.Context) error {
		status := http.StatusOK

		result, err := handler(c)
		if err != nil {
			code := http.StatusInternalServerError
			detail := err.Error()

			cause := errors.Cause(err)

			if httpErr, ok := cause.(*echo.HTTPError); ok {
				code = httpErr.Code
				if httpErr == err {
					detail = fmt.Sprint(httpErr.Message)
				}
			} else if cause == db.ErrNotFound {
				code = http.StatusNotFound
			}

			result = &model.SCIMError{
				Detail: detail,
				Status: code,
			}
			status = code
		} else if result == nil {
			return c.NoContent(http.StatusNoContent)
		}

		if s := c.Response().Status; status == http.StatusOK && s != 0 {
			status = s
		}

		c.Response().Header().Set("Content-Type", scimContentType)

		return c.JSON(status, result)
	}
}

func newBadRequestError(err error) error {
	if err == nil {
		return nil
	}
	return echo.NewHTTPError(http.StatusBadRequest, err.Error())
}

func newNotFoundError(err error) error {
	if err == nil {
		return nil
	}
	return echo.NewHTTPError(http.StatusNotFound, err.Error())
}

func newConflictError(err error) error {
	if err == nil {
		return nil
	}
	return echo.NewHTTPError(http.StatusConflict, err.Error())
}

// RegisterAPIHandler registers API handlers for SCIM endpoints.
func RegisterAPIHandler(e *echo.Echo, db *db.PgDB, c *Config, locationRoot *url.URL) {
	s := &service{c, db, locationRoot}

	users := e.Group(scimPathRoot+"/Users", middleware.BasicAuth(s.validateSCIMCredentials))
	users.POST("", route(s.PostUser))
	users.GET("", route(s.GetUsers))
	users.GET("/:user_id", route(s.GetUser))
	users.PUT("/:user_id", route(s.PutUser))
	users.PATCH("/:user_id", route(s.PatchUser))

	groups := e.Group(scimPathRoot+"/Groups", middleware.BasicAuth(s.validateSCIMCredentials))
	groups.GET("", route(s.GetGroups))
}
