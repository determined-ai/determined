//go:build integration
// +build integration

package workspace

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	cluster1 = "C1"
	cluster2 = "C2"
)

func TestMain(m *testing.M) {
	pgDB, _, err := db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestAddWorkspace(t *testing.T) {
	ctx := context.Background()
	curUserID, err := db.HackAddUser(ctx, &model.User{Username: uuid.NewString()})
	require.NoError(t, err)
	wkspState := model.WorkspaceStateDeleteFailed
	defaultComputePool := uuid.NewString()
	agentUID := int32(90)
	agentUser := uuid.NewString()

	tx, err := db.Bun().BeginTx(ctx, nil)
	require.NoError(t, err)

	cases := []struct {
		name    string
		wksp    *model.Workspace
		tx      *bun.Tx
		wantErr bool
	}{
		{
			"valid-wksp-no-tx",
			&model.Workspace{
				Name:               uuid.NewString(),
				Archived:           true,
				UserID:             curUserID,
				Immutable:          true,
				State:              &wkspState,
				AgentUID:           &agentUID,
				AgentUser:          &agentUser,
				DefaultComputePool: defaultComputePool,
			},
			nil,
			false,
		},
		{
			"valid-wksp-with-tx",
			&model.Workspace{
				Name:               uuid.NewString(),
				Archived:           true,
				UserID:             curUserID,
				Immutable:          true,
				State:              &wkspState,
				AgentUID:           &agentUID,
				AgentUser:          &agentUser,
				DefaultComputePool: defaultComputePool,
			},
			&tx,
			false,
		},
		{
			"no-user-id-no-tx",
			&model.Workspace{
				Name:               uuid.NewString(),
				Archived:           true,
				Immutable:          true,
				State:              &wkspState,
				AgentUID:           &agentUID,
				AgentUser:          &agentUser,
				DefaultComputePool: defaultComputePool,
			},
			nil,
			true,
		},
		{
			"no-user-id-with-tx",
			&model.Workspace{
				Name:               uuid.NewString(),
				Archived:           true,
				Immutable:          true,
				State:              &wkspState,
				AgentUID:           &agentUID,
				AgentUser:          &agentUser,
				DefaultComputePool: defaultComputePool,
			},
			&tx,
			true,
		},
		{
			"agent-id-no-agent-user-id",
			&model.Workspace{
				Name:               uuid.NewString(),
				Archived:           true,
				Immutable:          true,
				State:              &wkspState,
				AgentUID:           &agentUID,
				DefaultComputePool: defaultComputePool,
			},
			nil,
			true,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			err := AddWorkspace(ctx, test.wksp, test.tx)
			if !test.wantErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestGetNamespaceFromWorkspace(t *testing.T) {
	ctx := context.Background()
	wkspID1, wksp1 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID2, wksp2 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	_, wksp3 := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	b1 := model.WorkspaceNamespace{WorkspaceID: wkspID1, ClusterName: "C1", Namespace: "n1"}
	b2 := model.WorkspaceNamespace{WorkspaceID: wkspID1, ClusterName: "C2", Namespace: "n2"}
	b3 := model.WorkspaceNamespace{WorkspaceID: wkspID2, ClusterName: "C1", Namespace: "n3"}

	bindings := []model.WorkspaceNamespace{b1, b2, b3}

	_, err := db.Bun().NewInsert().Model(&bindings).Exec(ctx)
	require.NoError(t, err)

	tests := []struct {
		name          string
		workspaceName string
		clusterName   string
		namespaceName string
		wantErr       bool
	}{
		{"insert-and-check-1", wksp1, "C1", "n1", true},
		{"insert-and-check-2", wksp1, "C2", "n2", true},
		{"insert-and-check-3", wksp2, "C1", "n3", true},
		{"unknown-binding-1", wksp2, "C2", "", false},
		{"unknown-binding-2", wksp3, "C1", "", false},
		{"unknown-wksp", "test-unknown-wksp", "C1", "", false},
		{"unknown-cluster", wksp2, "test-unknown-cluster", "", false},
		{"unknown-wksp-cluster", "test-unknown-wksp", "test-unknown-cluster", "", false},
	}

	for _, tt := range tests {
		ns, err := GetNamespaceFromWorkspace(ctx, tt.workspaceName, tt.clusterName)
		if tt.wantErr {
			require.NoError(t, err)
			require.Equal(t, ns, tt.namespaceName)
		} else {
			require.ErrorContains(t, err, "no rows")
		}
	}
}

func TestGetAllNamespacesForRM(t *testing.T) {
	ctx := context.Background()
	wkspID1, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID2, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	b1 := model.WorkspaceNamespace{WorkspaceID: wkspID1, ClusterName: cluster1, Namespace: "n1"}
	b2 := model.WorkspaceNamespace{WorkspaceID: wkspID1, ClusterName: cluster2, Namespace: "n2"}
	b3 := model.WorkspaceNamespace{WorkspaceID: wkspID2, ClusterName: cluster1, Namespace: "n3"}

	bindings := []model.WorkspaceNamespace{b1, b2, b3}

	_, err := db.Bun().NewInsert().Model(&bindings).Exec(ctx)
	require.NoError(t, err)

	ns, err := GetAllNamespacesForRM(ctx, cluster1)
	require.NoError(t, err)
	require.Equal(t, []string{"n1", "n3"}, ns)

	ns, err = GetAllNamespacesForRM(ctx, cluster2)
	require.NoError(t, err)
	require.Equal(t, []string{"n2"}, ns)

	ns, err = GetAllNamespacesForRM(ctx, "cluster3")
	require.NoError(t, err)
	require.Equal(t, []string(nil), ns)
}

func TestAddWorkspaceNamespaceBinding(t *testing.T) {
	ctx := context.Background()
	wksp, userID := createWorkspace(ctx, t)

	clusterName := uuid.NewString()
	namespace := uuid.NewString()
	expectedWkspNmsp := &model.WorkspaceNamespace{
		WorkspaceID: wksp.ID,
		ClusterName: clusterName,
		Namespace:   namespace,
	}
	err := AddWorkspaceNamespaceBinding(ctx, expectedWkspNmsp, nil)
	require.NoError(t, err)

	// Verify that workspace-namespace binding must be unique.
	err = AddWorkspaceNamespaceBinding(ctx, expectedWkspNmsp, nil)
	require.Error(t, err)

	// Test in transaction.
	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		wksp := &model.Workspace{Name: uuid.NewString(), UserID: userID}
		err = AddWorkspace(ctx, wksp, nil)
		require.NoError(t, err)

		expectedWkspNmsp.WorkspaceID = wksp.ID
		err = AddWorkspaceNamespaceBinding(ctx, expectedWkspNmsp, &tx)
		require.NoError(t, err)

		return nil
	})

	require.NoError(t, err)
}

func TestGetWorkspaceNamespaceBindings(t *testing.T) {
	ctx := context.Background()
	wksp, _ := createWorkspace(ctx, t)

	// Set 3 workspace-namespace bindings for the workspace.
	expectedBindings := make(map[string]model.WorkspaceNamespace)

	for i := 0; i < 3; i++ {
		clusterName := uuid.NewString()
		namespace := uuid.NewString()
		wsns := &model.WorkspaceNamespace{
			WorkspaceID: wksp.ID,
			Namespace:   namespace,
			ClusterName: clusterName,
		}
		err := AddWorkspaceNamespaceBinding(ctx, wsns, nil)
		require.NoError(t, err)
		expectedBindings[clusterName] = *wsns
	}

	// Verify that the added workspace-namespace bindings are listed.
	var wsnsBindings []model.WorkspaceNamespace
	wsnsBindings, err := GetWorkspaceNamespaceBindings(ctx, wksp.ID)
	require.NoError(t, err)

	for _, binding := range wsnsBindings {
		clusterName := binding.ClusterName
		require.Equal(t, expectedBindings[clusterName], binding)
	}
}

func TestDeleteWorkspaceNamespaceBindings(t *testing.T) {
	// Create workspace with workspace-namespace bindings in the database.
	ctx := context.Background()
	wksp, _ := createWorkspace(ctx, t)
	tx, err := db.Bun().BeginTx(ctx, nil)
	require.NoError(t, err)
	// Set 4 workspace-namespace bindings for the workspace.
	expectedBindings := []*model.WorkspaceNamespace{}
	bindingsToMassDelete := []string{}
	numToMassDelete := 3
	for i := 0; i < 5; i++ {
		clusterName := uuid.NewString()
		namespace := uuid.NewString()
		if i < numToMassDelete {
			bindingsToMassDelete = append(bindingsToMassDelete, clusterName)
		}
		wsns := &model.WorkspaceNamespace{
			WorkspaceID: wksp.ID,
			Namespace:   namespace,
			ClusterName: clusterName,
		}
		err := AddWorkspaceNamespaceBinding(ctx, wsns, nil)
		require.NoError(t, err)
		expectedBindings = append(expectedBindings, wsns)
	}

	// Delete workspace-namespace bindings one by one.
	expectedMassDeletedBindings := []model.WorkspaceNamespace{}
	for i := 0; i < 5; i++ {
		binding := expectedBindings[i]
		if i < numToMassDelete {
			expectedMassDeletedBindings = append(expectedMassDeletedBindings, *binding)
		} else {
			expectedDeletedBindings := []model.WorkspaceNamespace{*binding}
			deletedBindings, err := DeleteWorkspaceNamespaceBindings(ctx, wksp.ID,
				[]string{binding.ClusterName}, &tx)
			require.NoError(t, err)
			require.Equal(t, expectedDeletedBindings, deletedBindings)
		}
	}

	// Delete the remaining workspace-namespace bindings in bulk.
	deletedBindings, err := DeleteWorkspaceNamespaceBindings(ctx, wksp.ID, bindingsToMassDelete,
		&tx)
	require.NoError(t, err)
	require.Equal(t, len(expectedMassDeletedBindings), len(deletedBindings))
	require.ElementsMatch(t, expectedMassDeletedBindings, deletedBindings)

	for _, binding := range expectedBindings {
		namespace := binding.Namespace
		clusterName := binding.ClusterName
		err := tx.NewSelect().
			Model(&model.WorkspaceNamespace{}).
			Where("workspace_id = ?", wksp.ID).
			Where("namespace = ?", namespace).
			Where("cluster_name = ?", clusterName).
			Scan(ctx)
		require.Error(t, err)
	}
	err = tx.Rollback()
	require.NoError(t, err)
}

func createWorkspace(ctx context.Context, t *testing.T) (*model.Workspace, model.UserID) {
	userID, err := db.HackAddUser(ctx, &model.User{Username: uuid.NewString()})
	require.NoError(t, err)
	wksp := &model.Workspace{Name: uuid.NewString(), UserID: userID}
	err = AddWorkspace(ctx, wksp, nil)
	require.NoError(t, err)
	return wksp, userID
}

func TestGetNumWorkspacesUsingNamespaceInCluster(t *testing.T) {
	ctx := context.Background()
	wkspID1, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID2, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")
	wkspID3, _ := db.RequireMockWorkspaceID(t, db.SingleDB(), "")

	bindings := []model.WorkspaceNamespace{
		{WorkspaceID: wkspID1, ClusterName: "test_C1", Namespace: "test_n1"},
		{WorkspaceID: wkspID2, ClusterName: "test_C1", Namespace: "test_n1"},
		{WorkspaceID: wkspID3, ClusterName: "test_C2", Namespace: "test_n1"},
		{WorkspaceID: wkspID1, ClusterName: "test_C1", Namespace: "test_n2"},
	}

	_, err := db.Bun().NewInsert().Model(&bindings).Exec(ctx)
	require.NoError(t, err)

	// valid combination
	n, err := GetNumWorkspacesUsingNamespaceInCluster(ctx, "test_C1", "test_n1")
	require.NoError(t, err)
	require.Equal(t, 2, n)

	// existing clusters and namespaces but invalid combination
	n, err = GetNumWorkspacesUsingNamespaceInCluster(ctx, "test_C2", "test_n2")
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// non-existent cluster
	n, err = GetNumWorkspacesUsingNamespaceInCluster(ctx, "test_C3", "test_n1")
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// non-existent namespace
	n, err = GetNumWorkspacesUsingNamespaceInCluster(ctx, "test_C1", "test_n3")
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// non-existent cluster and namespace
	n, err = GetNumWorkspacesUsingNamespaceInCluster(ctx, "test_C4", "test_n3")
	require.NoError(t, err)
	require.Equal(t, 0, n)

	// Clean up
	_, err = db.Bun().NewDelete().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id in (?)", bun.In([]int{wkspID1, wkspID2, wkspID3})).
		Exec(ctx)
	require.NoError(t, err)
}

func TestAbortUpdateAutoCreateNamespaceNameTrigger(t *testing.T) {
	// Create a workspace with an auto-created namespace name.
	ctx := context.Background()
	userID, err := db.HackAddUser(ctx, &model.User{Username: uuid.NewString()})
	require.NoError(t, err)
	wkspName := uuid.NewString()
	diff := "-diff"
	wkspAutoNmsp := wkspName + "-auto"
	wksp := &model.Workspace{
		Name:                     wkspName,
		UserID:                   userID,
		AutoCreatedNamespaceName: &wkspAutoNmsp,
	}
	err = AddWorkspace(ctx, wksp, nil)
	require.NoError(t, err)

	// Verify that we cannot update the workspace's auto-created namespace name.
	wkspAutoNmsp += diff
	_, err = db.Bun().NewUpdate().Model(wksp).Where("id = ?", wksp.ID).Exec(ctx)
	require.ErrorContains(t, err, "auto_created_namespace_name")

	var wkspNmsp string
	err = db.Bun().NewSelect().
		Model(&model.Workspace{}).
		Column("auto_created_namespace_name").
		Where("id = ?", wksp.ID).
		Scan(ctx, &wkspNmsp)
	require.NoError(t, err)
	require.NotEqual(t, wkspAutoNmsp, wkspNmsp)
	require.Equal(t, wkspName+"-auto", wkspNmsp)

	// Create a workspace with no auto-created namespace name.
	wksp, _ = createWorkspace(ctx, t)

	err = db.Bun().NewSelect().
		Model(&model.Workspace{}).
		Column("auto_created_namespace_name").
		Where("id = ?", wksp.ID).
		Scan(ctx, &wkspNmsp)
	require.NoError(t, err)
	require.Equal(t, "", wkspNmsp)

	// Verify that we can set the workspace's auto-created namespace name.
	wkspAutoNmsp = wksp.Name + "-auto"
	wksp.AutoCreatedNamespaceName = &wkspAutoNmsp
	_, err = db.Bun().NewUpdate().Model(wksp).Where("id = ?", wksp.ID).Exec(ctx)
	require.NoError(t, err)

	// Verify that we cannot update the workspace's auto-created namespace name.
	wkspAutoNmsp += diff
	_, err = db.Bun().NewUpdate().Model(wksp).Where("id = ?", wksp.ID).Exec(ctx)
	require.ErrorContains(t, err, "auto_created_namespace_name")

	err = db.Bun().NewSelect().
		Model(&model.Workspace{}).
		Column("auto_created_namespace_name").
		Where("id = ?", wksp.ID).
		Scan(ctx, &wkspNmsp)
	require.NoError(t, err)
	require.NotEqual(t, wkspAutoNmsp, wkspNmsp)
	require.Equal(t, wksp.Name+"-auto", wkspNmsp)

	// Verify that we cannot change the workspace's auto-created namespace name back to NULL.
	wksp.AutoCreatedNamespaceName = nil
	_, err = db.Bun().NewUpdate().Model(wksp).Where("id = ?", wksp.ID).Exec(ctx)
	require.ErrorContains(t, err, "auto_created_namespace_name")
}
