//go:build integration
// +build integration

package stream

import (
	"context"
	"testing"
	"time"

	"github.com/uptrace/bun"

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

func runStartupTest(t *testing.T, testCases []startupTestCase) {
	// setup test environment
	superCtx := context.TODO()
	ctx := context.TODO()
	testUser := model.User{Username: uuid.New().String()}
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	ps := NewPublisherSet(pgDB.URL)
	socket := newMockSocket()

	t.Cleanup(dbCleanup)
	errgrp := errgroupx.WithContext(ctx)
	trials := streamdata.GenerateStreamData()
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

func buildStartupMsg(syncID, knownType, known string, expIDs, trialIDs []int) StartupMsg {
	var knownKeySet KnownKeySet
	var subscriptionSpecSet SubscriptionSpecSet

	switch knownType {
	case "trials":
		knownKeySet.Trials = known
		subscriptionSpecSet.Trials = &TrialSubscriptionSpec{
			TrialIds:      trialIDs,
			ExperimentIds: expIDs,
			Since:         0,
		}
	case "experiments":
		knownKeySet.Experiments = known
		subscriptionSpecSet.Experiments = &ExperimentSubscriptionSpec{
			ExperimentIds: expIDs,
			Since:         0,
		}
	case "checkpoints":
		knownKeySet.Checkpoints = known
		subscriptionSpecSet.Checkpoints = &CheckpointSubscriptionSpec{
			TrialIDs:      trialIDs,
			ExperimentIDs: expIDs,
			Since:         0,
		}
	}
	return StartupMsg{
		SyncID:    syncID,
		Known:     knownKeySet,
		Subscribe: subscriptionSpecSet,
	}
}

func TestTrialStartup(t *testing.T) {
	trialUpsert := "key: trial, trial_id: 3, state: ERROR, experiment_id: 1, workspace_id: 2"

	testCases := []startupTestCase{
		{
			description:       "trial subscription with known trials",
			startupMsg:        buildStartupMsg("1", "trials", "1,2,3", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: trials_deleted, deleted: "},
		},
		{
			description:       "trial subscription with incomplete known trials",
			startupMsg:        buildStartupMsg("2", "trials", "1,2,4", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 2",
			expectedUpserts:   []string{trialUpsert},
			expectedDeletions: []string{"key: trials_deleted, deleted: 4"},
		},
		{
			description:       "trial subscription with trial ids subscription and known trials",
			startupMsg:        buildStartupMsg("3", "trials", "1,2,3,4", []int{}, []int{1, 2, 3, 4}),
			expectedSync:      "key: sync_msg, sync_id: 3",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: trials_deleted, deleted: 4"},
		},
		{
			description:       "trial subscription with trial ids subscription and incomplete known trials",
			startupMsg:        buildStartupMsg("4", "trials", "1,2,4", []int{}, []int{1, 2, 3, 4}),
			expectedSync:      "key: sync_msg, sync_id: 4",
			expectedUpserts:   []string{trialUpsert},
			expectedDeletions: []string{"key: trials_deleted, deleted: 4"},
		},
		{
			description:       "trial subscription with divergent known set and subscription",
			startupMsg:        buildStartupMsg("5", "trials", "1,2", []int{}, []int{3}),
			expectedSync:      "key: sync_msg, sync_id: 5",
			expectedUpserts:   []string{trialUpsert},
			expectedDeletions: []string{"key: trials_deleted, deleted: 1-2"},
		},
	}

	runStartupTest(t, testCases)
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
	t.Run(testCase.startupCase.description, func(t *testing.T) {
		basicStartupTest(t, testCase.startupCase, socket)
	})
	// execute provided queries on the db
	for i := range testCase.queries {
		_, err := testCase.queries[i].Exec(ctx)
		if err != nil {
			t.Errorf("%v failed to execute", testCase.queries)
		}
	}

	// read until we received the expected message
	data := []string{}
	socket.ReadUntilFound(t, &data, append(testCase.expectedUpserts, testCase.expectedDeletions...))
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

func runUpdateTest(t *testing.T, pgDB *db.PgDB, testCases []updateTestCase) {
	// setup test environment
	superCtx := context.TODO()
	ctx := context.TODO()
	testUser := model.User{Username: uuid.New().String()}
	ps := NewPublisherSet(pgDB.URL)
	socket := newMockSocket()

	// run migrations
	trials := streamdata.GenerateStreamData()
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

func TestTrialUpdate(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	trialToInsert := streamdata.Trial{
		ID:           4,
		ExperimentID: 1,
		State:        model.ErrorState,
		StartTime:    time.Now(),
	}
	taskJobID := model.JobID("test_job1")
	taskToInsert := model.Task{
		TaskID:    "1.4",
		JobID:     &taskJobID,
		TaskType:  "TRIAL",
		StartTime: time.Now(),
	}

	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description:       "startup case for: update trial while subscribed to its events",
				startupMsg:        buildStartupMsg("1", "trials", "1,2,3", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: trials_deleted, deleted: "},
			},
			description: "update trial while subscribed to its events",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("UPDATE trials SET state = 'CANCELED' WHERE id = 1"),
			},
			// TODO (bugfix): expected upsert has workspace_id 0 because workspace_id is not being populated on db modify
			expectedUpserts:   []string{"key: trial, trial_id: 1, state: CANCELED, experiment_id: 1, workspace_id: 0"},
			expectedDeletions: []string{},
			terminationMsg:    "key: trial, trial_id: 1, state: CANCELED, experiment_id: 1, workspace_id: 0",
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: insert trial while subscribed to its events",
				startupMsg:        buildStartupMsg("1", "trials", "1,2,3", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: trials_deleted, deleted: "},
			},
			description:       "insert trial while subscribed to its events",
			queries:           streamdata.GetAddTrialQueries(&taskToInsert, &trialToInsert),
			expectedUpserts:   []string{"key: trial, trial_id: 4, state: ERROR, experiment_id: 1, workspace_id: 0"},
			expectedDeletions: []string{}, // we don't expect any deletion messages after startup
			terminationMsg:    "key: trial, trial_id: 4, state: ERROR, experiment_id: 1, workspace_id: 0",
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: delete trial while subscribed to its events",
				startupMsg:        buildStartupMsg("1", "trials", "1,2,3,4", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: trials_deleted, deleted: "},
			},
			description: "delete trial while subscribed to its events",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("DELETE FROM trials WHERE id = 4"),
			},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: trials_deleted, deleted: 4"},
			terminationMsg:    "key: trials_deleted, deleted: 4",
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: change experiment project",
				startupMsg:        buildStartupMsg("1", "trials", "1,2,3,4", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: trials_deleted, deleted: 4"},
			},
			description: "change experiment project",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("UPDATE projects SET workspace_id = 1 WHERE name = 'test_project1'"),
			},
			expectedUpserts: []string{
				"key: trial, trial_id: 1, state: CANCELED, experiment_id: 1, workspace_id: 0",
				"key: trial, trial_id: 2, state: ERROR, experiment_id: 1, workspace_id: 0",
				"key: trial, trial_id: 3, state: ERROR, experiment_id: 1, workspace_id: 0",
			},
			expectedDeletions: []string{},
			terminationMsg:    "key: trial, trial_id: 3, state: ERROR, experiment_id: 1, workspace_id: 0",
		},
	}

	runUpdateTest(t, pgDB, testCases)
}

func TestCheckpointStartup(t *testing.T) {
	checkpointUpsert := "key: checkpoint, checkpoint_id: 2, state: COMPLETED, experiment_id: 1, workspace_id: 2"

	testCases := []startupTestCase{
		{
			description:       "checkpoint subscription with known checkpoints",
			startupMsg:        buildStartupMsg("1", "checkpoints", "1,2", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: "},
		},
		{
			description:       "checkpoint subscription with experiment id and known checkpoints",
			startupMsg:        buildStartupMsg("2", "checkpoints", "1,2,3", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 2",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 3"},
		},
		{
			description:       "checkpoint subscription with trial ids and known checkpoints",
			startupMsg:        buildStartupMsg("3", "checkpoints", "1,2,3", []int{}, []int{1, 2, 3}),
			expectedSync:      "key: sync_msg, sync_id: 3",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 3"},
		},
		{
			description:       "checkpoint subscription with incomplete known set",
			startupMsg:        buildStartupMsg("4", "checkpoints", "1,3", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 4",
			expectedUpserts:   []string{checkpointUpsert},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 3"},
		},
		{
			description:       "checkpoint subscription with incomplete known set using trial IDs",
			startupMsg:        buildStartupMsg("5", "checkpoints", "1,3", []int{}, []int{1, 2, 3}),
			expectedSync:      "key: sync_msg, sync_id: 5",
			expectedUpserts:   []string{checkpointUpsert},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 3"},
		},
		{
			description:       "trial subscription with divergent known set and subscription",
			startupMsg:        buildStartupMsg("6", "checkpoints", "1", []int{}, []int{2}),
			expectedSync:      "key: sync_msg, sync_id: 6",
			expectedUpserts:   []string{checkpointUpsert},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 1"},
		},
	}

	runStartupTest(t, testCases)
}

func TestCheckpointUpdate(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	baseStartupCase := startupTestCase{
		startupMsg:        buildStartupMsg("1", "checkpoints", "1", []int{1}, []int{}),
		expectedSync:      "key: sync_msg, sync_id: 1",
		expectedUpserts:   []string{"key: checkpoint, checkpoint_id: 2, state: COMPLETED, experiment_id: 1, workspace_id: 2"},
		expectedDeletions: []string{"key: checkpoints_deleted, deleted: "},
	}

	modCheckpoint := streamdata.Checkpoint{
		BaseModel:  bun.BaseModel{},
		ID:         1,
		State:      model.DeletedState,
		ReportTime: time.Time{},
	}

	newCheckpoint := model.CheckpointV2{
		UUID:       uuid.New(),
		TaskID:     model.TaskID("1.3"),
		ReportTime: time.Now(),
		State:      model.CompletedState,
	}

	testCases := []updateTestCase{
		{
			startupCase:       baseStartupCase,
			description:       "update checkpoint while subscribed to its events",
			queries:           []streamdata.ExecutableQuery{streamdata.GetUpdateCheckpointQuery(modCheckpoint)},
			expectedUpserts:   []string{"key: checkpoint, checkpoint_id: 1, state: DELETED, experiment_id: 1, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: insert checkpoint while subscribed to its events",
				startupMsg:        buildStartupMsg("2", "checkpoints", "1,2", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 2",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: checkpoints_deleted, deleted: "},
			},
			description: "insert checkpoint while subscribed to its events",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewInsert().Model(&newCheckpoint),
			},
			expectedUpserts: []string{
				"key: checkpoint, checkpoint_id: 3, state: COMPLETED, experiment_id: 1, workspace_id: 2",
			},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: delete checkpoint while subscribed to its events",
				startupMsg:        buildStartupMsg("3", "checkpoints", "1,2,3", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 3",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: checkpoints_deleted, deleted: "},
			},
			description: "delete checkpoint while subscribed to its events",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("DELETE FROM checkpoints_v2 WHERE id = 3"),
			},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: checkpoints_deleted, deleted: 3"},
		},
		{
			startupCase: startupTestCase{
				description:       "startup case for: change experiment project",
				startupMsg:        buildStartupMsg("4", "checkpoints", "1,2", []int{1}, []int{}),
				expectedSync:      "key: sync_msg, sync_id: 4",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: checkpoints_deleted, deleted: "},
			},
			description: "change experiment project",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewRaw("UPDATE projects SET workspace_id = 1 WHERE name = 'test_project1'"),
			},
			expectedUpserts: []string{
				// "key: checkpoint, checkpoint_id: 1, state: COMPLETED, experiment_id: 1, workspace_id: 2",
				// "key: checkpoint, checkpoint_id: 2, state: COMPLETED, experiment_id: 1, workspace_id: 2",
			},
			expectedDeletions: []string{},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}

func TestExperimentStartup(t *testing.T) {
	expUpsertString := "key: experiment, exp_id: 2, state: ERROR, project_id: 2, job_id: test_job2"
	testCases := []startupTestCase{
		{
			description:       "experiment subscription with experiment id",
			startupMsg:        buildStartupMsg("1", "experiments", "1", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: experiments_deleted, deleted: "},
		},
		{
			description:       "experiment subscription with extra known experiments",
			startupMsg:        buildStartupMsg("2", "experiments", "1,3,4", []int{1}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 2",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: experiments_deleted, deleted: 3-4"},
		},
		{
			description:       "experiment subscription with incomplete known experiments",
			startupMsg:        buildStartupMsg("3", "experiments", "1,4", []int{1, 2, 3, 4}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 3",
			expectedUpserts:   []string{expUpsertString},
			expectedDeletions: []string{"key: experiments_deleted, deleted: 4"},
		},
		{
			description:       "experiment subscription with divergent known set",
			startupMsg:        buildStartupMsg("4", "experiments", "1", []int{2}, []int{}),
			expectedSync:      "key: sync_msg, sync_id: 4",
			expectedUpserts:   []string{expUpsertString},
			expectedDeletions: []string{"key: experiments_deleted, deleted: 1"},
		},
	}

	runStartupTest(t, testCases)
}

func TestExperimentUpdate(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	baseStartupCase := startupTestCase{
		startupMsg: StartupMsg{
			SyncID: "1",
			Known: KnownKeySet{
				Experiments: "1",
			},
			Subscribe: SubscriptionSpecSet{
				Experiments: &ExperimentSubscriptionSpec{
					ExperimentIds: []int{1},
					Since:         0,
				},
			},
		},
		expectedSync:      "key: sync_msg, sync_id: 1",
		expectedUpserts:   []string{},
		expectedDeletions: []string{"key: experiments_deleted, deleted: "},
	}

	newExpStartupCase := startupTestCase{
		startupMsg: StartupMsg{
			SyncID: "2",
			Known: KnownKeySet{
				Experiments: "1",
			},
			Subscribe: SubscriptionSpecSet{
				Experiments: &ExperimentSubscriptionSpec{
					ExperimentIds: []int{1, 3},
					Since:         0,
				},
			},
		},
		expectedSync:      "key: sync_msg, sync_id: 2",
		expectedUpserts:   []string{},
		expectedDeletions: []string{"key: experiments_deleted, deleted: "},
	}

	deleteStartupCase := startupTestCase{
		startupMsg: StartupMsg{
			SyncID: "3",
			Known: KnownKeySet{
				Experiments: "1,2,3",
			},
			Subscribe: SubscriptionSpecSet{
				Experiments: &ExperimentSubscriptionSpec{
					ExperimentIds: []int{1, 2, 3},
					Since:         0,
				},
			},
		},
		expectedSync:      "key: sync_msg, sync_id: 3",
		expectedUpserts:   []string{},
		expectedDeletions: []string{"key: experiments_deleted, deleted: "},
	}

	uid := 1
	canceledExperiment := streamdata.Experiment{
		ID:                   1,
		JobID:                "test_job2",
		ModelDefinitionBytes: []byte{},
		OwnerID:              (*model.UserID)(&uid),
		State:                model.CanceledState,
		ProjectID:            2,
	}
	newExperiment3 := streamdata.Experiment{
		ID:                   3,
		JobID:                "test_job2",
		ModelDefinitionBytes: []byte{},
		OwnerID:              (*model.UserID)(&uid),
		State:                model.CanceledState,
		ProjectID:            2,
	}
	newExperiment4 := streamdata.Experiment{
		ID:                   4,
		JobID:                "test_job2",
		ModelDefinitionBytes: []byte{},
		OwnerID:              (*model.UserID)(&uid),
		State:                model.CanceledState,
		ProjectID:            2,
	}

	updateExpString := "key: experiment, exp_id: 1, state: CANCELED, project_id: 2, job_id: test_job2"

	testCases := []updateTestCase{
		{
			startupCase:       baseStartupCase,
			description:       "update experiment while subscribed to its events",
			queries:           []streamdata.ExecutableQuery{streamdata.GetUpdateExperimentQuery(canceledExperiment)},
			expectedUpserts:   []string{updateExpString},
			expectedDeletions: []string{},
		},
		{
			startupCase: newExpStartupCase,
			description: "add an experiment while subscribed to it",
			queries:     []streamdata.ExecutableQuery{streamdata.GetAddExperimentQuery(&newExperiment3)},
			expectedUpserts: []string{
				"key: experiment, exp_id: 3, state: CANCELED, project_id: 2, job_id: test_job2",
			},
			expectedDeletions: []string{},
		},
		{
			startupCase: baseStartupCase,
			description: "add an experiment while not subscribed to it and update another",
			queries: []streamdata.ExecutableQuery{
				streamdata.GetAddExperimentQuery(&newExperiment4),
				streamdata.GetUpdateExperimentQuery(canceledExperiment),
			},
			expectedUpserts:   []string{updateExpString},
			expectedDeletions: []string{},
		},
		{
			startupCase: deleteStartupCase,
			description: "delete experiment 3",
			queries: []streamdata.ExecutableQuery{
				streamdata.GetDeleteExperimentQuery(newExperiment3.ID),
			},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: experiments_deleted, deleted: 3"},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}
