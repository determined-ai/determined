//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/stream"
)

func TestStartupTrial(t *testing.T) {
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)

	defer cleanup()

	startupMessage := StartupMsg{
		Known: KnownKeySet{
			Trials: "1, 2, 3",
		},
		Subscribe: SubscriptionSpecSet{},
	}
	testStartup(t, startupMessage)
}

func testStartup(t *testing.T, startupMessage StartupMsg) {
	ctx := context.TODO()
	streamer := stream.NewStreamer()
	publisherSet := NewPublisherSet()
	subSet := NewSubscriptionSet(streamer, publisherSet)
	messages, err := subSet.Startup(startupMessage, ctx)
	require.NoError(t, err, "error running startup")
	fmt.Println(messages)
}
