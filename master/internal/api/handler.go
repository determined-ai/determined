package api

import (
	"fmt"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// equalASCIIFold returns true if s is equal to t with ASCII case folding as
// defined in RFC 4790.
func equalASCIIFold(s, t string) bool {
	for s != "" && t != "" {
		sr, size := utf8.DecodeRuneInString(s)
		s = s[size:]
		tr, size := utf8.DecodeRuneInString(t)
		t = t[size:]
		if sr == tr {
			continue
		}
		if 'A' <= sr && sr <= 'Z' {
			sr = sr + 'a' - 'A'
		}
		if 'A' <= tr && tr <= 'Z' {
			tr = tr + 'a' - 'A'
		}
		if sr != tr {
			return false
		}
	}
	return s == t
}

// Reference: https://github.com/gorilla/websocket/blob/main/server.go#L152
func checkOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]
	if len(origin) == 0 {
		return true
	}
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}
	return equalASCIIFold(u.Host, r.Host) || u.Hostname() == "localhost"
}

var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
}

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
