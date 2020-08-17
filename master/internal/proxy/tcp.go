package proxy

import (
	"net"
	"net/http"
	"net/url"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
)

func newSingleHostReverseTCPProxy(c echo.Context, t *url.URL) http.Handler {
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

		// Send the 200 OK response header.
		c.Response().WriteHeader(http.StatusOK)

		// Hijack the connection instead of providing a body.
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
