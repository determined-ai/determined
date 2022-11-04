package internal

import (
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/master/pkg/logger"
)

type agentAPIServer struct {
	// Configuration details.
	opts options.AgentOptions

	// Internal state.
	server *echo.Echo
}

func newAgentAPIServer(opts options.AgentOptions) *agentAPIServer {
	server := echo.New()
	server.Logger = logger.New()
	server.HidePort = true
	server.HideBanner = true
	server.Use(middleware.Recover())
	server.Pre(middleware.RemoveTrailingSlash())
	server.Use(otelecho.Middleware("determined-agent"))

	server.Any("/debug/pprof/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	server.Any("/debug/pprof/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	server.Any("/debug/pprof/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	server.Any("/debug/pprof/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	server.Any("/debug/pprof/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	return &agentAPIServer{
		opts:   opts,
		server: server,
	}
}

func (a *agentAPIServer) serve() error {
	bindAddr := fmt.Sprintf("%s:%d", a.opts.BindIP, a.opts.BindPort)
	logrus.Infof("starting agent server on [%s]", bindAddr)
	if a.opts.TLS {
		return a.server.StartTLS(bindAddr, a.opts.CertFile, a.opts.KeyFile)
	}
	return a.server.Start(bindAddr)
}

func (a *agentAPIServer) close() error {
	return a.server.Close()
}
