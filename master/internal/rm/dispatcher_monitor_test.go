package rm

import (
	"testing"

	launcher "github.hpe.com/hpe/hpc-ard-launcher-go/launcher"
	"gotest.tools/assert"
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
