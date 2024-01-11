package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ghodss/yaml"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/project"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/check"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/projectv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

func (a *apiServer) getGenericTaskLaunchParameters(
	ctx context.Context,
	contextDirectory []*utilv1.File,
	configYAML string,
	projectID int,
) (
	*tasks.GenericTaskSpec, []pkgCommand.LaunchWarning, []byte, error,
) {
	genericTaskSpec := &tasks.GenericTaskSpec{
		ProjectID: projectID,
	}

	// Validate the userModel and get the agent userModel group.
	userModel, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			nil,
			nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}

	proj, err := a.GetProjectByID(ctx, int32(genericTaskSpec.ProjectID), *userModel)
	if err != nil {
		return nil, nil, nil, err
	}
	agentUserGroup, err := user.GetAgentUserGroup(ctx, userModel.ID, int(proj.WorkspaceId))
	if err != nil {
		return nil, nil, nil, err
	}

	// Validate the resource configuration.
	resources := model.ParseJustResources([]byte(configYAML))

	poolName, launchWarnings, err := a.m.ResolveResources(resources.ResourcePool, resources.Slots, int(proj.WorkspaceId))
	if err != nil {
		return nil, nil, nil, err
	}
	// Get the base TaskSpec.
	taskSpec, err := a.m.fillTaskSpec(poolName, agentUserGroup, userModel)
	if err != nil {
		return nil, nil, nil, err
	}

	// Get the full configuration.
	taskConfig := model.DefaultConfigGenericTaskConfig(&taskSpec.TaskContainerDefaults)
	workDirInDefaults := taskConfig.WorkDir

	if err := yaml.UnmarshalStrict([]byte(configYAML), &taskConfig, yaml.DisallowUnknownFields); err != nil {
		return nil, nil, nil, fmt.Errorf("yaml unmarshaling generic task config: %w", err)
	}

	// Copy discovered (default) resource pool name and slot count.

	fillTaskConfig(resources.Slots, taskSpec, &taskConfig.Environment)
	taskConfig.Resources.RawResourcePool = &poolName
	taskConfig.Resources.RawSlots = &resources.Slots

	var contextDirectoryBytes []byte
	taskConfig.WorkDir, contextDirectoryBytes, err = fillContextDir(
		taskConfig.WorkDir,
		workDirInDefaults,
		contextDirectory,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	var token string
	token, err = getTaskSessionToken(ctx, userModel)
	if err != nil {
		return nil, nil, nil, err
	}

	taskSpec.UserSessionToken = token

	genericTaskSpec.Base = taskSpec
	genericTaskSpec.GenericTaskConfig = taskConfig

	genericTaskSpec.Base.ExtraEnvVars = map[string]string{
		"DET_TASK_TYPE": string(model.TaskTypeGeneric),
	}

	return genericTaskSpec, launchWarnings, contextDirectoryBytes, nil
}

func (a *apiServer) canCreateGenericTask(ctx context.Context, projectID int) error {
	userModel, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return err
	}

	errProjectNotFound := api.NotFoundErrs("project", fmt.Sprint(projectID), true)
	p := &projectv1.Project{}
	// Get project details
	if err := db.Bun().NewSelect().TableExpr("pe, projects AS p").With("pe", db.Bun().NewSelect().
		ModelTableExpr("experiments").ColumnExpr(`COUNT(*) AS num_experiments,
		SUM(CASE WHEN state = 'ACTIVE' THEN 1 ELSE 0 END) AS num_active_experiments,
		MAX(start_time) AS last_experiment_started_at`).Where("project_id = ?", projectID)).
		ColumnExpr(`p.id,p.name,p.workspace_id,p.description,p.immutable,
		p.notes,w.name AS workspace_name,p.error_message,
		(p.archived OR w.archived) AS archived,MAX(pe.num_experiments) AS num_experiments,
		MAX(pe.num_active_experiments) AS num_active_experiments,u.username,p.user_id`).
		Join("LEFT JOIN users AS u ON u.id = p.user_id").
		Join("LEFT JOIN workspaces AS w ON w.id = p.workspace_id").
		Where("p.id = ?", projectID).
		GroupExpr("p.id, u.username, w.archived, w.name").
		Scan(ctx, p); errors.Is(err, db.ErrNotFound) {
		return errProjectNotFound
	} else if err != nil {
		return err
	}
	if err := project.AuthZProvider.Get().CanGetProject(ctx, *userModel, p); err != nil {
		return authz.SubIfUnauthorized(err, errProjectNotFound)
	}

	if err := command.AuthZProvider.Get().CanCreateGenericTask(
		ctx, *userModel, model.AccessScopeID(p.WorkspaceId)); err != nil {
		return status.Errorf(codes.PermissionDenied, err.Error())
	}

	return nil
}

func (a *apiServer) CreateGenericTask(
	ctx context.Context, req *apiv1.CreateGenericTaskRequest,
) (*apiv1.CreateGenericTaskResponse, error) {
	var projectID int
	if req.ProjectId != nil {
		projectID = int(*req.ProjectId)
	} else {
		projectID = model.DefaultProjectID
	}

	if err := a.canCreateGenericTask(ctx, projectID); err != nil {
		return nil, err
	}

	genericTaskSpec, warnings, contextDirectoryBytes, err := a.getGenericTaskLaunchParameters(
		ctx, req.ContextDirectory, req.Config, projectID,
	)
	if err != nil {
		return nil, err
	}

	if err := check.Validate(genericTaskSpec.GenericTaskConfig); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid generic task config: %s",
			err.Error(),
		)
	}

	// Persist the task.
	taskID := model.NewTaskID()
	jobID := model.NewJobID()
	startTime := time.Now()
	err = db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := db.AddJobTx(ctx, tx, &model.Job{
			JobID:   jobID,
			JobType: model.JobTypeGeneric,
			OwnerID: &genericTaskSpec.Base.Owner.ID,
		}); err != nil {
			return fmt.Errorf("persisting job %v: %w", taskID, err)
		}

		configBytes, err := yaml.YAMLToJSON([]byte(req.Config))
		if err != nil {
			return fmt.Errorf("handling experiment config %v: %w", req.Config, err)
		}

		if err := db.AddTaskTx(ctx, tx, &model.Task{
			TaskID:     taskID,
			TaskType:   model.TaskTypeGeneric,
			StartTime:  startTime,
			JobID:      &jobID,
			LogVersion: model.CurrentTaskLogVersion,
			Config:     ptrs.Ptr(string(configBytes)),
			ParentID:   (*model.TaskID)(req.ParentId),
		}); err != nil {
			return fmt.Errorf("persisting task %v: %w", taskID, err)
		}

		// TODO persist config elemnts
		if contextDirectoryBytes == nil {
			contextDirectoryBytes = []byte{}
		}
		if _, err := tx.NewInsert().Model(&model.TaskContextDirectory{
			TaskID:           taskID,
			ContextDirectory: contextDirectoryBytes,
		}).Exec(ctx); err != nil {
			return fmt.Errorf("persisting context directory files: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("persisting task information: %w", err)
	}

	logCtx := logger.Context{
		"job-id":    jobID,
		"task-id":   taskID,
		"task-type": model.TaskTypeGeneric,
	}
	priorityChange := func(priority int) error {
		return nil
	}
	if err = tasklist.GroupPriorityChangeRegistry.Add(jobID, priorityChange); err != nil {
		return nil, err
	}

	onAllocationExit := func(ae *task.AllocationExited) {
		syslog := logrus.WithField("component", "genericTask").WithFields(logCtx.Fields())
		if err := a.m.db.CompleteTask(taskID, time.Now().UTC()); err != nil {
			syslog.WithError(err).Error("marking generic task complete")
		}
		if err := tasklist.GroupPriorityChangeRegistry.Delete(jobID); err != nil {
			syslog.WithError(err).Error("deleting group priority change registry")
		}
	}

	err = task.DefaultService.StartAllocation(logCtx, sproto.AllocateRequest{
		AllocationID:      model.AllocationID(fmt.Sprintf("%s.%d", taskID, 1)),
		TaskID:            taskID,
		JobID:             jobID,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              fmt.Sprintf("Generic Task %s", taskID),

		SlotsNeeded:  *genericTaskSpec.GenericTaskConfig.Resources.Slots(),
		ResourcePool: genericTaskSpec.GenericTaskConfig.Resources.ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: genericTaskSpec.GenericTaskConfig.Resources.MustFitSingleNode(),
		},

		Restore: false,
	}, a.m.db, a.m.rm, genericTaskSpec, onAllocationExit)
	if err != nil {
		return nil, err
	}

	jobservice.DefaultService.RegisterJob(jobID, genericTaskSpec)

	return &apiv1.CreateGenericTaskResponse{
		TaskId:   string(taskID),
		Warnings: pkgCommand.LaunchWarningToProto(warnings),
	}, nil
}
