package dispatcherrm

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	semver "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
)

// Blank user runs as launcher-configured user.
const blankImpersonatedUser = ""

// One time activity to create a manifest using SlurmResources carrier.
// This manifest is used on demand to retrieve details regarding HPC resources
// e.g., nodes, GPUs etc.
var hpcResourcesManifest = createSlurmResourcesManifest()

// One time activity to create a manifest using Slurm/PBSQueue carrier.
// This manifest is used on demand to retrieve details regarding
// pending/running HPC jobs.
var hpcQueueManifest = createHpcQueueManifest()

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

func (c *launcherAPIClient) getVersion(ctx context.Context) (v *semver.Version, err error) {
	defer recordAPITiming("get_version")()
	defer recordAPIErr("get_version")(err)

	resp, _, err := c.InfoApi.
		GetServerVersion(c.withAuth(ctx)).
		Execute() //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("getting launcher version: %w", err)
	}

	version, err := semver.NewVersion(strings.TrimSuffix(resp, "-SNAPSHOT"))
	if err != nil {
		return nil, fmt.Errorf("parsing semver version %s: %w", resp, err)
	}
	return version, nil
}

func (c *launcherAPIClient) launchHPCResourcesJob() (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	defer recordAPITiming("launch_hpc_resources_job")()
	defer recordAPIErr("launch_hpc_resources_job")(err)

	// Launch the HPC Resources manifest. Launch() method will ensure
	// the manifest is in the RUNNING state on successful completion.
	return c.LaunchApi.
		Launch(c.withAuth(context.TODO())).
		Manifest(hpcResourcesManifest).
		Impersonate(blankImpersonatedUser).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) launchHPCQueueJob() (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	defer recordAPITiming("launch_hpc_queue_job")()
	defer recordAPIErr("launch_hpc_queue_job")(err)

	// Launch the HPC Resources manifest. Launch() method will ensure
	// the manifest is in the RUNNING state on successful completion.
	return c.LaunchApi.
		Launch(c.withAuth(context.TODO())).
		Manifest(hpcQueueManifest).
		Impersonate(blankImpersonatedUser).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) terminateDispatch(
	owner string,
	id string,
) (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	defer recordAPITiming("terminate")()
	defer recordAPIErr("terminate")(err)

	info, resp, err = c.RunningApi.
		TerminateRunning(c.withAuth(context.TODO()), owner, id).
		Force(true).Execute() //nolint:bodyclose
	switch {
	case err != nil && resp != nil && resp.StatusCode == 404:
		c.log.Debugf("call to terminate missing dispatch %s: %s", id, err)
	case err != nil:
		return launcher.DispatchInfo{}, nil, fmt.Errorf("terminating dispatch %s: %w", id, err)
	default:
		c.log.Debugf("terminated dispatch %s", id)
	}
	return info, resp, nil
}

func (c *launcherAPIClient) deleteDispatch(owner, id string) (resp *http.Response, err error) {
	defer recordAPITiming("delete_env")()
	defer recordAPIErr("delete_env")(err)
	c.log.Debugf("deleting environment with DispatchID %s", id)

	resp, err = c.MonitoringApi.
		DeleteEnvironment(c.withAuth(context.TODO()), owner, id).
		Execute() //nolint:bodyclose
	switch {
	case err != nil && resp != nil && resp.StatusCode == 404:
		c.log.Debugf("try to delete environment with DispatchID %s but it is gone", id)
	case err != nil:
		return nil, fmt.Errorf("removing environment for Dispatch ID %s: %w", id, err)
	default:
		c.log.Debugf("deleted environment with DispatchID %s", id)
	}
	return resp, nil
}

func (c *launcherAPIClient) loadEnvironmentLog(owner, id, logFileName string) (
	log *os.File,
	resp *http.Response,
	err error,
) {
	defer recordAPITiming("load_log")()
	defer recordAPIErr("load_log")(err)

	log, resp, err = c.MonitoringApi.
		LoadEnvironmentLog(c.withAuth(context.TODO()), owner, id, logFileName).
		Execute() //nolint:bodyclose
	if err != nil {
		return nil, nil, fmt.Errorf(c.handleLauncherError(
			resp, "Failed to retrieve HPC Resource details", err))
	}
	return log, resp, nil
}

// handleLauncherError provides common error handling for REST API calls
// to the launcher in support of RM operations.
func (c *launcherAPIClient) handleLauncherError(r *http.Response,
	errPrefix string, err error,
) string {
	var msg string
	if r != nil {
		if r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden {
			msg = fmt.Sprintf("Failed to communicate with launcher due to error: "+
				"{%v}. Reloaded the auth token file {%s}. If this error persists, restart "+
				"the launcher service followed by a restart of the determined-master service.",
				err, c.authFile)
			c.reloadAuthToken()
		} else {
			msg = fmt.Sprintf("%s. Response: %v. ", errPrefix, r.Body)
		}
	} else {
		msg = fmt.Sprintf("Failed to communicate with launcher due to error: "+
			"{%v}. Verify that the launcher service is up and reachable.", err)
	}
	return msg
}

// CreateSlurmResourcesManifest creates a Manifest for SlurmResources Carrier.
// This Manifest is used to retrieve information about resources available on the HPC system.
func createSlurmResourcesManifest() launcher.Manifest {
	payload := launcher.NewPayloadWithDefaults()
	payload.SetName("DAI-HPC-Resources")
	payload.SetId("com.cray.analytics.capsules.hpc.resources")
	payload.SetVersion("latest")
	payload.SetCarriers([]string{slurmResourcesCarrier, pbsResourcesCarrier})

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("interactive")
	payload.SetLaunchParameters(*launchParameters)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName("DAI-HPC-Resources")

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return manifest
}

// CreateHpcQueueManifest creates a Manifest for Slurm/PBSQueue Carrier.
// This Manifest is used to retrieve information about pending/running jobs.
func createHpcQueueManifest() launcher.Manifest {
	payload := launcher.NewPayloadWithDefaults()
	payload.SetName("DAI-HPC-Queues")
	payload.SetId("com.cray.analytics.capsules.hpc.queue")
	payload.SetVersion("latest")
	payload.SetCarriers([]string{
		"com.cray.analytics.capsules.carriers.hpc.slurm.SlurmQueue",
		"com.cray.analytics.capsules.carriers.hpc.pbs.PbsQueue",
	})

	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("batch")
	payload.SetLaunchParameters(*launchParameters)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName("DAI-HPC-Queues")

	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return manifest
}
