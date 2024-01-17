package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

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

	if resources.Slots < 1 {
		resources.Slots = 1
	}

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

	if err := yaml.Unmarshal([]byte(configYAML), &taskConfig); err != nil {
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
	if err := a.m.db.QueryProto("get_project", p, projectID); errors.Is(err, db.ErrNotFound) {
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

	if req.InheritContext != nil && *req.InheritContext {
		if req.ParentId == nil {
			return nil, fmt.Errorf("could not inherit config directory since no parent task id provided")
		}
		contextDirectoryBytes, err = db.NonExperimentTasksContextDirectory(ctx, model.TaskID(*req.ParentId))
		if err != nil {
			return nil, err
		}
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
			State:      ptrs.Ptr(model.TaskStateActive),
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
		genericTaskSpec.GenericTaskConfig.Resources.SetPriority(&priority)
		return nil
	}
	if err = tasklist.GroupPriorityChangeRegistry.Add(jobID, priorityChange); err != nil {
		return nil, err
	}

	onAllocationExit := func(ae *task.AllocationExited) {
		syslog := logrus.WithField("component", "genericTask").WithFields(logCtx.Fields())
		isPaused, err := a.m.db.IsPaused(ctx, taskID)
		if err != nil {
			syslog.WithError(err).Error("on allocation exit")
		}
		if !isPaused {
			if err := a.m.db.CompleteGenericTask(taskID, time.Now().UTC()); err != nil {
				syslog.WithError(err).Error("marking generic task complete")
			}
			if err := tasklist.GroupPriorityChangeRegistry.Delete(jobID); err != nil {
				syslog.WithError(err).Error("deleting group priority change registry")
			}
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
			SingleAgent: genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode(),
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

func (a *apiServer) GetTaskChildren(
	ctx context.Context,
	taskID model.TaskID,
	overrideTasks []model.TaskState,
) ([]model.Task, error) {
	var query string
	if len(overrideTasks) > 0 {
		query = fmt.Sprintf(`
	WITH RECURSIVE cte as (
		SELECT * FROM tasks WHERE task_id='%s'
		UNION ALL
		SELECT t.* FROM tasks t INNER JOIN cte ON t.parent_id=cte.task_id
	`, taskID)
		for i, overrideTask := range overrideTasks {
			if i == 0 {
				query += fmt.Sprintf(` WHERE t.task_state !='%s'`, overrideTask)
			} else {
				query += fmt.Sprintf(` AND t.task_state !='%s'`, overrideTask)
			}
		}
		query += `)
	SELECT task_id, task_state, parent_id, job_id, config FROM cte`
	} else {
		query = fmt.Sprintf(`
	WITH RECURSIVE cte as (
		SELECT * FROM tasks WHERE task_id='%s'
		UNION ALL
		SELECT t.* FROM tasks t INNER JOIN cte ON t.parent_id=cte.task_id
	)
	SELECT task_id, task_state, parent_id, job_id, config FROM cte`, taskID)
	}

	var tasks []model.Task
	rows, err := db.Bun().QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	if rows.Err() != nil {
		return nil, err
	}
	err = db.Bun().ScanRows(ctx, rows, &tasks)
	if err != nil {
		return nil, err
	}
	return tasks, nil
}

func (a *apiServer) PropagateTaskState(
	ctx context.Context,
	taskID model.TaskID,
	state model.TaskState,
	overrideStates []model.TaskState,
) error {
	var query string
	if len(overrideStates) > 0 {
		query = fmt.Sprintf(`
	WITH RECURSIVE cte as (
		SELECT * FROM tasks WHERE task_id='%s'
		UNION ALL
		SELECT t.* FROM tasks t INNER JOIN cte ON t.parent_id=cte.task_id
	)
	UPDATE tasks SET task_state='%s' FROM cte WHERE cte.task_id=tasks.task_id`, taskID, state)
		for _, overrideState := range overrideStates {
			query += fmt.Sprintf(` AND cte.task_state!='%s'`, overrideState)
		}
		query += ";"
	} else {
		query = fmt.Sprintf(`
	WITH RECURSIVE cte as (
		SELECT * FROM tasks WHERE task_id='%s'
		UNION ALL
		SELECT t.* FROM tasks t INNER JOIN cte ON t.parent_id=cte.task_id
	)
	UPDATE tasks SET task_state='%s' FROM cte WHERE cte.task_id=tasks.task_id;`, taskID, state)
	}
	_, err := db.Bun().NewRaw(query).Exec(ctx)
	return err
}

func (a *apiServer) FindRoot(ctx context.Context, taskID model.TaskID) (model.TaskID, error) {
	out := struct {
		Root model.TaskID
	}{}
	query := fmt.Sprintf(`
	WITH RECURSIVE my_tree as (
		SELECT task_id, parent_id, task_id as root FROM tasks WHERE parent_id IS NULL
		UNION ALL
		SELECT t.task_id, t.parent_id, m.root FROM tasks t JOIN my_tree m on m.task_id=t.parent_id
	)
	SELECT root FROM my_tree WHERE task_id='%s'`, taskID)
	err := db.Bun().NewRaw(query).Scan(ctx, &out)
	return out.Root, err
}

func (a *apiServer) SetTaskState(ctx context.Context, taskID model.TaskID, state model.TaskState) error {
	_, err := db.Bun().NewUpdate().Table("tasks").
		Set("task_state = ?", state).
		Where("task_id = ?", taskID).
		Exec(ctx)
	return err
}

func (a *apiServer) SetPausedState(ctx context.Context, taskID model.TaskID) error {
	_, err := db.Bun().NewUpdate().Table("tasks").
		Set("task_state = ?", model.TaskStatePaused).
		Set("end_time = ?", time.Now().UTC()).
		Where("task_id = ?", taskID).
		Exec(ctx)
	return err
}

func (a *apiServer) SetResumedState(ctx context.Context, taskID model.TaskID) error {
	_, err := db.Bun().NewUpdate().Table("tasks").
		Set("task_state = ?", model.TaskStateActive).
		Set("end_time = NULL").
		Where("task_id = ?", taskID).
		Exec(ctx)
	return err
}

func (a *apiServer) KillGenericTask(
	ctx context.Context, req *apiv1.KillGenericTaskRequest,
) (*apiv1.KillGenericTaskResponse, error) {
	killTaskID := model.TaskID(req.TaskId)
	var taskModel model.Task
	err := db.Bun().NewSelect().Model(&taskModel).Where("task_id = ?", killTaskID).Scan(ctx)
	if err != nil {
		return nil, err
	}
	// Validate state
	overrideStates := []model.TaskState{model.TaskStateCanceled, model.TaskStateCompleted}
	if slices.Contains(overrideStates, *taskModel.State) {
		return nil, fmt.Errorf("cannot cancel task %s as it is in state '%s'", req.TaskId, *taskModel.State)
	}
	if req.KillFromRoot {
		rootID, err := a.FindRoot(ctx, model.TaskID(req.TaskId))
		if err != nil {
			return nil, err
		}
		killTaskID = rootID
	}
	err = a.PropagateTaskState(ctx, killTaskID, model.TaskStateStoppingCanceled, overrideStates)
	if err != nil {
		return nil, err
	}
	tasksToDelete, err := a.GetTaskChildren(ctx, killTaskID, overrideStates)
	if err != nil {
		return nil, err
	}
	for _, childTask := range tasksToDelete {
		if childTask.State == nil || *childTask.State != model.TaskStateCanceled {
			allocationID, err := a.GetAllocationFromTaskID(ctx, childTask.TaskID)
			if err != nil {
				return nil, err
			}
			err = task.DefaultService.Signal(model.AllocationID(allocationID), task.KillAllocation, "user requested task kill")
			if err != nil {
				return nil, err
			}
		}
	}
	return &apiv1.KillGenericTaskResponse{}, nil
}

func (a *apiServer) PauseGenericTask(
	ctx context.Context, req *apiv1.PauseGenericTaskRequest,
) (*apiv1.PauseGenericTaskResponse, error) {
	var taskModel model.Task
	err := db.Bun().NewSelect().Model(&taskModel).Where("task_id = ?", req.TaskId).Scan(ctx)
	if err != nil {
		return nil, err
	}
	// Check if the task is in a state which allows pausing.
	overrideStates := []model.TaskState{model.TaskStateCanceled, model.TaskStateCompleted, model.TaskStatePaused}
	// Validate state
	if slices.Contains(overrideStates, *taskModel.State) {
		return nil, fmt.Errorf("cannot pause task %s as it is in state '%s'", req.TaskId, *taskModel.State)
	}
	// Check for flag (default to false for root task)
	if taskModel.NoPause != nil && *taskModel.NoPause {
		return nil, fmt.Errorf("cannot pause task %s as it is flagged as not pausable", req.TaskId)
	}
	err = a.PropagateTaskState(ctx, model.TaskID(req.TaskId), model.TaskStateStoppingPaused, overrideStates)
	if err != nil {
		return nil, err
	}
	tasksToPause, err := a.GetTaskChildren(ctx, model.TaskID(req.TaskId), overrideStates)
	if err != nil {
		return nil, err
	}
	for _, childTask := range tasksToPause {
		if childTask.NoPause == nil || !*childTask.NoPause {
			allocationID, err := a.GetAllocationFromTaskID(ctx, childTask.TaskID)
			if err != nil {
				return nil, err
			}
			err = task.DefaultService.Signal(model.AllocationID(allocationID),
				task.TerminateAllocation,
				"user requested task kill")
			if err != nil {
				return nil, err
			}
			err = a.SetPausedState(ctx, childTask.TaskID)
			if err != nil {
				return nil, err
			}
		}
	}
	return &apiv1.PauseGenericTaskResponse{}, nil
}

func (a *apiServer) ResumeGenericTask(
	ctx context.Context, req *apiv1.ResumeGenericTaskRequest,
) (*apiv1.ResumeGenericTaskResponse, error) {
	var taskModel model.Task
	err := db.Bun().NewSelect().Model(&taskModel).Where("task_id = ?", req.TaskId).Scan(ctx)
	if err != nil {
		return nil, err
	}
	// Validate state
	if *taskModel.State != model.TaskStatePaused {
		return nil, fmt.Errorf("cannot unpause task %s as it is not in paused state", req.TaskId)
	}
	var projectID int
	if req.ProjectId != nil {
		projectID = int(*req.ProjectId)
	} else {
		projectID = model.DefaultProjectID
	}
	// Tasks (and child tasks) that are killed, completed should not be paused
	overrideStates := []model.TaskState{model.TaskStateCanceled, model.TaskStateCompleted}
	if err != nil {
		return nil, err
	}
	tasksToResume, err := a.GetTaskChildren(ctx, model.TaskID(req.TaskId), overrideStates)
	if err != nil {
		return nil, err
	}
	for _, childTask := range tasksToResume {
		genericTaskSpec, _, _, err := a.getGenericTaskLaunchParameters(
			ctx, nil, *childTask.Config, projectID,
		)
		if err != nil {
			return nil, err
		}
		// check if job still in registry
		_, exists := tasklist.GroupPriorityChangeRegistry.Load(*childTask.JobID)
		if !exists {
			priorityChange := func(priority int) error {
				genericTaskSpec.GenericTaskConfig.Resources.SetPriority(&priority)
				return nil
			}
			if err = tasklist.GroupPriorityChangeRegistry.Add(*childTask.JobID, priorityChange); err != nil {
				return nil, err
			}
		}
		allocationString, err := a.GetAllocationFromTaskID(ctx, childTask.TaskID)
		if err != nil {
			return nil, err
		}
		allocationID := model.AllocationID(allocationString)
		logCtx := logger.Context{
			"job-id":    childTask.JobID,
			"task-id":   childTask.TaskID,
			"task-type": model.TaskTypeGeneric,
		}
		onAllocationExit := func(ae *task.AllocationExited) {
			syslog := logrus.WithField("component", "genericTask").WithFields(logCtx.Fields())
			isPaused, err := a.m.db.IsPaused(ctx, childTask.TaskID)
			if err != nil {
				syslog.WithError(err).Error("on allocation exit")
			}
			if !isPaused {
				if err := a.m.db.CompleteGenericTask(childTask.TaskID, time.Now().UTC()); err != nil {
					syslog.WithError(err).Error("marking generic task complete")
				}
				if err := tasklist.GroupPriorityChangeRegistry.Delete(*childTask.JobID); err != nil {
					syslog.WithError(err).Error("deleting group priority change registry")
				}
			}
		}
		allocationSpecifier, err := allocationID.GetAllocationSpecifier()
		if err != nil {
			return nil, err
		}
		err = task.DefaultService.StartAllocation(
			logCtx, sproto.AllocateRequest{
				AllocationID:      model.AllocationID(fmt.Sprintf("%s.%d", childTask.TaskID, allocationSpecifier+1)),
				TaskID:            childTask.TaskID,
				JobID:             *childTask.JobID,
				JobSubmissionTime: time.Now().UTC(),
				RequestTime:       time.Now().UTC(),
				IsUserVisible:     true,
				Name:              fmt.Sprintf("Generic Task %s", childTask.TaskID),
				SlotsNeeded:       *genericTaskSpec.GenericTaskConfig.Resources.Slots(),
				ResourcePool:      genericTaskSpec.GenericTaskConfig.Resources.ResourcePool(),
				FittingRequirements: sproto.FittingRequirements{
					SingleAgent: genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode(),
				},
				Preemptible: true,
				Restore:     false,
			}, a.m.db, a.m.rm, genericTaskSpec, onAllocationExit)
		if err != nil {
			return nil, err
		}
		err = a.SetResumedState(ctx, childTask.TaskID)
		if err != nil {
			return nil, err
		}
	}
	return &apiv1.ResumeGenericTaskResponse{}, nil
}

func (a *apiServer) GetAllocationFromTaskID(ctx context.Context, taskID model.TaskID,
) (string, error) {
	allocation := model.Allocation{}
	err := db.Bun().NewSelect().Model(&allocation).
		ColumnExpr("allocation_id").
		Where("task_id = ?", taskID).
		OrderExpr("start_time DESC").Scan(ctx)
	if err != nil {
		return "", err
	}
	return string(allocation.AllocationID), nil
}
