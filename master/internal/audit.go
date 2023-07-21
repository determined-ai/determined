package internal

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"

	detContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/rbac/audit"
)

// LogrusLogFn is an interface for all the logrus Levelf log functions.
type LogrusLogFn func(format string, args ...interface{})

const proxyPrefix = "/proxy"

var debugMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodConnect: true,
	http.MethodHead:    true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
}

var infoMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPatch:  true,
	http.MethodPut:    true,
	http.MethodDelete: true,
}

var unauthorizedStatuses = map[int]bool{
	http.StatusUnauthorized: true,
	http.StatusForbidden:    true,
}

func auditLogMiddleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()
			res := c.Response()
			if err = next(c); err != nil {
				c.Error(err)
			}

			for path := range staticWebDirectoryPaths {
				if strings.HasPrefix(c.Path(), path) {
					return
				}
			}
			if strings.HasPrefix(c.Request().URL.Path, proxyPrefix) {
				return
			}

			unauthorized := false
			if unauthorizedStatuses[c.Response().Status] {
				unauthorized = true
			}

			errored := c.Response().Status >= 400

			fields := map[string]interface{}{
				"type":      "echo_audit_log",
				"remote_ip": c.RealIP(),
				// This has an implicit dependency that the context and auth middleware runs first.
				// This should always be true since we operate after the next() call.
				"determined_user": c.(*detContext.DetContext).GetUsername(),
				"unauthorized":    unauthorized,
			}

			var logFn LogrusLogFn
			switch method := c.Request().Method; {
			case infoMethods[method] || errored:
				logFn = log.WithFields(fields).Infof
			case debugMethods[method]:
				logFn = log.WithFields(fields).Debugf
			default:
				return
			}

			logFn("%s %s %d", req.Method, req.URL.Path, res.Status)

			return
		}
	})
}

func authzAuditLogMiddleware() echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			isProxiedToProto := strings.HasPrefix(c.Request().RequestURI, "/api/v1")
			if !isProxiedToProto {
				fields := log.Fields{"endpoint": c.Request().RequestURI}
				newCtx := context.WithValue(c.Request().Context(), audit.LogKey{}, fields)
				c.SetRequest(c.Request().WithContext(newCtx))
			}

			return next(c)
		}
	})
}
