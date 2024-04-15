//go:build integration
// +build integration

package stream

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/test/streamdata"
)

const (
	projects      = "projects"
	models        = "models"
	modelVersions = "modelversions"
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

// initializeStreamDB initializes a postgres database, performs current migrations and populates it with test data.
func initializeStreamDB(ctx context.Context, t *testing.T) *db.PgDB {
	pgDB, dbCleanup := db.MustResolveNewPostgresDatabase(t)
	t.Cleanup(dbCleanup)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	_, err := db.Bun().NewRaw(
		`INSERT INTO workspaces (name) VALUES ('test_workspace');
		INSERT INTO projects (name, workspace_id) VALUES ('test_project_1', 2);
		INSERT INTO models (name, workspace_id, creation_time, user_id) VALUES ('test_model_1', 2, NOW(), 1);
		INSERT INTO model_versions (name, version, model_id, creation_time, user_id, checkpoint_uuid) 
		VALUES ('test_model_version_1',1, 1, NOW(), 1, 
		uuid_in(md5(random()::text || random()::text)::cstring));
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
	socket.WriteOutbound(t, &testCase.startupMsg)

	// read messages collected during startup + sync msg.
	var data []string

	// constructed expected sync messages based on startup message.
	baseSyncMsg := fmt.Sprintf("key: sync_msg, sync_id: %s", testCase.startupMsg.SyncID)
	expectedSyncs := []string{
		baseSyncMsg + ", complete: false",
		baseSyncMsg + ", complete: true",
	}
	socket.ReadUntilFound(t, &data, expectedSyncs)
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
	errgrp.Go(ps.Start)
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
	subscriptionsMap map[string]map[string]interface{},
) StartupMsg {
	var knownKeySet KnownKeySet
	var subscriptionSpecSet SubscriptionSpecSet

	// populate knownKeySet
	for knownType, known := range knownsMap {
		switch knownType {
		case projects:
			knownKeySet.Projects = known
		case models:
			knownKeySet.Models = known
		case modelVersions:
			knownKeySet.ModelVersions = known
		}
	}

	// populate subscriptionSpec
	for subscriptionType, subscriptionIDs := range subscriptionsMap {
		switch subscriptionType {
		case projects:
			var projectIDs, workspaceIDs []int
			if subscriptionIDs[projects] != nil {
				projectIDs = subscriptionIDs[projects].([]int)
			}
			if subscriptionIDs["workspaces"] != nil {
				workspaceIDs = subscriptionIDs["workspaces"].([]int)
			}
			subscriptionSpecSet.Projects = &ProjectSubscriptionSpec{
				ProjectIDs:   projectIDs,
				WorkspaceIDs: workspaceIDs,
				Since:        0,
			}
		case models:
			var modelIDs, workspaceIDs, userIDs []int
			if subscriptionIDs[models] != nil {
				modelIDs = subscriptionIDs[models].([]int)
			}
			if subscriptionIDs["workspaces"] != nil {
				workspaceIDs = subscriptionIDs["workspaces"].([]int)
			}
			if subscriptionIDs["users"] != nil {
				userIDs = subscriptionIDs["users"].([]int)
			}
			subscriptionSpecSet.Models = &ModelSubscriptionSpec{
				ModelIDs:     modelIDs,
				WorkspaceIDs: workspaceIDs,
				UserIDs:      userIDs,
				Since:        0,
			}
		case modelVersions:
			var modelIDs, modelVersionIDs, userIDs []int
			if subscriptionIDs[models] != nil {
				modelIDs = subscriptionIDs[models].([]int)
			}
			if subscriptionIDs["versions"] != nil {
				modelVersionIDs = subscriptionIDs["versions"].([]int)
			}
			if subscriptionIDs["users"] != nil {
				userIDs = subscriptionIDs["users"].([]int)
			}
			subscriptionSpecSet.ModelVersion = &ModelVersionSubscriptionSpec{
				ModelIDs:        modelIDs,
				ModelVersionIDs: modelVersionIDs,
				UserIDs:         userIDs,
				Since:           0,
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
	fmt.Println("before queries")
	// execute provided queries on the db
	for i := range testCase.queries {
		_, err := testCase.queries[i].Exec(ctx)
		if err != nil {
			t.Errorf("%d %v failed to execute error", i, testCase.queries)
		}
	}

	// read until we received the expected message
	data := []string{}
	fmt.Printf("before readuntilfound\n")
	socket.ReadUntilFound(t, &data, append(testCase.expectedUpserts, testCase.expectedDeletions...))
	deletions, upserts, _ := splitMsgs(t, data)

	// validate messages collected at startup
	fmt.Printf("expectedUpserts: %+v, expectedDeletions: %+v\n",
		testCase.expectedUpserts, testCase.expectedDeletions)
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
	errgrp.Go(ps.Start)
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
				map[string]map[string]interface{}{projects: {projects: []int{1, 2}}},
			),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
		},
		{
			description: "project subscription with excess project id",
			startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2,3"},
				map[string]map[string]interface{}{projects: {projects: []int{1, 2, 3}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 3"},
		},
		{
			description: "project subscription with workspaces",
			startupMsg: buildStartupMsg("3", map[string]string{projects: "1,2"},
				map[string]map[string]interface{}{projects: {projects: []int{1, 2}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
		},
		{
			description: "project offline fall out",
			startupMsg: buildStartupMsg("4", map[string]string{projects: "1,2"},
				map[string]map[string]interface{}{projects: {projects: []int{1}}}),
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 2"},
		},
		{
			description: "project offline fall in",
			startupMsg: buildStartupMsg("5", map[string]string{projects: "1"},
				map[string]map[string]interface{}{projects: {projects: []int{1, 2}}}),
			expectedUpserts:   []string{"key: project, project_id: 2, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{"key: projects_deleted, deleted: "},
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
					map[string]map[string]interface{}{projects: {projects: []int{1, 2, 3}}}),
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description:       "create project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(testProject)},
			expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: update project 3",
				startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2"},
					map[string]map[string]interface{}{projects: {projects: []int{1, 2, 3}}}),
				expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description: "update project 3",
			queries:     []streamdata.ExecutableQuery{streamdata.GetUpdateProjectQuery(projectMod)},
			expectedUpserts: []string{
				"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1",
			},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup case for: delete project 3",
				startupMsg: buildStartupMsg("1", map[string]string{projects: "1,2"},
					map[string]map[string]interface{}{projects: {projects: []int{1, 2, 3}}}),
				expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 1"},
				expectedDeletions: []string{"key: projects_deleted, deleted: "},
			},
			description:       "delete project 3",
			queries:           []streamdata.ExecutableQuery{streamdata.GetDeleteProjectQuery(projectMod)},
			expectedUpserts:   []string{},
			expectedDeletions: []string{"key: projects_deleted, deleted: 3"},
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
					map[string]map[string]interface{}{projects: {"workspaces": []int{2}}},
				),
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"key: projects_deleted, deleted: ",
				},
			},
			description:       "online create project",
			queries:           []streamdata.ExecutableQuery{streamdata.GetAddProjectQuery(testProject)},
			expectedUpserts:   []string{"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2"},
			expectedDeletions: []string{},
		},
		{
			startupCase: startupTestCase{
				description: "startup test case for: online fall out project",
				startupMsg: buildStartupMsg(
					"4",
					map[string]string{projects: "2,3"},
					map[string]map[string]interface{}{projects: {"workspaces": []int{2}}},
				),
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
					map[string]string{projects: "2"},
					map[string]map[string]interface{}{projects: {"workspaces": []int{2}}},
				),
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

func TestMultipleSubscriptions(t *testing.T) {
	pgDB := initializeStreamDB(context.Background(), t)
	testProject := model.Project{
		Name:        uuid.NewString(),
		CreatedAt:   time.Now(),
		Archived:    false,
		WorkspaceID: 2,
		UserID:      1,
		State:       "UNSPECIFIED",
	}
	testModel := ModelMsg{
		ID:           2,
		Name:         uuid.NewString(),
		CreationTime: time.Now(),
		WorkspaceID:  2,
		UserID:       1,
	}
	testModelVersion := ModelVersionMsg{
		ID:             2,
		Name:           uuid.NewString(),
		CheckpointUUID: uuid.NewString(),
		Version:        2,
		ModelID:        1,
		UserID:         1,
	}
	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup test case for: multiple subscriptions",
				startupMsg: buildStartupMsg(
					"1",
					map[string]string{projects: "2", models: "1", modelVersions: "1"},
					map[string]map[string]interface{}{
						projects:      {"workspaces": []int{2}},
						models:        {"workspaces": []int{2}},
						modelVersions: {models: []int{1}},
					},
				),
				expectedUpserts: []string{},
				expectedDeletions: []string{
					"key: projects_deleted, deleted: ",
					"key: models_deleted, deleted: ",
					"key: modelversions_deleted, deleted: ",
				},
			},
			description: "multiple subscriptions",
			queries: []streamdata.ExecutableQuery{
				streamdata.GetAddProjectQuery(testProject),
				db.Bun().NewInsert().Model(&testModel),
				db.Bun().NewInsert().Model(&testModelVersion).ExcludeColumn("workspace_id"),
			},
			expectedUpserts: []string{
				"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2",
				"key: model, model_id: 2, workspace_id: 2",
				"key: modelversion, model_version_id: 2, model_id: 1, workspace_id: 2",
			},
			expectedDeletions: []string{},
		},
	}
	runUpdateTest(t, pgDB, testCases)
}

func TestSubscribeByUserID(t *testing.T) {
	pgDB := initializeStreamDB(context.Background(), t)
	testProject := model.Project{
		Name:        uuid.NewString(),
		CreatedAt:   time.Now(),
		Archived:    false,
		WorkspaceID: 2,
		UserID:      1,
		State:       "UNSPECIFIED",
	}
	testModel := ModelMsg{
		ID:           2,
		Name:         uuid.NewString(),
		CreationTime: time.Now(),
		WorkspaceID:  2,
		UserID:       1,
	}
	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup test case for: subscribe to models by user id",
				startupMsg: buildStartupMsg(
					"1",
					map[string]string{projects: "2", models: "1"},
					map[string]map[string]interface{}{projects: {"workspaces": []int{2}}, models: {"users": []int{1}}},
				),
				expectedUpserts:   []string{},
				expectedDeletions: []string{"key: projects_deleted, deleted: ", "key: models_deleted, deleted: "},
			},
			description: "subscribe to models by user id",
			queries: []streamdata.ExecutableQuery{
				streamdata.GetAddProjectQuery(testProject),
				db.Bun().NewInsert().Model(&testModel),
			},
			expectedUpserts: []string{
				"key: project, project_id: 3, state: UNSPECIFIED, workspace_id: 2",
				"key: model, model_id: 2, workspace_id: 2",
			},
			expectedDeletions: []string{},
		},
	}
	runUpdateTest(t, pgDB, testCases)
}

func TestSubscribeModelVersion(t *testing.T) {
	pgDB := initializeStreamDB(context.Background(), t)
	testCases := []updateTestCase{
		{
			startupCase: startupTestCase{
				description: "startup test case for: subcribe to model version by model id",
				startupMsg: buildStartupMsg(
					"3",
					map[string]string{modelVersions: ""},
					map[string]map[string]interface{}{
						modelVersions: {models: []int{1}},
					},
				),
				expectedUpserts: []string{
					"key: modelversion, model_version_id: 1, model_id: 1, workspace_id: ",
				},
				expectedDeletions: []string{
					"key: modelversions_deleted, deleted: ",
				},
			},
			description: "move parent model for model version would trigger an update",
			queries: []streamdata.ExecutableQuery{
				db.Bun().NewUpdate().Table("models").Set("workspace_id = ?", 1).Where("id = ?", 1),
			},
			expectedUpserts: []string{
				"key: modelversion, model_version_id: 1, model_id: 1, workspace_id: 1",
			},
			expectedDeletions: []string{},
		},
	}
	runUpdateTest(t, pgDB, testCases)
}
