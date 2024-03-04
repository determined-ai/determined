//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/master/test/streamdata"
)

const (
	projects = "projects"
)

func TestMockSocket(t *testing.T) {
	expectedMsg := StartupMsg{
		SyncID: uuid.NewString(),
		Known:  KnownKeySet{Projects: "1,2,3"},
		Subscribe: SubscriptionSpecSet{
			Projects: &ProjectSubscriptionSpec{
				ProjectIDs: []int{1},
				Since:      0,
			},
		},
	}

	// test WriteToServer
	socket := newMockSocket()
	socket.WriteToServer(t, &expectedMsg)

	// test ReadJSON
	actualMsg := StartupMsg{}
	err := socket.ReadJSON(&actualMsg)
	require.NoError(t, err)
	require.Equal(t, actualMsg.Known, expectedMsg.Known)
	require.Equal(t, actualMsg.Subscribe, expectedMsg.Subscribe)
	require.Equal(t, actualMsg.SyncID, expectedMsg.SyncID)
	require.Equal(t, 0, len(socket.toServer))

	// test write
	err = socket.Write("test")
	require.NoError(t, err)

	// test read incoming
	data := socket.ReadFromServer(t)
	require.Equal(t, "test", data)

	// test ReadUntil
	err = socket.Write("test")
	require.NoError(t, err)
	err = socket.Write("final")
	require.NoError(t, err)
	msgs := socket.ReadUntilFound(t, "final")
	require.NoError(t, err)
	require.Equal(t, 2, len(msgs))
	require.Equal(t, "test", msgs[0])
	require.Equal(t, "final", msgs[1])
}

// initializeStreamDB initializes a postgres database, performs current migrations and populates it with test data.
func initializeStreamDB(ctx context.Context, t *testing.T) *db.PgDB {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	_, err := db.Bun().NewRaw(
		`INSERT INTO workspaces (name) VALUES ('test_workspace');
		INSERT INTO projects (name, workspace_id) VALUES ('test_project_1', 2);
		`,
	).Exec(ctx)
	if err != nil {
		t.Errorf("failed to generate test data for streaming integration test: %s", err)
	}
	return pgDB
}

type startupTestCase struct {
	description       string
	startupMsg        StartupMsg
	expectedUpserts   []string
	expectedDeletions []string
}

// basicStartupTest sends a startup message and validates the result against the test case.
func basicStartupTest(t *testing.T, testCase startupTestCase, socket *mockSocket) {
	// write startup message
	socket.WriteToServer(t, &testCase.startupMsg)

	// read messages collected during startup + sync msg.

	// constructed expected sync messages based on startup message.
	baseSyncMsg := fmt.Sprintf("type: sync_msg, sync_id: %s", testCase.startupMsg.SyncID)
	expectedSyncs := []string{
		baseSyncMsg + ", complete: false",
		baseSyncMsg + ", complete: true",
	}
	data := socket.ReadUntilFound(t, expectedSyncs...)

	deletions, upserts, syncs := splitMsgs(t, data)
	// confirm these messages are the expected results
	validateMsgs(
		t,
		syncs,
		expectedSyncs,
		upserts,
		testCase.expectedUpserts,
		deletions,
		testCase.expectedDeletions,
	)
}

func runStartupTest(t *testing.T, pgDB *db.PgDB, testCases []startupTestCase) {
	// setup test environment
	superCtx := context.TODO()
	ctx := context.TODO()
	testUser := model.User{Username: uuid.New().String()}

	// setup and populate DB
	ps := NewPublisherSet(pgDB.URL)
	socket := newMockSocket()
	errgrp := errgroupx.WithContext(ctx)

	// start PublisherSet and connect as testUser
	errgrp.Go(ps.Run)
	errgrp.Go(func(ctx context.Context) error {
		return ps.streamHandler(superCtx, ctx, testUser, socket, testPrepareFunc)
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
		case projects:
			knownKeySet.Projects = known
		}
	}

	// populate subscriptionSpec
	for subscriptionType, subscriptionIDs := range subscriptionsMap {
		switch subscriptionType {
		case projects:
			subscriptionSpecSet.Projects = &ProjectSubscriptionSpec{
				ProjectIDs:   subscriptionIDs[projects],
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
	var expected []string
	expected = append(expected, testCase.expectedUpserts...)
	expected = append(expected, testCase.expectedDeletions...)
	data := socket.ReadUntilFound(t, expected...)
	deletions, upserts, _ := splitMsgs(t, data)

	// validate messages collected at startup
	validateMsgs(
		t,
		[]string{}, []string{}, // no sync message expected
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
	socket := newMockSocket()

	// create a new publisher set
	ps := NewPublisherSet(pgDB.URL)

	// start publisher set and connect as testUser
	errgrp := errgroupx.WithContext(ctx)
	errgrp.Go(ps.Run)
	errgrp.Go(func(ctx context.Context) error {
		return ps.streamHandler(superCtx, ctx, testUser, socket, testPrepareFunc)
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
	pgDB := initializeStreamDB(context.Background(), t)
	testCases := []startupTestCase{
		{
			description: "project subscription with project id",
			startupMsg: buildStartupMsg(
				"1",
				map[string]string{projects: "1,2"},
				map[string]map[string][]int{projects: {projects: {1, 2}}},
			),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: "},
		},
		{
			description: "project subscription with excess project id",
			startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2,3"},
				map[string]map[string][]int{projects: {projects: {1, 2, 3}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: 3"},
		},
		{
			description: "project subscription with workspaces",
			startupMsg: buildStartupMsg("3", map[string]string{projects: "1,2"},
				map[string]map[string][]int{projects: {projects: {1, 2}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: "},
		},
		{
			description: "project offline fall out",
			startupMsg: buildStartupMsg("4", map[string]string{projects: "1,2"},
				map[string]map[string][]int{projects: {projects: {1}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: 2"},
		},
		{
			description: "project offline fall in",
			startupMsg: buildStartupMsg("5", map[string]string{projects: "1"},
				map[string]map[string][]int{projects: {projects: {1, 2}}}),
			expectedUpserts:   []string{"type: project, project_id: 2, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{"type: projects_deleted, deleted: "},
		},
	}

	runStartupTest(t, pgDB, testCases)
}

func TestProjectUpdate(t *testing.T) {
	pgDB := initializeStreamDB(context.Background(), t)
	testProject := model.Project{
		Name:        uuid.NewString(),
		CreatedAt:   time.Now(),
		Archived:    false,
		WorkspaceID: 2,
		UserID:      1,
		State:       "UNSPECIFIED",
	}

	projectMod := model.Project{
		ID:          3,
		WorkspaceID: 1,
	}

	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup case for: create project 3",
				startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2"},
					map[string]map[string][]int{projects: {projects: {1, 2, 3}}}),
				expectedUpserts:   []string{},
				expectedDeletions: []string{"type: projects_deleted, deleted: "},
			},
			description:       "create project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(testProject)},
			expectedUpserts:   []string{"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: update project 3",
				startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2"},
					map[string]map[string][]int{projects: {projects: {1, 2, 3}}}),
				expectedUpserts:   []string{"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
				expectedDeletions: []string{"type: projects_deleted, deleted: "},
			},
			description: "update project 3",
			queries:     []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(projectMod)},
			expectedUpserts: []string{
				"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1",
			},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: delete project 3",
				startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2"},
					map[string]map[string][]int{projects: {projects: {1, 2, 3}}}),
				expectedUpserts:   []string{"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1"},
				expectedDeletions: []string{"type: projects_deleted, deleted: "},
			},
			description:       "delete project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetDeleteProjectQuery(projectMod)},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: 3"},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}

func TestOnlineChanges(t *testing.T) {
	pgDB := initializeStreamDB(context.Background(), t)
	testProject := model.Project{
		Name:        uuid.NewString(),
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
					map[string]string{projects: "2"},
					map[string]map[string][]int{projects: {"workspaces": {2}}},
				),
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"type: projects_deleted, deleted: ",
				},
			},
			description:       "online create project",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(testProject)},
			expectedUpserts:   []string{"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup test case for: online fall out project",
				startupMsg: buildStartupMsg(
					"4",
					map[string]string{projects: "2,3"},
					map[string]map[string][]int{projects: {"workspaces": {2}}},
				),
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"type: projects_deleted, deleted: ",
				},
			},
			description: "online fall out project",
			queries: []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(model.Project{
				ID:          3,
				WorkspaceID: 1,
			})},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"type: projects_deleted, deleted: 3"},
		},
		{
			startupCase: startupTestCase{
				description: "startup test case for: online fall in project",
				startupMsg: buildStartupMsg(
					"5",
					map[string]string{projects: "2"},
					map[string]map[string][]int{projects: {"workspaces": {2}}},
				),
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"type: projects_deleted, deleted: ",
				},
			},
			description: "online fall in project",
			queries: []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(model.Project{
				ID:          3,
				WorkspaceID: 2,
			})},
			expectedUpserts:   []string{"type: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
	}

	runUpdateTest(t, pgDB, testCases)
}
