//go:build integration
// +build integration

package stream

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/pkg/model"
)

func TestClientBeforeRunOne(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)

	// Startup a supervisor, but don't call ssup.Run() at all.
	dbURL := os.Getenv("DET_INTEGRATION_POSTGRES_URL")
	ssup := NewSupervisor(dbURL)

	// Pretend a websocket connection arrives...
	ctx := context.Background()
	testUser := model.User{Username: uuid.New().String()}
	socket := newMockSocket()

	errgrp := errgroupx.WithContext(ctx)
	defer errgrp.Cancel()

	// Connect the websocket to the server.
	errgrp.Go(func(ctx context.Context) error {
		defer socket.Close()
		return ssup.doWebsocket(ctx, testUser, socket, testPrepareFunc)
	})

	// Make sure our offline messages work and we get disconnect right afterwards.
	errgrp.Go(func(ctx context.Context) error {
		socket.WriteToServer(
			t,
			&StartupMsg{
				SyncID: "x",
				Known:  KnownKeySet{},
				Subscribe: SubscriptionSpecSet{
					Projects: &ProjectSubscriptionSpec{ProjectIDs: []int{1}},
				},
			},
		)
		socket.ReadUntilFound(
			t,
			"type: sync_msg, sync_id: x, complete: false",
			"type: project, project_id: 1, state: UNSPECIFIED, workspace_id: 1",
			"type: sync_msg, sync_id: x, complete: true",
		)
		// Since we start runOne with an already-canceled context, we expect the socket to break
		// as soon as the offline messages are sent.
		socket.AssertEOF(t)
		return nil
	})

	// Start a runOne that comes already canceled; we don't want the publisher to actually do
	// anything, we just need to test the synchronization logic.
	//
	// This also exercises the same codepaths as a websocket call which arrives after runOne crashes
	// and before the next runOne starts, because this PublisherSet is basically instantly crashed.
	// The result should be that all the offline messages are sent and the websocket breaks without
	// ony online messages.
	errgrp.Go(func(ctx context.Context) error {
		deadCtx, cancelFn := context.WithCancel(ctx)
		cancelFn()
		return ssup.runOne(deadCtx)
	})

	require.NoError(t, errgrp.Wait())
}
