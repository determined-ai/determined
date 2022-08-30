//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
)

var projectAuthZ *mocks.ProjectAuthZ

func projectNotFoundErr(id int) error {
	return status.Errorf(codes.NotFound, fmt.Sprintf("project (%d) not found", id))
}

func createTestExpWithProjectID(
	t *testing.T, api *apiServer, curUser model.User, projectID int,
) *model.Experiment {
	exp := &model.Experiment{
		JobID:                model.JobID(uuid.New().String()),
		State:                model.PausedState,
		OwnerID:              &curUser.ID,
		ProjectID:            projectID,
		StartTime:            time.Now(),
		ModelDefinitionBytes: []byte{10, 11, 12},
		Config: schemas.Merge(minExpConfig, expconf.ExperimentConfig{
			RawDescription: ptrs.Ptr("12345"),
			RawName:        expconf.Name{ptrs.Ptr("name")},
		}).(expconf.ExperimentConfig),
	}
	require.NoError(t, api.m.db.AddExperiment(exp))

	// Get experiment as our API mostly will to make it easier to mock.
	exp, err := api.m.db.ExperimentWithoutConfigByID(exp.ID)
	require.NoError(t, err)
	return exp
}

func SetupProjectAuthZTest(
	t *testing.T,
) (*apiServer, *mocks.ProjectAuthZ, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, workspaceAuthZ, curUser, ctx := SetupWorkspaceAuthZTest(t)

	if projectAuthZ == nil {
		projectAuthZ = &mocks.ProjectAuthZ{}
		project.AuthZProvider.Register("mock", projectAuthZ)
	}
	return api, projectAuthZ, workspaceAuthZ, curUser, ctx
}

func TestAuthZCanCreateProject(t *testing.T) {
	api, projectAuthZ, workspaceAuthZ, _, ctx := SetupProjectAuthZTest(t)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	resp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := int(resp.Workspace.Id)

	// Workspace deny returns same as 404.
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: -9999,
	})
	require.Equal(t, workspaceNotFoundErr(-9999).Error(), err.Error())

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t, workspaceNotFoundErr(workspaceID).Error(), err.Error())

	// Workspace error returns error unmodified.
	expectedErr := fmt.Errorf("canGetWorkspaceErr")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).
		Return(false, expectedErr).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())

	// Can view workspace but can't deny returns error wrapped in forbidden.
	expectedErr = status.Error(codes.PermissionDenied, "canGetWorkspaceDeny")
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything).
		Return(fmt.Errorf("canGetWorkspaceDeny")).Once()
	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name:        uuid.New().String(),
		WorkspaceId: int32(workspaceID),
	})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZGetProject(t *testing.T) {
	api, projectAuthZ, _, _, ctx := SetupProjectAuthZTest(t)

	// Deny returns same as 404,
	_, err := api.GetProject(ctx, &apiv1.GetProjectRequest{Id: -9999})
	require.Equal(t, projectNotFoundErr(-9999).Error(), err.Error())

	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.GetProject(ctx, &apiv1.GetProjectRequest{Id: 1})
	require.Equal(t, projectNotFoundErr(1).Error(), err.Error())

	// An error returned by CanGetProject is returned unmodified.
	expectedErr := fmt.Errorf("canGetProjectErr")
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, expectedErr).Once()
	_, err = api.GetProject(ctx, &apiv1.GetProjectRequest{Id: 1})
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZCanMoveProject(t *testing.T) {
	// Setup.
	api, projectAuthZ, workspaceAuthZ, _, ctx := SetupProjectAuthZTest(t)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	fromResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	toResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := toResp.Workspace.Id

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything).Return(nil).Once()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: uuid.New().String(), WorkspaceId: fromResp.Workspace.Id,
	})
	require.NoError(t, err)
	projectID := resp.Project.Id

	req := &apiv1.MoveProjectRequest{ProjectId: projectID, DestinationWorkspaceId: workspaceID}

	// Can't view project.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, projectNotFoundErr(int(projectID)).Error(), err.Error())

	// Can't view from workspace.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(true, nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, workspaceNotFoundErr(int(fromResp.Workspace.Id)).Error(), err.Error())

	// Can't move project.
	expectedErr := status.Error(codes.PermissionDenied, "canMoveProjectDeny")
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(true, nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Twice()
	projectAuthZ.On("CanMoveProject", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canMoveProjectDeny")).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZCanMoveProjectExperiments(t *testing.T) {
	// Setup.
	api, projectAuthZ, workspaceAuthZ, curUser, ctx := SetupProjectAuthZTest(t)

	workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
	wResp, err := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, err)
	workspaceID := wResp.Workspace.Id

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything).Return(nil).Once()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: uuid.New().String(), WorkspaceId: workspaceID,
	})
	require.NoError(t, err)
	srcProjectID := resp.Project.Id

	exp := createTestExpWithProjectID(t, api, curUser, int(srcProjectID))
	experimentID := exp.ID

	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
	projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything).Return(nil).Once()
	resp, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: uuid.New().String(), WorkspaceId: workspaceID,
	})
	require.NoError(t, err)
	destProjectID := resp.Project.Id

	req := &apiv1.MoveExperimentRequest{
		ExperimentId:         int32(experimentID),
		DestinationProjectId: destProjectID,
	}

	// Can't view destination project.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t, projectNotFoundErr(int(destProjectID)).Error(), err.Error())

	// Can't view source project
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(true, nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, nil).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t, projectNotFoundErr(int(srcProjectID)).Error(), err.Error())

	// Can't move experiment.
	expectedErr := status.Error(codes.PermissionDenied, "canMoveProjectExperimentsDeny")
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(true, nil).Twice()
	projectAuthZ.On("CanMoveProjectExperiments", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(fmt.Errorf("canMoveProjectExperimentsDeny")).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t, expectedErr.Error(), err.Error())
}

func TestAuthZRoutesGetProjectThenAction(t *testing.T) {
	api, projectAuthZ, workspaceAuthZ, _, ctx := SetupProjectAuthZTest(t)

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
		// Create a project and workspace.
		workspaceAuthZ.On("CanCreateWorkspace", mock.Anything, mock.Anything).Return(nil).Once()
		wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
		require.NoError(t, werr)

		workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything).Return(true, nil).Once()
		projectAuthZ.On("CanCreateProject", mock.Anything, mock.Anything).Return(nil).Once()
		resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
			Name: uuid.New().String(), WorkspaceId: wresp.Workspace.Id,
		})
		require.NoError(t, err)
		projectID := int(resp.Project.Id)

		// Project not found.
		err = curCase.IDToReqCall(-9999)
		require.Equal(t, projectNotFoundErr(-9999).Error(), err.Error())

		// Project can't be viewed.
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).Return(false, nil).Once()
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, projectNotFoundErr(projectID).Error(), err.Error())

		// Error checking if project errors during view check.
		expectedErr := fmt.Errorf("canGetProjectError")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).
			Return(false, expectedErr).Once()
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, expectedErr, err)

		// Can view but can't perform action.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything).
			Return(true, nil).Once()
		projectAuthZ.On(curCase.DenyFuncName, mock.Anything, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Deny"))
		err = curCase.IDToReqCall(projectID)
		require.Equal(t, expectedErr.Error(), err.Error())
	}
}
