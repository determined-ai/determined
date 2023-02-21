package dispatcherrm

import (
	"testing"
	"time"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func String(v string) *string { return &v }

func Test_getJobExitMessagesAndFiltering(t *testing.T) {
	tests := []struct {
		events           []launcher.Event
		expectedMessages []string
		filteredMessages []string
	}{
		{
			events: []launcher.Event{
				{
					// Suppressed to due the reporter
					Level:    String("ERROR"),
					Reporter: String(ignoredReporter),
					Message: String(
						"Failed to launch payload ai_cmd (com.cray.analytics.capsules.generic." +
							"container:latest) with carrier com.cray.analytics.capsules.carriers." +
							"hpc.pbs.SingularityOverPbs - PBS job is in a failed " +
							"state due to reason 'Failed':\n  Job Exited with Exit Code 0 in" +
							" response to receiving Signal 15\n",
					),
				},
				{
					// Expected, and matching filter
					Level: String("ERROR"),
					Reporter: String(
						"com.cray.analytics.capsules.carriers.hpc.pbs.SingularityOverPbs",
					),
					Message: String(
						"Failed to launch payload ai_cmd (com.cray.analytics.capsules.generic." +
							"container:latest) with carrier com.cray.analytics.capsules.carriers." +
							"hpc.pbs.SingularityOverPbs - PBS job is in a failed" +
							" state due to reason 'Failed':\n  Job Exited with Exit " +
							"Code 0 in response to receiving Signal 15\n",
					),
				},
				{
					// expected, but filtered due to invalid message format.
					Level:    String("ERROR"),
					Reporter: String("com.cray.analytics.capsules.carriers.hpc.pbs.SingularityOverPbs"),
					Message:  String("Not a recognized message format."),
				},
				{
					// Not expected because wrong logging level
					Level:    String("INFO"),
					Reporter: String("any"),
					Message: String(
						"Failed to launch payload ai_cmd (com.cray.analytics.capsules.generic." +
							"container:latest) with carrier com.cray.analytics.capsules.carriers." +
							"hpc.pbs.SingularityOverPbs - PBS job is in a failed" +
							" state due to reason 'Failed':\n  Job Exited with Exit " +
							"Code 0 in response to receiving Signal 15\n",
					),
				},
			},
			expectedMessages: []string{
				"Failed to launch payload ai_cmd (com.cray.analytics.capsules.generic." +
					"container:latest) with carrier com.cray.analytics.capsules.carriers." +
					"hpc.pbs.SingularityOverPbs - PBS job is in a failed state due to " +
					"reason 'Failed':\n  Job Exited with Exit Code 0 " +
					"in response to receiving Signal 15\n",
				"Not a recognized message format.",
			},
			filteredMessages: []string{
				"failed state due to reason 'Failed':\n  Job Exited with Exit Code " +
					"0 in response to receiving Signal 15\n",
			},
		},
		{
			events: []launcher.Event{
				{
					// Expected, and matching filter
					Level:    String("ERROR"),
					Reporter: String("com.cray.analytics.capsules.carriers.hpc.slurm.SingularityOverSlurm"),
					Message: String(
						"Slurm job is in a failed state due to reason 'NonZeroExitCode':\n" +
							"  Job Exited with Exit Code 1\n  INFO: Setting workdir to /run/determined/workdir\n",
					),
				},
			},
			expectedMessages: []string{
				"Slurm job is in a failed state due to reason 'NonZeroExitCode':\n" +
					"  Job Exited with Exit Code 1\n  INFO: Setting workdir to /run/determined/workdir\n",
			},
			filteredMessages: []string{
				"Slurm job is in a failed state due to reason 'NonZeroExitCode':\n" +
					"  Job Exited with Exit Code 1\n  INFO: Setting workdir to /run/determined/workdir\n",
			},
		},
	}

	for _, test := range tests {
		di := launcher.NewDispatchInfo()
		di.SetEvents(test.events)
		actualMessages := getJobExitMessages(*di)
		filteredMessages := filterOutSuperfluousMessages(actualMessages)

		assert.DeepEqual(t, actualMessages, test.expectedMessages)
		assert.DeepEqual(t, filteredMessages, test.filteredMessages)
	}
}

func Test_getJobID(t *testing.T) {
	var jobID string

	jobID = getJobID(map[string]interface{}{})
	assert.Equal(t, jobID, "")

	jobID = getJobID(map[string]interface{}{
		"job-id": 1234,
	})
	assert.Equal(t, jobID, "")

	jobID = getJobID(map[string]interface{}{
		"jobid": "1234",
	})
	assert.Equal(t, jobID, "")

	jobID = getJobID(map[string]interface{}{
		"job-id": "1234",
	})
	assert.Equal(t, jobID, "1234")
}

func Test_testAndSetBoolean(t *testing.T) {
	jobWatcher := getJobWatcher()

	processingWatchedJobs := false

	// Method should return the current value, which is false, and
	// change the value of "processingWatchesJobs" to true.
	result := testAndSetBoolean(&processingWatchedJobs, &jobWatcher.processingWatchedJobsMutex)

	assert.Equal(t, result, false)
	assert.Equal(t, processingWatchedJobs, true)

	processingWatchedJobs = true

	// Method should return the current value, which is true, and
	// change the value of "processingWatchesJobs" to true.
	result = testAndSetBoolean(&processingWatchedJobs, &jobWatcher.processingWatchedJobsMutex)

	assert.Equal(t, result, true)
	assert.Equal(t, processingWatchedJobs, true)
}

// Verifies that "allContainersRunning" returns true only when the job watcher
// has received a "NotifyContainerRunning" message from all the containers that
// are part of the job.
func Test_allContainersRunning(t *testing.T) {
	// Assume Slurm set the SLURM_NPROCS environment variable to 3, meaning
	// that there will be 3 containers running for the job. In the notification
	// message the SLURM_NPROCS environment variable is stored in "numPeers",
	// so use a similar variable name here for consistency.
	var numPeers int32 = 3

	ctx := getMockActorCtx()
	jobWatcher := getJobWatcher()
	job := getJob()

	// Add the job to the monitored jobs.
	jobWatcher.monitoredJobs[job.dispatcherID] = job

	// Since there have not been any "NotifyContainerRunning" messages sent
	// for this job, then we do not expect all containers to be running.
	assert.Equal(t, jobWatcher.allContainersRunning(jobWatcher.monitoredJobs[job.dispatcherID]), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// first container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 0, numPeers, "node001")

	assert.Equal(t, jobWatcher.allContainersRunning(jobWatcher.monitoredJobs[job.dispatcherID]), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// second container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 1, numPeers, "node002")

	assert.Equal(t, jobWatcher.allContainersRunning(jobWatcher.monitoredJobs[job.dispatcherID]), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// third container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 3, numPeers, "node003")

	// The job watcher has received "NotifyContainerRunning" messages from all
	// 3 containers, so "allContainersRunning()" should now return true.
	assert.Equal(t, jobWatcher.allContainersRunning(jobWatcher.monitoredJobs[job.dispatcherID]), true)
}

func getMockActorCtx() *actor.Context {
	var ctx *actor.Context
	sys := actor.NewSystem("")
	child, _ := sys.ActorOf(actor.Addr("child"), actor.ActorFunc(func(context *actor.Context) error {
		ctx = context
		return nil
	}))
	parent, _ := sys.ActorOf(actor.Addr("parent"), actor.ActorFunc(func(context *actor.Context) error {
		context.Ask(child, "").Get()
		return nil
	}))
	sys.Ask(parent, "").Get()
	return ctx
}

// getJobWatcher creates an instance of the dispatcher_monitor.
func getJobWatcher() *launcherMonitor {
	clientConfiguration := launcher.NewConfiguration()
	apiClient := launcher.NewAPIClient(clientConfiguration)
	authToken := "dummyToken"
	configFile := "dummyConfigFile"

	jobWatcher := newDispatchWatcher(apiClient, authToken, configFile)

	return jobWatcher
}

// getJob creates a test job instance of type launcherJob.
func getJob() launcherJob {
	user := "joeschmoe"
	dispatchID := "11ae54526b544bcd-8607d5744a7b1439"
	payloadName := "myPayload"

	job := launcherJob{
		user:                   user,
		dispatcherID:           dispatchID,
		payloadName:            payloadName,
		lastJobStatusCheckTime: time.Now(),
		totalContainers:        0,
		runningContainers:      make(map[int]containerInfo),
	}

	return job
}

// Test to check that major events in the dispatcher_monitor life cycle.
// This test checks the following events:
// - dispatcher_monitor launched successfully.
// - add a job to monitor.
// - new job is being monitored.
// - remove the job being monitored.
func TestMonitorJobOperations(t *testing.T) {
	jobWatcher := getJobWatcher()
	ctx := getMockActorCtx()
	go jobWatcher.watch(ctx)
	job := getJob()

	// Add the job to the monitored jobs.
	jobWatcher.monitorJob(job.user, job.dispatcherID, job.payloadName)
	// Wait for the job to be added to the monitored jobs with a timeout of 30 seconds.
	timeout := time.Now().Add(30 * time.Second)
	for !(jobWatcher.isJobBeingMonitored(job.dispatcherID)) {
		if time.Now().After(timeout) {
			assert.Assert(t, false, "Failed to monitor the job within the timeout limit.")
		}
	}
	// Check if the job is being monitored.
	jobWatcher.checkJob(job.dispatcherID)
	assert.Equal(t, jobWatcher.isJobBeingMonitored(job.dispatcherID), true,
		"Failed to monitor the job.")
	// Cancel job monitoring.
	jobWatcher.removeJob(job.dispatcherID)
	// Wait for the job to be removed from the monitored jobs with a timeout of 30 seconds.
	timeout = time.Now().Add(30 * time.Second)
	for jobWatcher.isJobBeingMonitored(job.dispatcherID) {
		if time.Now().After(timeout) {
			assert.Assert(t, false,
				"Failed to remove the job from the monitoring queue within the timeout limit.")
		}
	}
	// Check that job is not being monitored.
	assert.Equal(t, jobWatcher.isJobBeingMonitored(job.dispatcherID), false,
		"Failed to remove the job from the monitoring queue.")
}
