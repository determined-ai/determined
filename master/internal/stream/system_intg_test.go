//go:build integration
// +build integration

package stream

import (
	"context"
	"testing"
	"time"

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
			Projects: &ProjectSubscriptionSpec{
				ProjectIDs: []int{1},
				Since:      0,
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
	socket.ReadUntilFound(t, &msgs, []string{"final"})
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
	socket.ReadUntilFound(t, &data, []string{testCase.expectedSync})

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

func buildStartupMsg(
	syncID string,
	knownsMap map[string]string,
	subscriptionsMap map[string]map[string][]int,
) StartupMsg {
	var knownKeySet KnownKeySet
	var subscriptionSpecSet SubscriptionSpecSet

	// populate knownKeySet
	for knownType, known := range knownsMap {
		switch knownType {
		case ProjectsUpsertKey:
			knownKeySet.Projects = known
		}
	}

	// populate subscriptionSpec
	for subscriptionType, subscriptionIDs := range subscriptionsMap {
		switch subscriptionType {
		case ProjectsUpsertKey:
			subscriptionSpecSet.Projects = &ProjectSubscriptionSpec{
				ProjectIDs:   subscriptionIDs[ProjectsUpsertKey],
				WorkspaceIDs: subscriptionIDs["workspaces"],
				Since:        0,
			}
		}
	}

	return StartupMsg{
		SyncID:    syncID,
		Known:     knownKeySet,
		Subscribe: subscriptionSpecSet,
	}
}

type updateTestCase struct {
	startupCase       startupTestCase
	description       string
	queries           []streamdata.ExecutableQuery
	expectedUpserts   []string
	expectedDeletions []string
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
	t.Logf("Read and split")
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

func TestProjectStartup(t *testing.T) {
	testCases := []startupTestCase{
		{
			description: "project subscription with project id",
			startupMsg: buildStartupMsg("1", map[string]string{ProjectsUpsertKey: "1,2"},
				map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2}}}),
			expectedSync:      "key: sync_msg, sync_id: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
		},
		{
			description: "project subscription with excess project id",
			startupMsg: buildStartupMsg("1", map[string]string{ProjectsUpsertKey: "1,2,3"},
				map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2, 3}}}),
			expectedSync:      "key: sync_msg, sync_id: 1",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 3"},
		},
		{
			description: "project subscription with workspaces",
			startupMsg: buildStartupMsg("3", map[string]string{ProjectsUpsertKey: "1,2"},
				map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2}}}),
			expectedSync:      "key: sync_msg, sync_id: 3",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
		},
		{
			description: "project subscription with incomplete workspaces",
			startupMsg: buildStartupMsg("4", map[string]string{ProjectsUpsertKey: "1,2"},
				map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1}}}),
			expectedSync:      "key: sync_msg, sync_id: 4",
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 2"},
		},
		{
			description: "project subscription with incomplete workspaces",
			startupMsg: buildStartupMsg("5", map[string]string{ProjectsUpsertKey: "1"},
				map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2}}}),
			expectedSync:      "key: sync_msg, sync_id: 5",
			expectedUpserts:   []string{"key: project, project_id: 2, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
		},
	}

	runStartupTest(t, testCases)
}

func TestProjectUpdate(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	newProject3 := model.Project{
		Name:        "test project 3",
		CreatedAt:   time.Now(),
		Archived:    false,
		WorkspaceID: 2,
		UserID:      1,
		State:       "UNSPECIFIED",
	}

	project3Mod := model.Project{
		ID:          3,
		WorkspaceID: 1,
	}

	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup case for: create project 3",
				startupMsg: buildStartupMsg("1", map[string]string{ProjectsUpsertKey: "1,2"},
					map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2, 3}}}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description:       "create project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(newProject3)},
			expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: update project 3",
				startupMsg: buildStartupMsg("1", map[string]string{ProjectsUpsertKey: "1,2"},
					map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2, 3}}}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description: "update project 3",
			queries:     []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(project3Mod)},
			expectedUpserts: []string{
				"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1",
			},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: delete project 3",
				startupMsg: buildStartupMsg("1", map[string]string{ProjectsUpsertKey: "1,2"},
					map[string]map[string][]int{ProjectsUpsertKey: {ProjectsUpsertKey: {1, 2, 3}}}),
				expectedSync:      "key: sync_msg, sync_id: 1",
				expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1"},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description:       "delete project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetDeleteProjectQuery(project3Mod)},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 3"},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}

func TestUpdatesOutOfSpec(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	testCases := []updateTestCase{}

	runUpdateTest(t, pgDB, testCases)
}

func TestOfflineChanges(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	testCases := []updateTestCase{}

	runUpdateTest(t, pgDB, testCases)
}

func TestOnlineChanges(t *testing.T) {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)

	newProject3 := model.Project{
		Name:        "test project 3",
		CreatedAt:   time.Now(),
		Archived:    false,
		WorkspaceID: 2,
		UserID:      1,
		State:       "UNSPECIFIED",
	}

	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup test case for: online create project",
				startupMsg: buildStartupMsg(
					"3",
					map[string]string{ProjectsUpsertKey: "2"},
					map[string]map[string][]int{ProjectsUpsertKey: {"workspaces": {2}}},
				),
				expectedSync:    "key: sync_msg, sync_id: 3",
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"key: projects_deleted, deleted: ",
				},
			},
			description:       "online create project",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(newProject3)},
			expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup test case for: online fall out project",
				startupMsg: buildStartupMsg(
					"4",
					map[string]string{ProjectsUpsertKey: "2,3"},
					map[string]map[string][]int{ProjectsUpsertKey: {"workspaces": {2}}},
				),
				expectedSync:    "key: sync_msg, sync_id: 4",
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"key: projects_deleted, deleted: ",
				},
			},
			description: "online fall out project",
			queries: []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(model.Project{
				ID:          3,
				WorkspaceID: 1,
			})},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 3"},
		},
		{
			startupCase: startupTestCase{
				description: "startup test case for: online fall in project",
				startupMsg: buildStartupMsg(
					"5",
					map[string]string{ProjectsUpsertKey: "2"},
					map[string]map[string][]int{ProjectsUpsertKey: {"workspaces": {2}}},
				),
				expectedSync:    "key: sync_msg, sync_id: 5",
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"key: projects_deleted, deleted: ",
				},
			},
			description: "online fall in project",
			queries: []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(model.Project{
				ID:          3,
				WorkspaceID: 2,
			})},
			expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}
