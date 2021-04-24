package user

import (
	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/api"
)

// RegisterAPIHandler initializes and registers the API handlers for all command related features.
func RegisterAPIHandler(echo *echo.Echo, m *Service, middleware ...echo.MiddlewareFunc) {
	echo.POST("/logout", api.Route(m.postLogout), middleware...)
	echo.POST("/login", api.Route(m.postLogin))
	usersGroup := echo.Group("/users", middleware...)
	usersGroup.GET("", api.Route(m.getUsers))
	usersGroup.POST("", api.Route(m.postUser))
	usersGroup.GET("/me", api.Route(m.getMe))
	usersGroup.PATCH("/:username", api.Route(m.patchUser))
	usersGroup.PATCH("/:username/username", api.Route(m.patchUsername))
}
