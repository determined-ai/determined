package dispatcherrm

import (
	"testing"
	"time"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/assert"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/syncx/mapx"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

const (
	DispatchID1 = "dispatchID1"
	DispatchID2 = "dispatchID2"
	DispatchID3 = "dispatchID3"
	DispatchID4 = "dispatchID4"
	DispatchID5 = "dispatchID5"
)

const (
	HpcJobID1 = "hpcJobID1"
	HpcJobID2 = "hpcJobID2"
)

const NoneReasonCode = "None"

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

	jobWatcher, events := getJobWatcher()

	// Initialize to 0 for this test.
	numTimesWriteExperimentLogCalled := 0
	go func() {
		for e := range events {
			_, ok := e.(dispatchExpLogMessage)
			if !ok {
				continue
			}
			numTimesWriteExperimentLogCalled++
		}
	}()

	job := getJob("11ae54526b544bcd-8607d5744a7b1439", time.Now())

	// Add the job to the monitored jobs.
	jobWatcher.monitoredJobs.Store(job.dispatcherID, job)

	// Since there have not been any "NotifyContainerRunning" messages sent
	// for this job, then we do not expect all containers to be running.
	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// The job watcher receives a "NotifyContainerRunning" message from the
	// first container.
	jobWatcher.notifyContainerRunning(job.dispatcherID, 0, numPeers, "node001")

	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// Verify that we wrote a message to the experiment log indicating that
	// 1 out of 3 containers are running.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 1
	}, "numTimesWriteExperimentLogCalled != 1")

	// The job watcher receives a "NotifyContainerRunning" message from the
	// second container.
	jobWatcher.notifyContainerRunning(job.dispatcherID, 1, numPeers, "node002")

	assert.Equal(t, jobWatcher.allContainersRunning(job), false)

	// Verify that we wrote a message to the experiment log indicating that
	// 2 out of 3 containers are running.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 2
	}, "numTimesWriteExperimentLogCalled != 2")

	// The job watcher receives a "NotifyContainerRunning" message from the
	// third container.
	jobWatcher.notifyContainerRunning(job.dispatcherID, 3, numPeers, "node003")

	// The job watcher has received "NotifyContainerRunning" messages from all
	// 3 containers, so "allContainersRunning()" should now return true.
	assert.Equal(t, jobWatcher.allContainersRunning(job), true)

	// Verify that we wrote a message to the experiment log indicating that
	// 3 out of 3 containers are running.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 3
	}, "numTimesWriteExperimentLogCalled != 3")
}

// Verifies that "isJobBeingMonitored()" returns true when the job is being
// monitored; false otherwise.
func Test_isJobBeingMonitored(t *testing.T) {
	dispatchID := "11ae54526b544bcd-8607d5744a7b1439"

	jobWatcher, _ := getJobWatcher()
	job := getJob(dispatchID, time.Now())

	jobWatcher.addJobToMonitoredJobs(job)

	assert.Equal(t, false, jobWatcher.isJobBeingMonitored("some-fake-dispatch-id"))
	assert.Equal(t, true, jobWatcher.isJobBeingMonitored(dispatchID))
}

// getJobWatcher creates an instance of the dispatcher_monitor.
func getJobWatcher() (*launcherMonitor, <-chan launcherMonitorEvent) {
	events := make(chan launcherMonitorEvent, 64)
	dispatchIDToHPCJobID := mapx.New[string, string]()
	jobWatcher := newDispatchWatcher(&launcherAPIClient{
		log:       logrus.WithField("component", "dispatcher-test"),
		APIClient: launcher.NewAPIClient(launcher.NewConfiguration()),
		auth:      "dummyToken",
	}, &dispatchIDToHPCJobID, events)
	return jobWatcher, events
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
	jobWatcher, _ := getJobWatcher()
	go jobWatcher.watch()
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
	jobWatcher, _ := getJobWatcher()

	// Create the jobs, each with a different last job status check time.
	job1 := getJob(DispatchID1, time.Now().Add(time.Second*10))
	job2 := getJob(DispatchID2, time.Now().Add(time.Second*20))
	job3 := getJob(DispatchID3, time.Now().Add(time.Second*30))
	job4 := getJob(DispatchID4, time.Now().Add(time.Second*40))
	job5 := getJob(DispatchID5, time.Now().Add(time.Second*50))

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
	assert.Equal(t, sortedDispatchIDs[0], DispatchID1)
	assert.Equal(t, sortedDispatchIDs[1], DispatchID2)
	assert.Equal(t, sortedDispatchIDs[2], DispatchID3)
	assert.Equal(t, sortedDispatchIDs[3], DispatchID4)
	assert.Equal(t, sortedDispatchIDs[4], DispatchID5)
}

// Verifies the following behavior for "obtainJobStateFromWlmQueueDetails()":
//
//  1. Returns true if the job state is "PD" (pending) or "R" (running); false
//     otherwise.
//  2. Whenever the job is pending and the reason code changes, write the
//     reason description to the experiment log.
func Test_obtainJobStateFromWlmQueueDetails(t *testing.T) {
	// Initialize the map with some values.
	qStats := map[string]map[string]string{
		"hpcJobID1": {
			"state":      "PD",
			"reasonCode": "Resources",
			"reasonDesc": "The QOS resource limit has been reached.",
		},
		"hpcJobID2": {
			"state": "PD",
		},
	}

	jobWatcher, events := getJobWatcher()

	// Initialize to 0 for this test.
	numTimesWriteExperimentLogCalled := 0
	go func() {
		for e := range events {
			_, ok := e.(dispatchExpLogMessage)
			if !ok {
				continue
			}
			numTimesWriteExperimentLogCalled++
		}
	}()

	// Create a job.
	job := getJob(DispatchID1, time.Now())

	// Verify that a newly created job has an empty job pending reason code.
	assert.Equal(t, job.jobPendingReasonCode, "")

	// Map the dispatch ID to the HPC job ID, since the HPC job ID is used as
	// the key to the "qStats" map.
	jobWatcher.dispatchIDToHPCJobID.Store(DispatchID1, HpcJobID1)
	jobWatcher.dispatchIDToHPCJobID.Store(DispatchID2, HpcJobID2)

	retValue := jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 1
	}, "numTimesWriteExperimentLogCalled != 1")

	// Expect a return value of "true", since the job state was "PD" (pending).
	assert.Equal(t, retValue, true)

	// The job pending reason code should be "Resources", because that's what's
	// contained in the "qStats" for the HPC job ID.
	assert.Equal(t, job.jobPendingReasonCode, "Resources")

	// Call the function again with the same reason code as before.
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// We only write to the experiment log when the reason code changes, and
	// since the reason code has not changed, the count should still be 1.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 1
	}, "numTimesWriteExperimentLogCalled != 1")

	// Expect a return value of "true", since the job state was "PD" (pending).
	assert.Equal(t, retValue, true)

	// The job pending reason code should be "Resources", because that's what's
	// contained in the "qStats" for the HPC job ID.
	assert.Equal(t, job.jobPendingReasonCode, "Resources")

	// Change the reason code.
	qStats[HpcJobID1]["reasonCode"] = "NodeDown"
	qStats[HpcJobID1]["reasonDesc"] = "A node required by the job is down."

	// Call the function again with a different reason code than before.
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// We only write to the experiment log when the reason code changes, and
	// since the reason code changed, the count should now be 2.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 2
	}, "numTimesWriteExperimentLogCalled != 2")

	// Expect a return value of "true", since the job state was "PD" (pending).
	assert.Equal(t, retValue, true)

	// The job pending reason code should be "NodeDown", because that's what
	// we changed the "reasonCode" to.
	assert.Equal(t, job.jobPendingReasonCode, "NodeDown")

	// Change the job's state to "R" (running).
	qStats[HpcJobID1]["state"] = "R"
	qStats[HpcJobID1]["reasonCode"] = NoneReasonCode
	qStats[HpcJobID1]["reasonDesc"] = ""

	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// Expect a return value of "true", since the job state is "R" (running).
	assert.Equal(t, retValue, true)

	// The job pending reason code should be "None" now that the state has
	// transitioned from "PD" (pending) to "R" (running).
	assert.Equal(t, job.jobPendingReasonCode, NoneReasonCode)

	// Delete the "reasonCode" key from the map.
	qStats[HpcJobID1]["state"] = "R"
	delete(qStats[HpcJobID1], "reasonCode")

	// Here we verify that if the "qStat" map does not contain the "reasonCode"
	// key for the HPC job ID, that we don't blow up.
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID2, qStats, job)

	// Expect a return value of "true", since the job state is "R" (running).
	assert.Equal(t, retValue, true)
}

// Verifies that when a job is in the "Running" state with a reason code of
// "Prolog", a message is displayed only once in the experiment log that
// provides the description for the "Prolog" reason code.  And, when the
// reason code is no longer "Prolog", a message is displayed indicating that
// the condition has been cleared.
func Test_obtainJobStateFromWlmQueueDetailsWhenJobInRunningStateWithReasonProlog(t *testing.T) {
	// Initialize the map with some values.
	qStats := map[string]map[string]string{
		"hpcJobID1": {
			"state":      "R",
			"reasonCode": "Prolog",
			"reasonDesc": "The job's PrologSlurmctld program is still running.",
		},
	}

	jobWatcher, events := getJobWatcher()
	numTimesWriteExperimentLogCalled := 0
	writeExperimentLogMessageReceived := ""
	go func() {
		for e := range events {
			event, ok := e.(dispatchExpLogMessage)
			if !ok {
				continue
			}
			numTimesWriteExperimentLogCalled++
			writeExperimentLogMessageReceived = event.Message
		}
	}()

	// Initialize to 0 for this test.
	numTimesWriteExperimentLogCalled = 0

	// Create a job.
	job := getJob(DispatchID1, time.Now())

	// Map the dispatch ID to the HPC job ID, since the HPC job ID is used as
	// the key to the "qStats" map.
	jobWatcher.dispatchIDToHPCJobID.Store(DispatchID1, HpcJobID1)

	// Clear the message.
	writeExperimentLogMessageReceived = ""

	// Call the function again with a different reason code than before.
	retValue := jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// Expect a return value of "true", since the job state was "R" (running).
	assert.Equal(t, retValue, true)

	// The "writeExperimentLog()" function should have been called once.
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 1
	}, "numTimesWriteExperimentLogCalled != 1")

	assert.Equal(t, writeExperimentLogMessageReceived,
		"HPC job waiting on pre-condition: The job's PrologSlurmctld program is still running.")

	// Clear the message.
	writeExperimentLogMessageReceived = ""

	// Call the function again with the same reason code as before.
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// Expect a return value of "true", since the job state was "R" (running).
	assert.Equal(t, retValue, true)

	// The "writeExperimentLog()" function should not have been called again,
	// since neither the job state not reason code have changed since from
	// the previous time we called "obtainJobStateFromWlmQueueDetails()".
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 1
	}, "numTimesWriteExperimentLogCalled != 1")

	// The "writeExperimentLog()" function should not have been called, so
	// there should be no message set.
	assert.Equal(t, writeExperimentLogMessageReceived, "")

	// Clear the message.
	writeExperimentLogMessageReceived = ""

	// Change the reason code to "None", to pretend that the prolog script
	// completed, so the reason code is no longer "Prolog".
	qStats[HpcJobID1]["reasonCode"] = NoneReasonCode
	qStats[HpcJobID1]["reasonDesc"] = ""

	// Call the function again with the same reason code of "None".
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// Expect a return value of "true", since the job state was "R" (running).
	assert.Equal(t, retValue, true)

	// The "writeExperimentLog()" function should have been called again, since
	// the reason code changed from "Prolog" to "None" from the previous time
	// we called "obtainJobStateFromWlmQueueDetails()".
	assertConditionWithin(t, time.Second, func() bool {
		return numTimesWriteExperimentLogCalled == 2
	}, "numTimesWriteExperimentLogCalled != 2")

	assert.Equal(t, writeExperimentLogMessageReceived, "HPC job pre-condition cleared.")

	// Clear the message.
	writeExperimentLogMessageReceived = ""

	// Call the function again with the same reason code of "None" again.
	retValue = jobWatcher.obtainJobStateFromWlmQueueDetails(DispatchID1, qStats, job)

	// Expect a return value of "true", since the job state was "R" (running).
	assert.Equal(t, retValue, true)

	// The "writeExperimentLog()" function should not have been called again,
	// since neither the job state not reason code have changed since from
	// the previous time we called "obtainJobStateFromWlmQueueDetails()".
	assert.Equal(t, numTimesWriteExperimentLogCalled, 2)

	// The "writeExperimentLog()" function should not have been called, so
	// there should be no message set.
	assert.Equal(t, writeExperimentLogMessageReceived, "")
}

// TODO carolina/bradley: add to utils.
func assertConditionWithin(t *testing.T, timeout time.Duration, condition func() bool, msg string) {
	for i := 0; i < int(timeout/time.Millisecond); i++ {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	require.Fail(t, msg)
}

func TestConvertHpcStatus(t *testing.T) {
	jobWatcher, _ := getJobWatcher()
	assert.Equal(t, jobWatcher.convertHpcStatus("PENDING"), jobv1.State_STATE_QUEUED)
	assert.Equal(t, jobWatcher.convertHpcStatus("RUNNING"), jobv1.State_STATE_SCHEDULED)
	assert.Equal(t, jobWatcher.convertHpcStatus(""), jobv1.State_STATE_UNSPECIFIED)
}

func TestConvertExternalJobs(t *testing.T) {
	// External job with no job details
	jobDetails1 := map[string]string{
		"jobID": "1234",
	}

	jobWatcher, _ := getJobWatcher()

	assert.Assert(t, jobWatcher.convertExternalJobs("", nil) == nil)
	assert.Assert(t, jobWatcher.convertExternalJobs("asdf", nil) == nil)
	assert.Assert(t, jobWatcher.convertExternalJobs("", jobDetails1) == nil)

	jobWatcher.externalJobs.Store(jobDetails1["jobID"], jobDetails1)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 1)
	actualJobDetails := jobWatcher.fetchExternalJobs("")
	assert.Equal(t, actualJobDetails[0].JobId, jobDetails1["jobID"])
	assert.Equal(t, actualJobDetails[0].Type, jobv1.Type_TYPE_EXTERNAL)
	assert.Equal(t, actualJobDetails[0].Summary.State, jobv1.State_STATE_UNSPECIFIED)
	assert.Equal(t, actualJobDetails[0].Name, "Not Available")
	assert.Equal(t, actualJobDetails[0].Username, "Not Available")
	assert.Equal(t, actualJobDetails[0].ResourcePool, "Not Available")
	assert.Equal(t, actualJobDetails[0].RequestedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].AllocatedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].SubmissionTime.Seconds, timestamppb.New(time.Time{}).Seconds)
	jobWatcher.externalJobs.Delete(jobDetails1["jobID"])

	// External job with all job details
	jobDetails2 := map[string]string{
		"jobID":      "1234",
		"state":      "RUNNING",
		"name":       "hello_world",
		"partition":  "defq",
		"submitTime": "2023-05-10T02:35:12Z",
		"userName":   "testuser1",
		"gpuSlots":   "0",
		"cpuSlots":   "1",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	expectedSubmitTime, _ := time.Parse(time.RFC3339, jobDetails2["submitTime"])
	jobWatcher.externalJobs.Store(jobDetails2["jobID"], jobDetails2)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 1)
	actualJobDetails = jobWatcher.fetchExternalJobs("")
	assert.Equal(t, actualJobDetails[0].JobId, jobDetails2["jobID"])
	assert.Equal(t, actualJobDetails[0].Type, jobv1.Type_TYPE_EXTERNAL)
	assert.Equal(t, actualJobDetails[0].Summary.State, jobv1.State_STATE_SCHEDULED)
	assert.Equal(t, actualJobDetails[0].Name, jobDetails2["name"])
	assert.Equal(t, actualJobDetails[0].Username, jobDetails2["userName"])
	assert.Equal(t, actualJobDetails[0].ResourcePool, jobDetails2["partition"])
	assert.Equal(t, actualJobDetails[0].RequestedSlots, int32(1))
	assert.Equal(t, actualJobDetails[0].AllocatedSlots, int32(1))
	assert.Equal(t, actualJobDetails[0].SubmissionTime.Seconds,
		timestamppb.New(expectedSubmitTime).Seconds)
	jobWatcher.externalJobs.Delete(jobDetails2["jobID"])

	// External job with invalid job details
	jobDetails3 := map[string]string{
		"jobID":      "1234",
		"state":      "RUNNING",
		"name":       "hello_world",
		"partition":  "defq",
		"submitTime": "2023-05-10 02:35:12Z",
		"userName":   "testuser1",
		"gpuSlots":   "S",
		"cpuSlots":   "T",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobWatcher.externalJobs.Store(jobDetails3["jobID"], jobDetails3)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 1)
	actualJobDetails = jobWatcher.fetchExternalJobs("")
	assert.Equal(t, actualJobDetails[0].JobId, jobDetails3["jobID"])
	assert.Equal(t, actualJobDetails[0].Type, jobv1.Type_TYPE_EXTERNAL)
	assert.Equal(t, actualJobDetails[0].Summary.State, jobv1.State_STATE_SCHEDULED)
	assert.Equal(t, actualJobDetails[0].Name, jobDetails3["name"])
	assert.Equal(t, actualJobDetails[0].Username, jobDetails3["userName"])
	assert.Equal(t, actualJobDetails[0].ResourcePool, jobDetails3["partition"])
	assert.Equal(t, actualJobDetails[0].RequestedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].AllocatedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].SubmissionTime.Seconds, timestamppb.New(time.Time{}).Seconds)
	jobWatcher.externalJobs.Delete(jobDetails3["jobID"])
}

func TestAddAndFetchExternalJobs(t *testing.T) {
	jobWatcher, _ := getJobWatcher()
	jobDetails1 := map[string]string{
		"jobID":      "1234",
		"state":      "RUNNING",
		"name":       "hello_world",
		"partition":  "defq",
		"submitTime": "2023-05-10T02:35:12Z",
		"userName":   "testuser1",
		"gpuSlots":   "0",
		"cpuSlots":   "1",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobDetails2 := map[string]string{
		"jobID":      "1235",
		"state":      "PENDING",
		"name":       "hello_world_gpu",
		"partition":  "defq_GPU",
		"submitTime": "2023-05-10T02:35:12Z",
		"userName":   "testuser1",
		"gpuSlots":   "1",
		"cpuSlots":   "0",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobWatcher.externalJobs.Store(jobDetails1["jobID"], jobDetails1)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 1)
	_, jobFound := jobWatcher.externalJobs.Load(jobDetails1["jobID"])
	assert.Assert(t, jobFound)
	_, jobFound = jobWatcher.externalJobs.Load("asdf")
	assert.Equal(t, jobFound, false)
	externalJobs := jobWatcher.fetchExternalJobs("defq")
	assert.Equal(t, len(externalJobs), 1)
	assert.Equal(t, externalJobs[0].JobId, jobDetails1["jobID"])
	// We should get all external jobs when the resource pool value is empty.
	externalJobs = jobWatcher.fetchExternalJobs("")
	assert.Equal(t, len(externalJobs), 1)
	assert.Equal(t, externalJobs[0].JobId, jobDetails1["jobID"])
	jobWatcher.externalJobs.Store(jobDetails2["jobID"], jobDetails1)
	externalJobs = jobWatcher.fetchExternalJobs("")
	assert.Equal(t, len(externalJobs), 2)
}

func TestAddAndFetchExternalJobsInvalidInputs(t *testing.T) {
	jobWatcher, _ := getJobWatcher()
	jobDetails := map[string]string{
		"jobID":      "1234",
		"state":      "RUNNING",
		"name":       "hello_world",
		"partition":  "defq",
		"userName":   "testuser1",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobWatcher.externalJobs.Store(jobDetails["jobID"], jobDetails)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 1)
	actualJobDetails := jobWatcher.fetchExternalJobs("")
	assert.Equal(t, actualJobDetails[0].RequestedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].AllocatedSlots, int32(0))
	assert.Equal(t, actualJobDetails[0].SubmissionTime.Seconds, timestamppb.New(time.Time{}).Seconds)
}

func TestGetExternalJobQStats(t *testing.T) {
	jobWatcher, _ := getJobWatcher()
	jobDetails1 := map[string]string{
		"jobID":      "1234",
		"state":      "RUNNING",
		"name":       "hello_world",
		"partition":  "defq_GPU",
		"userName":   "testuser1",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobDetails2 := map[string]string{
		"jobID":      "1235",
		"state":      "PENDING",
		"name":       "hello_world",
		"partition":  "defq",
		"userName":   "testuser1",
		"reasonCode": "None",
		"reasonDesc": "",
	}
	jobWatcher.externalJobs.Store(jobDetails1["jobID"], jobDetails1)
	jobWatcher.externalJobs.Store(jobDetails2["jobID"], jobDetails2)
	assert.Equal(t, jobWatcher.externalJobs.Len(), 2, "Verify that two external jobs are added")

	actualJobDetails := jobWatcher.getExternalJobQStats("")
	assert.Equal(t, actualJobDetails.ScheduledCount, int32(1),
		"Verify that scheduled jobs count is 1 when processing all resource pools")
	assert.Equal(t, actualJobDetails.QueuedCount, int32(1),
		"Verify that queued jobs count is 1 when processing all resource pools")

	actualJobDetails = jobWatcher.getExternalJobQStats("defq")
	assert.Equal(t, actualJobDetails.ScheduledCount, int32(0),
		"Verify that scheduled jobs count is 0 when processing only defq resource pool")
	assert.Equal(t, actualJobDetails.QueuedCount, int32(1),
		"Verify that scheduled jobs count is 1 when processing only defq resource pool")
}
