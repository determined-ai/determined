package proxy

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"

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

// Clone returns a deep copy of the Service.
func (s Service) Clone() Service {
	sURL := *s.URL
	return Service{
		URL:                  &sURL,
		LastRequested:        s.LastRequested,
		ProxyTCP:             s.ProxyTCP,
		AllowUnauthenticated: s.AllowUnauthenticated,
	}
}

// ProxyHTTPAuth processes a proxy request, returning true if the request should terminate
// immediately and an error if one was encountered during authentication.
type ProxyHTTPAuth func(echo.Context) (done bool, err error)

// Proxy is an actor that proxies requests to registered services.
type Proxy struct {
	lock     sync.RWMutex
	HTTPAuth ProxyHTTPAuth
	services map[string]*Service
	syslog   *logrus.Entry
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
		services: make(map[string]*Service),
		syslog:   logrus.WithField("component", "proxy"),
	}
	err := LoadOrGenCA()
	if err != nil {
		logrus.Errorf("error generating key and cert: %t", err)
	}
	err = LoadOrGenSignedMasterCert()
	if err != nil {
		logrus.Errorf("error generating key and cert: %t", err)
	}
}

// Register registers the service name with the associated target URL. All requests with the
// format ".../:service-name/*" are forwarded to the service via the target URL.
func (p *Proxy) Register(serviceID string, url *url.URL, proxyTCP bool, unauth bool) {
	if serviceID == "" {
		return
	}
	p.lock.Lock()
	defer p.lock.Unlock()

	p.syslog.Infof("registering service: %s (%v)", serviceID, url)
	p.services[serviceID] = &Service{
		URL:                  url,
		LastRequested:        time.Now(),
		ProxyTCP:             proxyTCP,
		AllowUnauthenticated: unauth,
	}
}

// Unregister removes the service from the proxy. All future requests until the service name is
// registered again will be responded with a 404 response. If the service is not registered with
// the proxy, the message is ignored.
func (p *Proxy) Unregister(serviceID string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	delete(p.services, serviceID)
}

// ClearProxy erases all services from the proxy in case any handlers are still active.
func (p *Proxy) ClearProxy() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.services = make(map[string]*Service)
}

// GetService returns the Service, if any, given the serviceID key.
func (p *Proxy) GetService(serviceID string) *Service {
	p.lock.Lock()
	defer p.lock.Unlock()
	service := p.services[serviceID]
	if service == nil {
		return nil
	}
	service.LastRequested = time.Now()

	// Make a copy to avoid callers mutating the object outside of this locked method.
	clone := service.Clone()
	return &clone
}

func (p *Proxy) NewProxyHandler(serviceID string) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Look up the service name in the url path.
		serviceName := c.Param(serviceID)
		service := p.GetService(serviceName)

		if service == nil {
			return echo.NewHTTPError(http.StatusNotFound,
				fmt.Sprintf("service not found: %s", serviceName))
		}

		if !service.AllowUnauthenticated {
			switch done, err := p.HTTPAuth(c); {
			case err != nil:
				return err
			case done:
				return nil
			}
		}

		// Set proxy headers and log them.
		req := c.Request()
		logHeaders := func() {
			headers := make(map[string]string)
			for name, values := range req.Header {
				headers[name] = strings.Join(values, ", ")
			}
			headers["X-Real-IP"] = c.RealIP()
			headers["X-Forwarded-Proto"] = c.Scheme()
			if c.IsWebSocket() {
				headers["X-Forwarded-For"] = c.RealIP()
			}
			log.Printf("Proxying request to: %s, Headers: %+v", service.URL, headers)
		}

		logHeaders() // Call to log headers

		// Proxy the request to the target host.
		var proxy http.Handler
		switch {
		case service.ProxyTCP:
			proxy = newSingleHostReverseTCPOverWebSocketProxy(c, service.URL)
			fmt.Println("Proxying TCP over WebSocket to:", service.URL)
		case c.IsWebSocket():
			proxy = newSingleHostReverseWebSocketProxy(c, service.URL)
			fmt.Println("Proxying WebSocket to:", service.URL)
		default:
			newProxy, err := setUpProxy(service.URL)
			if err != nil {
				return err
			}
			proxy = newProxy
			fmt.Println("Proxying HTTP to:", service.URL)
		}

		proxy.ServeHTTP(c.Response(), req)

		return nil
	}
}

func setUpProxy(serviceURL *url.URL) (*httputil.ReverseProxy, error) {
	// here
	proxy := httputil.NewSingleHostReverseProxy(serviceURL)
	if serviceURL.Scheme != https {
		return proxy, nil
	}
	keyBytes, certBytes, err := MasterKeyAndCert()
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, err
	}

	masterCaBytes, err := MasterCACert()
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(masterCaBytes)

	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = &tls.Config{
		RootCAs:            caCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, //nolint:gosec
		VerifyConnection:   VerifyMasterSigned,
	}
	proxy.Transport = transport

	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = serviceURL.Host
	}

	return proxy, nil
}

// Summaries returns a snapshot of the registered services.
func (p *Proxy) Summaries() map[string]Service {
	p.lock.RLock()
	defer p.lock.RUnlock()

	snapshot := make(map[string]Service)
	for id, service := range p.services {
		snapshot[id] = service.Clone()
	}
	return snapshot
}

// Summary returns a snapshot of a specific registered service.
func (p *Proxy) Summary(id string) (Service, bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	service, ok := p.services[id]
	if !ok {
		return Service{}, false
	}
	return service.Clone(), true
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
