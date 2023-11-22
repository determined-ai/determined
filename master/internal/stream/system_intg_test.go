//go:build integration
// +build integration

package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
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
	err := socket.WriteOutbound(&expectedMsg)
	require.NoError(t, err)

	// test ReadJSON
	actualMsg := StartupMsg{}
	err = socket.ReadJSON(&actualMsg)
	require.NoError(t, err)
	require.Equal(t, actualMsg.Known, expectedMsg.Known)
	require.Equal(t, actualMsg.Subscribe, expectedMsg.Subscribe)
	require.Equal(t, actualMsg.SyncID, expectedMsg.SyncID)
	require.Equal(t, 0, len(socket.outbound))

	// test write
	err = socket.Write("test")
	require.NoError(t, err)

	// test read incoming
	var data interface{}
	err = socket.ReadIncoming(&data)
	require.NoError(t, err)
	dataStr, ok := data.(string)
	require.True(t, ok)
	require.Equal(t, "test", dataStr)

	// test ReadUntil
	err = socket.Write("test")
	require.NoError(t, err)
	err = socket.Write(SyncMsg{SyncID: "1"})
	require.NoError(t, err)
	var msgs []interface{}
	socket.ReadUntil(t, "testing ReadUntil", &msgs, SyncMsg{SyncID: "1"})
	require.NoError(t, err)
	require.Equal(t, 2, len(msgs))
	testStr, ok := msgs[0].(string)
	require.True(t, ok)
	require.Equal(t, "test", testStr)
	syncMsg, ok := msgs[1].(SyncMsg)
	require.True(t, ok)
	require.Equal(t, SyncMsg{SyncID: "1"}, syncMsg)
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

type startupTestCase[M stream.Msg] struct {
	description       string
	startupMsg        StartupMsg
	expectedSync      SyncMsg
	expectedUpserts   []M
	expectedDeletions []string
}

func TestTrialStartup(t *testing.T) {
	testCases := []startupTestCase[*TrialMsg]{
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
			expectedSync:      SyncMsg{SyncID: "1"},
			expectedUpserts:   []*TrialMsg{},
			expectedDeletions: []string{""},
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
			expectedSync:      SyncMsg{SyncID: "2"},
			expectedUpserts:   []*TrialMsg{{ID: 3, ExperimentID: 1, State: model.ErrorState}},
			expectedDeletions: []string{"4"},
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
			expectedSync:      SyncMsg{SyncID: "3"},
			expectedUpserts:   []*TrialMsg{},
			expectedDeletions: []string{"4"},
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
			expectedSync:      SyncMsg{SyncID: "4"},
			expectedUpserts:   []*TrialMsg{{ID: 3, ExperimentID: 1, State: model.ErrorState}},
			expectedDeletions: []string{"4"},
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
			expectedSync:      SyncMsg{SyncID: "5"},
			expectedUpserts:   []*TrialMsg{{ID: 3, ExperimentID: 1, State: model.ErrorState}},
			expectedDeletions: []string{"1-2"},
		},
	}

	superCtx, ctx, testUser, ps, socket, pgDB, dbCleanup := setupStreamTest(t)
	defer dbCleanup()
	errgrp := errgroupx.WithContext(ctx)

	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	// start publisher set and connect as testUser
	errgrp.Go(ps.Start)
	errgrp.Go(func(ctx context.Context) error {
		return ps.entrypoint(superCtx, ctx, testUser, socket, simpleUpsert)
	})

	// handles each provided test case
	testBody := func(ctx context.Context, testCase startupTestCase[*TrialMsg]) error {
		// write startup message
		if err := socket.WriteOutbound(&testCase.startupMsg); err != nil {
			return fmt.Errorf("%s: %s", testCase.description, err)
		}

		// read messages collected during startup + sync msg
		var data []interface{}
		socket.ReadUntil(t, testCase.description, &data, testCase.expectedSync)
		deletions, upserts, syncs := splitMsgs[*TrialMsg](t, testCase.description, data)
		if len(syncs) != 1 {
			return fmt.Errorf("%s: did not receive expected number of upsert messages: expected %d, actual: %d",
				testCase.description,
				1,
				len(syncs),
			)
		}

		// confirm these messages are the expected results
		validateMsgs(
			t,
			testCase.description,
			syncs[0],
			testCase.expectedSync,
			upserts,
			testCase.expectedUpserts,
			deletions,
			testCase.expectedDeletions,
		)
		return nil
	}

	errgrp.Go(func(ctx context.Context) error {
		// clean up socket & errgroup
		defer func() {
			socket.Close()
			errgrp.Cancel()
		}()

		for i := range testCases {
			err := testBody(ctx, testCases[i])
			if err != nil {
				return err
			}
		}
		return nil
	},
	)
	require.NoError(t, errgrp.Wait())
}

type updateTestCase[M stream.Msg] struct {
	startupCase       startupTestCase[M]
	description       string
	queries           []streamdata.ExecutableQuery
	expectedSync      SyncMsg
	expectedUpserts   []M
	expectedDeletions []string
	terminationMsg    interface{}
}

func TestTrialUpdate(t *testing.T) {
	superCtx, ctx, testUser, ps, socket, pgDB, dbCleanup := setupStreamTest(t)
	defer dbCleanup()
	errgrp := errgroupx.WithContext(ctx)

	baseStartupCase := startupTestCase[*TrialMsg]{
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
		expectedSync:      SyncMsg{SyncID: "1"},
		expectedUpserts:   []*TrialMsg{},
		expectedDeletions: []string{""},
	}

	canceledTrial := streamdata.Trial{
		ID:           1,
		ExperimentID: 1,
		State:        model.CanceledState,
	}

	testCases := []updateTestCase[*TrialMsg]{
		{
			startupCase:  baseStartupCase,
			description:  "update trial while subscribed to its events",
			queries:      []streamdata.ExecutableQuery{streamdata.GetUpdateTrialQuery(canceledTrial)},
			expectedSync: SyncMsg{SyncID: "1"},
			expectedUpserts: []*TrialMsg{
				{
					ID:           canceledTrial.ID,
					ExperimentID: canceledTrial.ExperimentID,
					State:        canceledTrial.State,
				},
			},
			expectedDeletions: []string{},
			terminationMsg: stream.UpsertMsg{
				Msg: &TrialMsg{
					ID:           canceledTrial.ID,
					ExperimentID: canceledTrial.ExperimentID,
					State:        canceledTrial.State,
				},
			},
		},
	}

	// run migrations
	trials := streamdata.GenerateStreamTrials()
	trials.MustMigrate(t, pgDB, "file://../../static/migrations")

	// start publisher set and connect as testUser
	errgrp.Go(ps.Start)
	errgrp.Go(func(ctx context.Context) error {
		return ps.entrypoint(superCtx, ctx, testUser, socket, simpleUpsert)
	})

	testBody := func(ctx context.Context, testCase updateTestCase[*TrialMsg]) error {
		// write startup message
		if err := socket.WriteOutbound(&testCase.startupCase.startupMsg); err != nil {
			return err
		}

		// read messages collected during startup + sync msg
		var data []interface{}
		socket.ReadUntil(t, testCase.description, &data, testCase.startupCase.expectedSync)
		deletions, upserts, syncs := splitMsgs[*TrialMsg](t, testCase.description, data)
		if len(syncs) != 1 {
			return fmt.Errorf("%s: did not receive expected number of sync messages: expected %d, actual: %d",
				testCase.description,
				1,
				len(syncs),
			)
		}

		// validate messages collected at startup
		validateMsgs[*TrialMsg](
			t,
			testCase.description,
			syncs[0],
			testCase.startupCase.expectedSync,
			upserts,
			testCase.startupCase.expectedUpserts,
			deletions,
			testCase.startupCase.expectedDeletions,
		)

		// execute provided queries on the db
		for i := range testCase.queries {
			_, err := testCase.queries[i].Exec(ctx)
			if err != nil {
				return fmt.Errorf("%s: %v failed to execute", testCase.description, testCase.queries)
			}
		}

		// read until we received the expected message
		data = []interface{}{}
		socket.ReadUntil(t, testCase.description, &data, testCase.terminationMsg)
		deletions, upserts, _ = splitMsgs[*TrialMsg](t, testCase.description, data)

		// validate messages collected at startup
		validateMsgs[*TrialMsg](
			t,
			testCase.description,
			syncs[0],
			testCase.expectedSync,
			upserts,
			testCase.expectedUpserts,
			deletions,
			testCase.expectedDeletions,
		)
		return nil
	}

	errgrp.Go(
		func(ctx context.Context) error {
			// clean up socket & errgroup
			defer func() {
				socket.Close()
				errgrp.Cancel()
			}()

			for i := range testCases {
				err := testBody(ctx, testCases[i])
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
	require.NoError(t, errgrp.Wait())
}
