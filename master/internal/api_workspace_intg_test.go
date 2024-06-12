//go:build integration
// +build integration

package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
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
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/multirm"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/workspacev1"
)

const noName = ""

var defaultNamespace = defaultKubernetesNamespace

func newProtoStruct(t *testing.T, in map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(in)
	require.NoError(t, err)
	return s
}

func TestPostWorkspace(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	cluster1 := "k8sCluster1"
	namespace := uuid.NewString()
	badClusterName := "nonExistentCluster"
	badNamespace := "nonexistentNamespace"

	errorCases := []struct {
		name        string
		req         *apiv1.PostWorkspaceRequest
		setupMockRM func(*mocks.ResourceManager)
		multiRM     bool
	}{
		{
			"Min-error",
			&apiv1.PostWorkspaceRequest{Name: ""},
			func(*mocks.ResourceManager) {},
			false,
		},
		{
			"Max-error",
			&apiv1.PostWorkspaceRequest{Name: string(make([]byte, 81))},
			func(*mocks.ResourceManager) {},
			false,
		},
		{
			"Invalid-config-no-bucket-given",
			&apiv1.PostWorkspaceRequest{
				Name: uuid.NewString(),
				CheckpointStorageConfig: newProtoStruct(t, map[string]any{
					"type": "s3",
				}),
			}, func(*mocks.ResourceManager) {},
			false,
		},
		{
			"Invalid-config-bad-prefix",
			&apiv1.PostWorkspaceRequest{
				Name: uuid.NewString(),
				CheckpointStorageConfig: newProtoStruct(t, map[string]any{
					"type":   "s3",
					"bucket": "bucketbucket",
					"prefix": "./../.",
				}),
			},
			func(*mocks.ResourceManager) {},
			false,
		},
		{
			"cluster-name-no-namespace-multiRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: noName},
			},
			func(*mocks.ResourceManager) {},
			true,
		},
		{
			"namespace-no-cluster-name-multiRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{noName: namespace},
			},
			func(mockRM *mocks.ResourceManager) {},
			true,
		},
		{
			"invalid-cluster-name-valid-namespace-multiRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{badClusterName: namespace},
			},
			func(*mocks.ResourceManager) {},
			true,
		},
		{
			"invalid-namespace-valid-cluster-name-multiRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: badNamespace},
			},
			func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", badNamespace, cluster1).
					Return(fmt.Errorf("Invalid namespace name")).
					Once()
			},
			true,
		},
		{
			"invalid-cluster-name-valid-namespace-kubernetesRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{badClusterName: namespace},
			},
			func(mockRM *mocks.ResourceManager) {},
			false,
		},
		{
			"valid-cluster-name-no-namespace-kubernetesRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: noName},
			},
			func(*mocks.ResourceManager) {},
			false,
		},
		{
			"invalid-namespace-valid-cluster-name-kubernetesRM",
			&apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: namespace},
			},
			func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, cluster1).
					Return(fmt.Errorf("Invalid namespace name")).
					Once()
			},
			false,
		},
	}

	for _, test := range errorCases {
		t.Run(test.name, func(t *testing.T) {
			mockRM := MockRM()
			test.setupMockRM(mockRM)
			setMasterRms(api, mockRM, cluster1, test.multiRM)

			_, err := api.PostWorkspace(ctx, test.req)
			require.Error(t, err, "Test %s failed", test.name)
		})
	}

	type SuccessCase struct {
		name                 string
		req                  *apiv1.PostWorkspaceRequest
		setupMockRM          func(*mocks.ResourceManager)
		multiRM              bool
		expectedWksp         *workspacev1.Workspace
		expectedWsNsBindings map[string]*workspacev1.WorkspaceNamespace
		defaultRMName        string
	}

	successCases := []SuccessCase{
		// TODO PostWorkspace should returned pinnedAt.
		{
			name: "valid-workspace",
			req: &apiv1.PostWorkspaceRequest{
				Name: uuid.NewString(),
				CheckpointStorageConfig: newProtoStruct(t, map[string]any{
					"type":       "s3",
					"bucket":     "bucketofrain",
					"secret_key": "thisisasecret",
				}),
				DefaultComputePool: "testRP",
				DefaultAuxPool:     "testRP",
			},
			setupMockRM: func(*mocks.ResourceManager) {},
			multiRM:     false,
			expectedWksp: &workspacev1.Workspace{
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
			},
			expectedWsNsBindings: nil,
		},
		{
			name: "valid-cluster-name-valid-namespace-multiRM",
			req: &apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, cluster1).Return(nil).Once()
				mockRM.On("DefaultNamespace", cluster1).Return(&defaultNamespace, nil).Once()
			},
			multiRM: true,
			expectedWksp: &workspacev1.Workspace{
				Username: curUser.Username,
				Pinned:   true,
				UserId:   int32(curUser.ID),
			},
			expectedWsNsBindings: map[string]*workspacev1.WorkspaceNamespace{
				cluster1: {
					Namespace:   namespace,
					ClusterName: cluster1,
				},
			},
			defaultRMName: cluster1,
		},
		{
			name: "valid-cluster-name-valid-namespace-kubernetesRM",
			req: &apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{cluster1: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, cluster1).Return(nil).Once()
				mockRM.On("DefaultNamespace", cluster1).Return(&defaultNamespace, nil).Once()
			},
			multiRM: false,
			expectedWksp: &workspacev1.Workspace{
				Username: curUser.Username,
				Pinned:   true,
				UserId:   int32(curUser.ID),
			},
			expectedWsNsBindings: map[string]*workspacev1.WorkspaceNamespace{
				cluster1: {
					Namespace:   namespace,
					ClusterName: cluster1,
				},
			},
			defaultRMName: cluster1,
		},
		{
			name: "valid-namespace-no-cluster-name-kubernetesRM",
			req: &apiv1.PostWorkspaceRequest{
				Name:                  uuid.NewString(),
				ClusterNamespacePairs: map[string]string{noName: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, noName).Return(nil).Once()
				mockRM.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
			},
			multiRM: false,
			expectedWksp: &workspacev1.Workspace{
				Username: curUser.Username,
				Pinned:   true,
				UserId:   int32(curUser.ID),
			},
			expectedWsNsBindings: map[string]*workspacev1.WorkspaceNamespace{
				noName: {
					Namespace:   namespace,
					ClusterName: noName,
				},
			},
		},
	}

	for _, test := range successCases {
		t.Run(test.name, func(t *testing.T) {
			mockRM := MockRM()
			test.setupMockRM(mockRM)
			setMasterRms(api, mockRM, test.defaultRMName, test.multiRM)

			resp, err := api.PostWorkspace(ctx, test.req)
			require.NoError(t, err, "Test %s failed", test.name)
			require.Equal(t, len(test.expectedWsNsBindings), len(resp.NamespaceBindings))

			// Workspace returned correctly?
			test.expectedWksp.Id = resp.Workspace.Id
			test.expectedWksp.Name = test.req.Name

			proto.Equal(test.expectedWksp, resp.Workspace)
			require.Equal(t, test.expectedWksp, resp.Workspace)

			// Workspace-namespace bindings returned correctly?
			for cluster, wsnsBinding := range resp.NamespaceBindings {
				expectedWsNsBinding, ok := test.expectedWsNsBindings[cluster]
				require.True(t, ok)

				expectedWsNsBinding.WorkspaceId = resp.Workspace.Id
				proto.Equal(expectedWsNsBinding, wsnsBinding)
				require.Equal(t, expectedWsNsBinding, wsnsBinding)

				// Workspace-namespace binding successfully added to the database?
				var wsns model.WorkspaceNamespace
				err = db.Bun().NewSelect().Model(&model.WorkspaceNamespace{}).
					Where("workspace_id = ?", resp.Workspace.Id).
					Where("namespace LIKE ?", expectedWsNsBinding.Namespace).
					Scan(ctx, &wsns)
				require.NoError(t, err)
			}

			// Workspace persisted correctly?
			getWorkResp, err := api.GetWorkspace(ctx, &apiv1.GetWorkspaceRequest{Id: resp.Workspace.Id})
			require.NoError(t, err)
			require.NotNil(t, getWorkResp.Workspace.PinnedAt)
			getWorkResp.Workspace.PinnedAt = nil // Can't check timestamp exactly.
			require.Equal(t, test.expectedWksp, getWorkResp.Workspace)
		})
	}
}

func setMasterRms(api *apiServer, mockRM *mocks.ResourceManager, mockRMName string, multiRM bool) {
	api.m.rm = mockRM
	api.m.allRms = map[string]rm.ResourceManager{
		mockRMName: mockRM,
	}
	if multiRM {
		mockRM2 := MockRM()
		cluster2 := "k8sCluster2"
		api.m.allRms[cluster2] = mockRM2
		multiRMRouter := multirm.New(mockRMName, api.m.allRms)
		api.m.rm = multiRMRouter
	}
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
	require.Empty(t, resp)

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
	require.Empty(t, resp)
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
	clusterName := uuid.NewString()
	namespace := uuid.NewString()
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
		{"CanSetWorkspaceNamespaceBindings", func(id int) error {
			api.m.allRms = map[string]rm.ResourceManager{clusterName: api.m.rm}
			_, err := api.SetWorkspaceNamespaceBindings(ctx,
				&apiv1.SetWorkspaceNamespaceBindingsRequest{
					WorkspaceId:           int32(id),
					ClusterNamespacePairs: map[string]string{clusterName: namespace},
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
		require.Equal(t, apiPkg.NotFoundErrs("workspace", strconv.Itoa(id), true).Error(),
			curCase.IDToReqCall(id).Error())

		// A error returned by CanGetWorkspace is returned unmodified.
		cantGetWorkspaceErr := fmt.Errorf("canGetWorkspaceError")
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(cantGetWorkspaceErr).Once()
		require.Equal(t, cantGetWorkspaceErr, curCase.IDToReqCall(id))

		// Deny with permission to view returns error wrapped in forbidden.
		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()

		expectedErr := status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
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
	mockRM := MockRM()
	// set up api server
	api, _, ctx := setupAPITest(t, nil, mockRM)
	api.m.allRms = map[string]rm.ResourceManager{noName: mockRM}
	// set up command service - required for successful DeleteWorkspaceRequest calls
	cs, err := command.NewService(api.m.db, api.m.rm)
	require.NoError(t, err)
	command.SetDefaultService(cs)
	// create workspace with namespace binding
	workspaceName := uuid.New().String()
	namespaceName := uuid.New().String()
	mockRM.On("VerifyNamespaceExists", namespaceName, noName).Return(nil).Once()
	mockRM.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name:                  workspaceName,
		ClusterNamespacePairs: map[string]string{noName: namespaceName},
	})
	require.NoError(t, err)

	var wsns model.WorkspaceNamespace

	findWsNsQuery := db.Bun().NewSelect().Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", resp.Workspace.Id).
		Where("namespace LIKE ?", namespaceName)

	// verify existence of workspace-namespace binding.
	err = findWsNsQuery.Scan(ctx, &wsns)
	require.NoError(t, err)

	// delete workspace without models
	wkspID := resp.Workspace.Id
	autoGeneratedNamespaceName, err := getAutoGeneratedNamespaceName(ctx, int(wkspID))
	require.NoError(t, err)
	mockRM.On("DeleteNamespace", *autoGeneratedNamespaceName).Return(nil).Once()
	_, err = api.DeleteWorkspace(ctx, &apiv1.DeleteWorkspaceRequest{
		Id: resp.Workspace.Id,
	})
	require.NoError(t, err)

	// verify workspace-namespace bindings are removed.
	err = findWsNsQuery.Scan(ctx, &wsns)
	require.Error(t, err)
	require.Equal(t, err, sql.ErrNoRows)

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

func TestSetWorkspaceNamespaceBindings(t *testing.T) {
	testSetWkspNmspBindingsErrorCases(t)
	testSetWkspNmspBindingsSuccessCases(t)
}

func testSetWkspNmspBindingsErrorCases(t *testing.T) {
	mockRM := MockRM()
	api, _, ctx := setupAPITest(t, nil, mockRM)
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.NewString()})
	require.NoError(t, err)
	wkspID := resp.Workspace.Id

	badNamespace := uuid.NewString()
	badClusterName := uuid.NewString()
	namespace := uuid.NewString()
	cluster1 := uuid.NewString()

	type ErrorCases struct {
		name        string
		req         *apiv1.SetWorkspaceNamespaceBindingsRequest
		setupMockRM func(*mocks.ResourceManager)
		multiRM     bool
	}

	errorCases := []ErrorCases{
		{
			name: "invalid-wksp-id",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId: -1,
			},
			setupMockRM: func(*mocks.ResourceManager) {},
			multiRM:     false,
		},
		{
			name: "invalid-namespace-valid-cluster-name-multiRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{cluster1: badNamespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", badNamespace, cluster1).
					Return(fmt.Errorf("namespace does not exist")).Once()
			},
			multiRM: true,
		},
		{
			name: "invalid-cluster-name-valid-namespace-multiRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{badClusterName: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {},
			multiRM:     true,
		},
		{
			name: "no-cluster-name-valid-namespace-multiRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{noName: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {},
			multiRM:     true,
		},
		{
			name: "no-namespace-valid-cluster-name-multiRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{cluster1: noName},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {},
			multiRM:     true,
		},
		{
			name: "invalid-namespace-valid-cluster-name-kubernetesRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{cluster1: badNamespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", badNamespace, cluster1).
					Return(fmt.Errorf("Namespace does not exist")).Once()
			},
			multiRM: false,
		},
		{
			name: "invalid-namespace-no-cluster-name-kubernetesRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{noName: badNamespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", badNamespace, noName).
					Return(fmt.Errorf("namespace does not exist")).Once()
			},
			multiRM: false,
		},
		{
			name: "invalid-cluster-name-valid-namespace-kubernetesRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{badClusterName: namespace},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {},
			multiRM:     false,
		},
		{
			name: "no-namespace-valid-cluster-name-kubernetesRM",
			req: &apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{cluster1: noName},
			},
			setupMockRM: func(mockRM *mocks.ResourceManager) {},
			multiRM:     false,
		},
	}

	for _, test := range errorCases {
		t.Run(test.name, func(t *testing.T) {
			mockRM := MockRM()
			test.setupMockRM(mockRM)
			setMasterRms(api, mockRM, cluster1, test.multiRM)
			_, err := api.SetWorkspaceNamespaceBindings(ctx, test.req)
			require.Error(t, err)
		})
	}
}

func testSetWkspNmspBindingsSuccessCases(t *testing.T) {
	mockRM := MockRM()
	api, _, ctx := setupAPITest(t, nil, mockRM)
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.NewString()})
	require.NoError(t, err)
	wkspID := resp.Workspace.Id

	cluster1 := uuid.NewString()
	cluster2 := uuid.NewString()
	namespace1 := "validnamespace1"
	namespace2 := "validnamespace2"
	newNamespace1Name := "validnamespace3"

	// Setup MultiRM.
	mockRM1 := MockRM()
	mockRM2 := MockRM()
	api.m.allRms = map[string]rm.ResourceManager{
		cluster1: mockRM1,
		cluster2: mockRM2,
	}
	api.m.rm = multirm.New(cluster1, api.m.allRms)

	// Set a workspace-namespace binding for the initially unbound workspace.
	mockRM1.On("VerifyNamespaceExists", namespace1, cluster1).Return(nil).Once()
	mockRM1.On("DefaultNamespace", cluster1).Return(&defaultNamespace, nil).Once()
	respWsNs, err := api.SetWorkspaceNamespaceBindings(
		ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{cluster1: namespace1},
		},
	)
	require.NoError(t, err)

	// Correct workspace-namespace bindings returned?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		cluster1: {
			WorkspaceId: wkspID,
			Namespace:   namespace1,
			ClusterName: cluster1,
		},
	}, respWsNs.NamespaceBindings)

	getBindingsForWkspQuery := db.Bun().NewSelect().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID)

	// Workspace-namespace binding exists in the database?
	numBindings, err := getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	// Create an additional workspace-namespace binding for a different cluster.
	mockRM2.On("VerifyNamespaceExists", namespace2, cluster2).Return(nil).Once()
	mockRM2.On("DefaultNamespace", cluster2).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx, &apiv1.SetWorkspaceNamespaceBindingsRequest{
		WorkspaceId:           wkspID,
		ClusterNamespacePairs: map[string]string{cluster2: namespace2},
	})
	require.NoError(t, err)

	// Correct workspace-namespace binding returned?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		cluster2: {
			WorkspaceId: wkspID,
			Namespace:   namespace2,
			ClusterName: cluster2,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, numBindings)

	// Change the namespace name of the first workspace-namespace binding and verify no error.
	mockRM1.On("VerifyNamespaceExists", newNamespace1Name, cluster1).Return(nil).Once()
	mockRM1.On("DefaultNamespace", cluster1).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{cluster1: newNamespace1Name},
		})
	require.NoError(t, err)

	// Workspace-namespace binding changed successfully?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		cluster1: {
			WorkspaceId: wkspID,
			Namespace:   newNamespace1Name,
			ClusterName: cluster1,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, numBindings)

	// Set workspace-namespace binding to itself (effectively leaving the binding the same).
	mockRM2.On("VerifyNamespaceExists", namespace2, cluster2).Return(nil).Once()
	mockRM2.On("DefaultNamespace", cluster2).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{cluster2: namespace2},
		})
	require.NoError(t, err)

	// Workspace-namespace binding left unchanged?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		cluster2: {
			WorkspaceId: wkspID,
			Namespace:   namespace2,
			ClusterName: cluster2,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, numBindings)

	// Correct number of workspace-namespace bindings for each cluster?
	numBindings, err = db.Bun().NewSelect().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Where("cluster_name LIKE ?", cluster1).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	numBindings, err = db.Bun().NewSelect().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Where("cluster_name LIKE ?", cluster2).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	// Setup KubernetesRM.
	api.m.rm = mockRM1
	delete(api.m.allRms, cluster1)
	delete(api.m.allRms, cluster2)
	api.m.allRms[noName] = mockRM1

	mockRM1.On("VerifyNamespaceExists", namespace1, noName).Return(nil)
	mockRM1.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()

	// Create workspace with namespace-binding.
	resp, err = api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name:                  uuid.NewString(),
		ClusterNamespacePairs: map[string]string{noName: namespace1},
	})
	require.NoError(t, err)
	wkspID = resp.Workspace.Id

	getBindingsForWkspQuery = db.Bun().NewSelect().
		Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID)

	// Correct number of workspace-namespace bindings exist in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	// Modify workspace-namespace binding for workspace
	mockRM1.On("VerifyNamespaceExists", newNamespace1Name, noName).Return(nil).Once()
	mockRM1.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{noName: newNamespace1Name},
		})
	require.NoError(t, err)

	// Workspace-namespace binding modified correctly?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		noName: {
			WorkspaceId: wkspID,
			Namespace:   newNamespace1Name,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	// Set workspace-namespace binding to itself (effectively leaving the binding the same).
	mockRM1.On("VerifyNamespaceExists", newNamespace1Name, noName).Return(nil).Once()
	mockRM1.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{noName: newNamespace1Name},
		})
	require.NoError(t, err)

	// Workspace-namespace binding left unchanged?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		noName: {
			WorkspaceId: wkspID,
			Namespace:   newNamespace1Name,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBindings)

	// Set workspace-namespace binding to the default namespace.
	mockRM1.On("VerifyNamespaceExists", defaultNamespace, noName).Return(nil).Once()
	mockRM1.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
	mockRM1.On("RemoveEmptyNamespace", newNamespace1Name, noName).Return(nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{noName: defaultNamespace},
		})
	require.NoError(t, err)

	// Workspace-namespace binding returned correctly?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		noName: {
			WorkspaceId: wkspID,
			Namespace:   defaultNamespace,
		},
	}, respWsNs.NamespaceBindings)

	// workspace-namespace binding removed from db?
	err = db.Bun().NewSelect().Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Where("cluster_name = ?", noName).
		Scan(ctx)
	require.Error(t, err)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBindings)

	// Again, try to bind the workspace to the default namespace verify nothing changed.
	mockRM1.On("VerifyNamespaceExists", defaultNamespace, noName).Return(nil).Once()
	mockRM1.On("DefaultNamespace", noName).Return(&defaultNamespace, nil).Once()
	respWsNs, err = api.SetWorkspaceNamespaceBindings(ctx,
		&apiv1.SetWorkspaceNamespaceBindingsRequest{
			WorkspaceId:           wkspID,
			ClusterNamespacePairs: map[string]string{noName: defaultNamespace},
		})
	require.NoError(t, err)

	// Workspace-namespace binding returned correctly?
	verifyCorrectWorkspaceNamespaceBindings(t, map[string]*workspacev1.WorkspaceNamespace{
		noName: {
			WorkspaceId: wkspID,
			Namespace:   defaultNamespace,
		},
	}, respWsNs.NamespaceBindings)

	// Correct number of workspace-namespace bindings in the database?
	numBindings, err = getBindingsForWkspQuery.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBindings)
}

func verifyCorrectWorkspaceNamespaceBindings(t *testing.T, expectedWsNsBindings,
	actualWsNsBindings map[string]*workspacev1.WorkspaceNamespace,
) {
	require.Equal(t, len(expectedWsNsBindings), len(actualWsNsBindings))

	for clusterName, wsnsBinding := range actualWsNsBindings {
		expectedWsNsBinding, ok := expectedWsNsBindings[clusterName]
		require.True(t, ok)
		// Correct workspace-namespace binding returned?
		proto.Equal(expectedWsNsBinding, wsnsBinding)
		require.Equal(t, expectedWsNsBinding, wsnsBinding)
	}
}

func TestListWorkspaceNamespaceBindings(t *testing.T) {
	mockRM := MockRM()
	api, _, ctx := setupAPITest(t, nil, mockRM)

	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{
		Name: uuid.NewString(),
	})
	require.NoError(t, err)
	wkspID := resp.Workspace.Id

	// Setup Kubernetes RM.
	emptyClusterName := ""
	allRms := map[string]rm.ResourceManager{emptyClusterName: mockRM}
	api.m.allRms = allRms

	defaultKubernetesNamespaceName := defaultKubernetesNamespace
	mockRM.On("DefaultNamespace", emptyClusterName).Return(&defaultKubernetesNamespaceName, nil)

	// Test list bindings single RM with default Kubernetes namespace (not explicitly added in the
	// database).
	wsNsResp, err := api.ListWorkspaceNamespaceBindings(
		ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, 1)

	expectedWsNsBindings := map[string]string{emptyClusterName: defaultKubernetesNamespaceName}
	require.Equal(t, expectedWsNsBindings, wsNsResp.NamespaceBindings)

	// Test list bindings with namespace and no cluster name (Kubernetes RM).
	namespace1Name := "namespace1"
	mockRM.On("VerifyNamespaceExists", namespace1Name, noName).Return(nil).Once()

	_, err = api.SetWorkspaceNamespaceBindings(ctx, &apiv1.SetWorkspaceNamespaceBindingsRequest{
		WorkspaceId:           wkspID,
		ClusterNamespacePairs: map[string]string{"": namespace1Name},
	},
	)
	require.NoError(t, err)

	wsNsResp, err = api.ListWorkspaceNamespaceBindings(ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, 1)

	expectedWsNsBindings = map[string]string{emptyClusterName: namespace1Name}
	require.Equal(t, expectedWsNsBindings, wsNsResp.NamespaceBindings)

	// Give Kubernetes RM a cluster name. A binding including the new RM's default namespace
	// and cluster name should be added to the list. The previously added binding should still be
	// listed with its binding labeled "stale".
	delete(api.m.allRms, emptyClusterName)
	cluster1Name := "cluster1"
	mockRM.On("DefaultNamespace", cluster1Name).Return(&defaultKubernetesNamespaceName, nil)
	api.m.allRms[cluster1Name] = mockRM

	staleBindings := map[string]int{emptyClusterName + staleLabel: 1}

	wsNsResp, err = api.ListWorkspaceNamespaceBindings(ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, 2)

	expectedBindings := map[string]string{
		emptyClusterName + staleLabel: namespace1Name,
		cluster1Name:                  defaultKubernetesNamespaceName,
	}

	for cluster, ns := range wsNsResp.NamespaceBindings {
		require.Contains(t, expectedBindings, cluster)
		require.Equal(t, expectedBindings[cluster], ns)
	}

	// Setup MultiRM

	// Add more resource manageers which each have a stored workspace-namespace binding.
	numRms := 5
	for i := 2; i <= numRms; i++ {
		mockRM := MockRM()
		clusterName := "cluster" + strconv.Itoa(i)
		namespaceName := "namespace" + strconv.Itoa(i)
		defaultNamespaceName := "default-namespace" + strconv.Itoa(i)
		mockRM.On("DefaultNamespace", clusterName).Return(&defaultNamespaceName, nil)
		mockRM.On("VerifyNamespaceExists", namespaceName, clusterName).Return(nil).Once()
		allRms[clusterName] = mockRM

		expectedBindings[clusterName] = defaultNamespaceName
	}

	api.m.allRms = allRms
	multRMRouter := multirm.New(cluster1Name, api.m.allRms)
	api.m.rm = multRMRouter

	// Test MultiRM list bindings with explicitly set default namespace names (bindings not
	// explicitly set in the database).
	wsNsResp, err = api.ListWorkspaceNamespaceBindings(
		ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, numRms+len(staleBindings))

	for cluster, ns := range wsNsResp.NamespaceBindings {
		require.Contains(t, expectedBindings, cluster)
		require.Equal(t, expectedBindings[cluster], ns)
	}

	for i := 2; i <= numRms; i++ {
		cluster := "cluster" + strconv.Itoa(i)
		namespace := "namespace" + strconv.Itoa(i)
		_, err = api.SetWorkspaceNamespaceBindings(
			ctx,
			&apiv1.SetWorkspaceNamespaceBindingsRequest{
				WorkspaceId:           wkspID,
				ClusterNamespacePairs: map[string]string{cluster: namespace},
			},
		)
		require.NoError(t, err)

		expectedBindings[cluster] = namespace
	}

	// Are all workspace-namespacebindings correctly listed for each respective cluster?
	wsNsResp, err = api.ListWorkspaceNamespaceBindings(
		ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, numRms+len(staleBindings))

	for cluster, ns := range wsNsResp.NamespaceBindings {
		require.Contains(t, expectedBindings, cluster)
		require.Equal(t, expectedBindings[cluster], ns)
	}

	// Remove two of the additional resource managers.
	for i := 2; i <= 3; i++ {
		clusterName := "cluster" + strconv.Itoa(i)
		delete(api.m.allRms, clusterName)
		namespace, ok := expectedBindings[clusterName]
		require.True(t, ok)
		delete(expectedBindings, clusterName)
		expectedBindings[clusterName+staleLabel] = namespace
		staleBindings[clusterName] = 1
	}

	// Are the removed resource managers' workspace-namespace bindings labeled as stale?
	wsNsResp, err = api.ListWorkspaceNamespaceBindings(
		ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, len(api.m.allRms)+len(staleBindings))

	for cluster, ns := range wsNsResp.NamespaceBindings {
		require.Contains(t, expectedBindings, cluster)
		require.Equal(t, expectedBindings[cluster], ns)
	}

	// Remove the stale bindings.
	_, err = db.Bun().NewDelete().Model(&model.WorkspaceNamespace{}).
		Where("workspace_id = ?", wkspID).
		Where("cluster_name LIKE ?", emptyClusterName).
		WhereOr("cluster_name LIKE ?", "cluster2").
		WhereOr("cluster_name LIKE ?", "cluster3").
		Exec(ctx)
	require.NoError(t, err)

	// Stale bindings no longer listed?
	wsNsResp, err = api.ListWorkspaceNamespaceBindings(
		ctx,
		&apiv1.ListWorkspaceNamespaceBindingsRequest{Id: wkspID},
	)
	require.NoError(t, err)
	require.Len(t, wsNsResp.NamespaceBindings, 3)
	require.NoError(t, err)

	for cluster, ns := range wsNsResp.NamespaceBindings {
		require.Contains(t, expectedBindings, cluster)
		require.Equal(t, expectedBindings[cluster], ns)
	}
}

func TestBasicRBACWorkspacePerms(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	curUser.Admin = false
	err := user.Update(ctx, &curUser, []string{"admin"}, nil)
	require.NoError(t, err)
	namespace := uuid.NewString()
	clusterName := uuid.NewString()

	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	wkspID := resp.Workspace.Id

	cases := []struct {
		name          string
		setupMockRM   func(*mocks.ResourceManager)
		funcToExecute func() error
	}{
		{
			"set-wksp-namespace-binding",
			func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, clusterName).Return(nil).Once()
				mockRM.On("DefaultNamespace", clusterName).Return(&defaultNamespace, nil).Once()
			},
			func() error {
				api.m.allRms = map[string]rm.ResourceManager{clusterName: api.m.rm}
				_, err := api.SetWorkspaceNamespaceBindings(ctx,
					&apiv1.SetWorkspaceNamespaceBindingsRequest{
						WorkspaceId:           wkspID,
						ClusterNamespacePairs: map[string]string{clusterName: namespace},
					})
				return err
			},
		},
		{
			"post-wksp-with-namespace-binding",
			func(mockRM *mocks.ResourceManager) {
				mockRM.On("VerifyNamespaceExists", namespace, clusterName).Return(nil).Once()
				mockRM.On("DefaultNamespace", clusterName).Return(&defaultNamespace, nil).Once()
			},
			func() error {
				api.m.allRms = map[string]rm.ResourceManager{clusterName: api.m.rm}
				_, err := api.PostWorkspace(ctx,
					&apiv1.PostWorkspaceRequest{
						Name:                  uuid.NewString(),
						ClusterNamespacePairs: map[string]string{clusterName: namespace},
					})
				return err
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			mockRM := MockRM()
			test.setupMockRM(mockRM)
			api.m.rm = mockRM
			err := test.funcToExecute()
			require.Error(t, err)
		})
	}
}
