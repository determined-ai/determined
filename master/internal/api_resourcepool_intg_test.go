//go:build integration
// +build integration

package internal

import (
	"context"
	"testing"

	"github.com/uptrace/bun"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

const (
	testPoolName       = "testRP"
	testPool2Name      = "testRP2"
	testWorkspaceName  = "bindings_test_workspace_1"
	testWorkspace2Name = "bindings_test_workspace_2"
)

func setupWorkspaces(ctx context.Context, t *testing.T, api *apiServer) []int32 {
	w1, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: testWorkspaceName})
	require.NoError(t, err)
	w2, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: testWorkspace2Name})
	require.NoError(t, err)

	return []int32{w1.Workspace.Id, w2.Workspace.Id}
}

func cleanupWorkspaces(ctx context.Context) {
	workspaces := []string{testWorkspaceName, testWorkspace2Name}
	_, err := db.Bun().NewDelete().Table("workspaces").
		Where("name IN (?)", bun.In(workspaces)).
		Exec(ctx)
	if err != nil {
		logrus.Errorf("Error deleting the following workspaces: %s", workspaces)
	}
}

func TestPostBindingFails(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	_ = setupWorkspaces(ctx, t, api)

	// TODO (eliu): add some tests for workspaceIDs
	// test resource pools on workspaces that do not exist
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Once()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Once()
	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{"nonexistent_workspace"},
	})
	require.ErrorContains(t, err, "the following workspaces do not exist: [nonexistent_workspace]")

	// test resource pool doesn't exist
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Once()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Twice()
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{},
		}, nil).Once()
	_, err = api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName, testWorkspace2Name},
	})

	require.ErrorContains(t, err, "pool with name testRP doesn't exist")

	// test resource pool is a default resource pool
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{PoolName: testPoolName}, nil).Twice()

	_, err = api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName, testWorkspace2Name},
	})

	require.ErrorContains(t, err, "default resource pool testRP cannot be bound to any workspace")

	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{PoolName: testPoolName}, nil).Once()

	_, err = api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName, testWorkspace2Name},
	})

	require.ErrorContains(t, err, "default resource pool testRP cannot be bound to any workspace")

	// test no resource pool specified
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{PoolName: testPoolName}, nil).Once()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{PoolName: testPoolName}, nil).Once()
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Once()
	_, err = api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: "",
		WorkspaceNames:   []string{testWorkspaceName, testWorkspace2Name},
	})

	require.ErrorContains(t, err, "doesn't exist")

	require.True(t, mockRM.AssertExpectations(t))
}

func TestPostBindingSucceeds(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	_ = setupWorkspaces(ctx, t, api)

	// bind first resource pool
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Twice()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Twice()
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Twice()

	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName},
	})
	require.NoError(t, err)

	// test bind no workspaces specified
	_, err = api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{},
	})

	require.NoError(t, err)

	require.True(t, mockRM.AssertExpectations(t))
}

func TestListWorkspacesBoundToRPFails(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	_ = setupWorkspaces(ctx, t, api)

	// bind first workspace
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Once()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Once()
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Times(3)

	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName},
	})
	require.NoError(t, err)

	// fail listing a non-existent resource pool
	_, err = api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: "nonExistentRP",
	})
	require.Error(t, err)

	_, err = api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{})
	require.Error(t, err)

	require.True(t, mockRM.AssertExpectations(t))
}

func TestListWorkspacesBoundToRPSucceeds(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	workspaceIDs := setupWorkspaces(ctx, t, api)

	// test bind resource pool to workspace
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Once()
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Once()
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Twice()

	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName},
	})
	require.NoError(t, err)

	// test listing the workspace that was just bound
	resp, err := api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: testPoolName,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.WorkspaceIds))
	require.Equal(t, workspaceIDs[0], resp.WorkspaceIds[0])

	// test listing on resource pool that has no bindings
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{
				{Name: testPoolName}, {Name: testPool2Name},
			},
		}, nil).Once()
	resp, err = api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: testPool2Name,
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.WorkspaceIds))

	require.True(t, mockRM.AssertExpectations(t))
}

func TestPatchBindingsSucceeds(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	workspaceIDs := setupWorkspaces(ctx, t, api)

	// setup first binding
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Times(4)
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Times(4)
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Times(7)

	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName},
	})
	require.NoError(t, err)

	// test patch binding with empty workspaces
	_, err = api.OverwriteRPWorkspaceBindings(ctx, &apiv1.OverwriteRPWorkspaceBindingsRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{},
	})
	require.NoError(t, err)

	resp, err := api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: testPoolName,
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.WorkspaceIds))

	// test patch binding with different workspace
	_, err = api.OverwriteRPWorkspaceBindings(ctx, &apiv1.OverwriteRPWorkspaceBindingsRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspace2Name},
	})
	require.NoError(t, err)
	resp, err = api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: testPoolName,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.WorkspaceIds))
	require.Equal(t, workspaceIDs[1], resp.WorkspaceIds[0])

	// test patch binding with different workspaceID
	_, err = api.OverwriteRPWorkspaceBindings(ctx, &apiv1.OverwriteRPWorkspaceBindingsRequest{
		ResourcePoolName: testPoolName,
		WorkspaceIds:     []int32{workspaceIDs[0]},
		WorkspaceNames:   []string{testWorkspace2Name},
	})
	require.NoError(t, err)
	resp, err = api.ListWorkspacesBoundToRP(ctx, &apiv1.ListWorkspacesBoundToRPRequest{
		ResourcePoolName: testPoolName,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(resp.WorkspaceIds))
	expectedIds := set.FromSlice[int32](workspaceIDs)
	for _, id := range resp.WorkspaceIds {
		require.True(t, expectedIds.Contains(id))
	}

	require.True(t, mockRM.AssertExpectations(t))
}

func TestDeleteBindingsSucceeds(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	var mockRM mocks.ResourceManager
	api.m.rm = &mockRM

	defer func() { cleanupWorkspaces(ctx) }()

	workspaceIDs := setupWorkspaces(ctx, t, api)

	// TODO: fix all comments
	// setup first binding
	mockRM.On("GetDefaultComputeResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultComputeResourcePoolResponse{}, nil).Times(1)
	mockRM.On("GetDefaultAuxResourcePool", mock.Anything, mock.Anything).
		Return(sproto.GetDefaultAuxResourcePoolResponse{}, nil).Times(1)
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).
		Return(&apiv1.GetResourcePoolsResponse{
			ResourcePools: []*resourcepoolv1.ResourcePool{{Name: testPoolName}},
		}, nil).Times(3)

	_, err := api.BindRPToWorkspace(ctx, &apiv1.BindRPToWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspaceName, testWorkspace2Name},
	})
	require.NoError(t, err)

	_, err = api.UnbindRPFromWorkspace(ctx, &apiv1.UnbindRPFromWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceIds:     []int32{workspaceIDs[0]},
	})
	require.NoError(t, err)

	listReq := &apiv1.ListWorkspacesBoundToRPRequest{ResourcePoolName: testPoolName}
	resp, err := api.ListWorkspacesBoundToRP(ctx, listReq)
	require.NoError(t, err)
	require.Equal(t, 1, len(resp.WorkspaceIds))
	require.Equal(t, workspaceIDs[1], resp.WorkspaceIds[0])

	_, err = api.UnbindRPFromWorkspace(ctx, &apiv1.UnbindRPFromWorkspaceRequest{
		ResourcePoolName: testPoolName,
		WorkspaceNames:   []string{testWorkspace2Name},
	})
	require.NoError(t, err)

	resp, err = api.ListWorkspacesBoundToRP(ctx, listReq)
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.WorkspaceIds))

	require.True(t, mockRM.AssertExpectations(t))
}
