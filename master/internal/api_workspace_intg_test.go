//go:build integration
// +build integration

package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
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
	// TODO PostWorkspace should returned pinnedAt.
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: workspaceName,
		CheckpointStorageConfig: newProtoStruct(t, map[string]any{
			"type":       "s3",
			"bucket":     "bucketofrain",
			"secret_key": "thisisasecret",
		}),
		DefaultComputePool: "testRP",
		DefaultAuxPool:     "testRP",
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
		DefaultComputePool: "testRP",
		DefaultAuxPool:     "testRP",
	}
	proto.Equal(expected, resp.Workspace)
	require.Equal(t, expected, resp.Workspace)

	// Workspace persisted correctly?
	getWorkResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: resp.Workspace.Id})
	require.NoError(t, err)
	require.NotNil(t, getWorkResp.Workspace.PinnedAt)
	getWorkResp.Workspace.PinnedAt = nil // Can't check timestamp exactly.
	proto.Equal(expected, getWorkResp.Workspace)
	require.Equal(t, expected, getWorkResp.Workspace)
}

func TestGetWorkspaces(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))
	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")

	api, curUser, ctx := setupAPITest(t, pgDB)

	w0Resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: "w0name"})
	require.NoError(t, err)
	w0 := int(w0Resp.Workspace.Id)

	w1, p0 := createProjectAndWorkspace(ctx, t, api)

	w2, p1 := createProjectAndWorkspace(ctx, t, api)
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(w2),
	})
	require.NoError(t, err)

	createTestExpWithProjectID(t, api, curUser, p0)
	createTestExpWithProjectID(t, api, curUser, p0)
	createTestExpWithProjectID(t, api, curUser, p1)
	createTestExpWithProjectID(t, api, curUser, p1)

	cases := []struct {
		name     string
		req      *apiv1.GetWorkspacesRequest
		expected []int
	}{
		{"empty request", &apiv1.GetWorkspacesRequest{}, []int{1, w0, w1, w2}},
		{"id desc request", &apiv1.GetWorkspacesRequest{
			OrderBy: apiv1.OrderBy_ORDER_BY_DESC,
		}, []int{w2, w1, w0, 1}},
		{"w0 name", &apiv1.GetWorkspacesRequest{
			Name: "w0name",
		}, []int{w0}},
		{"w0 name subset doesn't match", &apiv1.GetWorkspacesRequest{
			Name: "0nam",
		}, []int{}},
		{"w0 name case insensitive", &apiv1.GetWorkspacesRequest{
			Name: "w0nAMe",
		}, []int{w0}},
		{"archive false", &apiv1.GetWorkspacesRequest{
			Archived: wrapperspb.Bool(false),
		}, []int{1, w0, w1, w2}},
		{"archive true", &apiv1.GetWorkspacesRequest{
			Archived: wrapperspb.Bool(true),
		}, []int{}},
		{"users determined", &apiv1.GetWorkspacesRequest{
			Users: []string{"determined"},
		}, []int{}},
		{"users admin", &apiv1.GetWorkspacesRequest{
			Users: []string{"admin"},
		}, []int{1}},
		{"users determined", &apiv1.GetWorkspacesRequest{
			Users: []string{curUser.Username},
		}, []int{w0, w1, w2}},
		{"userID determined", &apiv1.GetWorkspacesRequest{
			UserIds: []int32{2},
		}, []int{}},
		{"userID admin", &apiv1.GetWorkspacesRequest{
			UserIds: []int32{1},
		}, []int{1}},
		{"userID determined", &apiv1.GetWorkspacesRequest{
			UserIds: []int32{int32(curUser.ID)},
		}, []int{w0, w1, w2}},
		{"w0 name case sensitive", &apiv1.GetWorkspacesRequest{
			NameCaseSensitive: "w0name",
		}, []int{w0}},
		{"w0 name case sensitive subset doesn't match", &apiv1.GetWorkspacesRequest{
			NameCaseSensitive: "0nam",
		}, []int{}},
		{"w0 name case sensative doesn't match", &apiv1.GetWorkspacesRequest{
			NameCaseSensitive: "w0nAMe",
		}, []int{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Make expected workspaces from GetWorkspace endpoint.
			expectedWorkspaces := []*workspacev1.Workspace{} // Do empty and not null.
			for _, w := range c.expected {
				getResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: int32(w)})
				require.NoError(t, err)
				getResp.Workspace.CheckpointStorageConfig = nil // Not returned in list endpoint.
				expectedWorkspaces = append(expectedWorkspaces, getResp.Workspace)
			}
			expectedJSON, err := json.MarshalIndent(expectedWorkspaces, "", "  ")
			require.NoError(t, err)

			actual, err := api.GetWorkspaces(ctx, c.req)
			require.NoError(t, err)
			actualJSON, err := json.MarshalIndent(actual.Workspaces, "", "  ")
			require.NoError(t, err)

			require.Equal(t, string(expectedJSON), string(actualJSON))
		})
	}
}

// This should eventually be in internal/workspaces.
func TestWorkspacesIDsByExperimentIDs(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	resp, err := workspace.WorkspacesIDsByExperimentIDs(ctx, nil)
	require.NoError(t, err)
	require.Len(t, resp, 0)

	w0, p0 := createProjectAndWorkspace(ctx, t, api)
	w1, p1 := createProjectAndWorkspace(ctx, t, api)

	e0 := createTestExpWithProjectID(t, api, curUser, p0)
	e1 := createTestExpWithProjectID(t, api, curUser, p1)
	e2 := createTestExpWithProjectID(t, api, curUser, p0)

	resp, err = workspace.WorkspacesIDsByExperimentIDs(ctx, []int{e0.ID, e1.ID, e2.ID})
	require.NoError(t, err)
	require.Equal(t, []int{w0, w1, w0}, resp)

	resp, err = workspace.WorkspacesIDsByExperimentIDs(ctx, []int{e0.ID, e1.ID, -1})
	require.Error(t, err)
	require.Len(t, resp, 0)
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

func TestPostWorkspaceRBACWorkspaceAdminAssigned(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)

	for _, enabled := range []bool{true, false} {
		for _, id := range []int{2, 5} {
			config.GetMasterConfig().Security.AuthZ.AssignWorkspaceCreator.RoleID = id
			config.GetMasterConfig().Security.AuthZ.AssignWorkspaceCreator.Enabled = enabled

			// Create workspace.
			workspaceName := uuid.New().String()
			wresp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: workspaceName})
			require.NoError(t, err)

			// Did workspace admin get assigned to the scope?
			resp, err := api.GetPermissionsSummary(ctx, &apiv1.GetPermissionsSummaryRequest{})
			require.NoError(t, err)

			var role *rbacv1.Role
			for _, r := range resp.Roles {
				if int(r.RoleId) == id {
					role = r
					break
				}
			}
			require.NotNilf(t, role, "did not find roleID in MyPermissions", id)

			shouldFail := true
			for _, assign := range resp.Assignments {
				if assign.RoleId == role.RoleId {
					if enabled {
						require.Contains(t, assign.ScopeWorkspaceIds, wresp.Workspace.Id)
					} else {
						require.NotContains(t, assign.ScopeWorkspaceIds, wresp.Workspace.Id)
					}
					shouldFail = false
				}
			}

			if shouldFail {
				require.Fail(t, "did not find workspace admin in assignments")
			}
		}
	}
}

// pgdb can be nil to use the singleton database for testing.
func setupWorkspaceAuthZTest(
	t *testing.T, pgdb *db.PgDB,
	altMockRM ...*mocks.ResourceManager,
) (*apiServer, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, _, curUser, ctx := setupUserAuthzTest(t, pgdb, altMockRM...)

	if wAuthZ == nil {
		wAuthZ = &mocks.WorkspaceAuthZ{}
		workspace.AuthZProvider.Register("mock", wAuthZ)
	}
	return api, wAuthZ, curUser, ctx
}

func setupWorkspaceAuthZ() *mocks.WorkspaceAuthZ {
	if wAuthZ == nil {
		wAuthZ = &mocks.WorkspaceAuthZ{}
		workspace.AuthZProvider.Register("mock", wAuthZ)
	}
	return wAuthZ
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
	require.NotNil(t, getResp.Workspace.PinnedAt)
	getResp.Workspace.PinnedAt = nil // Can't check timestamp exactly.
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

func TestWorkspaceHasModels(t *testing.T) {
	// set up api server to use for integration testing
	api, _, ctx := setupAPITest(t, nil)

	// create workspace for test
	workspaceName := uuid.New().String()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: workspaceName})
	require.NoError(t, err)

	// confirm workspace does not have any models
	exists, err := api.workspaceHasModels(ctx, resp.Workspace.Id)
	require.NoError(t, err)
	assert.False(t, exists)

	// add model to workspace
	modelName := uuid.New().String()
	_, err = api.PostModel(ctx, &apiv1.PostModelRequest{Name: modelName, WorkspaceName: &workspaceName})
	require.NoError(t, err) // no error creating model
	exists, err = api.workspaceHasModels(ctx, resp.Workspace.Id)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestDeleteWorkspace(t *testing.T) {
	// set up api server
	api, _, ctx := setupAPITest(t, nil)

	// set up command service - required for successful DeleteWorkspaceRequest calls
	cs, err := command.NewService(api.m.db, api.m.rm)
	require.NoError(t, err)
	command.SetDefaultService(cs)

	// create workspace
	workspaceName := uuid.New().String()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: workspaceName})
	require.NoError(t, err)

	// delete workspace without models
	_, err = api.DeleteWorkspace(ctx, &apiv1.DeleteWorkspaceRequest{
		Id: resp.Workspace.Id,
	})
	require.NoError(t, err)

	// create another workspace, and add a model
	workspaceName = uuid.New().String()
	resp, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: workspaceName})
	require.NoError(t, err)
	_, err = api.PostModel(ctx, &apiv1.PostModelRequest{Name: uuid.New().String(), WorkspaceName: &workspaceName})
	require.NoError(t, err)

	// delete should fail because workspace has models
	_, err = api.DeleteWorkspace(ctx, &apiv1.DeleteWorkspaceRequest{
		Id: resp.Workspace.Id,
	})
	require.Error(t, err)
}
