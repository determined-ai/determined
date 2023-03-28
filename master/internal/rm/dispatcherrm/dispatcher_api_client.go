package dispatcherrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
)

type launcherAPIClient struct {
	*launcher.APIClient

	log      *logrus.Entry
	mu       sync.RWMutex
	auth     string
	authFile string
}

func newLauncherAPIClient(cfg *config.DispatcherResourceManagerConfig) (*launcherAPIClient, error) {
	log := logrus.WithField("component", "launcher-api-client")

	lcfg := launcher.NewConfiguration()
	lcfg.Host = fmt.Sprintf("%s:%d", cfg.LauncherHost, cfg.LauncherPort)
	lcfg.Scheme = cfg.LauncherProtocol // "http" or "https"
	if cfg.Security != nil {
		lcfg.HTTPClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: cfg.Security.TLS.SkipVerify, //nolint:gosec
				},
			},
		}
	}

	c := &launcherAPIClient{
		log:       log,
		APIClient: launcher.NewAPIClient(lcfg),
		authFile:  cfg.LauncherAuthFile,
	}

	err := c.loadAuthToken()
	if err != nil {
		return nil, fmt.Errorf("initial setup: %w", err)
	}

	return c, nil
}

// Return a context with launcher API auth added.
func (c *launcherAPIClient) withAuth(ctx context.Context) context.Context {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return context.WithValue(ctx, launcher.ContextAccessToken, c.auth)
}

func (c *launcherAPIClient) loadAuthToken() error {
	if len(c.authFile) > 0 {
		auth, err := os.ReadFile(c.authFile)
		if err != nil {
			return fmt.Errorf(
				"configuration resource_manager.auth_file (%s) not readable: %w",
				c.authFile, err,
			)
		}

		c.mu.Lock()
		defer c.mu.Unlock()
		c.auth = string(auth)
	}
	return nil
}

func (c *launcherAPIClient) reloadAuthToken() {
	err := c.loadAuthToken()
	if err != nil {
		c.log.WithError(err).Errorf("reloading auth token from %s", c.authFile)
		return
	}
}

// handleServiceQueryError provides common error handling for REST API calls
// to the launcher in support of RM operations.
func (c *launcherAPIClient) handleServiceQueryError(r *http.Response, err error) {
	if r != nil {
		if r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden {
			c.log.Errorf("Failed to communicate with launcher due to error: "+
				"{%v}. Reloaded the auth token file {%s}. If this error persists, restart "+
				"the launcher service followed by a restart of the determined-master service.",
				err, c.authFile)
			c.reloadAuthToken()
		} else {
			c.log.Errorf("Failed to retrieve HPC resources or queue data from launcher due to error: "+
				"{%v}, response: {%v}. ", err, r.Body)
		}
	} else {
		c.log.Errorf("Failed to communicate with launcher due to error: "+
			"{%v}. Verify that the launcher service is up and reachable.", err)
	}
}
