//go:build integration
// +build integration

package stream

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/streamdata"
)

func TestMockSocket(t *testing.T) {
	expectedMsg := StartupMsg{
		SyncID: uuid.NewString(),
		Known:  KnownKeySet{Trials: "1,2,3"},
		Subscribe: SubscriptionSpecSet{
			Trials: &TrialSubscriptionSpec{
				ExperimentIds: []int{1},
				Since:         0,
			},
		},
	}

	// test WriteOutbound
	socket := newMockSocket()
	socket.WriteOutbound(t, &expectedMsg)

	// test ReadJSON
	actualMsg := StartupMsg{}
	err := socket.ReadJSON(&actualMsg)
	require.NoError(t, err)
	require.Equal(t, actualMsg.Known, expectedMsg.Known)
	require.Equal(t, actualMsg.Subscribe, expectedMsg.Subscribe)
	require.Equal(t, actualMsg.SyncID, expectedMsg.SyncID)
	require.Equal(t, 0, len(socket.outbound))

	// test write
	err = socket.Write("test")
	require.NoError(t, err)

	// test read incoming
	var data string
	socket.ReadIncoming(t, &data)
	require.Equal(t, "test", data)

	// test ReadUntil
	err = socket.Write("test")
	require.NoError(t, err)
	err = socket.Write("final")
	require.NoError(t, err)
	var msgs []string
	socket.ReadUntil(t, &msgs, "final")
	require.NoError(t, err)
	require.Equal(t, 2, len(msgs))
	require.Equal(t, "test", msgs[0])
	require.Equal(t, "final", msgs[1])
}

// setupStreamTest creates and sets up all the entities needed for testing streaming updates.
func setupStreamTest(t *testing.T) (
	superCtx, ctx context.Context,
	testUser model.User,
	ps *PublisherSet,
	socket *mockSocket,
	pgDB *db.PgDB,
	dbCleanup func(),
) {
	superCtx = context.TODO()
	ctx = context.TODO()
	testUser = model.User{Username: uuid.New().String()}
	pgDB, dbCleanup = db.MustResolveNewPostgresDatabase(t)
	ps = NewPublisherSet(pgDB.URL)
	socket = newMockSocket()

	return superCtx, ctx, testUser, ps, socket, pgDB, dbCleanup
}

type startupTestCase struct {
	description       string
	startupMsg        StartupMsg
	expectedSync      string
	expectedUpserts   []string
	expectedDeletions []string
}

// basicStartupTest sends a startup message and validates the result against the test case.
func basicStartupTest(t *testing.T, testCase startupTestCase, socket *mockSocket) {
	// write startup message
	socket.WriteOutbound(t, &testCase.startupMsg)

	// read messages collected during startup + sync msg
	var data []string
	socket.ReadUntil(t, &data, testCase.expectedSync)
	deletions, upserts, syncs := splitMsgs(t, data)
	require.Len(t, syncs, 1)

	// confirm these messages are the expected results
	validateMsgs(
		t,
		syncs[0],
		testCase.expectedSync,
		upserts,
		testCase.expectedUpserts,
		deletions,
		testCase.expectedDeletions,
	)
}

func TestTrialStartup(t *testing.T) {
	testCases := []startupTestCase{
		{
			description: "trial subscription with experiment id and known trials",
			startupMsg: StartupMsg{
				SyncID: "1",
				Known: KnownKeySet{
					Trials: "1,2,3",
				},
				Subscribe: SubscriptionSpecSet{
					Trials: &TrialSubscriptionSpec{
						ExperimentIds: []int{1}, // trials 1,2,3 exist in experiment 1
						Since:         0,
					},
				},
			},
			expectedSync:      "sync_msg: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"trials_deleted: "},
		},
		{
			description: "trial subscription with experiment id and incomplete known trials",
			startupMsg: StartupMsg{
				SyncID: "2",
				Known: KnownKeySet{
					Trials: "1,2,4", // 3 is not known, and 4 does not exist
				},
				Subscribe: SubscriptionSpecSet{
					Trials: &TrialSubscriptionSpec{
						ExperimentIds: []int{1},
						Since:         0,
					},
				},
			},
			expectedSync:      "sync_msg: 2",
			expectedUpserts:   []string{"trial (3): ERROR 1 1"},
			expectedDeletions: []string{"trials_deleted: 4"},
		},
		{
			description: "trial subscription with trial ids and known trials",
			startupMsg: StartupMsg{
				SyncID: "3",
				Known: KnownKeySet{
					Trials: "1,2,3,4",
				},
				Subscribe: SubscriptionSpecSet{
					Trials: &TrialSubscriptionSpec{
						TrialIds: []int{1, 2, 3, 4}, // Subscribe to all known trials, but 4 doesn't exist
						Since:    0,
					},
				},
			},
			expectedSync:      "sync_msg: 3",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"trials_deleted: 4"},
		},
		{
			description: "trial subscription with trial ids and incomplete known trials",
			startupMsg: StartupMsg{
				SyncID: "4",
				Known: KnownKeySet{
					Trials: "1,2,4", // 3 is not known, and 4 does not exist
				},
				Subscribe: SubscriptionSpecSet{
					Trials: &TrialSubscriptionSpec{
						TrialIds: []int{1, 2, 3, 4},
						Since:    0,
					},
				},
			},
			expectedSync:      "sync_msg: 4",
			expectedUpserts:   []string{"trial (3): ERROR 1 1"},
			expectedDeletions: []string{"trials_deleted: 4"},
		},
		{
			description: "trial subscription with divergent known set",
			startupMsg: StartupMsg{
				SyncID: "5",
				Known: KnownKeySet{
					Trials: "1,2",
				},
				Subscribe: SubscriptionSpecSet{
					Trials: &TrialSubscriptionSpec{
						TrialIds: []int{3},
					},
				},
			},
			expectedSync:      "sync_msg: 5",
			expectedUpserts:   []string{"trial (3): ERROR 1 1"},
			expectedDeletions: []string{"trials_deleted: 1-2"},
		},
	}

	// setup test environment
	superCtx, ctx, testUser, ps, socket, pgDB, dbCleanup := setupStreamTest(t)
	t.Cleanup(dbCleanup)
	errgrp := errgroupx.WithContext(ctx)
	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	// start publisher set and connect as testUser
	errgrp.Go(ps.Start)
	errgrp.Go(func(ctx context.Context) error {
		return ps.entrypoint(superCtx, ctx, testUser, socket, testPrepareFunc)
	})

	func() {
		// clean up socket & errgroup
		defer func() {
			socket.Close()
			errgrp.Cancel()
		}()

	TestLoop:
		for i := range testCases {
			select {
			case <-ctx.Done():
				break TestLoop
			default:
				t.Run(testCases[i].description, func(t *testing.T) {
					basicStartupTest(t, testCases[i], socket)
				})
			}
		}
	}()

	require.NoError(t, errgrp.Wait())
}

type updateTestCase struct {
	startupCase       startupTestCase
	description       string
	queries           []streamdata.ExecutableQuery
	expectedUpserts   []string
	expectedDeletions []string
	terminationMsg    string
}

// basicUpdateTest runs startup case, executed provided queries, and validates the results.
func basicUpdateTest(
	ctx context.Context,
	t *testing.T,
	testCase updateTestCase,
	socket *mockSocket,
) {
	basicStartupTest(t, testCase.startupCase, socket)

	// execute provided queries on the db
	for i := range testCase.queries {
		_, err := testCase.queries[i].Exec(ctx)
		if err != nil {
			t.Errorf("%v failed to execute", testCase.queries)
		}
	}

	// read until we received the expected message
	data := []string{}
	socket.ReadUntil(t, &data, testCase.terminationMsg)
	deletions, upserts, _ := splitMsgs(t, data)

	// validate messages collected at startup
	validateMsgs(
		t, "", "", // no sync message expected
		upserts,
		testCase.expectedUpserts,
		deletions,
		testCase.expectedDeletions,
	)
}

func TestTrialUpdate(t *testing.T) {
	superCtx, ctx, testUser, ps, socket, pgDB, dbCleanup := setupStreamTest(t)
	t.Cleanup(dbCleanup)

	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				startupMsg: StartupMsg{
					SyncID: "1",
					Known: KnownKeySet{
						Trials: "1,2,3",
					},
					Subscribe: SubscriptionSpecSet{
						Trials: &TrialSubscriptionSpec{
							ExperimentIds: []int{1},
							Since:         0,
						},
					},
				},
				expectedSync:      "sync_msg: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"trials_deleted: "},
			},
			description: "update trial while subscribed to its events",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("UPDATE trials SET state = 'CANCELED' WHERE id = 1"),
			},
			expectedUpserts:   []string{"trial (1): CANCELED 1 0"},
			expectedDeletions: []string{}, // we don't expect any deletion messages after startup
			terminationMsg:    "trial (1): CANCELED 1 0",
		},
	}

	// run migrations
	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	// start publisher set and connect as testUser
	errgrp := errgroupx.WithContext(ctx)
	errgrp.Go(ps.Start)
	errgrp.Go(func(ctx context.Context) error {
		return ps.entrypoint(superCtx, ctx, testUser, socket, testPrepareFunc)
	})

	func() {
		// clean up socket & errgroup
		defer func() {
			socket.Close()
			errgrp.Cancel()
		}()

		for i := range testCases {
			t.Run(
				testCases[i].description,
				func(t *testing.T) {
					basicUpdateTest(ctx, t, testCases[i], socket)
				},
			)
		}
	}()

	require.NoError(t, errgrp.Wait())
}
