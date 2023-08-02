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
	"github.com/uptrace/bun"
	"google.golang.org/protobuf/types/known/wrapperspb"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

var pAuthZ *mocks.ProjectAuthZ

func isMockAuthZ() bool {
	return config.GetMasterConfig().Security.AuthZ.Type == "mock"
}

// pgdb can be nil to use the singleton database for testing.
func setupProjectAuthZTest(
	t *testing.T, pgdb *db.PgDB,
) (*apiServer, *mocks.ProjectAuthZ, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, workspaceAuthZ, curUser, ctx := setupWorkspaceAuthZTest(t, pgdb)

	if pAuthZ == nil {
		pAuthZ = &mocks.ProjectAuthZ{}
		project.AuthZProvider.Register("mock", pAuthZ)
	}
	return api, pAuthZ, workspaceAuthZ, curUser, ctx
}

func createProjectAndWorkspace(ctx context.Context, t *testing.T, api *apiServer) (int, int) {
	if isMockAuthZ() {
		wAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	}
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	if isMockAuthZ() {
		wAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
	}
	if isMockAuthZ() {
		pAuthZ.On("CanCreateProject", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
	}
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: uuid.New().String(), WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	return int(wresp.Workspace.Id), int(resp.Project.Id)
}

func TestAuthZCanCreateProject(t *testing.T) {
	api, projectAuthZ, workspaceAuthZ, _, ctx := setupProjectAuthZTest(t, nil)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).
		Return(nil).Once()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := int(resp.Workspace.Id)

	// Workspace deny returns same as 404.
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: -9999,
	})
	require.Equal(t, apiPkg.NotFoundErrs("workspace", "-9999", true).Error(), err.Error())

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t,
		apiPkg.NotFoundErrs("workspace", fmt.Sprint(workspaceID), true).Error(), err.Error())

	// Workspace error returns error unmodified.
	expectedErr := fmt.Errorf("canGetWorkspaceErr")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Can view workspace but can't deny returns error wrapped in forbidden.
	expectedErr = status.Error(codes.PermissionDenied, "canGetWorkspaceDeny")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canGetWorkspaceDeny")).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZGetProject(t *testing.T) {
	api, projectAuthZ, _, _, ctx := setupProjectAuthZTest(t, nil)

	// Deny returns same as 404,
	_, err := api.GetProject(ctx, &apiv1.GetProjectRequest{Id: -9999})
	require.Equal(t, apiPkg.NotFoundErrs("project", "-9999", true).Error(), err.Error())

	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.GetProject(ctx, &apiv1.GetProjectRequest{Id: 1})
	require.Equal(t, apiPkg.NotFoundErrs("project", "1", true).Error(), err.Error())

	// An error returned by CanGetProject is returned unmodified.
	expectedErr := fmt.Errorf("canGetProjectErr")
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr).Once()
	_, err = api.GetProject(ctx, &apiv1.GetProjectRequest{Id: 1})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZCanMoveProject(t *testing.T) {
	// Setup.
	api, projectAuthZ, workspaceAuthZ, _, ctx := setupProjectAuthZTest(t, nil)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	fromResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).
		Return(nil).Once()
	toResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := toResp.Workspace.Id

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: uuid.New().String(), WorkspaceId: fromResp.Workspace.Id,
	})
	require.NoError(t, err)
	projectID := resp.Project.Id

	req := &apiv1.MoveProjectRequest{ProjectId: projectID, DestinationWorkspaceId: workspaceID}

	// Can't view project.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t,
		apiPkg.NotFoundErrs("project", fmt.Sprint(projectID), true).Error(), err.Error())

	// Can't view from workspace.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, apiPkg.NotFoundErrs("workspace",
		fmt.Sprint(int(fromResp.Workspace.Id)), true).Error(), err.Error())

	// Can't move project.
	expectedErr := status.Error(codes.PermissionDenied, "canMoveProjectDeny")
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()
	projectAuthZ.On("CanMoveProject", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything).Return(fmt.Errorf("canMoveProjectDeny")).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZCanMoveProjectExperiments(t *testing.T) {
	// Setup.
	api, authZExp, projectAuthZ, curUser, ctx := setupExpAuthTest(t, nil)

	_, srcProjectID := createProjectAndWorkspace(ctx, t, api)
	_, destProjectID := createProjectAndWorkspace(ctx, t, api)
	exp := createTestExpWithProjectID(t, api, curUser, srcProjectID)
	experimentID := exp.ID

	req := &apiv1.MoveExperimentRequest{
		ExperimentId:         int32(experimentID),
		DestinationProjectId: int32(destProjectID),
	}

	// Can't view source project.
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err := api.MoveExperiment(ctx, req)
	require.Equal(t,
		apiPkg.NotFoundErrs("project", fmt.Sprint(srcProjectID), true).Error(), err.Error())

	// Can't view destination project
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t,
		apiPkg.NotFoundErrs("project", fmt.Sprint(destProjectID), true).Error(), err.Error())

	// Can't create experiment in destination project.
	expectedErr := status.Error(codes.PermissionDenied, "canCreateExperimentDeny")
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()
	authZExp.On("CanCreateExperiment", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canCreateExperimentDeny")).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())

	// Can't view and delete experiments from source projects.
	resQuery := &bun.SelectQuery{}
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Twice()
	authZExp.On("CanCreateExperiment", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	authZExp.On("FilterExperimentsQuery", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		[]rbacv1.PermissionType{
			rbacv1.PermissionType_PERMISSION_TYPE_VIEW_EXPERIMENT_METADATA,
			rbacv1.PermissionType_PERMISSION_TYPE_DELETE_EXPERIMENT,
		}).
		Return(resQuery, expectedErr).Once().Run(func(args mock.Arguments) {
		q := args.Get(3).(*bun.SelectQuery)
		*resQuery = *q
	})
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZRoutesGetProjectThenAction(t *testing.T) {
	api, projectAuthZ, _, _, ctx := setupProjectAuthZTest(t, nil)

	cases := []struct {
		DenyFuncName string
		IDToReqCall  func(id int) error
	}{
		{"CanSetProjectNotes", func(id int) error {
			_, err := api.AddProjectNote(ctx, &apiv1.AddProjectNoteRequest{
				Note:      &projectv1.Note{Name: "x", Contents: "y"},
				ProjectId: int32(id),
			})
			return err
		}},
		{"CanSetProjectNotes", func(id int) error {
			_, err := api.PutProjectNotes(ctx, &apiv1.PutProjectNotesRequest{
				Notes:     []*projectv1.Note{{Name: "x", Contents: "y"}},
				ProjectId: int32(id),
			})
			return err
		}},
		{"CanSetProjectName", func(id int) error {
			_, err := api.PatchProject(ctx, &apiv1.PatchProjectRequest{
				Project: &projectv1.PatchProject{Name: wrapperspb.String("newman")},
				Id:      int32(id),
			})
			return err
		}},
		{"CanSetProjectDescription", func(id int) error {
			_, err := api.PatchProject(ctx, &apiv1.PatchProjectRequest{
				Project: &projectv1.PatchProject{Description: wrapperspb.String("newman")},
				Id:      int32(id),
			})
			return err
		}},
		{"CanDeleteProject", func(id int) error {
			_, err := api.DeleteProject(ctx, &apiv1.DeleteProjectRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanArchiveProject", func(id int) error {
			_, err := api.ArchiveProject(ctx, &apiv1.ArchiveProjectRequest{
				Id: int32(id),
			})
			return err
		}},
		{"CanUnarchiveProject", func(id int) error {
			_, err := api.UnarchiveProject(ctx, &apiv1.UnarchiveProjectRequest{
				Id: int32(id),
			})
			return err
		}},
	}

	for _, curCase := range cases {
		_, projectID := createProjectAndWorkspace(ctx, t, api)

		// Project not found.
		err := curCase.IDToReqCall(-9999)
		require.Equal(t, apiPkg.NotFoundErrs("project", "-9999", true).Error(), err.Error())

		// Project can't be viewed.
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, apiPkg.NotFoundErrs("project", fmt.Sprint(projectID), true).Error(),
			err.Error())

		// Error checking if project errors during view check.
		expectedErr := fmt.Errorf("canGetProjectError")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(expectedErr).Once()
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, expectedErr, err)

		// Can view but can't perform action.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		projectAuthZ.On(curCase.DenyFuncName, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Deny"))
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, expectedErr.Error(), err.Error())
	}
}

func TestGetProjectByActivity(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	_, projectID := createProjectAndWorkspace(ctx, t, api)

	_, err := api.PostUserActivity(ctx, &apiv1.PostUserActivityRequest{
		ActivityType: userv1.ActivityType_ACTIVITY_TYPE_GET,
		EntityType:   userv1.EntityType_ENTITY_TYPE_PROJECT,
		EntityId:     int32(projectID),
	})

	require.NoError(t, err)

	resp, err := api.GetProjectsByUserActivity(ctx, &apiv1.GetProjectsByUserActivityRequest{
		Limit: 1,
	})
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, 1, len(resp.Projects))
}
