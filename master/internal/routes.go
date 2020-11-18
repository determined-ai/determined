package internal

import (
	"net/http"

	"github.com/labstack/echo"
)

type redirect struct {
	src    string
	dest   string
	method string
	code   int
}

var redirects = [...]redirect{
	{
		src:    "/",
		dest:   webuiBaseRoute,
		method: http.MethodGet,
		code:   http.StatusMovedPermanently,
	},
}

func setupEchoRedirects(m *Master) {
	for idx := range redirects {
		func(r redirect) {
			m.echo.Router().Add(r.method, r.src, func(c echo.Context) error {
				return c.Redirect(r.code, r.dest)
			})
		}(redirects[idx])
	}
}
