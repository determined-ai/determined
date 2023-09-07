//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
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
			Trials: "1,2,3,4",
		},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				// TrialIds: []int{1, 2, 3},
				ExperimentIds: []int{1}, // trials 1,2,3
				Since:         0,
			},
		},
	}
	testStartup(t, startupMessage, []int{})
}

func testStartup(t *testing.T, startupMessage StartupMsg, expectedIDs []int) {
	ctx := context.TODO()
	streamer := stream.NewStreamer()
	publisherSet := NewPublisherSet()
	subSet := NewSubscriptionSet(streamer, publisherSet)
	messages, err := subSet.Startup(startupMessage, ctx)
	require.NoError(t, err, "error running startup")

	//expectedIDsMap := make(map[int]bool, len(expectedIDs))
	//for _, id := range messages {
	//	expectedIDsMap[id] = false
	//}

	fmt.Println(len(messages))
	for _, msg := range messages {
		fmt.Println(msg)
	}

	// messages[0]
	fmt.Println(messages[0])
}
