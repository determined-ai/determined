package resourcemanagers

// Follow launcher jobs to completion and report status back to Determined.

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
)

const (
	pollLoopIntervalSecs       = 10
	minItemPollingIntervalSecs = pollLoopIntervalSecs
)

// A list of WARNING/ERROR level messages that we're interested in, because they contain
// the root cause of the error.  The last matching pattern is used.
var messagePatternsOfInterest = []*regexp.Regexp{
	// Remove the carrier prefix and "()" will contain just the Slurm error message.
	// The (?s) is a non-capturing option that allows . to match newlines.
	// This provides the additional SBATCH error context in the message
	regexp.MustCompile("com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm" +
		" - Slurm job is in a (?s)(.+)"),

	// Whatever matches what's inside the "()" will contain the root cause of the SLURM error.
	// The (?s) is a non-capturing option that allows . to match newlines.
	// This provides the additional SBATCH error context in the message
	regexp.MustCompile("Slurm job process terminated with exit code \\d+:\n*(?s)(.+)"),
}

// launcherJob describes a new launcher job, the progress of which we need to track.
type launcherJob struct {
	user         string
	dispatcherID string
	payloadName  string
	timestamp    time.Time
}

// launcherMonitor describes the monitoring of jobs created by the launcher.
type launcherMonitor struct {
	monitoredJobs     map[string]launcherJob
	jobsToRemove      map[string]bool
	apiClient         *launcher.APIClient
	newLauncherJob    chan launcherJob
	removeLauncherJob chan launcherJob
	checkLauncherJob  chan launcherJob
	schedulerTick     *time.Ticker
	authToken         string
	mu                sync.RWMutex
}

// newDispatchWatcher initiates the process of monitoring the progress of launched jobs.
func newDispatchWatcher(apiClient *launcher.APIClient, authToken string) *launcherMonitor {
	return &launcherMonitor{
		monitoredJobs:     map[string]launcherJob{},
		jobsToRemove:      map[string]bool{},
		apiClient:         apiClient,
		newLauncherJob:    make(chan launcherJob),
		removeLauncherJob: make(chan launcherJob),
		checkLauncherJob:  make(chan launcherJob),
		// Poll job status this often
		schedulerTick: time.NewTicker(time.Second * pollLoopIntervalSecs),
		authToken:     authToken,
	}
}

// monitorJob adds the specified job to the collection of jobs whose status is monitored.
func (m *launcherMonitor) monitorJob(user string, dispatchID string, payloadName string) {
	m.newLauncherJob <- launcherJob{
		user:         user,
		dispatcherID: dispatchID,
		payloadName:  payloadName,
		timestamp:    time.Now(),
	}
}

// removeJob removes the specified job from the collection of jobs whose status is monitored.
func (m *launcherMonitor) removeJob(dispatchID string) {
	m.removeLauncherJob <- launcherJob{
		dispatcherID: dispatchID,
	}
}

// Return a starting context for the API client call that includes the authToken
// (may be empty if disabled).
func (m *launcherMonitor) authContext(ctx *actor.Context) context.Context {
	return context.WithValue(context.Background(), launcher.ContextAccessToken, m.authToken)
}

// watch runs asynchronously as a go routine. It receives instructions as
// to what jobs to monitor, and when to monitor them, via channels.
func (m *launcherMonitor) watch(ctx *actor.Context) {
	for {
		select {
		case msg := <-m.newLauncherJob:
			ctx.Log().Infof("Starting monitoring of %s", msg.dispatcherID)
			// Add job to collection of those being monitored.
			m.monitoredJobs[msg.dispatcherID] = msg

		case msg := <-m.removeLauncherJob:
			// Save the job to be removed in map. This job will deleted later when processing watched jobs.
			_ = m.updateJobStatus(ctx, m.monitoredJobs[msg.dispatcherID])
			m.jobsToRemove[msg.dispatcherID] = true

		case msg := <-m.checkLauncherJob:
			// Check the status of the given job.
			_ = m.updateJobStatus(ctx, m.monitoredJobs[msg.dispatcherID])

		case <-m.schedulerTick.C:
			m.processWatchedJobs(ctx)
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
func (m *launcherMonitor) processWatchedJobs(ctx *actor.Context) {
	// Obtain a RW lock, so that "isJobBeingMonitored()" doesn't try to iterate
	// through the "monitoredJobs" list while we're manipulating it.
	m.mu.Lock()
	defer m.mu.Unlock()

	// Loop through the jobs in the monitoredJobs map and update status accordingly
	for _, job := range m.monitoredJobs {
		// Check if the current job is in the jobsToRemove map. If it is, then delete the
		// job from both monitoredJobs map and jobsToRemove map and continue to process
		// the next job.
		if _, ok := m.jobsToRemove[job.dispatcherID]; ok {
			ctx.Log().Infof("Stopping monitoring of %s", job.dispatcherID)
			delete(m.monitoredJobs, job.dispatcherID)
			delete(m.jobsToRemove, job.dispatcherID)
			continue
		}

		if m.shouldSkip(job) {
			continue
		}

		if removeJob := m.updateJobStatus(ctx, job); removeJob {
			ctx.Log().Infof("Stopping monitoring of %s", job.dispatcherID)
			delete(m.monitoredJobs, job.dispatcherID)
			continue
		}

		job.timestamp = time.Now()
	}
	// There are chances that jobsToRemove might still have some elements remaining.
	// These values are stale and can be removed safely.
	if len(m.jobsToRemove) > 0 {
		m.jobsToRemove = map[string]bool{}
	}
}

func (m *launcherMonitor) updateJobStatus(ctx *actor.Context, job launcherJob) bool {
	removeJob := false
	dispatchID := job.dispatcherID
	owner := job.user
	ctx.Log().Debugf("Checking status of launcher job %s", dispatchID)

	resp, r, err := m.apiClient.MonitoringApi.
		GetEnvironmentStatus(m.authContext(ctx), owner, dispatchID).
		Refresh(true).
		Execute()
	if err != nil {
		if r != nil && r.StatusCode == 404 {
			ctx.Log().Infof("DispatchID %s is either COMPLETE or TERMINATED", dispatchID)
			removeJob = true
		} else {
			ctx.Log().WithError(err).Infof("error when calling `GetEnvironmentStatus` for %s:\n%v",
				dispatchID, r)
		}
		return removeJob
	}

	ctx.Log().Infof("DispatchID %s state: %s", dispatchID, *resp.State)

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
			// few lines of the error and output logs into the failure message.
			exitMessages, _ = m.getTaskLogsFromDispatcher(ctx, &job, "error.log")
			outputMessages, _ := m.getTaskLogsFromDispatcher(ctx, &job, "output.log")
			exitMessages = append(exitMessages, outputMessages...)
		}

		ctx.Log().Debugf("Send status to DAI: %d, messages %s", exitStatus, exitMessages)
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
		ctx.Tell(ctx.Self(), DispatchStateChange{
			DispatchID: dispatchID,
			State:      *resp.State,
		})
	}
	return removeJob
}

// shouldSkip returns true if we should not get the status of the specified job
// this time around the polling loop. The skip is computed on the time elapsed since
// either the time the job was added to the list of those to monitor, or the time
// of the last sample, for the following reasons. If insufficient time has elapsed
// since the job was launched then the launcher GetEnvironmentStatus REST API
// may block awaiting the job status to be come available, so reduce the likelihood of this by
// requiring a minimum time before the first status fetch. If we do encounter a delay
// for the above (or any other) reason, then the next scheduling tick may arrive
// soon thereafter, resulting in the overhead of unnecessarily rapid polling.
// Avoid the latter by applying a rate limit to each job.
func (*launcherMonitor) shouldSkip(job launcherJob) bool {
	durationSinceJobAddedOrLastStatusCollection := time.Now().Sub(job.timestamp).Seconds()
	return durationSinceJobAddedOrLastStatusCollection < minItemPollingIntervalSecs
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
		case "UNKNOWN":
		case "PENDING": // Successfully launched; pending Slurm scheduling
		case "RUNNING": // Job is executing
		case "TERMINATING": // User-initiated termination in process
			return 0, nil, false
		case "TERMINATED": // User-initiated termination complete
			return 1, getJobExitMessages(resp), true
		case "FAILED":
			return 1, getJobExitMessages(resp), true // exit status TBD -- use 1 for now
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
// The logRange expression can be used to limit the size of the logs returned.
// For example "lines=-30" is the last 30 lines of the file.
func (m *launcherMonitor) getTaskLogsFromDispatcher(
	ctx *actor.Context, job *launcherJob, baseLogName string,
) ([]string, error) {
	dispatchID := job.dispatcherID

	// By default show limited lines, on debug/trace levels show more
	linesToShow := 15
	if ctx.Log().Logger.Level == logrus.DebugLevel {
		linesToShow = 100
	} else if ctx.Log().Logger.Level == logrus.TraceLevel {
		linesToShow = 1000
	}
	// The number of lines from error/output logs to display on failure.
	logRange := fmt.Sprintf("lines=-%d", linesToShow)

	// Compose the file name
	logFileName := fmt.Sprintf("%s-%s", job.payloadName, baseLogName)

	logFile, httpResponse, err := m.apiClient.MonitoringApi.LoadEnvironmentLog(
		m.authContext(ctx), job.user, dispatchID, logFileName,
	).Range_(logRange).Execute()
	if err != nil {
		ctx.Log().WithError(err).Warnf("unable to access %s for dispatch", logFileName)
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
			ctx.Log().Errorf("logFile.Stat() failed: %s", err.Error())
			return []string{}, nil
		}
		contentLength = int(fileStat.Size())
	} else {
		contentLength, err = strconv.Atoi(contentLengthStr)
		if err != nil {
			ctx.Log().Errorf("atoi(Content-Length) failed: %s", err.Error())
			return []string{}, err
		}
		if contentLength == 0 {
			ctx.Log().Debugf("No content yet for %s", logFileName)
			return []string{}, nil
		}
	}

	buffer := make([]byte, contentLength)
	bytesRead, err := logFile.Read(buffer)
	if err != nil || bytesRead != contentLength {
		ctx.Log().WithError(err).Errorf(
			"Failed to read full http response: read %d != contentLength %d",
			bytesRead, contentLength)
		return nil, err
	}
	return strings.Split(string(buffer), "\n"), nil
}
