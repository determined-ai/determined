package api

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

var (
	upgrader = websocket.Upgrader{}
)

// Route returns an echo compatible handler for JSON requests.
func Route(handler func(c echo.Context) (interface{}, error)) echo.HandlerFunc {
	return func(c echo.Context) error {
		result, err := handler(c)
		if err != nil {
			if httpErr, ok := errors.Cause(err).(*echo.HTTPError); ok {
				msg := err.Error()
				if httpErr == err {
					msg = fmt.Sprint(httpErr.Message)
				}
				return echo.NewHTTPError(httpErr.Code, msg)
			}
			return err
		}
		if result == nil {
			return c.NoContent(http.StatusNoContent)
		}

		switch typed := result.(type) {
		case []byte:
			return c.JSONBlob(http.StatusOK, typed)
		default:
			return c.JSON(http.StatusOK, result)
		}
	}
}

// WebSocketRoute upgrades incoming requests to websocket requests.
func WebSocketRoute(handler func(socket *websocket.Conn, c echo.Context) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			c.Logger().Error("websocket connection error: ", err)
			return nil
		}
		err = handler(ws, c)
		if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			c.Logger().Error("websocket handler error: ", err)
		}
		return nil
	}
}
