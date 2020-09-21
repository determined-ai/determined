package api

import "github.com/labstack/echo"

// AddCORSHeader enables CORS for request's origin.
func AddCORSHeader(c echo.Context) {
	if origin := c.Request().Header.Get("Origin"); origin != "" {
		c.Response().Header().Set("Access-Control-Allow-Origin", origin)
	}
}
