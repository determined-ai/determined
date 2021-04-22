package api

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CORSWithTargetedOrigin builds on labstack/echo CORS by dynamically setting the origin header to
// the request's origin.
func CORSWithTargetedOrigin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		origin := c.Request().Header.Get(echo.HeaderOrigin)
		if origin == "" {
			origin = "*"
		}
		origins := []string{origin}
		config := middleware.CORSConfig{AllowOrigins: origins, AllowCredentials: true}
		return middleware.CORSWithConfig(config)(next)(c)
	}
}
