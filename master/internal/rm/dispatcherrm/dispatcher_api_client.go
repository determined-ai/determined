package dispatcherrm

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/sirupsen/logrus"
	"github.hpe.com/hpe/hpc-ard-launcher-go/launcher"

	"github.com/determined-ai/determined/master/internal/config"
)

// Blank user runs as launcher-configured user.
const (
	blankImpersonatedUser = ""
	resourceQueryName     = "DAI-HPC-Resources"
	queueQueryName        = "DAI-HPC-Queues"
)

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
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: cfg.Security.TLS.SkipVerify, //nolint:gosec
		}

		client := cleanhttp.DefaultClient()
		client.Transport = transport

		lcfg.HTTPClient = client
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

func (c *launcherAPIClient) getVersion(
	ctx context.Context,
	launcherAPILogger *logrus.Entry,
) (v *semver.Version, err error) {
	launcherAPILogger = launcherAPILogger.WithField("api-name", "getVersion")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
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

func (c *launcherAPIClient) launchDispatcherJob(
	manifest *launcher.Manifest,
	impersonatedUser string,
	allocationID string,
	launcherAPILogger *logrus.Entry,
) (dispatchInfo launcher.DispatchInfo, response *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", allocationID).
		WithField("api-name", "launchDispatcherJob")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("launch_dispatcher_job")()
	defer recordAPIErr("launch_dispatcher_job")(err)

	/*
	 * "Launch()" waits until the job has been submitted to the Workload manager
	 * (i.e., "Slurm" or "PBS) before returning, thereby guaranteeing that the
	 * launcher has created the environment files/directories and that we have
	 * the HPC job ID.
	 *
	 * The "manifest" describes the job to be launched and includes any environment
	 * variables, mount points, etc., that are needed by the job.
	 *
	 * The "impersonatedUser" is the user that we want to run the job as on the cluster.
	 * Of course, that user must be known to the cluster as either a local Linux user
	 * (e.g. "/etc/passwd"), LDAP, or some other authentication mechanism.
	 */
	return c.LaunchApi.
		Launch(c.withAuth(context.TODO())).
		Manifest(*manifest).
		Impersonate(impersonatedUser).
		DispatchId(allocationID).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) getEnvironmentStatus(
	owner string,
	dispatchID string,
	launcherAPILogger *logrus.Entry,
) (dispatchInfo launcher.DispatchInfo, response *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).
		WithField("api-name", "getEnvironmentStatus")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("get_environment_status")()
	defer recordAPIErr("get_environment_status")(err)

	return c.MonitoringApi.
		GetEnvironmentStatus(c.withAuth(context.TODO()), owner, dispatchID).
		Refresh(true).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) getEnvironmentDetails(
	owner string,
	dispatchID string,
	launcherAPILogger *logrus.Entry,
) (manifest launcher.Manifest, response *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).
		WithField("api-name", "getEnvironmentDetails")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("get_environment_details")()
	defer recordAPIErr("get_environment_details")(err)

	return c.MonitoringApi.
		GetEnvironmentDetails(c.withAuth(context.TODO()), owner, dispatchID).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) launchHPCResourcesJob(launcherAPILogger *logrus.Entry) (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	launcherAPILogger = launcherAPILogger.WithField("api-name", "launchHPCResourcesJob")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
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

func (c *launcherAPIClient) launchHPCQueueJob(launcherAPILogger *logrus.Entry) (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	launcherAPILogger = launcherAPILogger.WithField("api-name", "launchHPCQueueJob")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
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

func (c *launcherAPIClient) listAllTerminated(
	launcherAPILogger *logrus.Entry,
) (dispatchInfo map[string][]launcher.DispatchInfo, response *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("api-name", "listAllTerminated")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("list_all_terminated")()
	defer recordAPIErr("list_all_terminated")(err)

	return c.TerminatedApi.
		ListAllTerminated(c.withAuth(context.TODO())).
		EventLimit(0).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) listAllRunning(
	launcherAPILogger *logrus.Entry,
) (dispatchInfo map[string][]launcher.DispatchInfo, response *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("api-name", "listAllRunning")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("list_all_running")()
	defer recordAPIErr("list_all_running")(err)

	return c.RunningApi.
		ListAllRunning(c.withAuth(context.TODO())).
		EventLimit(0).
		Execute() //nolint:bodyclose
}

func (c *launcherAPIClient) terminateDispatch(
	owner string,
	dispatchID string,
	launcherAPILogger *logrus.Entry,
) (
	info launcher.DispatchInfo,
	resp *http.Response,
	err error,
) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).
		WithField("api-name", "terminateDispatch")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("terminate")()
	defer recordAPIErr("terminate")(err)

	info, resp, err = c.RunningApi.
		TerminateRunning(c.withAuth(context.TODO()), owner, dispatchID).
		Force(true).Execute() //nolint:bodyclose
	switch {
	case err != nil && resp != nil && resp.StatusCode == 404:
		launcherAPILogger.WithError(err).Debug("attempt to terminate dispatch but it is gone")
	case err != nil:
		return launcher.DispatchInfo{}, nil, fmt.Errorf("terminating dispatch %s: %w", dispatchID, err)
	default:
		launcherAPILogger.Debug("terminated dispatch")
	}
	return info, resp, nil
}

func (c *launcherAPIClient) deleteDispatch(
	owner,
	dispatchID string,
	launcherAPILogger *logrus.Entry,
) (resp *http.Response, err error) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).
		WithField("api-name", "deleteDispatch")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("delete_env")()
	defer recordAPIErr("delete_env")(err)

	launcherAPILogger.Debug("deleting environment")

	resp, err = c.MonitoringApi.
		DeleteEnvironment(c.withAuth(context.TODO()), owner, dispatchID).
		Execute() //nolint:bodyclose
	switch {
	case err != nil && resp != nil && resp.StatusCode == 404:
		launcherAPILogger.Debug("try to delete environment but it is gone")
	case err != nil:
		return nil, fmt.Errorf("removing environment for Dispatch ID %s: %w", dispatchID, err)
	default:
		launcherAPILogger.Debug("deleted environment")
	}
	return resp, nil
}

func (c *launcherAPIClient) loadEnvironmentLog(
	owner string,
	dispatchID string,
	logFileName string,
	launcherAPILogger *logrus.Entry,
) (data string, resp *http.Response, err error,
) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).WithField("api-name", "loadEnvironmentLog")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("load_environment_log")()
	defer recordAPIErr("load_environment_log")(err)

	data, resp, err = c.MonitoringApi.
		LoadEnvironmentLog(c.withAuth(context.TODO()), owner, dispatchID, logFileName).
		Execute() //nolint:bodyclose
	if err != nil {
		return data, nil, fmt.Errorf(c.handleLauncherError(
			resp, "Failed to retrieve HPC Resource details", err))
	}
	return data, resp, nil
}

func (c *launcherAPIClient) loadEnvironmentLogWithRange(
	owner string,
	dispatchID string,
	logFileName string,
	logRange string,
	launcherAPILogger *logrus.Entry) (
	data string, httpResponse *http.Response, err error,
) {
	launcherAPILogger = launcherAPILogger.WithField("dispatch-id", dispatchID).
		WithField("api-name", "loadEnvironmentLogWithRange")

	defer c.logExcessiveAPIResponseTimes(launcherAPILogger)()
	defer recordAPITiming("launch_environment_log_with_range")()
	defer recordAPIErr("launch_environment_log_with_range")(err)

	return c.MonitoringApi.
		LoadEnvironmentLog(c.withAuth(context.TODO()), owner, dispatchID, logFileName).
		Range_(logRange).
		Execute() //nolint:bodyclose
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
	payload.SetName(resourceQueryName)
	payload.SetId("com.cray.analytics.capsules.hpc.resources")
	payload.SetVersion("latest")
	payload.SetCarriers([]string{slurmResourcesCarrier, pbsResourcesCarrier})

	// Create payload launch parameters
	launchParameters := launcher.NewLaunchParameters()
	launchParameters.SetMode("interactive")
	payload.SetLaunchParameters(*launchParameters)

	clientMetadata := launcher.NewClientMetadataWithDefaults()
	clientMetadata.SetName(resourceQueryName)

	// Create & populate the manifest
	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return manifest
}

// CreateHpcQueueManifest creates a Manifest for Slurm/PBSQueue Carrier.
// This Manifest is used to retrieve information about pending/running jobs.
func createHpcQueueManifest() launcher.Manifest {
	payload := launcher.NewPayloadWithDefaults()
	payload.SetName(queueQueryName)
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
	clientMetadata.SetName(queueQueryName)

	manifest := *launcher.NewManifest("v1", *clientMetadata)
	manifest.SetPayloads([]launcher.Payload{*payload})

	return manifest
}

// If we have a BadRequest/InternalServerError with a details
// message in the response body, return it after appling our
// filterOutSuperfluousMessages cleanup method; otherwise return an
// empty string ("").
func extractDetailsFromResponse(resp *http.Response, err error) string {
	if resp == nil {
		return err.Error()
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusInternalServerError {
		openAPIErr, ok := err.(launcher.GenericOpenAPIError)
		if !ok {
			return err.Error()
		}
		// Unmarshal the error body into a struct to access detail.
		var errorBody struct {
			Detail string `json:"detail"`
		}
		if parseErr := json.Unmarshal(openAPIErr.Body(), &errorBody); parseErr != nil {
			return err.Error()
		}

		messages := filterOutSuperfluousMessages([]string{errorBody.Detail})
		if len(messages) > 0 {
			return strings.Join(messages, "\n")
		}
		return errorBody.Detail
	}
	return err.Error()
}

func (c *launcherAPIClient) logExcessiveAPIResponseTimes(launcherAPILogger *logrus.Entry) func() {
	// The time that the launcher API call was made, so we can track how long
	// the API call is taking to return.
	startTime := time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-time.After(10 * time.Second):
				launcherAPILogger.Warnf("after %.2fs, API call has still not completed",
					time.Since(startTime).Seconds())
			case <-ctx.Done():
				elapsed := time.Since(startTime)

				// Only log a message if it took longer than 10 seconds for the API
				// call to complete to avoid filling up the logs.
				if elapsed.Seconds() >= 10 {
					launcherAPILogger.Infof("after %.2fs, API call has completed",
						elapsed.Seconds())
				}

				return
			}
		}
	}()

	// When "logExcessiveAPIResponseTimes()" is called from a "defer", this is
	// the real function that gets deferred until the calling function returns.
	// The "cancel()" will cause the "ctx->Done()" in the goroutine above to
	// return a value, which will result in the goroutine terminating.
	return func() {
		cancel()
		wg.Wait()
	}
}
