package proxy

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/pkg/errors"
)

func asyncCopyToWebSocket(dst *websocket.Conn, src io.Reader) chan error {
	return asyncRun(func() error {
		for buf := make([]byte, 4096); ; {
			n, err := src.Read(buf)
			if err != nil {
				return err
			}
			if err := dst.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return err
			}
		}
	})
}

func asyncCopyFromWebSocket(dst io.Writer, src *websocket.Conn) chan error {
	return asyncRun(func() error {
		for {
			switch tp, buf, err := src.ReadMessage(); {
			case err != nil:
				return err
			case tp == websocket.BinaryMessage:
				if _, err := dst.Write(buf); err != nil {
					return err
				}
			}
		}
	})
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

		copyReqErr := asyncCopyToWebSocket(ws, out)
		copyResErr := asyncCopyFromWebSocket(out, ws)

		if cerr := <-copyReqErr; cerr != nil {
			c.Logger().Errorf("error copying request body for %v: %v", t, cerr)
		}
		if cerr := <-copyResErr; cerr != nil {
			c.Logger().Errorf("error copying response body for %v: %v", t, cerr)
		}
	})
}
