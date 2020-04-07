package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Proxy-specific actor messages.
type (
	// Register registers the service name with the associated target URL. All requests with the
	// format ".../:service-name/*" are forwarded to the service via the target URL.
	Register struct {
		Service string
		Target  *url.URL
	}
	// Unregister removes the service from the proxy. All future requests until the service name is
	// registered again will be responded with a 404 response. If the service is not registered with
	// the proxy, the message is ignored.
	Unregister struct{ Service string }
	// NewHandler returns a middleware function for proxying requests to services running in the
	// cluster.
	NewHandler struct{ ServiceKey string }
)

// Proxy is an actor that proxies requests to registered services.
type Proxy struct {
	lock     sync.RWMutex
	services map[string]*url.URL
}

// Receive implements the actor.Actor interface.
func (p *Proxy) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		p.services = make(map[string]*url.URL)
	case Register:
		if msg.Service == "" {
			return nil
		}
		p.lock.Lock()
		defer p.lock.Unlock()
		ctx.Log().Infof("registering service: %s (%v)", msg.Service, msg.Target)
		p.services[msg.Service] = msg.Target

		if ctx.ExpectingResponse() {
			ctx.Respond(nil)
		}
	case Unregister:
		p.lock.Lock()
		defer p.lock.Unlock()
		delete(p.services, msg.Service)
	case NewHandler:
		ctx.Respond(p.newHandler(msg.ServiceKey))
	case actor.PostStop:
		p.lock.Lock()
		defer p.lock.Unlock()
		// Erase all services from the proxy in case any handlers are still active.
		p.services = nil
	}
	return nil
}

func (p *Proxy) newHandler(serviceKey string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Look up the service name in the url path.
		serviceName := c.Param(serviceKey)
		target, ok := func() (*url.URL, bool) {
			p.lock.RLock()
			defer p.lock.RUnlock()
			target, ok := p.services[serviceName]
			return target, ok
		}()
		if !ok {
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
		if c.IsWebSocket() {
			proxy = newSingleHostReverseWebSocketProxy(c, target)
		} else {
			proxy = httputil.NewSingleHostReverseProxy(target)
		}
		proxy.ServeHTTP(c.Response(), req)

		return nil
	}
}
