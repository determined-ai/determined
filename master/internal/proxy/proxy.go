package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Service represents a registered service. The LastRequested field is used by
// the Tensorboard manager to spin down idle instances of Tensorboard.
type Service struct {
	URL                  *url.URL
	LastRequested        time.Time
	ProxyTCP             bool
	AllowUnauthenticated bool
}

// ProxyHTTPAuth processes a proxy request, returning true if the request should terminate
// immediately and an error if one was encountered during authentication.
type ProxyHTTPAuth func(echo.Context) (done bool, err error)

// Proxy is an actor that proxies requests to registered services.
type Proxy struct {
	HTTPAuth ProxyHTTPAuth
	Services map[string]*Service
	Syslog   *logrus.Entry
}

// DefaultProxy is the global proxy singleton.
var DefaultProxy *Proxy

// InitProxy initializes the global proxy.
func InitProxy(httpAuth ProxyHTTPAuth) {
	if DefaultProxy != nil {
		logrus.Warn(
			"detected re-initialization of Proxy that should never occur outside of tests",
		)
	}
	DefaultProxy = &Proxy{
		HTTPAuth: httpAuth,
		Services: make(map[string]*Service),
		Syslog:   logrus.WithTime(time.Now()), // NIT: How else to initialize syslog?
	}
}

// Register registers the service name with the associated target URL. All requests with the
// format ".../:service-name/*" are forwarded to the service via the target URL.
func Register(serviceID string, url *url.URL, proxyTCP bool, unauth bool) {
	if serviceID == "" {
		return
	}
	// NIT CAROLINA: When to fail?
	DefaultProxy.Syslog.Infof("registering service: %s (%v)", serviceID, url)
	DefaultProxy.Services[serviceID] = &Service{
		URL:                  url,
		LastRequested:        time.Now(),
		ProxyTCP:             proxyTCP,
		AllowUnauthenticated: unauth,
	}
	return
}

// Unregister removes the service from the proxy. All future requests until the service name is
// registered again will be responded with a 404 response. If the service is not registered with
// the proxy, the message is ignored.
func Unregister(serviceID string) {
	delete(DefaultProxy.Services, serviceID)
}

// ClearProxy erases all services from the proxy in case any handlers are still active.
func ClearProxy() {
	DefaultProxy.Services = nil
}

// GetService returns the Service, if any, given the serviceID key.
func GetService(serviceID string) *Service {
	service := DefaultProxy.Services[serviceID]
	if service == nil {
		return nil
	}
	service.LastRequested = time.Now()

	// Make a copy to avoid callers mutating the object outside of this locked method.
	sURL := *service.URL
	return &Service{
		URL:                  &sURL,
		LastRequested:        service.LastRequested,
		ProxyTCP:             service.ProxyTCP,
		AllowUnauthenticated: service.AllowUnauthenticated,
	}
}

// NewProxyHandler returns a middleware function for proxying HTTP-like traffic to services
// running in the cluster. Services an HTTP request through the /proxy/:service/* route.
func NewProxyHandler(serviceID string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Look up the service name in the url path.
		serviceName := c.Param(serviceID)
		service := GetService(serviceName)

		if service == nil {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("service not found: %s", serviceName))
		}

		if !service.AllowUnauthenticated {
			switch done, err := DefaultProxy.HTTPAuth(c); {
			case err != nil:
				return err
			case done:
				return nil
			}
		}

		// Set proxy headers.
		req := c.Request()
		if req.Header.Get(echo.HeaderXRealIP) == "" {
			req.Header.Set(echo.HeaderXRealIP, c.RealIP())
		}
		if req.Header.Get(echo.HeaderXForwardedProto) == "" {
			req.Header.Set(echo.HeaderXForwardedProto, c.Scheme())
		}
		if c.IsWebSocket() && req.Header.Get(echo.HeaderXForwardedFor) == "" {
			req.Header.Set(echo.HeaderXForwardedFor, c.RealIP())
		}

		// Proxy the request to the target host.
		var proxy http.Handler
		switch {
		case service.ProxyTCP:
			proxy = newSingleHostReverseTCPOverWebSocketProxy(c, service.URL)
		case c.IsWebSocket():
			proxy = newSingleHostReverseWebSocketProxy(c, service.URL)
		default:
			proxy = httputil.NewSingleHostReverseProxy(service.URL)
		}
		proxy.ServeHTTP(c.Response(), req)

		return nil
	}
}

// GetSummary returns a snapshot of the registered services.
func GetSummary() map[string]Service {
	snapshot := make(map[string]Service)

	for id, service := range DefaultProxy.Services {
		sURL := *service.URL
		snapshot[id] = Service{
			URL:                  &sURL,
			LastRequested:        service.LastRequested,
			ProxyTCP:             service.ProxyTCP,
			AllowUnauthenticated: service.AllowUnauthenticated,
		}
	}

	return snapshot
}

func asyncCopy(dst io.Writer, src io.Reader) chan error {
	errs := make(chan error, 1)
	go func() {
		defer close(errs)
		_, err := io.Copy(dst, src)
		if err != io.EOF {
			errs <- err
		}
	}()
	return errs
}
