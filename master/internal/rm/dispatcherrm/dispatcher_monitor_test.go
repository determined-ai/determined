package dispatcherrm

import (
	"testing"
	"time"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"gotest.tools/assert"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
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
	job := getJob("11ae54526b544bcd-8607d5744a7b1439", time.Now())

	// Add the job to the monitored jobs.
	jobWatcher.monitoredJobs.Store(job.dispatcherID, job)

	// Since there have not been any "NotifyContainerRunning" messages sent
	// for this job, then we do not expect all containers to be running.
	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// first container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 0, numPeers, "node001")

	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// second container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 1, numPeers, "node002")

	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// third container.
	jobWatcher.notifyContainerRunning(ctx, job.dispatcherID, 3, numPeers, "node003")

	// The job watcher has received "NotifyContainerRunning" messages from all
	// 3 containers, so "allContainersRunning()" should now return true.
	assert.Equal(t, jobWatcher.allContainersRunning(job), true)
}

// Verifies that "isJobBeingMonitored()" returns true when the job is being
// monitored; false otherwise.
func Test_isJobBeingMonitored(t *testing.T) {
	dispatchID := "11ae54526b544bcd-8607d5744a7b1439"

	jobWatcher := getJobWatcher()
	job := getJob(dispatchID, time.Now())

	jobWatcher.addJobToMonitoredJobs(job)

	assert.Equal(t, false, jobWatcher.isJobBeingMonitored("some-fake-dispatch-id"))
	assert.Equal(t, true, jobWatcher.isJobBeingMonitored(dispatchID))
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
	jobWatcher := newDispatchWatcher(&launcherAPIClient{
		log:       logrus.WithField("component", "dispatcher-test"),
		APIClient: launcher.NewAPIClient(launcher.NewConfiguration()),
		auth:      "dummyToken",
	})

	jobWatcher.rm = &dispatcherResourceManager{
		wlmType: pbsSchedulerType,
	}

	return jobWatcher
}

// getJob creates a test job instance of type launcherJob.
func getJob(dispatchID string, lastJobStatusCheckTime time.Time) *launcherJob {
	user := "joeschmoe"
	payloadName := "myPayload"

	job := launcherJob{
		user:                   user,
		dispatcherID:           dispatchID,
		payloadName:            payloadName,
		lastJobStatusCheckTime: lastJobStatusCheckTime,
		totalContainers:        0,
		runningContainers:      mapx.New[int, containerInfo](),
	}

	return &job
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
	job := getJob("11ae54526b544bcd-8607d5744a7b1439", time.Now())

	// Add the job to the monitored jobs.
	jobWatcher.monitorJob(job.user, job.dispatcherID, job.payloadName, false)
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

// Verifies that "getDispatchIDsSortedByLastJobStatusCheckTime()" returns an
// array of dispatch IDs, sorted by the time that the jobs status was last
// checked.
func Test_getDispatchIDsSortedByLastJobStatusCheckTime(t *testing.T) {
	jobWatcher := getJobWatcher()

	dispatchID1 := "1"
	dispatchID2 := "2"
	dispatchID3 := "3"
	dispatchID4 := "4"
	dispatchID5 := "5"

	// Create the jobs, each with a different last job status check time.
	job1 := getJob(dispatchID1, time.Now().Add(time.Second*10))
	job2 := getJob(dispatchID2, time.Now().Add(time.Second*20))
	job3 := getJob(dispatchID3, time.Now().Add(time.Second*30))
	job4 := getJob(dispatchID4, time.Now().Add(time.Second*40))
	job5 := getJob(dispatchID5, time.Now().Add(time.Second*50))

	// Store the jobs in the map in random order.
	jobWatcher.monitoredJobs.Store(job2.dispatcherID, job2)
	jobWatcher.monitoredJobs.Store(job4.dispatcherID, job4)
	jobWatcher.monitoredJobs.Store(job3.dispatcherID, job3)
	jobWatcher.monitoredJobs.Store(job1.dispatcherID, job1)
	jobWatcher.monitoredJobs.Store(job5.dispatcherID, job5)

	// Get the dispatch IDs of the jobs in the "monitoredJobs" map, sorted by
	// the last job status check time, which in our case, was the time we
	// created the job.
	sortedDispatchIDs := jobWatcher.getDispatchIDsSortedByLastJobStatusCheckTime()

	// Verify that we got back the same number of dispatch IDs.
	assert.Equal(t, len(sortedDispatchIDs), 5)

	// Verify that the dispatch IDs are, in fact, sorted by timestamp.
	assert.Equal(t, sortedDispatchIDs[0], dispatchID1)
	assert.Equal(t, sortedDispatchIDs[1], dispatchID2)
	assert.Equal(t, sortedDispatchIDs[2], dispatchID3)
	assert.Equal(t, sortedDispatchIDs[3], dispatchID4)
	assert.Equal(t, sortedDispatchIDs[4], dispatchID5)
}
