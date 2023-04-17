package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

// websocketReadWriter exposes an io.ReadWriter interface to a WebSocket connection that is only
// being used for binary communication.
type websocketReadWriter struct {
	ws  *websocket.Conn
	buf *bytes.Buffer
}

func (w *websocketReadWriter) Read(buf []byte) (int, error) {
	if w.buf.Len() > 0 {
		b, err := w.buf.Read(buf)
		if err != nil {
			return 0, fmt.Errorf("error reading from websocket buffer: %w", err)
		}
		return b, nil
	}
	for {
		switch msg, data, err := w.ws.ReadMessage(); {
		case err != nil:
			return 0, fmt.Errorf("error reading message from websocket: %w", err)
		case msg == websocket.CloseMessage:
			return 0, io.EOF
		case msg == websocket.BinaryMessage:
			if len(data) > 0 {
				w.buf.Write(data)
				b, err := w.buf.Read(buf)
				if err != nil {
					return 0, fmt.Errorf("error reading from websocket buffer binary msg: %w", err)
				}
				return b, nil
			}
		}
	}
}

func (w *websocketReadWriter) Write(buf []byte) (int, error) {
	if err := w.ws.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		return 0, fmt.Errorf("error writing websocket binary message: %w", err)
	}
	return len(buf), nil
}

func newSingleHostReverseTCPOverWebSocketProxy(c echo.Context, t *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Make sure we can open the connection to the remote host.
		out, err := net.Dial("tcp", t.Host)
		if err != nil {
			c.Error(echo.NewHTTPError(http.StatusBadGateway,
				errors.Errorf("error dialing to %v: %v", t, err)))
			return
		}
		defer func() {
			if cerr := out.Close(); cerr != nil {
				c.Logger().Error(cerr)
			}
		}()

		ws, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			c.Error(echo.NewHTTPError(http.StatusBadGateway, errors.Wrap(err, "error upgrading")))
			return
		}

		rw := &websocketReadWriter{ws: ws, buf: new(bytes.Buffer)}
		copyReqErr := asyncCopy(rw, out)
		copyResErr := asyncCopy(out, rw)

		if cerr := <-copyReqErr; cerr != nil {
			c.Logger().Errorf("error copying request body for %v: %v", t, cerr)
		}
		if cerr := <-copyResErr; cerr != nil {
			c.Logger().Errorf("error copying response body for %v: %v", t, cerr)
		}
	})
}
