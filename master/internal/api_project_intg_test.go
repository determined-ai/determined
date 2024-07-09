//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

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
	"github.com/determined-ai/determined/master/pkg/random"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/rbacv1"
	"github.com/determined-ai/determined/proto/pkg/userv1"
)

const mockType = "mock"

var pAuthZ *mocks.ProjectAuthZ

func isMockAuthZ() bool {
	return config.GetMasterConfig().Security.AuthZ.Type == mockType
}

// pgdb can be nil to use the singleton database for testing.
func setupProjectAuthZTest(
	t *testing.T, pgdb *db.PgDB,
	altMockRM ...*mocks.ResourceManager,
) (*apiServer, *mocks.ProjectAuthZ, *mocks.WorkspaceAuthZ, model.User, context.Context) {
	api, workspaceAuthZ, curUser, ctx := setupWorkspaceAuthZTest(t, pgdb, altMockRM...)

	if pAuthZ == nil {
		pAuthZ = &mocks.ProjectAuthZ{}
		project.AuthZProvider.Register(mockType, pAuthZ)
	}
	return api, pAuthZ, workspaceAuthZ, curUser, ctx
}

func createProjectAndWorkspace(ctx context.Context, t *testing.T, api *apiServer) (wkspID int, projID int) {
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
		apiPkg.NotFoundErrs("workspace", strconv.Itoa(workspaceID), true).Error(), err.Error())

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
		apiPkg.NotFoundErrs("project", strconv.Itoa(int(projectID)), true).Error(), err.Error())

	// Can't view from workspace.
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	workspaceAuthZ.On("CanGetWorkspace", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.MoveProject(ctx, req)
	require.Equal(t, apiPkg.NotFoundErrs("workspace",
		strconv.Itoa(int(fromResp.Workspace.Id)), true).Error(), err.Error())

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
		apiPkg.NotFoundErrs("project", strconv.Itoa(srcProjectID), true).Error(), err.Error())

	// Can't view destination project
	authZExp.On("CanGetExperiment", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Once()
	projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
		Return(authz2.PermissionDeniedError{}).Once()
	_, err = api.MoveExperiment(ctx, req)
	require.Equal(t,
		apiPkg.NotFoundErrs("project", strconv.Itoa(destProjectID), true).Error(), err.Error())

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
		{"CanSetProjectKey", func(id int) error {
			_, err := api.PatchProject(ctx, &apiv1.PatchProjectRequest{
				Project: &projectv1.PatchProject{Key: wrapperspb.String("newma")},
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
		require.Error(t, err)
		require.Equal(t, apiPkg.NotFoundErrs("project", "-9999", true).Error(), err.Error())

		// Project can't be viewed.
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(authz2.PermissionDeniedError{}).Once()
		err = curCase.IDToReqCall(projectID)
		require.Error(t, err)
		require.Equal(t, apiPkg.NotFoundErrs("project", strconv.Itoa(projectID), true).Error(),
			err.Error())

		// Error checking if project errors during view check.
		expectedErr := fmt.Errorf("canGetProjectError")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(expectedErr).Once()
		err = curCase.IDToReqCall(projectID)
		require.Error(t, err)
		require.Equal(t, expectedErr, err)

		// Can view but can't perform action.
		expectedErr = status.Error(codes.PermissionDenied, curCase.DenyFuncName+"Deny")
		projectAuthZ.On("CanGetProject", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		projectAuthZ.On(curCase.DenyFuncName, mock.Anything, mock.Anything, mock.Anything).
			Return(fmt.Errorf(curCase.DenyFuncName + "Deny"))
		err = curCase.IDToReqCall(projectID)
		require.Error(t, err)
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
	require.Len(t, resp.Projects, 1)
}

func TestGetProjectColumnsRuns(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)

	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)

	exp1 := createTestExpWithProjectID(t, api, curUser, projectIDInt)
	exp2 := createTestExpWithProjectID(t, api, curUser, projectIDInt)

	hyperparameters1 := map[string]any{"global_batch_size": 1}

	task1 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task1))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp1.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters1,
	}, task1.TaskID))

	getColumnsReq := &apiv1.GetProjectColumnsRequest{
		Id:        int32(projectIDInt),
		TableType: apiv1.TableType_TABLE_TYPE_RUN.Enum(),
	}

	getColumnsResp, err := api.GetProjectColumns(ctx, getColumnsReq)
	require.NoError(t, err)
	require.Len(t, getColumnsResp.Columns, len(defaultRunsTableColumns)+1)
	for i, column := range defaultRunsTableColumns {
		require.Equal(t, column, getColumnsResp.Columns[i])
	}
	expectedHparam := &projectv1.ProjectColumn{
		Column:   "hp.global_batch_size",
		Location: projectv1.LocationType_LOCATION_TYPE_RUN_HYPERPARAMETERS,
		Type:     projectv1.ColumnType_COLUMN_TYPE_NUMBER,
	}
	require.Equal(t, expectedHparam, getColumnsResp.Columns[len(getColumnsResp.Columns)-1])

	hyperparameters2 := map[string]any{"test1": map[string]any{"test2": "text_val"}}
	task2 := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task2))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.PausedState,
		ExperimentID: exp2.ID,
		StartTime:    time.Now(),
		HParams:      hyperparameters2,
	}, task2.TaskID))

	getColumnsResp, err = api.GetProjectColumns(ctx, getColumnsReq)
	require.NoError(t, err)
	require.Len(t, getColumnsResp.Columns, len(defaultRunsTableColumns)+2)
	expectedHparam = &projectv1.ProjectColumn{
		Column:   "hp.test1.test2",
		Location: projectv1.LocationType_LOCATION_TYPE_RUN_HYPERPARAMETERS,
		Type:     projectv1.ColumnType_COLUMN_TYPE_TEXT,
	}
	require.Equal(t, expectedHparam, getColumnsResp.Columns[len(getColumnsResp.Columns)-1])
}

func TestCreateProjectWithoutProjectKey(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	projectKeyPrefix := strings.ToUpper(projectName[:project.MaxProjectKeyPrefixLength])
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)
	require.Equal(t, projectKeyPrefix, resp.Project.Key[:project.MaxProjectKeyPrefixLength])
}

func TestCreateProjectWithProjectKey(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	projectKey := random.String(project.MaxProjectKeyLength)
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id, Key: &projectKey,
	})
	require.NoError(t, err)

	// Check that the project key is generated correctly.
	err = db.Bun().NewSelect().
		Column("key").
		Table("projects").
		Where("id = ?", resp.Project.Id).
		Scan(ctx, &resp.Project.Key)
	require.NoError(t, err)
	require.Equal(t, projectKey, resp.Project.Key)
}

func TestCreateProjectWithDuplicateProjectKey(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	projectKey := random.String(project.MaxProjectKeyLength)
	_, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id, Key: &projectKey,
	})
	require.NoError(t, err)

	_, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName + "2", WorkspaceId: wresp.Workspace.Id, Key: &projectKey,
	})
	require.Error(t, err)
	require.ErrorContains(t, err, fmt.Sprintf("project key %s is already in use", projectKey))
}

func TestCreateProjectWithDefaultKeyAndDuplicatePrefix(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := uuid.New().String()
	projectKeyPrefix := strings.ToUpper(projectName[:project.MaxProjectKeyPrefixLength])
	resp1, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)
	require.Equal(t, projectKeyPrefix, resp1.Project.Key[:project.MaxProjectKeyPrefixLength])

	resp2, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName + "2", WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)
	require.NoError(t, err)
	require.Equal(t, projectKeyPrefix, resp2.Project.Key[:project.MaxProjectKeyPrefixLength])
}

func TestConcurrentProjectKeyGenerationAttempts(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)
	numRequests := 5
	errgrp := errgroupx.WithContext(ctx)
	for i := 0; i < numRequests; i++ {
		projectName := "test-project" + uuid.New().String()
		errgrp.Go(func(context.Context) error {
			_, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
				Name: projectName, WorkspaceId: wresp.Workspace.Id,
			})
			require.NoError(t, err)
			return err
		})

		require.NoError(t, errgrp.Wait())
		t.Cleanup(func() {
			_, err := db.Bun().NewDelete().Table("projects").Where("workspace_id = ?", wresp.Workspace.Id).Exec(ctx)
			require.NoError(t, err)
			_, err = db.Bun().NewDelete().Table("workspaces").Where("id = ?", wresp.Workspace.Id).Exec(ctx)
			require.NoError(t, err)
		})
	}
}

func TestPatchProject(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	newName := uuid.New().String()
	newDescription := uuid.New().String()
	newKey := random.String(project.MaxProjectKeyLength)
	_, err = api.PatchProject(ctx, &apiv1.PatchProjectRequest{
		Id: resp.Project.Id,
		Project: &projectv1.PatchProject{
			Name:        wrapperspb.String(newName),
			Description: wrapperspb.String(newDescription),
			Key:         wrapperspb.String(newKey),
		},
	})
	require.NoError(t, err)

	// Check that the project was updated correctly.
	var project model.Project
	err = db.Bun().NewSelect().
		Model(&project).
		Where("id = ?", resp.Project.Id).
		Scan(ctx)
	require.NoError(t, err)
	require.Equal(t, newName, project.Name)
	require.Equal(t, newDescription, project.Description)
	require.Equal(t, strings.ToUpper(newKey), project.Key)
}

func TestPatchProjectRecordRedirect(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	oldKey := resp.Project.Key

	exp := createTestExpWithProjectID(t, api, curUser, int(resp.Project.Id))
	task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
	require.NoError(t, db.AddTask(ctx, task))
	require.NoError(t, db.AddTrial(ctx, &model.Trial{
		State:        model.CompletedState,
		ExperimentID: exp.ID,
		StartTime:    time.Now(),
	}, task.TaskID))

	newName := uuid.New().String()
	newDescription := uuid.New().String()
	newKey := random.String(project.MaxProjectKeyLength)
	_, err = api.PatchProject(ctx, &apiv1.PatchProjectRequest{
		Id: resp.Project.Id,
		Project: &projectv1.PatchProject{
			Name:        wrapperspb.String(newName),
			Description: wrapperspb.String(newDescription),
			Key:         wrapperspb.String(newKey),
		},
	})
	require.NoError(t, err)

	// Check that new local id is recorded in redirect table
	var numRuns int
	err = db.Bun().NewSelect().
		Table("runs").
		ColumnExpr("COUNT(*) as num_runs").
		Where("project_id = ?", resp.Project.Id).
		Scan(ctx, &numRuns)
	require.NoError(t, err)

	var numRunsRedirect int
	err = db.Bun().NewSelect().
		Table("local_id_redirect").
		ColumnExpr("COUNT(*) as num_runs_redirect").
		Where("project_key = ?", resp.Project.Key).
		Scan(ctx, &numRunsRedirect)
	require.NoError(t, err)
	require.Equal(t, numRuns, numRunsRedirect)

	// Allow project to go back to old key
	_, err = api.PatchProject(ctx, &apiv1.PatchProjectRequest{
		Id: resp.Project.Id,
		Project: &projectv1.PatchProject{
			Key: wrapperspb.String(oldKey),
		},
	})
	require.NoError(t, err)

	// Do not allow new project to take old key
	newProjectName := "test-project-new" + uuid.New().String()
	resp, err = api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: newProjectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	_, err = api.PatchProject(ctx, &apiv1.PatchProjectRequest{
		Id: resp.Project.Id,
		Project: &projectv1.PatchProject{
			Key: wrapperspb.String(oldKey),
		},
	})
	require.Error(t, err)
	require.Equal(t, status.Errorf(codes.AlreadyExists,
		"error updating project %s, provided key '%s' already in use in redirect table", newProjectName, oldKey), err)
}

func TestPatchProjectWithDuplicateProjectKey(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp1, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	projectName = "test-project" + uuid.New().String()
	resp2, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	_, err = api.PatchProject(ctx, &apiv1.PatchProjectRequest{
		Id: resp2.Project.Id,
		Project: &projectv1.PatchProject{
			Key: wrapperspb.String(resp1.Project.Key),
		},
	})
	require.Error(t, err)
	require.Equal(t, status.Errorf(codes.AlreadyExists, "project key %s is already in use", resp1.Project.Key), err)
}

func TestPatchProjectWithConcurrent(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	newName := "new-name"
	newDescription := "new-description"
	errgrp := errgroupx.WithContext(ctx)
	for i := 0; i < 20; i++ {
		newKey := random.String(project.MaxProjectKeyLength)
		errgrp.Go(func(context.Context) error {
			_, err := api.PatchProject(ctx, &apiv1.PatchProjectRequest{
				Id: resp.Project.Id,
				Project: &projectv1.PatchProject{
					Name:        wrapperspb.String(newName),
					Description: wrapperspb.String(newDescription),
					Key:         wrapperspb.String(newKey),
				},
			})
			require.NoError(t, err)
			return err
		})
	}
	require.NoError(t, errgrp.Wait())
}

func TestPatchProjectWithInvalidProjectKey(t *testing.T) {
	api, _, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	type TestCase struct {
		Description string
		Key         string
		Err         string
	}
	testCases := []TestCase{
		{
			Description: "empty key",
			Key:         "",
			Err:         "project key cannot be empty",
		},
		{
			Description: "key with special characters",
			Key:         "!@#$%",
			Err:         "project key can only contain alphanumeric characters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Description, func(t *testing.T) {
			_, err := api.PatchProject(ctx, &apiv1.PatchProjectRequest{
				Id: resp.Project.Id,
				Project: &projectv1.PatchProject{
					Key: wrapperspb.String(tc.Key),
				},
			})
			require.Error(t, err)
			require.ErrorContains(t, err, tc.Err)
		})
	}
}

func TestGetProjectByID(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	wresp, werr := api.PostWorkspace(ctx, &apiv1.PostWorkspaceRequest{Name: uuid.New().String()})
	require.NoError(t, werr)

	projectName := "test-project" + uuid.New().String()
	resp, err := api.PostProject(ctx, &apiv1.PostProjectRequest{
		Name: projectName, WorkspaceId: wresp.Workspace.Id,
	})
	require.NoError(t, err)

	project, err := api.GetProjectByID(ctx, resp.Project.Id, curUser)
	require.NoError(t, err)
	require.Equal(t, wresp.Workspace.Name, project.WorkspaceName)
	require.Equal(t, wresp.Workspace.Id, project.WorkspaceId)
	require.Equal(t, projectName, project.Name)
	require.Equal(t, resp.Project.Id, project.Id)
}

func TestGetMetadataValues(t *testing.T) {
	api, curUser, ctx := setupAPITest(t, nil)
	_, projectIDInt := createProjectAndWorkspace(ctx, t, api)
	projectID := int32(projectIDInt)
	exp := createTestExpWithProjectID(t, api, curUser, int(projectID))

	numRuns := 4
	for i := 0; i < numRuns; i++ {
		task := &model.Task{TaskType: model.TaskTypeTrial, TaskID: model.NewTaskID()}
		require.NoError(t, db.AddTask(context.Background(), task))
		require.NoError(t, db.AddTrial(context.Background(), &model.Trial{
			State:        model.PausedState,
			ExperimentID: exp.ID,
			StartTime:    time.Now(),
		}, task.TaskID))
	}

	resp, err := api.SearchRuns(ctx, &apiv1.SearchRunsRequest{ProjectId: &projectID})
	require.NoError(t, err)

	// Add metadata
	rawMetadata1 := map[string]any{
		"test_key": "test_value1",
		"nested": map[string]any{
			"nested_key": "nested_value1",
		},
	}
	rawMetadata2 := map[string]any{
		"test_key": "test_value1",
		"nested": map[string]any{
			"nested_key": "nested_value2",
		},
	}
	rawMetadata3 := map[string]any{
		"test_key": "test_value2",
		"nested": map[string]any{
			"nested_key": "nested_value2",
		},
	}
	rawMetadata4 := map[string]any{
		"test_key": "test_value3",
		"nested": map[string]any{
			"nested_key": "nested_value1",
		},
	}

	rawMetadata := []map[string]any{}
	rawMetadata = append(rawMetadata, rawMetadata1, rawMetadata2, rawMetadata3, rawMetadata4)
	for i := 0; i < numRuns; i++ {
		metadata := newProtoStruct(t, rawMetadata[i])
		_, err = api.PostRunMetadata(ctx, &apiv1.PostRunMetadataRequest{
			RunId:    resp.Runs[i].Id,
			Metadata: metadata,
		})
		require.NoError(t, err)
	}

	getMetadataResp, err := api.GetMetadataValues(ctx, &apiv1.GetMetadataValuesRequest{
		Key: "test_key", ProjectId: projectID,
	})
	require.NoError(t, err)
	require.Len(t, getMetadataResp.Values, 3)

	getMetadataResp, err = api.GetMetadataValues(ctx, &apiv1.GetMetadataValuesRequest{
		Key: "nested.nested_key", ProjectId: projectID,
	})
	require.NoError(t, err)
	require.Len(t, getMetadataResp.Values, 2)
}
