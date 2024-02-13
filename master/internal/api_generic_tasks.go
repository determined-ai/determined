package internal

import (
	"context"
	"fmt"
	"time"

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

func getConfigBytes(config []byte, forkedConfig []byte) ([]byte, error) {
	if len(config) == 0 {
		return forkedConfig, nil
	}
	if len(forkedConfig) == 0 {
		return config, nil
	}
	var master map[string]interface{}
	if err := yaml.Unmarshal(forkedConfig, &master); err != nil {
		return nil, err
	}

	var override map[string]interface{}
	if err := yaml.Unmarshal(config, &override); err != nil {
		return nil, err
	}

	for k, v := range override {
		master[k] = v
	}

	out, err := yaml.Marshal(master)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (a *apiServer) getGenericTaskLaunchParameters(
	ctx context.Context,
	contextDirectory []*utilv1.File,
	projectID int,
	configBytes []byte,
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

	genericTaskSpec.WorkspaceID = int(proj.WorkspaceId)

	// Validate the resource configuration.
	resources := model.ParseJustResources(configBytes)

	if resources.Slots < 0 {
		return nil, nil, nil, fmt.Errorf("resource slots must be >= 0")
	}
	isSingleNode := resources.IsSingleNode != nil && *resources.IsSingleNode
	poolName, launchWarnings, err := a.m.ResolveResources(resources.ResourcePool,
		resources.Slots,
		int(proj.WorkspaceId),
		isSingleNode)
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
	if err := yaml.UnmarshalStrict(configBytes, &taskConfig, yaml.DisallowUnknownFields); err != nil {
		return nil, nil, nil, fmt.Errorf("yaml unmarshaling generic task config: %w", err)
	}
	workDirInDefaults := taskConfig.WorkDir

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
	projectExperimentsQuery := db.Bun().NewSelect().
		ModelTableExpr("experiments").
		ColumnExpr(`
			COUNT(*) AS num_experiments,
			SUM(CASE WHEN state = 'ACTIVE' THEN 1 ELSE 0 END) AS num_active_experiments,
			MAX(start_time) AS last_experiment_started_at`).
		Where("project_id = ?", projectID)
	if err := db.Bun().NewSelect().
		TableExpr("pe, projects AS p").
		With("pe", projectExperimentsQuery).
		ColumnExpr(`
			p.id, p.name, p.workspace_id, p.description, p.immutable,
			p.notes, w.name AS workspace_name, p.error_message,
			(p.archived OR w.archived) AS archived,
			MAX(pe.num_experiments) AS num_experiments,
			MAX(pe.num_active_experiments) AS num_active_experiments, u.username, p.user_id`).
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

	// forkedConfig denotes the config of the task we are forking from
	var forkedConfig []byte
	var forkedContextDirectory []byte
	if req.ForkedFrom != nil {
		getTaskReq := &apiv1.GetGenericTaskConfigRequest{
			TaskId: *req.ForkedFrom,
		}
		resp, err := a.GetGenericTaskConfig(ctx, getTaskReq)
		if err != nil {
			return nil, err
		}

		forkedConfig = []byte(resp.Config)
		if err != nil {
			return nil, err
		}

		if len(req.ContextDirectory) == 0 {
			contextDirectoryResp, err := a.GetTaskContextDirectory(ctx, &apiv1.GetTaskContextDirectoryRequest{
				TaskId: *req.ForkedFrom,
			})
			if err != nil {
				return nil, err
			}
			forkedContextDirectory = []byte(contextDirectoryResp.B64Tgz)
		}
	}

	if len(forkedConfig) == 0 && len(req.Config) == 0 {
		return nil, status.Error(codes.InvalidArgument, "No config file nor forked task provided")
	}
	configBytes, err := getConfigBytes([]byte(req.Config), forkedConfig)
	if err != nil {
		return nil, err
	}
	genericTaskSpec, warnings, contextDirectoryBytes, err := a.getGenericTaskLaunchParameters(
		ctx, req.ContextDirectory, projectID, configBytes,
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
	if len(contextDirectoryBytes) == 0 {
		contextDirectoryBytes = forkedContextDirectory
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

		genericTaskSpec.RegisteredTime = startTime
		genericTaskSpec.JobID = jobID

		configBytesJSON, err := yaml.YAMLToJSON(configBytes)
		if err != nil {
			return err
		}
		if err := db.AddTaskTx(ctx, tx, &model.Task{
			TaskID:     taskID,
			TaskType:   model.TaskTypeGeneric,
			StartTime:  startTime,
			JobID:      &jobID,
			LogVersion: model.CurrentTaskLogVersion,
			ForkedFrom: req.ForkedFrom,
			Config:     ptrs.Ptr(string(configBytesJSON)),
			ParentID:   (*model.TaskID)(req.ParentId),
			State:      ptrs.Ptr(model.TaskStateActive),
			NoPause:    req.NoPause,
		}); err != nil {
			return fmt.Errorf("persisting task %v: %w", taskID, err)
		}

		// Persist context directory
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

	onAllocationExit := getGenericTaskOnAllocationExit(ctx, taskID, jobID, logCtx)

	allocationID := model.AllocationID(fmt.Sprintf("%s.%d", taskID, 1))
	isSingleNode := genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode() != nil &&
		*genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode()
	err = task.DefaultService.StartAllocation(logCtx, sproto.AllocateRequest{
		AllocationID:      allocationID,
		TaskID:            taskID,
		JobID:             jobID,
		JobSubmissionTime: startTime,
		IsUserVisible:     true,
		Name:              fmt.Sprintf("Generic Task %s", taskID),

		SlotsNeeded:  *genericTaskSpec.GenericTaskConfig.Resources.Slots(),
		ResourcePool: genericTaskSpec.GenericTaskConfig.Resources.ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: isSingleNode,
		},

		Restore: false,
	}, a.m.db, a.m.rm, genericTaskSpec, onAllocationExit)
	if err != nil {
		return nil, err
	}

	err = persistGenericTaskSpec(ctx, taskID, *genericTaskSpec, allocationID)
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
	SELECT task_id, task_state, parent_id, job_id FROM cte`
	} else {
		query = fmt.Sprintf(`
	WITH RECURSIVE cte as (
		SELECT * FROM tasks WHERE task_id='%s'
		UNION ALL
		SELECT t.* FROM tasks t INNER JOIN cte ON t.parent_id=cte.task_id
	)
	SELECT task_id, task_state, parent_id, job_id FROM cte`, taskID)
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

func (a *apiServer) KillGenericTask(
	ctx context.Context, req *apiv1.KillGenericTaskRequest,
) (*apiv1.KillGenericTaskResponse, error) {
	killTaskID := model.TaskID(req.TaskId)
	var taskModel model.Task
	err := db.Bun().NewSelect().Model(&taskModel).
		Where("task_id = ?", killTaskID).
		Where("task_type = ?", model.TaskTypeGeneric).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s (make sure task is of type GENERIC)", err)
	}
	if taskModel.TaskType != model.TaskTypeGeneric {
		return nil, fmt.Errorf("this operation is currently only supported for generic tasks")
	}
	// Validate state
	if taskModel.State == nil {
		return nil, fmt.Errorf("task state is NULL")
	}
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
			allocationID, err := getAllocationFromTaskID(ctx, childTask.TaskID)
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
	err := db.Bun().NewSelect().Model(&taskModel).
		Where("task_id = ?", req.TaskId).
		Where("task_type = ?", model.TaskTypeGeneric).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s (make sure task is of type GENERIC)", err)
	}
	// Check if the task is in a state which allows pausing.
	overrideStates := []model.TaskState{
		model.TaskStateCanceled,
		model.TaskStateCompleted,
		model.TaskStatePaused,
		model.TaskStateError,
		model.TaskStateStoppingError,
		model.TaskStateStoppingCanceled,
		model.TaskStateStoppingCompleted,
	}
	// Validate state
	if slices.Contains(overrideStates, *taskModel.State) {
		return nil, fmt.Errorf("cannot pause task %s as it is in state '%s'", req.TaskId, *taskModel.State)
	}
	// Check for flag (default to false for root task)
	if taskModel.NoPause != nil && *taskModel.NoPause {
		return nil, fmt.Errorf("cannot pause task %s with `no_pause` set to true", req.TaskId)
	}
	err = a.PropagateTaskState(ctx, model.TaskID(req.TaskId), model.TaskStateStoppingPaused, overrideStates)
	if err != nil {
		return nil, err
	}
	tasksToPause, err := a.GetTaskChildren(ctx, model.TaskID(req.TaskId), overrideStates)
	if err != nil {
		return nil, err
	}
	for _, pausingTask := range tasksToPause {
		// If task is not the root we default 'nil' no_pause as true
		if pausingTask.TaskID != model.TaskID(req.TaskId) && (pausingTask.NoPause == nil || *pausingTask.NoPause) {
			continue
		}
		allocationID, err := getAllocationFromTaskID(ctx, pausingTask.TaskID)
		if err != nil {
			return nil, err
		}
		err = task.DefaultService.Signal(model.AllocationID(allocationID),
			task.TerminateAllocation,
			"user requested pause")
		if err != nil {
			return nil, err
		}
	}
	return &apiv1.PauseGenericTaskResponse{}, nil
}

func (a *apiServer) UnpauseGenericTask(
	ctx context.Context, req *apiv1.UnpauseGenericTaskRequest,
) (*apiv1.UnpauseGenericTaskResponse, error) {
	var taskModel model.Task
	err := db.Bun().NewSelect().Model(&taskModel).
		Where("task_id = ?", req.TaskId).
		Where("task_type = ?", model.TaskTypeGeneric).Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s (make sure task is of type GENERIC)", err)
	}
	// Validate state
	if *taskModel.State != model.TaskStatePaused && *taskModel.State != model.TaskStateStoppingPaused {
		return nil, fmt.Errorf("cannot unpause task %s as it is not in paused state", req.TaskId)
	}
	// Tasks (and child tasks) that are killed, completed, or exit with an error should not be resumed
	overrideStates := []model.TaskState{
		model.TaskStateCanceled,
		model.TaskStateCompleted,
		model.TaskStateError,
		model.TaskStateStoppingError,
		model.TaskStateStoppingCanceled,
		model.TaskStateStoppingCompleted,
	}
	tasksToResume, err := a.GetTaskChildren(ctx, model.TaskID(req.TaskId), overrideStates)
	if err != nil {
		return nil, err
	}
	for _, resumingTask := range tasksToResume {
		allocationString, genericTaskSpec, err := getGenericTaskSpec(ctx, resumingTask.TaskID)
		if err != nil {
			return nil, fmt.Errorf("%s (retrieving generic task spec)", err)
		}
		if genericTaskSpec == nil {
			return nil, fmt.Errorf("could not retrieve task spec for task: %s", resumingTask.TaskID)
		}
		// check if job still in registry
		_, exists := tasklist.GroupPriorityChangeRegistry.Load(*resumingTask.JobID)
		if !exists {
			priorityChange := func(priority int) error {
				genericTaskSpec.GenericTaskConfig.Resources.SetPriority(&priority)
				return nil
			}
			if err = tasklist.GroupPriorityChangeRegistry.Add(*resumingTask.JobID, priorityChange); err != nil {
				return nil, err
			}
		}
		allocationID := model.AllocationID(allocationString)
		logCtx := logger.Context{
			"job-id":    resumingTask.JobID,
			"task-id":   resumingTask.TaskID,
			"task-type": model.TaskTypeGeneric,
		}
		onAllocationExit := getGenericTaskOnAllocationExit(ctx, resumingTask.TaskID, *resumingTask.JobID, logCtx)
		allocationSpecifier, err := allocationID.GetAllocationSpecifier()
		if err != nil {
			return nil, err
		}
		resumingAllocationID := model.AllocationID(fmt.Sprintf("%s.%d", resumingTask.TaskID, allocationSpecifier+1))
		isSingleNode := genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode() != nil &&
			*genericTaskSpec.GenericTaskConfig.Resources.IsSingleNode()
		err = task.DefaultService.StartAllocation(
			logCtx, sproto.AllocateRequest{
				AllocationID:      resumingAllocationID,
				TaskID:            resumingTask.TaskID,
				JobID:             *resumingTask.JobID,
				JobSubmissionTime: time.Now().UTC(),
				RequestTime:       time.Now().UTC(),
				IsUserVisible:     true,
				Name:              fmt.Sprintf("Generic Task %s", resumingTask.TaskID),
				SlotsNeeded:       *genericTaskSpec.GenericTaskConfig.Resources.Slots(),
				ResourcePool:      genericTaskSpec.GenericTaskConfig.Resources.ResourcePool(),
				FittingRequirements: sproto.FittingRequirements{
					SingleAgent: isSingleNode,
				},
				Preemptible: true,
				Restore:     false,
			}, a.m.db, a.m.rm, genericTaskSpec, onAllocationExit)
		if err != nil {
			return nil, err
		}
		err = persistGenericTaskSpec(ctx, resumingTask.TaskID, *genericTaskSpec, resumingAllocationID)
		if err != nil {
			return nil, err
		}
		err = setUnpauseState(ctx, resumingTask.TaskID)
		if err != nil {
			return nil, err
		}
	}
	return &apiv1.UnpauseGenericTaskResponse{}, nil
}

func setUnpauseState(ctx context.Context, taskID model.TaskID) error {
	_, err := db.Bun().NewUpdate().Table("tasks").
		Set("task_state = ?", model.TaskStateActive).
		Set("end_time = NULL").
		Where("task_id = ?", taskID).
		Exec(ctx)
	return err
}

func getAllocationFromTaskID(ctx context.Context, taskID model.TaskID,
) (string, error) {
	allocation := model.Allocation{}
	err := db.Bun().NewSelect().Model(&allocation).
		ColumnExpr("allocation_id").
		Where("task_id = ?", taskID).
		OrderExpr("start_time DESC").
		Scan(ctx)
	if err != nil {
		return "", err
	}
	return string(allocation.AllocationID), nil
}

func persistGenericTaskSpec(ctx context.Context,
	taskID model.TaskID,
	generciTaskSpec tasks.GenericTaskSpec,
	allocationID model.AllocationID,
) error {
	snapshot := &command.CommandSnapshot{
		TaskID:             taskID,
		RegisteredTime:     time.Now().UTC(),
		AllocationID:       allocationID,
		GenericCommandSpec: tasks.GenericCommandSpec{},
		GenericTaskSpec:    &generciTaskSpec,
	}

	_, err := db.Bun().NewInsert().Model(snapshot).
		On("CONFLICT (task_id) DO UPDATE").Exec(ctx)
	return err
}

func getGenericTaskSpec(ctx context.Context, taskID model.TaskID,
) (string, *tasks.GenericTaskSpec, error) {
	snapshot := command.CommandSnapshot{}

	err := db.Bun().NewSelect().Model(&snapshot).
		ColumnExpr("allocation_id, generic_task_spec").
		Where("task_id = ?", taskID).Scan(ctx)
	if err != nil {
		return "", nil, err
	}
	return string(snapshot.AllocationID), snapshot.GenericTaskSpec, nil
}
