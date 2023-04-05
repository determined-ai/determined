package cluster

import (
	"net/http"

	detContext "github.com/determined-ai/determined/master/internal/context"

	"github.com/labstack/echo/v4"
)

// CanGetUsageDetails returns an echo middleware that checks if the user has permission to get
// usage details.
func CanGetUsageDetails() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			user := c.(*detContext.DetContext).MustGetUser()
			permErr, err := AuthZProvider.Get().CanGetUsageDetails(c.Request().Context(), &user)
			if err != nil {
				return err
			}
			if permErr != nil {
				return echo.NewHTTPError(http.StatusForbidden, permErr.Error())
			}
			return next(c)
		}
	})
}
