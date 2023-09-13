//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/stream"
	"github.com/determined-ai/determined/master/test/streamdata"
)

func TestStartupTrial(t *testing.T) {
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)

	defer cleanup()

	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	startupMessage := StartupMsg{
		Known: KnownKeySet{
			Trials: "1,2,3",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				// TrialIds: []int{1, 2, 3},
				ExperimentIds: []int{1}, // trials 1,2,3
				Since:         0,
			},
		},
	}
	messages := testStartup(t, startupMessage)
	deletions, trialMsgs, err := splitDeletionsAndTrials(messages)
	require.NoError(t, err)
	require.Equal(t, 1, len(deletions), "did not receive 1 deletion message")
	require.Equal(t, "0", deletions[0], "expected deleted trials to be 0, not %s", deletions[0])
	require.Equal(t, 0, len(trialMsgs), "received unexpected trial message")
}

func splitDeletionsAndTrials(messages []interface{}) ([]string, []*TrialMsg, error) {
	var deletions []string
	var trialMsgs []*TrialMsg
	for _, msg := range messages {
		if deletion, ok := msg.(string); ok {
			deletions = append(deletions, deletion)
		} else if trialMsg, ok := msg.(*TrialMsg); ok {
			trialMsgs = append(trialMsgs, trialMsg)
		} else {
			return nil, nil, fmt.Errorf("expected a string or *TrialMsg, but received %t",
				reflect.TypeOf(msg))
		}
	}
	return deletions, trialMsgs, nil
}

func testStartup(t *testing.T, startupMessage StartupMsg) []interface{} {
	ctx := context.TODO()
	streamer := stream.NewStreamer()
	publisherSet := NewPublisherSet()
	subSet := NewSubscriptionSet(streamer, publisherSet, func(i interface{}) interface{} {
		return i
	},
		func(s string, s2 string) interface{} {
			return s2
		},
	)
	messages, err := subSet.Startup(startupMessage, ctx)
	require.NoError(t, err, "error running startup")

	return messages
}
