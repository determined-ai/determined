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

func setupEchoRoutes(m *Master) {
	for _, aRedirect := range redirects {
		aRedirect := aRedirect
		m.echo.Router().Add(aRedirect.method, aRedirect.src, func(c echo.Context) error {
			return c.Redirect(aRedirect.code, aRedirect.dest)
		})
	}
}
