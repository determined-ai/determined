package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/archive"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8sV1 "k8s.io/api/core/v1"
)

func ResolveResources(a *apiServer, resourcePool string, slots int, workspaceID int) (string, []pkgCommand.LaunchWarning, error) {

	poolName, err := a.m.rm.ResolveResourcePool(
		resourcePool, workspaceID, slots)
	if err != nil {
		return "", nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err = a.m.rm.ValidateResources(poolName, slots, true); err != nil {
		return "nil", nil, fmt.Errorf("validating resources: %v", err)
	}

	launchWarnings, err := a.m.rm.ValidateResourcePoolAvailability(
		&sproto.ValidateResourcePoolAvailabilityRequest{
			Name:  poolName,
			Slots: slots,
		},
	)
	if err != nil {
		return "", launchWarnings, fmt.Errorf("checking resource availability: %v", err.Error())
	}
	if a.m.config.ResourceManager.AgentRM != nil &&
		a.m.config.LaunchError &&
		len(launchWarnings) > 0 {
		return "", nil, errors.New("slots requested exceeds cluster capacity")
	}

	return poolName, launchWarnings, nil
}

func fillTaskSpec(a *apiServer, poolName string, agentUserGroup *model.AgentUserGroup, userModel *model.User) (tasks.TaskSpec, error) {
	taskContainerDefaults, err := a.m.rm.TaskContainerDefaults(
		poolName,
		a.m.config.TaskContainerDefaults,
	)
	if err != nil {
		return tasks.TaskSpec{}, fmt.Errorf("getting TaskContainerDefaults: %v", err)
	}
	taskSpec := *a.m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = userModel
	return taskSpec, nil
}

func fillTaskConfig(resourcePoolDest **string, poolName string, resourceSlotsDest **int, slots int, taskSpec tasks.TaskSpec, environment *model.Environment) {
	*resourcePoolDest = &poolName
	*resourceSlotsDest = &slots

	taskContainerPodSpec := taskSpec.TaskContainerDefaults.GPUPodSpec
	if slots == 0 {
		taskContainerPodSpec = taskSpec.TaskContainerDefaults.CPUPodSpec
	}
	environment.PodSpec = (*k8sV1.Pod)(schemas.Merge(
		(*expconf.PodSpec)(environment.PodSpec),
		(*expconf.PodSpec)(taskContainerPodSpec),
	))
}

func fillContextDir(configWorkDirDest **string, defaultWorkDir *string, contextDirectory []*utilv1.File) ([]byte, error) {
	var contextDirectoryBytes []byte
	if len(contextDirectory) > 0 {
		userFiles := filesToArchive(contextDirectory)

		workdirSetInReq := *configWorkDirDest != nil &&
			(defaultWorkDir == nil || *defaultWorkDir != **configWorkDirDest)
		if workdirSetInReq {
			return nil, status.Errorf(codes.InvalidArgument,
				"cannot set work_dir and context directory at the same time")
		}
		*configWorkDirDest = nil

		var err error
		contextDirectoryBytes, err = archive.ToTarGz(userFiles)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				fmt.Errorf("compressing files context files: %w", err).Error())
		}
	}
	return contextDirectoryBytes, nil
}

func getTaskSessionToken(ctx context.Context, userModel *model.User) (string, error) {
	extConfig := config.GetMasterConfig().InternalConfig.ExternalSessions
	var token string
	var err error
	if extConfig.Enabled() {
		token, err = grpcutil.GetUserExternalToken(ctx)
		if err != nil {
			return "", status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to get external user token").Error())
		}
	} else {
		token, err = user.StartSession(ctx, userModel)
		if err != nil {
			return "", status.Errorf(codes.Internal,
				errors.Wrapf(err,
					"unable to create user session inside task").Error())
		}
	}
	return token, nil
}
