package generictask

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type APIServer struct{}

func (a *APIServer) getGenericTaskLaunchParameters(
	ctx context.Context,
	config string,
	contextDirectory []*utilv1.File,
	configYAML string,
	projectID *int,
) (
	*command.CreateGeneric, []pkgCommand.LaunchWarning, error,
) {
	var err error
	taskSpec := tasks.GenericTaskSpec{}

	taskSpec.ProjectID = model.DefaultProjectID
	if projectID != nil {
		taskSpec.ProjectID = *projectID
	}

	// Validate the userModel and get the agent userModel group.
	userModel, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil,
			nil,
			status.Errorf(codes.Unauthenticated, "failed to get the user: %s", err)
	}

	workspaceID := 1 // TODO convert projectID to workspaceID here
	agentUserGroup, err := user.GetAgentUserGroup(ctx, userModel.ID, workspaceID)
	if err != nil {
		return nil, nil, err
	}

	// Validate the resource configuration.
	resources := model.ParseJustResources([]byte(config))

	poolName, err := a.m.rm.ResolveResourcePool(
		resources.ResourcePool, workspaceID, resources.SlotsPerTask)
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	launchWarnings, err := a.m.rm.ValidateResourcePoolAvailability(
		&sproto.ValidateResourcePoolAvailabilityRequest{
			Name:  poolName,
			Slots: resources.SlotsPerTask,
		},
	)
	if err != nil {
		return nil, launchWarnings, fmt.Errorf("checking resource availability: %v", err.Error())
	}
	if a.m.config.ResourceManager.AgentRM != nil &&
		a.m.config.LaunchError &&
		len(launchWarnings) > 0 {
		return nil, nil, fmt.Errorf("slots requested exceeds cluster capacity")
	}

	// Get the base TaskSpec.
	taskContainerDefaults, err := a.m.rm.TaskContainerDefaults(
		poolName,
		a.m.config.TaskContainerDefaults,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("getting TaskContainerDefaults: %v", err)
	}
	taskSpec := *a.m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = userModel

	// Get the full configuration.
	config := model.DefaultConfig(&taskSpec.TaskContainerDefaults)

	workDirInDefaults := config.WorkDir
	if len(configBytes) != 0 {
		dec := json.NewDecoder(bytes.NewBuffer(configBytes))
		dec.DisallowUnknownFields()

		if err := dec.Decode(&config); err != nil {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				errors.Wrapf(err,
					"unable to decode the merged config: %s", string(configBytes)).Error())
		}
	}
	// Copy discovered (default) resource pool name and slot count.
	config.Resources.ResourcePool = poolName
	config.Resources.SlotsPerTrial = resources.SlotsPerTask

	taskContainerPodSpec := taskSpec.TaskContainerDefaults.GPUPodSpec
	if config.Resources.SlotsPerTask == 0 {
		taskContainerPodSpec = taskSpec.TaskContainerDefaults.CPUPodSpec
	}
	config.Environment.PodSpec = (*k8sV1.Pod)(schemas.Merge(
		(*expconf.PodSpec)(config.Environment.PodSpec),
		(*expconf.PodSpec)(taskContainerPodSpec),
	))

	var contextDirectory []byte
	if len(req.Files) > 0 {
		userFiles := filesToArchive(contextDirectory)

		workdirSetInReq := config.WorkDir != nil &&
			(workDirInDefaults == nil || *workDirInDefaults != *config.WorkDir)
		if workdirSetInReq {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				"cannot set work_dir and context directory at the same time")
		}
		config.WorkDir = nil

		contextDirectory, err = archive.ToTarGz(userFiles)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.InvalidArgument,
				fmt.Errorf("compressing files context files: %w", err).Error())
		}
	}

	extConfig := mconfig.GetMasterConfig().InternalConfig.ExternalSessions
	var token string
	if extConfig.Enabled() {
		token, err = grpcutil.GetUserExternalToken(ctx)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to get external user token").Error())
		}
		err = nil
	} else {
		token, err = user.StartSession(ctx, userModel)
		if err != nil {
			return nil, launchWarnings, status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to create user session inside task").Error())
		}
	}
	taskSpec.UserSessionToken = token

	cmdSpec.Base = taskSpec
	cmdSpec.Config = config

	return &command.CreateGeneric{
		Spec:             &cmdSpec,
		ContextDirectory: contextDirectory,
	}, launchWarnings, nil
}

func (a *APIServer) CreateGenericTask(
	ctx context.Context, req *apiv1.CreateGenericTaskRequest,
) (*apiv1.CreateGenericTaskResponse, error) {
	user, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	// Parse launch commnads.
	getGenericTaskLaunchParameters(ctx)
	// TODO rbac check.

	// Maybe fill in description?

	// TODO do we need to wrap entrypoint with a custom wrapper?
	// If we do it feels like a weird place to do it

	if err := check.Validate(genericTaskConfig); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"invalid generic task config: %s",
			err.Error(),
		)
	}

	launchReq.Spec.Base.ExtraEnvVars = map[string]string{
		"DET_TASK_TYPE": string(model.TaskTypeGeneric),
	}

	// Persist the task.

	// TODO actually create the task

	return &apiv1.CreateGenericTaskResponse{}, nil
}
