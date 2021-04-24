package proxy

import (
	"net"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func newSingleHostReverseWebSocketProxy(c echo.Context, t *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		in, _, err := c.Response().Hijack()
		if err != nil {
			c.Error(errors.Errorf("error hijacking connection to %v: %v", t, err))
			return
		}
		defer func() {
			if cerr := in.Close(); cerr != nil {
				c.Logger().Error(cerr)
			}
		}()

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

		err = r.Write(out)
		if err != nil {
			c.Error(echo.NewHTTPError(http.StatusBadGateway,
				errors.Errorf("error copying headers for %v: %v", t, err)))
			return
		}

		copyReqErr := asyncCopy(out, in)
		copyResErr := asyncCopy(in, out)
		if cerr := <-copyReqErr; cerr != nil {
			c.Logger().Errorf("error copying request body for %v: %v", t, cerr)
		}
		if cerr := <-copyResErr; cerr != nil {
			c.Logger().Errorf("error copying response body for %v: %v", t, cerr)
		}
	})
}
