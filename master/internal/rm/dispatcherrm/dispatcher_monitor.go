package dispatcherrm

// Follow launcher jobs to completion and report status back to Determined.

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/pkg/mathx"
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
	"github.com/determined-ai/determined/proto/pkg/jobv1"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
)

//nolint:lll
const (
	pollLoopInterval         = time.Duration(10) * time.Second
	ignoredReporter          = "com.cray.analytics.capsules.dispatcher.shasta.ShastaDispatcher"
	errorLinesToRetrieve     = 500
	errorLinesToDisplay      = 15
	slotsValueNotAvailable   = 0
	notAvailable             = "Not Available"
	keyMissingErrorMessage   = "The '%s' key is missing from the job details for external job id %s"
	invalidValueErrorMessage = "Invalid '%s' value '%s' for external job id %s: %s"
)

// Keys for External Job Details Map.
const (
	jobState      = "state"
	jobName       = "name"
	jobPartition  = "partition"
	jobSubmitTime = "submitTime"
	jobUserName   = "userName"
	jobGPUSlots   = "gpuSlots"
	jobCPUSlots   = "cpuSlots"
)

// SlurmPrologReasonCode is the Slurm Prolog Reason Code.
const SlurmPrologReasonCode = "Prolog"

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
	jobPendingReasonCode   string
	lastJobStatusCheckTime time.Time
	totalContainers        int
	runningContainers      mapx.Map[int, containerInfo]
	jobWasTerminated       bool
	launchInProgress       bool // Launch proceeding concurrent with monitoring
}

// launcherMonitorEvent is a union of all events emitted by the launcherMonitor.
type launcherMonitorEvent interface {
	launcherMonitorEvent()
}

func (dispatchExpLogMessage) launcherMonitorEvent() {}
func (DispatchExited) launcherMonitorEvent()        {}
func (DispatchStateChange) launcherMonitorEvent()   {}

// launcherMonitor describes the monitoring of jobs created by the launcher.
type launcherMonitor struct {
	// system dependencies
	syslog    *logrus.Entry
	apiClient *launcherAPIClient

	// configuration details.
	outbox chan<- launcherMonitorEvent

	// immutable state.
	schedulerTick *time.Ticker

	// mutable internal state. requires a lock if not threadsafe already.
	monitoredJobs         mapx.Map[string, *launcherJob]
	jobsToRemove          mapx.Map[string, struct{}]
	newLauncherJob        chan *launcherJob
	removeLauncherJob     chan *launcherJob
	checkLauncherJob      chan *launcherJob
	processingWatchedJobs atomic.Bool
	dispatchIDToHPCJobID  *mapx.Map[string, string]
	externalJobs          mapx.Map[string, map[string]string]
}

// dispatchLastJobStatusCheckTime is used to sort the dispatches by the time
// that the job status was last checked.
type dispatchLastJobStatusCheckTime struct {
	dispatchID             string
	lastJobStatusCheckTime time.Time
}

func newDispatchWatcher(
	apiClient *launcherAPIClient,
	dispatchIDToHPCJobID *mapx.Map[string, string],
	outbox chan<- launcherMonitorEvent,
) *launcherMonitor {
	return &launcherMonitor{
		syslog: logrus.WithField("component", "dispatchwatcher"),
		outbox: outbox,

		monitoredJobs:     mapx.New[string, *launcherJob](),
		jobsToRemove:      mapx.New[string, struct{}](),
		apiClient:         apiClient,
		externalJobs:      mapx.New[string, map[string]string](),
		newLauncherJob:    make(chan *launcherJob),
		removeLauncherJob: make(chan *launcherJob),
		checkLauncherJob:  make(chan *launcherJob),
		// Poll job status this often
		schedulerTick:        time.NewTicker(pollLoopInterval),
		dispatchIDToHPCJobID: dispatchIDToHPCJobID,
	}
}

// monitorJob adds the specified job to the collection of jobs whose status is monitored.
// payload name may be empty (when reconnecting), monitor will retrieve if necessary.
// When launchPending is true, a later call to notifyJobLaunched is required once the
// launcher has initiated the launch and the dispatchID will be valid for status checks.
func (m *launcherMonitor) monitorJob(
	user string, dispatchID string, payloadName string, launchPending bool,
) {
	m.newLauncherJob <- &launcherJob{
		user:                   user,
		dispatcherID:           dispatchID,
		payloadName:            payloadName,
		jobPendingReasonCode:   "",
		lastJobStatusCheckTime: time.Now(),
		totalContainers:        0,
		runningContainers:      mapx.New[int, containerInfo](),
		jobWasTerminated:       false,
		launchInProgress:       launchPending,
	}
}

// When monitoring commences with parameter isLaunching:true specified, 404/NOT_FOUND
// status returns are ignored for this job until notifyJobLaunched is called to enable
// normal monitoring.
func (m *launcherMonitor) notifyJobLaunched(dispatchID string) {
	if job, ok := m.monitoredJobs.Load(dispatchID); ok {
		job.launchInProgress = false
		return
	}

	m.syslog.Errorf("Could not find dispatchID %s to mark as launched", dispatchID)
}

// removeJob removes the specified job from the collection of jobs whose status is monitored.
func (m *launcherMonitor) removeJob(dispatchID string) {
	m.removeLauncherJob <- &launcherJob{
		dispatcherID: dispatchID,
	}
}

// checkJob checks the status of the specified job from the collection of jobs whose status is
// being monitored.
func (m *launcherMonitor) checkJob(dispatchID string) {
	m.checkLauncherJob <- &launcherJob{
		dispatcherID: dispatchID,
	}
}

// watch runs asynchronously as a go routine. It receives instructions as
// to what jobs to monitor, and when to monitor them, via channels.
func (m *launcherMonitor) watch() {
	defer close(m.outbox)

	// Indicates whether the "processWatchedJobs()" goroutine is already running,
	// so we don't run a second goroutine while the previous goroutine is still
	// running.
	m.processingWatchedJobs.Store(false)

	for {
		select {
		case msg := <-m.newLauncherJob:
			m.syslog.Infof("Starting monitoring of %s", msg.dispatcherID)
			// Add job to collection of those being monitored.
			m.addJobToMonitoredJobs(msg)

		case msg := <-m.removeLauncherJob:
			// Don't think the "removeLauncherJob" message is ever sent, but
			// leaving code here in case we had plans to use it at some point.
			m.syslog.Infof("Received removeLauncherJob message for dispatchID %s", msg.dispatcherID)

			job, ok := m.getJobByDispatchID(msg.dispatcherID)

			if ok {
				_ = m.updateJobStatus(job)

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
				_ = m.updateJobStatus(job)
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
				go m.processWatchedJobs()
			} else {
				//nolint:lll
				m.syslog.Debugf("Skipping calling the processWatchedJobs() goroutine, as the previous goroutine is still running")
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

// Receive notification that one of the ranks of the job has started
// execution.  It generates experiment log messages as the ranks
// start, and updates job info so we know when all have started.
func (m *launcherMonitor) notifyContainerRunning(
	dispatchID string,
	rank int32,
	numPeers int32,
	nodeName string,
) {
	job, ok := m.monitoredJobs.Load(dispatchID)

	if !ok {
		m.syslog.Errorf("Could not find dispatchID %s in the monitored jobs", dispatchID)
		return
	}

	// The "notifyContainerRunning()" function is called from a message handler,
	// so we do not expect there to be two instances of the function running
	// concurrently and trying to set these variables at the same time. Even if
	// there was, both instances would be setting the variables to the exact
	// same values.
	job.launchInProgress = false
	job.totalContainers = int(numPeers)

	switch existingEntry, ok := job.runningContainers.Load(int(rank)); {
	case !ok:
		job.runningContainers.Store(int(rank), containerInfo{nodeName: nodeName})

		startedMsg := fmt.Sprintf("%d out of %d containers running on nodes %s",
			job.runningContainers.Len(),
			job.totalContainers,
			getNodesThatAreRunningContainer(job))

		// Show message in the experiment log.
		if job.totalContainers > 1 {
			m.outbox <- dispatchExpLogMessage{
				DispatchID: dispatchID,
				Message:    startedMsg,
			}
		}

		// Show message in the master log. Comes in very handy during triaging
		// to know not only how many containers are running, but also which
		// nodes they are running on. Sometimes jobs will fail on certain nodes,
		// due to differences in configuration on those nodes, and it helps to
		// be able to quickly find out which nodes the job was running on so we
		// can check the configuration on those nodes.
		m.syslog.WithField("dispatch-id", dispatchID).
			Info(startedMsg)

	// Rank already existed in the map. This is not expected, as each container
	// should only send the notification that it's running only once.
	case existingEntry.nodeName != nodeName:
		// Two different containers running on two different nodes
		// reported the same rank number. That is unexpected, since
		// each container is supposed to have a unique rank.
		m.syslog.Errorf("For dispatchID %s, received a notification that the container "+
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
		m.syslog.Warnf("For dispatchID %s, received multiple notifications that the "+
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

	job.runningContainers.WithLock(func(inmap map[int]containerInfo) {
		for _, v := range inmap {
			if sb.Len() > 0 {
				sb.WriteString(",")
			}

			sb.WriteString(v.nodeName)
		}
	})

	return sb.String()
}

// Returns true if all the containers have notified the Determined Master that
// they are running; false otherwise.
func (m *launcherMonitor) allContainersRunning(job *launcherJob) bool {
	// Since each container has a unique ID, the number of running containers
	// is the number of unique IDs in the "runningContainers" map.
	numContainersRunning := job.runningContainers.Len()

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
	_, ok := m.monitoredJobs.Load(dispatchID)

	return ok
}

// processWatchedJobs is called periodically to poll for the completion status
// of launched jobs. The exit status of any completed job is reported to Determined; such
// jobs are them removed from further consideration.
func (m *launcherMonitor) processWatchedJobs() {
	defer m.processingWatchedJobs.Store(false)

	var job *launcherJob
	var ok bool

	qStats := m.queuesFromCluster()

	sortedDispatchIDs := m.getDispatchIDsSortedByLastJobStatusCheckTime()

	// Loop through the jobs in the monitoredJobs map and update status accordingly
	for _, dispatchID := range sortedDispatchIDs {
		if job, ok = m.getJobByDispatchID(dispatchID); !ok {
			m.syslog.Warnf("dispatcher_monitor did not find job for dispatchID %s", dispatchID)
			continue
		}

		// In each case below, remove the processed job details from the qStats map. This will
		// leave only the external jobs in the qStats map after all DispatchIDs are processed.
		// The qStats map with just the external jobs will be processed in the next for loop.

		if m.isJobBeingRemoved(dispatchID) {
			m.removeJobFromMonitoredList(dispatchID)
			delete(qStats, job.hpcJobID)
			continue
		}

		if m.obtainJobStateFromWlmQueueDetails(dispatchID, qStats, job) {
			m.updateLastJobStatusCheckTime(dispatchID)
			delete(qStats, job.hpcJobID)
			continue // An optimization to avoid per-job use of updateJobStatus (below)
		}

		if removeJob := m.updateJobStatus(job); removeJob {
			m.removeJobFromMonitoredList(dispatchID)
			delete(qStats, job.hpcJobID)
			continue
		}

		m.updateLastJobStatusCheckTime(dispatchID)
		delete(qStats, job.hpcJobID)
	}

	// There are chances that jobsToRemove might still have some elements remaining.
	// These values are stale and can be removed safely.
	m.clearJobsToRemoveMap()

	// Refresh the external jobs map.
	m.externalJobs.Clear()
	for qJobID, qJobDetails := range qStats {
		m.externalJobs.Store(qJobID, qJobDetails)
	}
}

// obtainJobStateFromWlmQueueDetails gets the state of the specified dispatch from
// the supplied WLM queue details. If found the associated job state will be published.
func (m *launcherMonitor) obtainJobStateFromWlmQueueDetails(
	dispatchID string,
	qStats map[string]map[string]string,
	job *launcherJob,
) bool {
	hpcJobID, _ := m.dispatchIDToHPCJobID.Load(dispatchID)
	nativeState := qStats[hpcJobID]["state"]

	// Provides information as to why a job is in a particular state.
	reasonCode := qStats[hpcJobID]["reasonCode"]
	reasonDesc := qStats[hpcJobID]["reasonDesc"]

	m.syslog.WithField("dispatch-id", dispatchID).
		WithField("hpc-job-id", hpcJobID).
		WithField("native-state", nativeState).
		WithField("state-reason-code", reasonCode).
		Debugf("job state from HPC queue stats")

	switch {
	case nativeState == "PD" || strings.ToLower(nativeState) == "pending":
		m.publishJobState(launcher.PENDING, job, dispatchID, hpcJobID)

		m.processReasonCodeForPendingJobs(dispatchID, reasonCode, reasonDesc, job)

		return true
	case nativeState == "R" || strings.ToLower(nativeState) == "running":
		m.publishJobState(launcher.RUNNING, job, dispatchID, hpcJobID)

		// The launcher treats a Slurm "CG" (completing) state, as a "running"
		// state, so the reason codes for jobs that are in the "completing"
		// state are also handled in this function.
		m.processReasonCodeForRunningJobs(dispatchID, reasonCode, reasonDesc, job)

		return true
	}

	job.jobPendingReasonCode = ""

	return false
}

// Provides additional information in the experiment log based on the
// reason code.
func (m *launcherMonitor) processReasonCodeForPendingJobs(
	dispatchID string,
	reasonCode string,
	reasonDesc string,
	job *launcherJob,
) {
	// Only log a message for this reason code if we did not already log
	// it. This avoids logging the same message over and over again.
	if reasonCode != job.jobPendingReasonCode {
		m.outbox <- dispatchExpLogMessage{
			DispatchID: dispatchID,
			Message:    "HPC job waiting to be scheduled: " + reasonDesc,
		}

		// Avoid repeated logging to the experiment log by storing the
		// last reason code for which we logged a message. If the
		// reason code changes, then that will cause a new message to
		// be logged.
		job.jobPendingReasonCode = reasonCode
	}
}

// Provides additional information in the experiment log based on the
// reason code.
func (m *launcherMonitor) processReasonCodeForRunningJobs(
	dispatchID string,
	reasonCode string,
	reasonDesc string,
	job *launcherJob,
) {
	// Only log a message for this reason code if we did not already log
	// it. This avoids logging the same message over and over again.
	if reasonCode != job.jobPendingReasonCode {
		// Is the prolog script running?
		if reasonCode == SlurmPrologReasonCode {
			// Although Slurm reports the job as running, the container runtime
			// (e.g., "singularity", "podman", "enroot") doesn't get executed
			// until the prolog script, if any, finishes. While the prolog
			// script is running, the user will see a state of "Pulling", even
			// if the image file already exists on the local filesystem and is
			// not actually being pulled from an external repository. Therefore,
			// write a message to the experiment log to let the user know that
			// the reason why their job shows a state of "Pulling" is because
			// the container runtime is not running yet, and won't run, until
			// the prolog script finishes.
			m.outbox <- dispatchExpLogMessage{
				DispatchID: dispatchID,
				Message:    "HPC job waiting on pre-condition: " + reasonDesc,
			}
		} else if job.jobPendingReasonCode == SlurmPrologReasonCode {
			// The reason code is no longer "Prolog", so let the user know that,
			// if the job state is "Pulling", it's no longer because the prolog
			// script is running, and it's because the image is actually getting
			// pulled.
			m.outbox <- dispatchExpLogMessage{
				DispatchID: dispatchID,
				Message:    "HPC job pre-condition cleared.",
			}
		}

		// Avoid repeated logging to the experiment log by storing
		// the last reason code for which we logged a message. If the
		// reason code changes, then that will cause a new message to
		// be logged.
		job.jobPendingReasonCode = reasonCode
	}
}

// queuesFromCluster fetches the latest job queue information from the cluster.
func (m *launcherMonitor) queuesFromCluster() map[string]map[string]string {
	result := map[string]map[string]string{}
	if m.monitoredJobs.Len() == 0 {
		return result // Nothing to get of interest in this case
	}
	m.syslog.Debugf("Fetching HPC queue state")

	dispatchInfo, r, err := m.apiClient.launchHPCQueueJob() //nolint:bodyclose
	if err != nil {
		m.syslog.Errorf(m.apiClient.handleLauncherError(r,
			"Failed to retrieve HPC job queue from launcher", err))
		return result
	}
	dispatchID := dispatchInfo.GetDispatchId()
	owner := dispatchInfo.GetLaunchingUser()
	defer func() {
		_, _, err := m.apiClient.terminateDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			m.syslog.WithError(err).Errorf("failed to terminate dispatchID {%s}", dispatchID)
			return
		}

		_, err = m.apiClient.deleteDispatch(owner, dispatchID) //nolint:bodyclose
		if err != nil {
			m.syslog.WithError(err).Errorf("failed to delete dispatchID {%s}", dispatchID)
			return
		}
	}()

	resp, _, err := m.apiClient.loadEnvironmentLog( //nolint:bodyclose
		owner, dispatchID, "slurm-queue-info")
	if err != nil {
		m.syslog.WithError(err).Errorf("failed to retrieve HPC job queue details. response: {%v}", resp)
		return result
	}
	resourcesBytes := []byte(resp)
	if err = yaml.Unmarshal(resourcesBytes, &result); err != nil {
		m.syslog.WithError(err).Errorf("failed to parse HPC job queue details")
		return result
	}
	m.syslog.Debugf("HPC queue state done, size %d", len(result))
	return result
}

func (m *launcherMonitor) addJobToMonitoredJobs(job *launcherJob) {
	m.monitoredJobs.Store(job.dispatcherID, job)
}

func (m *launcherMonitor) markJobForRemoval(dispatchID string) {
	m.jobsToRemove.Store(dispatchID, struct{}{})
}

func (m *launcherMonitor) getJobByDispatchID(dispatchID string) (*launcherJob, bool) {
	job, ok := m.monitoredJobs.Load(dispatchID)

	return job, ok
}

func (m *launcherMonitor) isJobBeingRemoved(dispatchID string) bool {
	_, ok := m.jobsToRemove.Load(dispatchID)

	return ok
}

func (m *launcherMonitor) markJobAsTerminated(dispatchID string) {
	job, ok := m.monitoredJobs.Load(dispatchID)

	if ok {
		job.jobWasTerminated = true
	} else {
		m.syslog.Tracef("Cannot mark job with dispatchID %s for termination, "+
			"because it is not found in the monitored jobs",
			dispatchID)
	}
}

func (m *launcherMonitor) removeJobFromMonitoredList(dispatchID string) {
	m.syslog.Infof("Stopping monitoring of %s", dispatchID)
	m.monitoredJobs.Delete(dispatchID)
	m.jobsToRemove.Delete(dispatchID)
}

func (m *launcherMonitor) updateLastJobStatusCheckTime(dispatchID string) {
	m.monitoredJobs.WithLock(func(inmap map[string]*launcherJob) {
		if job, ok := inmap[dispatchID]; ok {
			job.lastJobStatusCheckTime = time.Now()
		}
	})
}

func (m *launcherMonitor) clearJobsToRemoveMap() {
	m.jobsToRemove.WithLock(func(inmap map[string]struct{}) {
		for k := range inmap {
			delete(inmap, k)
		}
	})
}

// Returns an array of dispatch IDs sorted by the time that the last
// status check was made to the launcher.
func (m *launcherMonitor) getDispatchIDsSortedByLastJobStatusCheckTime() []string {
	var dispatches []dispatchLastJobStatusCheckTime

	// With the lock held, copy the dispatch ID and the last job status check
	// time into an array.
	m.monitoredJobs.WithLock(func(inmap map[string]*launcherJob) {
		dispatches = make([]dispatchLastJobStatusCheckTime, 0, len(inmap))

		for k, v := range inmap {
			dispatchWithTime := dispatchLastJobStatusCheckTime{
				dispatchID:             k,
				lastJobStatusCheckTime: v.lastJobStatusCheckTime,
			}

			dispatches = append(dispatches, dispatchWithTime)
		}
	})

	// With the lock no longer held, sort by the last job status check time.
	sort.SliceStable(dispatches, func(i, j int) bool {
		a := dispatches[i].lastJobStatusCheckTime
		b := dispatches[j].lastJobStatusCheckTime

		return a.Before(b)
	})

	// Create an array to store only the dispatch IDs.
	dispatchIDs := make([]string, len(dispatches))

	// Copy the dispatch IDs into an array that is sorted by the last job
	// status check time.
	for i, dispatch := range dispatches {
		dispatchIDs[i] = dispatch.dispatchID
	}

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

	var output []string
	output = append(output, "Job terminated. Most recent output from the stderr log is:")
	for i, line := range messages {
		if strings.Contains(line, "Failed to download model definition from master.") {
			// Return up to linesToShow after message
			return append(output, messages[i:mathx.Min(len(messages), i+linesToShow)]...)
		}
	}

	// Return the last linesToShow of the log
	return append(output, messages[mathx.Clamp(0, len(messages)-linesToShow, len(messages)):]...)
}

/*
Processes the job state and sends DispatchStateChange on each call.
updateJobStatus returns true when the job has finished or no longer exists,
and monitoring should stop.
*/
func (m *launcherMonitor) updateJobStatus(job *launcherJob) bool {
	removeJob := false
	dispatchID := job.dispatcherID
	owner := job.user

	m.syslog.WithField("dispatch-id", dispatchID).Debug("Checking status of launcher job")

	resp, ok := m.getDispatchStatus(owner, dispatchID, job.launchInProgress)

	// Dispatch was not found.
	if !ok {
		missingDispatchMsg := "The job was canceled"
		if job.jobWasTerminated {
			m.syslog.WithField("dispatch-id", dispatchID).Info(missingDispatchMsg)
		} else {
			missingDispatchMsg = "The job was lost"
			m.syslog.WithField("dispatch-id", dispatchID).Error(missingDispatchMsg)
		}

		m.outbox <- DispatchExited{
			DispatchID: dispatchID,
			ExitCode:   -1,
			Message:    missingDispatchMsg,
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

		// Insert the last few lines of the error log into the failure message.
		if exitStatus != 0 {
			var errMessages []string
			errMessages, _ = m.getTaskLogsFromDispatcher(job, "error.log", errorLinesToRetrieve)
			if m.allContainersRunning(job) {
				// If all containers are running, then show limited output.
				// Otherwise show all collected lines which would include the
				// container downloading logs.
				errMessages = minimizeErrorLog(errMessages)
			}
			exitMessages = append(exitMessages, errMessages...)
		}

		//nolint:lll
		m.syslog.Debugf("For dispatchID %s, sending job termination status to DAI: exitCode=%d, messages=%s",
			dispatchID,
			exitStatus,
			exitMessages)

		m.outbox <- DispatchExited{
			DispatchID: dispatchID,
			ExitCode:   exitStatus,
			Message:    strings.Join(exitMessages, "\n") + "\n",
		}

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
		m.publishJobState(*resp.State, job, dispatchID, getJobID(resp.GetAdditionalPropertiesField()))
	}
	return removeJob
}

// publishJobState publishes the state of the specified job to the rest of the system.
func (m *launcherMonitor) publishJobState(
	notifyState launcher.DispatchState,
	job *launcherJob,
	dispatchID string,
	hpcJobID string,
) {
	isPullingImage := notifyState == launcher.RUNNING && !m.allContainersRunning(job)

	m.syslog.Debugf("For dispatchID %s, sending DAI a job state of %s (pulling=%t)",
		dispatchID,
		notifyState,
		isPullingImage)

	m.outbox <- DispatchStateChange{
		DispatchID:     dispatchID,
		State:          notifyState,
		IsPullingImage: isPullingImage,
		HPCJobID:       job.hpcJobID,
	}
}

/*
Return the DispatchInfo (possibly invalid/empty), caller must check fields for
existence (e.g. GetStateOk()) before use.   A second value will be true if the
dispatch exists, or false if it no longer exists (i.e. 404).
*/
func (m *launcherMonitor) getDispatchStatus(
	owner string,
	dispatchID string,
	ignoreNotFound bool,
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
			m.syslog.WithField("dispatch-id", dispatchID).
				Infof("The job status could not be obtained because the launcher returned HTTP code 404")

			// No details, but we know dispatch does not exist (false unless ignoreNotFound)
			return launcher.DispatchInfo{}, ignoreNotFound
		}

		if r != nil && (r.StatusCode == http.StatusUnauthorized ||
			r.StatusCode == http.StatusForbidden) {
			//nolint:lll
			m.syslog.WithField("dispatch-id", dispatchID).
				WithError(err).
				Infof("Failed to `GetEnvironmentStatus` due to error {%v}. "+
					"Reloaded the auth token file {%s}. If this error persists, restart "+
					"the launcher service followed by a restart of the determined-master service.",
					err, m.apiClient.authFile)
			m.apiClient.reloadAuthToken()
		} else {
			m.syslog.WithField("dispatch-id", dispatchID).
				WithError(err).
				Infof("An error occurred when calling `GetEnvironmentStatus`:\n%v", r)
		}
		// No details, but job may still exist
		return launcher.DispatchInfo{}, true
	}

	m.syslog.Debugf("DispatchID %s state: %s", dispatchID, *resp.State)
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
		case "MISSING": // Unexpected job state, assuming job is terminated
			return -1,
				append(getJobExitMessages(resp), "HPC launcher job lost. Assuming job terminated."),
				true
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
		if event.Reporter == nil || event.Message == nil || event.Level == nil {
			continue
		}
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
	job *launcherJob,
	baseLogName string,
	linesToShow int,
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
			m.syslog.WithError(err).Warnf(
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
		m.syslog.WithError(err).Warnf("For dispatchID %s, unable to access %s, response {%v}",
			dispatchID, logFileName, httpResponse)
		return []string{}, err
	}
	return strings.Split(logFile, "\n"), nil
}

/*
Return true if the specified dispatch is in a non-terminal (still running) state.
Or launch is still in progress.
*/
func (m *launcherMonitor) isDispatchInProgress(owner string, dispatchID string) bool {
	ignoreNotFound := false
	if job, ok := m.getJobByDispatchID(dispatchID); ok {
		ignoreNotFound = job.launchInProgress
	}

	resp, ok := m.getDispatchStatus(owner, dispatchID, ignoreNotFound)

	// Dispatch was not found.
	if !ok {
		// We know it does not exist so not in progress
		return false
	}
	_, _, exited := calculateJobExitStatus(resp)
	return !exited
}

// getRequestedSlots retrieves the value for Requested Slots using the provided job details map.
// It looks for GPU slots value first and CPU slots value later. It will return the first valid
// value. In case of errors, warning messages are logged and default value of zero is returned.
func (m *launcherMonitor) getRequestedSlots(qJobID string, qJobDetails map[string]string) int32 {
	var requestedSlots int64 = slotsValueNotAvailable
	var err error
	keys := []string{jobGPUSlots, jobCPUSlots}
	// Look for GPU slots first and CPU slots later.
	// Return the first valid value.
	for _, key := range keys {
		if len(qJobDetails[key]) == 0 {
			m.syslog.Warnf(keyMissingErrorMessage, key, qJobID)
		} else {
			requestedSlots, err = strconv.ParseInt(qJobDetails[key], 10, 32)
			if err != nil {
				m.syslog.Warnf(invalidValueErrorMessage, key, qJobDetails[key], qJobID, err.Error())
				requestedSlots = slotsValueNotAvailable
			}
			if requestedSlots != slotsValueNotAvailable {
				break
			}
		}
	}
	return int32(requestedSlots)
}

// getSubmitTime retrieves the value for Job Submission Time using the provided job details map.
// In case of errors, warning messages are logged and default value of zero timestamp is returned.
func (m *launcherMonitor) getSubmitTime(qJobID string, qJobDetails map[string]string) time.Time {
	submitTime := time.Time{}
	var err error
	if len(qJobDetails[jobSubmitTime]) == 0 {
		m.syslog.Warnf(keyMissingErrorMessage, jobSubmitTime, qJobID)
	} else {
		submitTime, err = time.Parse(time.RFC3339, qJobDetails[jobSubmitTime])
		if err != nil {
			m.syslog.Warnf(invalidValueErrorMessage, jobSubmitTime, qJobDetails[jobSubmitTime],
				qJobID, err.Error())
			submitTime = time.Time{}
		}
	}
	return submitTime
}

// convertHpcStatus converts the HPC WLM status into a DeterminedAI job status.
func (m *launcherMonitor) convertHpcStatus(slurmStatus string) jobv1.State {
	switch slurmStatus {
	case "PENDING":
		return jobv1.State_STATE_QUEUED
	case "RUNNING":
		return jobv1.State_STATE_SCHEDULED
	default:
		return jobv1.State_STATE_UNSPECIFIED
	}
}

// getJobSummary retrieves the value for Job State using the provided job details map. It returns
// jobv1.JobSummary value which contains member State indicating the job state. In case of errors,
// warning messages are logged and default value of JobSummary with State set to
// jobv1.State_STATE_UNSPECIFIED is returned.
func (m *launcherMonitor) getJobSummary(
	qJobID string, qJobDetails map[string]string,
) *jobv1.JobSummary {
	jobSummary := &jobv1.JobSummary{
		State: jobv1.State_STATE_UNSPECIFIED,
	}
	if len(qJobDetails[jobState]) == 0 {
		m.syslog.Warnf(keyMissingErrorMessage, jobState, qJobID)
	} else {
		jobSummary = &jobv1.JobSummary{
			State: m.convertHpcStatus(qJobDetails[jobState]),
		}
	}
	return jobSummary
}

// getValueFromJobDetails retrieves the string values for the provided key from the given job
// details map. In case of errors, warning messages are logged and default value 'Not Available'
// is returned.
func (m *launcherMonitor) getValueFromJobDetails(
	qJobID string, qJobDetails map[string]string, key string,
) string {
	value := notAvailable
	if len(qJobDetails[key]) == 0 {
		m.syslog.Warnf(keyMissingErrorMessage, key, qJobID)
	} else {
		value = qJobDetails[key]
	}
	return value
}

// convertExternalJobs returns a new jobv1.Job instance created using the provided external job
// details.
func (m *launcherMonitor) convertExternalJobs(
	qJobID string, qJobDetails map[string]string,
) *jobv1.Job {
	if len(qJobID) == 0 {
		m.syslog.Warn("Cannot convert the external job. Job ID is missing.")
		return nil
	}
	if qJobDetails == nil {
		m.syslog.Warn("Cannot convert the external job. Job Details are missing.")
		return nil
	}

	requestedSlots := m.getRequestedSlots(qJobID, qJobDetails)
	submitTime := m.getSubmitTime(qJobID, qJobDetails)
	jobSummary := m.getJobSummary(qJobID, qJobDetails)
	name := m.getValueFromJobDetails(qJobID, qJobDetails, jobName)
	userName := m.getValueFromJobDetails(qJobID, qJobDetails, jobUserName)
	partition := m.getValueFromJobDetails(qJobID, qJobDetails, jobPartition)

	var allocatedSlots int32 = slotsValueNotAvailable
	if jobSummary.State == jobv1.State_STATE_SCHEDULED {
		allocatedSlots = requestedSlots
	}

	job := jobv1.Job{
		JobId:          qJobID,
		Type:           jobv1.Type_TYPE_EXTERNAL,
		Summary:        jobSummary,
		Name:           name,
		Username:       userName,
		SubmissionTime: timestamppb.New(submitTime),
		AllocatedSlots: allocatedSlots,
		RequestedSlots: requestedSlots,
		ResourcePool:   partition,
	}
	return &job
}

// fetchExternalJobs returns a list of external jobs in the jobv1.Job format.
// If a resource pool name is provided the list is filtered to return only the jobs related to the
// provided resource pool.
func (m *launcherMonitor) fetchExternalJobs(resourcePool string) []*jobv1.Job {
	externalJobs := []*jobv1.Job{}
	m.externalJobs.WithLock(func(inmap map[string]map[string]string) {
		for jobID, jobDetails := range inmap {
			if len(resourcePool) == 0 || jobDetails["partition"] == resourcePool {
				convertedExternalJob := m.convertExternalJobs(jobID, jobDetails)
				if convertedExternalJob != nil {
					externalJobs = append(externalJobs, convertedExternalJob)
				}
			}
		}
	})
	return externalJobs
}
