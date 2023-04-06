package dispatcherrm

// Follow launcher jobs to completion and report status back to Determined.

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/mathx"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
)

//nolint:lll
const (
	pollLoopInterval     = time.Duration(10) * time.Second
	ignoredReporter      = "com.cray.analytics.capsules.dispatcher.shasta.ShastaDispatcher"
	errorLinesToRetrieve = 200
	errorLinesToDisplay  = 15
)

// A list of WARNING/ERROR level messages that we're interested in, because they contain
// the root cause of the error.  The last matching pattern is used.
var messagePatternsOfInterest = []*regexp.Regexp{
	// Remove the carrier prefix and "()" will contain just the Slurm error message.
	// The (?s) is a non-capturing option that allows . to match newlines.
	// This provides the additional SBATCH error context in the message
	// (?:x) is not capturing prefix for alternating word patterns.
	regexp.MustCompile("com.cray.analytics.capsules.carriers.hpc.\\S+" +
		" - (?:Slurm|Pbs|PBS) job is in a (?s)(.+)"),
	regexp.MustCompile("(?s)(Slurm job is in a .+)"),

	// Whatever matches what's inside the "()" will contain the root cause of the SLURM error.
	// The (?s) is a non-capturing option that allows . to match newlines.
	// This provides the additional SBATCH error context in the message
	regexp.MustCompile("(?:Slurm|Pbs|PBS) job process terminated with exit code \\d+:\n*(?s)(.+)"),
}

// containerInfo stores the data sent by the container in the
// "NotifyContainerRunning" message, so that we can keep track of which
// containers are running.
type containerInfo struct {
	nodeName string
}

// launcherJob describes a new launcher job, the progress of which we need to track.
type launcherJob struct {
	user                   string
	dispatcherID           string
	hpcJobID               string
	payloadName            string
	lastJobStatusCheckTime time.Time
	totalContainers        int
	runningContainers      map[int]containerInfo
	jobWasTerminated       bool
}

// launcherMonitor describes the monitoring of jobs created by the launcher.
type launcherMonitor struct {
	monitoredJobs         map[string]*launcherJob
	jobsToRemove          map[string]bool
	apiClient             *launcherAPIClient
	newLauncherJob        chan launcherJob
	removeLauncherJob     chan launcherJob
	checkLauncherJob      chan launcherJob
	schedulerTick         *time.Ticker
	mu                    sync.RWMutex
	processingWatchedJobs atomic.Bool
	rm                    *dispatcherResourceManager
}

func newDispatchWatcher(apiClient *launcherAPIClient) *launcherMonitor {
	return &launcherMonitor{
		monitoredJobs:     map[string]*launcherJob{},
		jobsToRemove:      map[string]bool{},
		apiClient:         apiClient,
		newLauncherJob:    make(chan launcherJob),
		removeLauncherJob: make(chan launcherJob),
		checkLauncherJob:  make(chan launcherJob),
		// Poll job status this often
		schedulerTick: time.NewTicker(pollLoopInterval),
	}
}

// monitorJob adds the specified job to the collection of jobs whose status is monitored.
// payload name may be empty (when reconnecting), monitor will retrieve if necessary.
func (m *launcherMonitor) monitorJob(user string, dispatchID string, payloadName string) {
	m.newLauncherJob <- launcherJob{
		user:                   user,
		dispatcherID:           dispatchID,
		payloadName:            payloadName,
		lastJobStatusCheckTime: time.Now(),
		totalContainers:        0,
		runningContainers:      make(map[int]containerInfo),
		jobWasTerminated:       false,
	}
}

// removeJob removes the specified job from the collection of jobs whose status is monitored.
func (m *launcherMonitor) removeJob(dispatchID string) {
	m.removeLauncherJob <- launcherJob{
		dispatcherID: dispatchID,
	}
}

// checkJob checks the status of the specified job from the collection of jobs whose status is
// being monitored.
func (m *launcherMonitor) checkJob(dispatchID string) {
	m.checkLauncherJob <- launcherJob{
		dispatcherID: dispatchID,
	}
}

// watch runs asynchronously as a go routine. It receives instructions as
// to what jobs to monitor, and when to monitor them, via channels.
func (m *launcherMonitor) watch(ctx *actor.Context) {
	// Indicates whether the "processWatchedJobs()" goroutine is already running,
	// so we don't run a second goroutine while the previous goroutine is still
	// running.
	m.processingWatchedJobs.Store(false)

	for {
		select {
		case msg := <-m.newLauncherJob:
			ctx.Log().Infof("Starting monitoring of %s", msg.dispatcherID)
			// Add job to collection of those being monitored.
			m.addJobToMonitoredJobs(&msg)

		case msg := <-m.removeLauncherJob:
			// Don't think the "removeLauncherJob" message is ever sent, but
			// leaving code here in case we had plans to use it at some point.
			ctx.Log().Infof("Received removeLauncherJob message for dispatchID %s", msg.dispatcherID)

			job, ok := m.getJobByDispatchID(msg.dispatcherID)

			if ok {
				_ = m.updateJobStatus(ctx, job)

				// Save the job to be removed in map. This job will deleted
				// later when processing watched jobs.
				m.markJobForRemoval(msg.dispatcherID)
			}

		case msg := <-m.checkLauncherJob:
			// Don't think the "checkLauncherJob" message is ever sent, but
			// leaving code here in case we had plans to use it at some point.
			job, ok := m.getJobByDispatchID(msg.dispatcherID)

			if ok {
				// Check the status of the given job.
				_ = m.updateJobStatus(ctx, job)
			}

		case <-m.schedulerTick.C:
			// Protect against running another "processWatchedJobs()" goroutine
			// while the previous one is still running. The "schedulerTick"
			// message is received every 10 seconds, as per the value we set
			// "pollLoopIntervalSecs". If we have a lot of jobs to monitor, it
			// is possible that "processWatchedJobs()" will still be querying
			// the launcher for job status when the next "schedulerTick" message
			// arrives 10 seconds later.
			//
			// We really don't need a mutex for testing and setting the
			// "processingWatchedJobs" boolean, because the "watch()" method
			// is single threaded, but the GO race detector, if enabled, might
			// complain if it thinks the "watch()" and the
			// "processWatchedJobs()" goroutine are trying to access the
			// "processingWatchedJobs" boolean variable concurrently.
			if m.processingWatchedJobs.CompareAndSwap(false, true) {
				// Run the "processWatchedJobs()" method as a goroutine to
				// prevent the "for-loop" in the "watch()" from hanging while
				// we're polling the job status of the monitored jobs.
				// The "processingWatchedJobs" boolean variable will be set
				// back to false by "processWatchedJobs()" when it is finished.
				go m.processWatchedJobs(ctx)
			} else {
				//nolint:lll
				ctx.Log().Debugf("Skipping calling the processWatchedJobs() goroutine, as the previous goroutine is still running")
			}
		}
	}
}

// This function filters out the noise from the error message, such that only the information that's
// useful to idenfify the root cause is shown in the master output.
// <p>
// When there's a job error, the launcher may send back too much information.
// <p>
// For example,
// <p>
// Capsule corujor/DAI-singularity-over-Slurm:unknown submitted for launch by user corujor
// Attempting to launch Payload DAI-task-container_exp-388-trial-365 ...
// Failed to launch payload DAI-task-container_exp-388-trial-365 ... Slurm job process terminated
// with exit code 1:
// sbatch: error: Batch job submission failed: Requested GRES option unsupported by configured
// SelectType plugin
// Failed to launch payload DAI-task-container_exp-388-trial-365 ... with any of the specified
// carriers
// Transitioned environment from state PENDING to FAILED
// Failed to launch capsule
// <p>
// Much of this information is of no value to the user and will only serve as noise.
// <p>
// In this example, the actual root cause of the failure is the line:
// <p>
// sbatch: error: Batch job submission failed: Requested GRES option unsupported by configured
// SelectType plugin
// <p>
// Therefore, we should only be returning the root cause and nothing else.
func filterOutSuperfluousMessages(allMessages []string) []string {
	// A list of messages that matched the pattern(s).
	messagesMatchingPattern := make([]string, 0)

	// The error messages that are returned from the launcher will be on multiple lines. Iterate
	// through all the lines of output.
	for _, msg := range allMessages {
		// Iterate through all the message patterns to see if the error message matches any of them.
		for _, messagePatternOfInterest := range messagePatternsOfInterest {
			// Does this error message line match any of the patterns we're looking for?
			matches := messagePatternOfInterest.FindAllStringSubmatch(msg, -1)

			// The 1st element (i.e., "matches[0][0]") contains the entire messasge that matched.
			// The 2nd element (i.e., "matches[0][1]") contains the substring we want.
			if len(matches) > 0 && len(matches[0]) >= 2 {
				messagesMatchingPattern = append(messagesMatchingPattern, matches[0][1])
			}
		}
	}

	return messagesMatchingPattern
}

func (m *launcherMonitor) notifyContainerRunning(
	ctx *actor.Context,
	dispatchID string,
	rank int32,
	numPeers int32,
	nodeName string,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.monitoredJobs[dispatchID]

	if !ok {
		ctx.Log().Errorf("Could not find dispatchID %s in the monitored jobs", dispatchID)
		return
	}

	job.totalContainers = int(numPeers)

	switch existingEntry, ok := job.runningContainers[int(rank)]; {
	case !ok:
		job.runningContainers[int(rank)] = containerInfo{nodeName: nodeName}

		ctx.Log().Infof("For dispatchID %s, %d out of %d containers running on nodes %s",
			dispatchID,
			len(job.runningContainers),
			job.totalContainers,
			getNodesThatAreRunningContainer(job))

	// Rank already existed in the map. This is not expected, as each container
	// should only send the notification that it's running only once.
	case existingEntry.nodeName != nodeName:
		// Two different containers running on two different nodes
		// reported the same rank number. That is unexpected, since
		// each container is supposed to have a unique rank.
		ctx.Log().Errorf("For dispatchID %s, received a notification that the container "+
			"is running for rank %d from two different nodes, %s and %s",
			dispatchID,
			rank,
			existingEntry.nodeName,
			nodeName)

	default:
		// A container reported its rank more than once. This is
		// unexpected, since each container should only send one
		// notification that it's running.  Unless, for some reason,
		// the Workload Manager (e.g., Slurm, PBS, etc), restarted
		// the job.
		ctx.Log().Warnf("For dispatchID %s, received multiple notifications that the "+
			"container is running for rank %d from node %s",
			dispatchID,
			rank,
			existingEntry.nodeName)
	}
}

// Returns a comma separated string containing the nodes that have notified
// the Determined master that the are running the container. Used for logging
// purposes only.
func getNodesThatAreRunningContainer(job *launcherJob) string {
	var sb strings.Builder

	for _, v := range job.runningContainers {
		if sb.Len() > 0 {
			sb.WriteString(",")
		}

		sb.WriteString(v.nodeName)
	}

	return sb.String()
}

// Returns true if all the containers have notified the Determined Master that
// they are running; false otherwise.
func (m *launcherMonitor) allContainersRunning(job *launcherJob) bool {
	// Since each container has a unique ID, the number of running containers
	// is the number of unique IDs in the "runningContainers" map.
	numContainersRunning := len(job.runningContainers)

	// Initially "numContainersRunning" and "job.totalContainers" will be
	// zero, until we receive our first notification from one of the
	// containers.  Therefore, in order to prevent "0 == 0" from falsely
	// returning true, make sure that at least one container has notified
	// the master, by adding "numContainersRunning > 0" so that the
	// value of "job.totalContainers" is set by the first container to
	// send a notification.
	if numContainersRunning > 0 && numContainersRunning == job.totalContainers {
		return true
	}

	return false
}

// Returns true if the job with the given dispatch ID is being monitored by
// the job watcher; false otherwise.
func (m *launcherMonitor) isJobBeingMonitored(dispatchID string) bool {
	// Obtain a read lock, so that "processedWatchedJobs()" doesn't manipulate
	// the "monitoredJobs" list while we're iterating through it. Because it's
	// a read lock, another thread will still be able to come in here and
	// iterate the "monitoredJobs" list.
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, job := range m.monitoredJobs {
		if job.dispatcherID == dispatchID {
			return true
		}
	}

	return false
}

// processWatchedJobs is called periodically to poll for the completion status
// of launched jobs. The exit status of any completed job is reported to Determined; such
// jobs are them removed from further consideration.
func (m *launcherMonitor) processWatchedJobs(
	ctx *actor.Context,
) {
	defer m.processingWatchedJobs.Store(false)

	var job *launcherJob
	var ok bool

	qStats := m.queuesFromCluster(ctx)

	sortedDispatchIDs := m.getDispatchIDsSortedByLastJobStatusCheckTime(m.monitoredJobs, ctx)

	// Loop through the jobs in the monitoredJobs map and update status accordingly
	for _, dispatchID := range sortedDispatchIDs {
		if m.isJobBeingRemoved(dispatchID) {
			m.removeJobFromMonitoredList(dispatchID, ctx)
			continue
		}

		if job, ok = m.getJobByDispatchID(dispatchID); !ok {
			ctx.Log().Warnf("dispatcher_monitor did not find job for dispatchID %s", dispatchID)
			continue
		}

		if m.obtainJobStateFromWlmQueueDetails(dispatchID, qStats, ctx, job) {
			continue // An optimization to avoid per-job use of updateJobStatus (below)
		}

		if removeJob := m.updateJobStatus(ctx, job); removeJob {
			m.removeJobFromMonitoredList(dispatchID, ctx)
			continue
		}

		m.updateLastJobStatusCheckTime(dispatchID)
	}

	// There are chances that jobsToRemove might still have some elements remaining.
	// These values are stale and can be removed safely.
	m.clearJobsToRemoveMap()
}

// obtainJobStateFromWlmQueueDetails gets the state of the specified dispatch from
// the supplied WLM queue details. If found the associated job state will be published.
func (m *launcherMonitor) obtainJobStateFromWlmQueueDetails(
	dispatchID string, qStats map[string]map[string]string,
	ctx *actor.Context, job *launcherJob,
) bool {
	hpcJobID, _ := m.rm.getHpcJobIDFromDispatchID(dispatchID)
	nativeState := qStats[hpcJobID]["state"]
	ctx.Log().WithField("dispatch-id", dispatchID).
		WithField("hpc-job-id", hpcJobID).
		WithField("native-state", nativeState).
		Debugf("job state from HPC queue stats")
	switch {
	case nativeState == "PD" || strings.ToLower(nativeState) == "pending":
		m.publishJobState(launcher.PENDING, job, ctx, dispatchID, hpcJobID)
		return true
	case nativeState == "R" || strings.ToLower(nativeState) == "running":
		m.publishJobState(launcher.RUNNING, job, ctx, dispatchID, hpcJobID)
		return true
	}
	return false
}

// queuesFromCluster fetches the latest job queue information from the cluster.
func (m *launcherMonitor) queuesFromCluster(ctx *actor.Context) map[string]map[string]string {
	result := map[string]map[string]string{}
	if len(m.monitoredJobs) == 0 {
		return result // Nothing to get of interest in this case
	}
	ctx.Log().Debugf("Fetching HPC queue state")

	dispatchInfo, r, err := m.apiClient.launchHPCQueueJob() //nolint:bodyclose
	if err != nil {
		m.apiClient.handleServiceQueryError(r, err)
		return result
	}
	dispatchID := dispatchInfo.GetDispatchId()
	owner := dispatchInfo.GetLaunchingUser()
	defer func() {
		_, _, err := m.apiClient.terminateDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			ctx.Log().WithError(err).Errorf("failed to terminate dispatchID {%s}", dispatchID)
			return
		}

		_, err = m.apiClient.deleteDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			ctx.Log().WithError(err).Errorf("failed to delete dispatchID {%s}", dispatchID)
			return
		}
	}()

	resp, _, err := m.apiClient.loadEnvironmentLog( //nolint:bodyclose
		owner, dispatchID, "slurm-queue-info")
	if err != nil {
		ctx.Log().WithError(err).Errorf("failed to retrieve HPC job queue details. response: {%v}", resp)
		return result
	}

	// Parse the carrier output file to a map of properties per job.
	resourcesBytes, err := io.ReadAll(resp)
	if err != nil {
		ctx.Log().WithError(err).Errorf("failed to read HPC job queue details")
		return result
	}
	if err = yaml.Unmarshal(resourcesBytes, &result); err != nil {
		ctx.Log().WithError(err).Errorf("failed to parse HPC job queue details")
		return result
	}
	ctx.Log().Debugf("HPC queue state done, size %d", len(result))
	return result
}

func (m *launcherMonitor) addJobToMonitoredJobs(job *launcherJob) {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	m.monitoredJobs[job.dispatcherID] = job
}

func (m *launcherMonitor) markJobForRemoval(dispatchID string) {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	m.jobsToRemove[dispatchID] = true
}

func (m *launcherMonitor) getJobByDispatchID(dispatchID string) (*launcherJob, bool) {
	// Obtain a read lock.
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.monitoredJobs[dispatchID]

	return job, ok
}

func (m *launcherMonitor) isJobBeingRemoved(dispatchID string) bool {
	// Obtain a read lock.
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.jobsToRemove[dispatchID]

	return ok
}

func (m *launcherMonitor) markJobAsTerminated(ctx *actor.Context, dispatchID string) {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.monitoredJobs[dispatchID]

	if ok {
		job.jobWasTerminated = true
	} else {
		ctx.Log().Tracef("Cannot mark job with dispatchID %s for termination, "+
			"because it is not found in the monitored jobs",
			dispatchID)
	}
}

func (m *launcherMonitor) removeJobFromMonitoredList(dispatchID string, ctx *actor.Context) {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx.Log().Infof("Stopping monitoring of %s", dispatchID)
	delete(m.monitoredJobs, dispatchID)
	delete(m.jobsToRemove, dispatchID)
}

func (m *launcherMonitor) updateLastJobStatusCheckTime(dispatchID string) {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	if job, ok := m.monitoredJobs[dispatchID]; ok {
		job.lastJobStatusCheckTime = time.Now()
	}
}

func (m *launcherMonitor) clearJobsToRemoveMap() {
	// Obtain a read/write lock.
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.jobsToRemove) > 0 {
		m.jobsToRemove = map[string]bool{}
	}
}

// Returns an array of dispatch IDs sorted by the time that the last
// status check was made to the launcher.
func (m *launcherMonitor) getDispatchIDsSortedByLastJobStatusCheckTime(
	monitoredJobs map[string]*launcherJob,
	ctx *actor.Context,
) []string {
	// Obtain a read lock.
	m.mu.RLock()
	defer m.mu.RUnlock()

	dispatchIDs := make([]string, 0, len(monitoredJobs))

	// Get the keys to the monitored jobs map. The key is the dispatch IDs.
	for key := range monitoredJobs {
		dispatchIDs = append(dispatchIDs, key)
	}

	// Sort by the time we last asked the launcher for job status.
	sort.SliceStable(dispatchIDs, func(i, j int) bool {
		a := monitoredJobs[dispatchIDs[i]].lastJobStatusCheckTime
		b := monitoredJobs[dispatchIDs[j]].lastJobStatusCheckTime

		return a.Before(b)
	})

	return dispatchIDs
}

func getJobID(additionalProperties map[string]interface{}) string {
	tagValue, ok := additionalProperties["job-id"]
	if !ok {
		return ""
	}

	// Ensure that the tag value is a string.
	typed, ok := tagValue.(string)
	if !ok {
		return ""
	}
	return typed
}

/*
Error logs may have large python stack traces, if we have a
misconfiguration error, prune messages before that to make the error
clearer for the user.  Limit the number of lines to errorLinesToDisplay.
In debug/trace show increased number of lines.
*/
func minimizeErrorLog(messages []string) []string {
	// By default we show limited lines, on debug/trace levels show more
	linesToShow := errorLinesToDisplay
	if logrus.GetLevel() == logrus.DebugLevel {
		linesToShow = 100
	} else if logrus.GetLevel() == logrus.TraceLevel {
		linesToShow = 1000
	}

	for i, line := range messages {
		if strings.Contains(line, "Failed to download model definition from master.") {
			// Return up to linesToShow after message
			return messages[i:mathx.Min(len(messages), i+linesToShow)]
		}
	}

	// Return the last linesToShow of the log
	return messages[mathx.Clamp(0, len(messages)-linesToShow, len(messages)):]
}

/*
Processes the job state and sends DispatchStateChange on each call.
updateJobStatus returns true when the job has finished or no longer exists,
and monitoring should stop.
*/
func (m *launcherMonitor) updateJobStatus(ctx *actor.Context, job *launcherJob) bool {
	removeJob := false
	dispatchID := job.dispatcherID
	owner := job.user

	ctx.Log().WithField("dispatch-id", dispatchID).Debug("Checking status of launcher job")

	resp, ok := m.getDispatchStatus(ctx, owner, dispatchID)

	// Dispatch was not found.
	if !ok {
		if job.jobWasTerminated {
			ctx.Log().WithField("dispatch-id", dispatchID).Infof("The job was canceled")

			ctx.Tell(ctx.Self(), DispatchExited{
				DispatchID: dispatchID,
				ExitCode:   -1,
				Message:    "Job was canceled",
			})
		}

		return true
	}

	if _, gotResponse := resp.GetStateOk(); !gotResponse {
		return false
	}

	if exitStatus, exitMessages, ok := calculateJobExitStatus(resp); ok {
		// Try to filter out messages that offer no value to the user, leaving only the
		// message that identifies the root cause of the error.
		filteredMessages := filterOutSuperfluousMessages(exitMessages)

		// If we were able to filter out the root cause of the error, then replace the messages
		// that we're going to propagate upstream with the filtered messages. Otherwise, we'll
		// simply propagate the original messages upstream, which will likely include a lot of
		// noise that the user doesn't care about, but we have no choice in this case.
		if len(filteredMessages) > 0 {
			exitMessages = filteredMessages
		}

		if exitStatus != 0 && len(exitMessages) == 0 {
			// If we have no messages, it may be a connection failure from the container
			// and we will have no logs to assist in diagnosis, so insert the last
			// few lines of the error log into the failure message.
			exitMessages, _ = m.getTaskLogsFromDispatcher(ctx, job, "error.log", errorLinesToRetrieve)
			exitMessages = minimizeErrorLog(exitMessages)
		}

		//nolint:lll
		ctx.Log().Debugf("For dispatchID %s, sending job termination status to DAI: exitCode=%d, messages=%s",
			dispatchID,
			exitStatus,
			exitMessages)

		ctx.Tell(ctx.Self(), DispatchExited{
			DispatchID: dispatchID,
			ExitCode:   exitStatus,
			Message:    strings.Join(exitMessages, "\n") + "\n",
		})

		// If status sent, remove this job form the monitored list as we are done.
		// I tried this approach:
		//
		// monitoredLauncherJobs.Remove(e)
		//
		// but it seemed to cause other jobs in the list to be not processed, so instead
		// keep list of jobs to be removed for later.
		removeJob = true
	} else {
		// Copy the HPC job ID, which is the ID that Slurm/PBS generate
		// to track the jobs they run.
		job.hpcJobID = getJobID(resp.GetAdditionalPropertiesField())

		// From the launcher's perspective, a job is running when the Workload
		// Manager (e.g., Slurm, PBS, etc) starts the job. However, from the
		// Determined perspective, a job is not running until the all
		// containers that are part of the job are being run by
		// Singularity/Podman.  If the image does not already exist on the
		// compute node, then Singularity/Podman will first need to pull down
		// the image from the Internet before they can run the container.
		// While there is at least one image being pulled (i.e., one container
		// that's not running), then Determined will report a state of
		// "Pulling".  When all the containers are running, then Determined
		// will report a state of "Running".
		m.publishJobState(
			*resp.State, job, ctx, dispatchID, getJobID(resp.GetAdditionalPropertiesField()))
	}
	return removeJob
}

// publishJobState publishes the state of the specified job to the rest of the system.
func (m *launcherMonitor) publishJobState(
	notifyState launcher.DispatchState,
	job *launcherJob, ctx *actor.Context, dispatchID string, hpcJobID string,
) {
	isPullingImage := notifyState == launcher.RUNNING && !m.allContainersRunning(job)

	ctx.Log().Debugf("For dispatchID %s, sending DAI a job state of %s (pulling=%t)",
		dispatchID,
		notifyState,
		isPullingImage)

	ctx.Tell(ctx.Self(), DispatchStateChange{
		DispatchID:     dispatchID,
		State:          notifyState,
		IsPullingImage: isPullingImage,
		HPCJobID:       job.hpcJobID,
	})
}

/*
Return the DispatchInfo (possibly invalid/empty), caller must check fields for
existence (e.g. GetStateOk()) before use.   A second value will be true if the
dispatch exists, or false if it no longer exists (i.e. 404).
*/
func (m *launcherMonitor) getDispatchStatus(
	ctx *actor.Context, owner string,
	dispatchID string,
) (dispatchInfo launcher.DispatchInfo, dispatchFound bool) {
	resp, r, err := m.apiClient.MonitoringApi.
		GetEnvironmentStatus(m.apiClient.withAuth(context.TODO()), owner, dispatchID).
		Refresh(true).
		Execute() //nolint:bodyclose
	if err != nil {
		// This may happen if the job is canceled before the launcher creates
		// the environment files containing status. Wouldn't expect this to
		// happen now that we're starting the job using "Launch()" API, instead
		// of "LaunchAsync()", but we're still seeing it.
		if r != nil && r.StatusCode == 404 {
			//nolint:lll
			ctx.Log().WithField("dispatch-id", dispatchID).
				Infof("The job status could not be obtained because the launcher returned HTTP code 404")

			// No details, but we know dispatch does not exist
			return launcher.DispatchInfo{}, false
		}

		if r != nil && (r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden) {
			//nolint:lll
			ctx.Log().WithField("dispatch-id", dispatchID).
				WithError(err).
				Infof("Failed to `GetEnvironmentStatus` due to error {%v}. "+
					"Reloaded the auth token file {%s}. If this error persists, restart "+
					"the launcher service followed by a restart of the determined-master service.",
					err, m.apiClient.authFile)
			m.apiClient.reloadAuthToken()
		} else {
			ctx.Log().WithField("dispatch-id", dispatchID).
				WithError(err).
				Infof("An error occurred when calling `GetEnvironmentStatus`:\n%v", r)
		}
		// No details, but job may still exist
		return launcher.DispatchInfo{}, true
	}

	ctx.Log().Debugf("DispatchID %s state: %s", dispatchID, *resp.State)
	// We have details, need to process them
	return resp, true
}

type exitCode int

// calculateJobExitStatus determines  an exit status for the specified job. If the job is not
// in a terminal state, there is no exit status (and monitoring continues).
// If in a terminal state, also return the job messages.
func calculateJobExitStatus(
	resp launcher.DispatchInfo,
) (exitCode, []string, bool) {
	state, ok := resp.GetStateOk()
	if ok {
		// TODO(HAL-2813): Track and send more of these state changes with sendStatusToDetermined.
		switch *state {
		case "TERMINATED": // User-initiated termination complete
			return 1, getJobExitMessages(resp), true
		case "FAILED":
			// exit status TBD -- use -1 to skip printing incorrect (exit code 1)
			return -1, getJobExitMessages(resp), true
		case "COMPLETED": // Normal completion
			return 0, getJobExitMessages(resp), true
		default:
			return 0, nil, false
		}
	}
	return 0, nil, false
}

// getJobExitMessages returns the job messages from the event array (if any).
func getJobExitMessages(resp launcher.DispatchInfo) []string {
	var result []string
	for _, event := range resp.GetEvents() {
		if *event.Reporter == "com.cray.analytics.capsules.dispatcher.shasta.ShastaDispatcher" {
			// Ignore general dispatcher messages, only want carrier messages
			continue
		}

		// Only need messages that help diagnose the failure
		if *event.Level == "WARNING" || *event.Level == "ERROR" {
			result = append(result, *event.Message)
		}
	}
	return result
}

// getTaskLogsFromDispatcher is used to read the logs direct from the dispatcher.
// It is used on job failure when no messages have been relayed from the job
// as a last-chance to provide context for the failure.
// The baseLogName string is error.log, output.log, submission.log (etc), the
// prefix is taken from the job payload name.
func (m *launcherMonitor) getTaskLogsFromDispatcher(
	ctx *actor.Context, job *launcherJob, baseLogName string, linesToShow int,
) ([]string, error) {
	dispatchID := job.dispatcherID

	// The logRange expression can be used to limit the size of the logs returned.
	// For example "lines=-30" is the last 30 lines of the file.
	logRange := fmt.Sprintf("lines=-%d", linesToShow)

	// If we re-connect to a running job, we've lost the payload name
	// So in the rare case that we fail to start and need to display
	// the log file content, read the payload name from the launcher.
	if len(job.payloadName) == 0 {
		manifest, resp, err := m.apiClient.MonitoringApi.
			GetEnvironmentDetails(m.apiClient.withAuth(context.TODO()), job.user, dispatchID).
			Execute() //nolint:bodyclose
		if err != nil {
			ctx.Log().WithError(err).Warnf(
				"For dispatchID %s, unable to access environment details, response {%v}",
				dispatchID, resp)
			return []string{}, err
		}
		for _, p := range *manifest.Payloads {
			job.payloadName = *p.Name
		}
	}

	// Compose the file name
	logFileName := fmt.Sprintf("%s-%s", job.payloadName, baseLogName)

	logFile, httpResponse, err := m.apiClient.MonitoringApi.
		LoadEnvironmentLog(m.apiClient.withAuth(context.TODO()), job.user, dispatchID, logFileName).
		Range_(logRange).
		Execute() //nolint:bodyclose
	if err != nil {
		ctx.Log().WithError(err).Warnf("For dispatchID %s, unable to access %s, response {%v}",
			dispatchID, logFileName, httpResponse)
		return []string{}, err
	}

	contentLength := 0
	// Content-Length is not always set sometimes only Content-Range
	contentLengthStr := httpResponse.Header.Get("Content-Length")
	if len(contentLengthStr) == 0 {
		// No specified length header just read the whole http response
		var fileStat fs.FileInfo
		fileStat, err = logFile.Stat()
		if err != nil {
			ctx.Log().Errorf("For dispatchID %s, logFile.Stat() failed: %s", dispatchID, err.Error())
			return []string{}, nil
		}
		contentLength = int(fileStat.Size())
	} else {
		contentLength, err = strconv.Atoi(contentLengthStr)
		if err != nil {
			ctx.Log().Errorf("For dispatchID %s, atoi(Content-Length) failed: %s", dispatchID, err.Error())
			return []string{}, err
		}
		if contentLength == 0 {
			ctx.Log().Debugf("For dispatchID %s, no content yet for %s", dispatchID, logFileName)
			return []string{}, nil
		}
	}

	buffer := make([]byte, contentLength)
	bytesRead, err := logFile.Read(buffer)
	if err != nil || bytesRead != contentLength {
		ctx.Log().WithError(err).Errorf(
			"For dispatcID %s, failed to read full http response: read %d != contentLength %d",
			dispatchID, bytesRead, contentLength)
		return nil, err
	}
	return strings.Split(string(buffer), "\n"), nil
}

/*
Return true if the specified dispatch is in a non-terminal (still running) state.
*/
func (m *launcherMonitor) isDispatchInProgress(
	ctx *actor.Context,
	owner string,
	dispatchID string,
) bool {
	resp, ok := m.getDispatchStatus(ctx, owner, dispatchID)

	// Dispatch was not found.
	if !ok {
		// We know it does not exist so not in progress
		return false
	}
	_, _, exited := calculateJobExitStatus(resp)
	return !exited
}
