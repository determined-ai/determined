//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

var workspaceAuthZ *mocks.WorkspaceAuthZ

func workspaceNotFoundErr(id int) error {
	return status.Errorf(codes.NotFound, fmt.Sprintf("workspace (%d) not found", id))
}

func SetupWorkspaceAuthZTest(
	t *testing.T,
) (*apiServer, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, _, curUser, ctx := SetupUserAuthzTest(t)

	if workspaceAuthZ == nil {
		workspaceAuthZ = &mocks.WorkspaceAuthZ{}
		workspace.AuthZProvider.Register("mock", workspaceAuthZ)
	}
	return api, workspaceAuthZ, curUser, ctx
}

func TestAuthzGetWorkspace(t *testing.T) {
	api, workspaceAuthZ, _, ctx := SetupWorkspaceAuthZTest(t)
	// Deny returns same as 404.
	_, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: -9999})
	require.Equal(t, workspaceNotFoundErr(-9999).Error(), err.Error())

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
	require.Equal(t, workspaceNotFoundErr(1).Error(), err.Error())

	// A error returned by CanGetWorkspace is returned unmodified.
	expectedErr := fmt.Errorf("canGetWorkspaceError")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).
		Return(false, expectedErr).Once()
	_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
	require.Equal(t, expectedErr, err)
}

func TestAuthzGetWorkspaceProjects(t *testing.T) {
	api, workspaceAuthZ, _, ctx := SetupWorkspaceAuthZTest(t)

	// Deny with error returns error unmodified.
	expectedErr := fmt.Errorf("filterWorkspaceProjectsError")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	workspaceAuthZ.On("FilterWorkspaceProjects", mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := api.GetWorkspaceProjects(ctx, &apiv1.GetWorkspaceProjectsRequest{Id: 1})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever the filtering returned.
	expected := []*projectv1.Project{{Name: "test"}}
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	workspaceAuthZ.On("FilterWorkspaceProjects", mock.Anything, mock.Anything).
		Return(expected, nil).Once()
	resp, err := api.GetWorkspaceProjects(ctx, &apiv1.GetWorkspaceProjectsRequest{Id: 1})
	require.NoError(t, err)
	require.Equal(t, expected, resp.Projects)
}

func TestAuthzGetWorkspaces(t *testing.T) {
	api, workspaceAuthZ, _, ctx := SetupWorkspaceAuthZTest(t)

	// Deny with error returns error unmodified.
	expectedErr := fmt.Errorf("filterWorkspaceError")
	workspaceAuthZ.On("FilterWorkspaces", mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := api.GetWorkspaces(ctx, &apiv1.GetWorkspacesRequest{})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever the filtering returned.
	expected := []*workspacev1.Workspace{{Name: "test"}}
	workspaceAuthZ.On("FilterWorkspaces", mock.Anything, mock.Anything).
		Return(expected, nil).Once()
	resp, err := api.GetWorkspaces(ctx, &apiv1.GetWorkspacesRequest{})
	require.NoError(t, err)
	require.Equal(t, expected, resp.Workspaces)
}

func TestAuthzPostWorkspace(t *testing.T) {
	api, workspaceAuthZ, _, ctx := SetupWorkspaceAuthZTest(t)

	// Deny returns error wrapped in forbidden.
	expectedErr := status.Error(codes.PermissionDenied, "canCreateWorkspaceDeny")
	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything).
		Return(fmt.Errorf("canCreateWorkspaceDeny")).Once()
	_, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Allow allows the workspace to be created and gotten.
	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything).Return(nil).Once()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything).Return(nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	getResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: resp.Workspace.Id})
	require.NoError(t, err)
	require.Equal(t, resp.Workspace, getResp.Workspace)
}

func TestAuthzWorkspaceGetThenActionRoutes(t *testing.T) {
	api, workspaceAuthZ, _, ctx := SetupWorkspaceAuthZTest(t)
	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
		{"CanSetWorkspacesName", func(id int) error {
			_, err := api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
				Id: int32(id),
				Workspace: &workspacev1.PatchWorkspace{
					Name: wrapperspb.String(uuid.New().String()),
				},
			})
			return err
		}},
		{"CanDeleteWorkspace", func(id int) error {
			_, err := api.DeleteWorkspace(ctx, &apiv1.DeleteWorkspaceRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanArchiveWorkspace", func(id int) error {
			_, err := api.ArchiveWorkspace(ctx, &apiv1.ArchiveWorkspaceRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanUnarchiveWorkspace", func(id int) error {
			_, err := api.UnarchiveWorkspace(ctx, &apiv1.UnarchiveWorkspaceRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanPinWorkspace", func(id int) error {
			_, err := api.PinWorkspace(ctx, &apiv1.PinWorkspaceRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanUnpinWorkspace", func(id int) error {
			_, err := api.UnpinWorkspace(ctx, &apiv1.UnpinWorkspaceRequest{
				Id: int32(id),
			})
			return err
		}},
	}

	for _, curCase := range cases {
		// Create workspace to test with.
		workspaceAuthZ.On("CanCreateWorkspace", mock.Anything).Return(nil).Once()
		resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
		require.NoError(t, err)
		id := int(resp.Workspace.Id)

		// Bad ID gives not found.
		require.Equal(t, workspaceNotFoundErr(-9999), curCase.IDToReqCall(-9999))

		// Without permission to view returns not found.
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(false, nil).Once()
		require.Equal(t, workspaceNotFoundErr(id).Error(), curCase.IDToReqCall(id).Error())

		// A error returned by CanGetWorkspace is returned unmodified.
		cantGetWorkspaceErr := fmt.Errorf("canGetWorkspaceError")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).
			Return(false, cantGetWorkspaceErr).Once()
		require.Equal(t, cantGetWorkspaceErr, curCase.IDToReqCall(id))

		// Deny with permission to view returns error wrapped in forbidden.
		expectedErr := status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
		workspaceAuthZ.On(curCase.DenyFuncName, mock.Anything, mock.Anything).
			Return(fmt.Errorf("%sDeny", curCase.DenyFuncName)).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(id).Error())
	}
}
