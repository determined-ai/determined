//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

func newProtoStruct(t *testing.T, in map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(in)
	require.NoError(t, err)
	return s
}

func TestPostWorkspace(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	// Name min error.
	_, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: ""})
	require.Error(t, err)

	// Name max error.
	_, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: string(make([]byte, 81))})
	require.Error(t, err)

	// Invalid configs.
	_, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.New().String(),
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type": "s3",
		}),
	})
	require.Error(t, err)

	_, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.New().String(),
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type":   "s3",
			"bucket": "bucketbucket",
			"prefix": "./../.",
		}),
	})
	require.Error(t, err)

	// Valid workspace.
	workspaceName := uuid.New().String()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: workspaceName,
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type":       "s3",
			"bucket":     "bucketofrain",
			"secret_key": "thisisasecret",
		}),
	})
	require.NoError(t, err)

	// Workspace returned correctly?
	expected := &workspacev1.Workspace{
		Id:             resp.Workspace.Id,
		Name:           workspaceName,
		Archived:       false,
		Username:       curUser.Username,
		Immutable:      false,
		NumProjects:    0,
		Pinned:         true,
		UserId:         int32(curUser.ID),
		NumExperiments: 0,
		State:          workspacev1.WorkspaceState_WORKSPACE_STATE_UNSPECIFIED,
		ErrorMessage:   "",
		AgentUserGroup: nil,
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type":                 "s3",
			"bucket":               "bucketofrain",
			"secret_key":           "********",
			"access_key":           nil,
			"endpoint_url":         nil,
			"prefix":               nil,
			"save_experiment_best": nil,
			"save_trial_best":      nil,
			"save_trial_latest":    nil,
		}),
	}
	proto.Equal(expected, resp.Workspace)
	require.Equal(t, expected, resp.Workspace)

	// Workspace persisted correctly?
	getWorkResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: resp.Workspace.Id})
	require.NoError(t, err)
	proto.Equal(expected, getWorkResp.Workspace)
	require.Equal(t, expected, getWorkResp.Workspace)
}

func TestPatchWorkspace(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := resp.Workspace.Id

	// Ensure created without checkpoint config.
	getWorkResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: workspaceID})
	require.NoError(t, err)
	require.Nil(t, getWorkResp.Workspace.CheckpointStorageConfig)

	// Try adding invalid workspace configs.
	_, err = api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
		Workspace: &workspacev1.PatchWorkspace{
			CheckpointStorageConfig: newProtoStruct(t, map[string]any{
				"type": "shared_fs",
			}),
		},
	})
	require.Error(t, err)
	_, err = api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
		Workspace: &workspacev1.PatchWorkspace{
			CheckpointStorageConfig: newProtoStruct(t, map[string]any{
				"type":   "s3",
				"bucket": "bucketbucket",
				"prefix": "../../..",
			}),
		},
	})
	require.Error(t, err)

	// Patch with valid workspace config.
	patchResp, err := api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
		Id: workspaceID,
		Workspace: &workspacev1.PatchWorkspace{
			CheckpointStorageConfig: newProtoStruct(t, map[string]any{
				"type":                 "s3",
				"bucket":               "bucketofrain",
				"secret_key":           "keyyyyy",
				"save_experiment_best": 4,
				"save_trial_best":      2,
			}),
		},
	})
	require.NoError(t, err)

	// Correct response returned by patch?
	expected := newProtoStruct(t, map[string]any{
		"type":                 "s3",
		"bucket":               "bucketofrain",
		"secret_key":           "********",
		"access_key":           nil,
		"endpoint_url":         nil,
		"prefix":               nil,
		"save_experiment_best": 4,
		"save_trial_best":      2,
		"save_trial_latest":    nil,
	})
	proto.Equal(expected, patchResp.Workspace.CheckpointStorageConfig)
	require.Equal(t, expected, patchResp.Workspace.CheckpointStorageConfig)

	// Change persisted?
	getWorkResp, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: workspaceID})
	require.NoError(t, err)
	proto.Equal(expected, getWorkResp.Workspace.CheckpointStorageConfig)
	require.Equal(t, expected, getWorkResp.Workspace.CheckpointStorageConfig)
}

var wAuthZ *mocks.WorkspaceAuthZ

// pgdb can be nil to use the singleton database for testing.
func setupWorkspaceAuthZTest(
	t *testing.T, pgdb *db.PgDB,
) (*apiServer, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, _, curUser, ctx := setupUserAuthzTest(t, pgdb)

	if wAuthZ == nil {
		wAuthZ = &mocks.WorkspaceAuthZ{}
		workspace.AuthZProvider.Register("mock", wAuthZ)
	}
	return api, wAuthZ, curUser, ctx
}

func TestAuthzGetWorkspace(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)
	// Deny returns same as 404.
	_, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: -9999})
	require.Equal(t, apiPkg.NotFoundErrs("workspace", "-9999", true).Error(), err.Error())

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
	require.Equal(t, apiPkg.NotFoundErrs("workspace", "1", true).Error(), err.Error())

	// A error returned by CanGetWorkspace is returned unmodified.
	expectedErr := fmt.Errorf("canGetWorkspaceError")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr).Once()
	_, err = api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: 1})
	require.Equal(t, expectedErr, err)
}

func TestAuthzGetWorkspaceProjects(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)

	// Deny with error returns error unmodified.
	expectedErr := fmt.Errorf("filterWorkspaceProjectsError")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	workspaceAuthZ.On("FilterWorkspaceProjects", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := api.GetWorkspaceProjects(ctx, &apiv1.GetWorkspaceProjectsRequest{Id: 1})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever the filtering returned.
	expected := []*projectv1.Project{{Name: "test"}}
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	workspaceAuthZ.On("FilterWorkspaceProjects", mock.Anything, mock.Anything, mock.Anything).
		Return(expected, nil).Once()
	resp, err := api.GetWorkspaceProjects(ctx, &apiv1.GetWorkspaceProjectsRequest{Id: 1})
	require.NoError(t, err)
	require.Equal(t, expected, resp.Projects)
}

func TestAuthzGetWorkspaces(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)

	// Deny with error returns error unmodified.
	expectedErr := fmt.Errorf("filterWorkspaceError")
	workspaceAuthZ.On("FilterWorkspaces", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, expectedErr).Once()
	_, err := api.GetWorkspaces(ctx, &apiv1.GetWorkspacesRequest{})
	require.Equal(t, expectedErr, err)

	// Nil error returns whatever the filtering returned.
	expected := []*workspacev1.Workspace{{Name: "test"}}
	workspaceAuthZ.On("FilterWorkspaces", mock.Anything, mock.Anything, mock.Anything).
		Return(expected, nil).Once()
	resp, err := api.GetWorkspaces(ctx, &apiv1.GetWorkspacesRequest{})
	require.NoError(t, err)
	require.Equal(t, expected, resp.Workspaces)
}

func TestAuthzPostWorkspace(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)

	// Deny returns error wrapped in forbidden.
	expectedErr := status.Error(codes.PermissionDenied, "canCreateWorkspaceDeny")
	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).
		Return(fmt.Errorf("canCreateWorkspaceDeny")).Once()
	_, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Allow allows the workspace to be created and gotten.
	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	getResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: resp.Workspace.Id})
	require.NoError(t, err)
	proto.Equal(resp.Workspace, getResp.Workspace)
	require.Equal(t, resp.Workspace, getResp.Workspace)

	// Tried to create with checkpoint storage config.
	expectedErr = status.Error(codes.PermissionDenied, "storageConfDeny")
	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything).Return(nil).Once()
	workspaceAuthZ.On("CanCreateWorkspaceWithCheckpointStorageConfig",
		mock.Anything, mock.Anything).Return(fmt.Errorf("storageConfDeny"))
	_, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.New().String(),
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type": "s3",
		}),
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthzWorkspaceGetThenActionRoutes(t *testing.T) {
	api, workspaceAuthZ, _, ctx := setupWorkspaceAuthZTest(t, nil)
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
		{"CanSetWorkspacesCheckpointStorageConfig", func(id int) error {
			_, err := api.PatchWorkspace(ctx, &apiv1.PatchWorkspaceRequest{
				Id: int32(id),
				Workspace: &workspacev1.PatchWorkspace{
					CheckpointStorageConfig: newProtoStruct(t, map[string]any{
						"type":   "s3",
						"bucket": "bucketbucket",
					}),
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
		workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
		resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
		require.NoError(t, err)
		id := int(resp.Workspace.Id)

		// Bad ID gives not found.
		require.Equal(t, apiPkg.NotFoundErrs("workspace", "-9999", true), curCase.IDToReqCall(-9999))

		// Without permission to view returns not found.
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		require.Equal(t, apiPkg.NotFoundErrs("workspace", fmt.Sprint(id), true).Error(),
			curCase.IDToReqCall(id).Error())

		// A error returned by CanGetWorkspace is returned unmodified.
		cantGetWorkspaceErr := fmt.Errorf("canGetWorkspaceError")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(cantGetWorkspaceErr).Once()
		require.Equal(t, cantGetWorkspaceErr, curCase.IDToReqCall(id))

		// Deny with permission to view returns error wrapped in forbidden.
		expectedErr := status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		workspaceAuthZ.On(curCase.DenyFuncName, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Errorf("%sDeny", curCase.DenyFuncName)).Once()
		require.Equal(t, expectedErr.Error(), curCase.IDToReqCall(id).Error())
	}
}

func TestAuthZListRPsBoundToWorkspace(t *testing.T) {
	return
}
