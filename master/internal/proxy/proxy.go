package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Proxy-specific actor messages.
type (
	// Register registers the service name with the associated target URL. All requests with the
	// format ".../:service-name/*" are forwarded to the service via the target URL.
	Register struct {
		ServiceID string
		URL       *url.URL
	}
	// Unregister removes the service from the proxy. All future requests until the service name is
	// registered again will be responded with a 404 response. If the service is not registered with
	// the proxy, the message is ignored.
	Unregister struct{ ServiceID string }
	// NewProxyHandler returns a middleware function for proxying HTTP-like traffic to services
	// running in the cluster.
	NewProxyHandler struct{ ServiceID string }

	// GetSummary returns a snapshot of the registered services.
	GetSummary struct{}
)

// Service represents a registered service. The LastRequested field is used by
// the Tensorboard manager to spin down idle instances of Tensorboard.
type Service struct {
	URL           *url.URL
	LastRequested time.Time
}

// Proxy is an actor that proxies requests to registered services.
type Proxy struct {
	lock     sync.RWMutex
	services map[string]*Service
}

// Receive implements the actor.Actor interface.
func (p *Proxy) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		p.services = make(map[string]*Service)
	case Register:
		if msg.ServiceID == "" {
			return nil
		}
		p.lock.Lock()
		defer p.lock.Unlock()
		ctx.Log().Infof("registering service: %s (%v)", msg.ServiceID, msg.URL)
		p.services[msg.ServiceID] = &Service{msg.URL, time.Now()}

		if ctx.ExpectingResponse() {
			ctx.Respond(nil)
		}
	case Unregister:
		p.lock.Lock()
		defer p.lock.Unlock()
		delete(p.services, msg.ServiceID)
	case NewProxyHandler:
		ctx.Respond(p.newProxyHandler(msg.ServiceID))
	case GetSummary:
		ctx.Respond(p.getSummary())
	case actor.PostStop:
		p.lock.Lock()
		defer p.lock.Unlock()
		// Erase all services from the proxy in case any handlers are still active.
		p.services = nil
	}
	return nil
}

func (p *Proxy) getTargetURL(serviceName string) *url.URL {
	p.lock.Lock()
	defer p.lock.Unlock()
	service := p.services[serviceName]
	if service == nil {
		return nil
	}
	service.LastRequested = time.Now()
	// Make a copy to avoid callers mutating the url outside of this locked
	// method.
	sURL := *service.URL
	return &sURL
}

// Service an HTTP request through the /proxy/:service/* route.
func (p *Proxy) newProxyHandler(serviceID string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Look up the service name in the url path.
		serviceName := c.Param(serviceID)
		serviceURL := p.getTargetURL(serviceName)
		if serviceURL == nil {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("service not found: %s", serviceName))
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
		case strings.HasSuffix(c.Request().URL.Path, "tcp"):
			proxy = newSingleHostReverseTCPOverWebSocketProxy(c, serviceURL)
		case c.IsWebSocket():
			proxy = newSingleHostReverseWebSocketProxy(c, serviceURL)
		default:
			proxy = httputil.NewSingleHostReverseProxy(serviceURL)
		}
		proxy.ServeHTTP(c.Response(), req)

		return nil
	}
}

func (p *Proxy) getSummary() map[string]Service {
	p.lock.RLock()
	defer p.lock.RUnlock()
	snapshot := make(map[string]Service)

	for id, service := range p.services {
		sURL := *service.URL
		snapshot[id] = Service{&sURL, service.LastRequested}
	}

	return snapshot
}

func asyncRun(f func() error) chan error {
	errs := make(chan error, 1)
	go func() {
		defer close(errs)
		errs <- f()
	}()
	return errs
}

func asyncCopy(dst io.Writer, src io.Reader) chan error {
	return asyncRun(func() error {
		if _, err := io.Copy(dst, src); err != nil && err != io.EOF {
			return err
		}
		return nil
	})
}
