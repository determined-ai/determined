package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/archive"
	pkgCommand "github.com/determined-ai/determined/master/pkg/command"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

// ResolveResources - Validate ResoucePool and check for availability.
func (m *Master) ResolveResources(
	resourcePool string,
	slots int,
	workspaceID int,
	isSingleNode bool,
) (string, []pkgCommand.LaunchWarning, error) {
	poolName, err := m.rm.ResolveResourcePool(
		resourcePool, workspaceID, slots)
	if err != nil {
		return "", nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	if err = m.rm.ValidateResources(poolName, slots, isSingleNode); err != nil {
		return "", nil, fmt.Errorf("validating resources: %v", err)
	}

	launchWarnings, err := m.rm.ValidateResourcePoolAvailability(
		&sproto.ValidateResourcePoolAvailabilityRequest{
			Name:  poolName,
			Slots: slots,
		},
	)
	if err != nil {
		return "", launchWarnings, fmt.Errorf("checking resource availability: %v", err.Error())
	}
	if m.config.ResourceManager.AgentRM != nil &&
		m.config.LaunchError &&
		len(launchWarnings) > 0 {
		return "", nil, errors.New("slots requested exceeds cluster capacity")
	}

	return poolName, launchWarnings, nil
}

// Fill and return TaskSpec.
func (m *Master) fillTaskSpec(
	poolName string,
	agentUserGroup *model.AgentUserGroup,
	userModel *model.User,
) (tasks.TaskSpec, error) {
	taskContainerDefaults, err := m.rm.TaskContainerDefaults(
		poolName,
		m.config.TaskContainerDefaults,
	)
	if err != nil {
		return tasks.TaskSpec{}, fmt.Errorf("getting TaskContainerDefaults: %v", err)
	}
	taskSpec := *m.taskSpec
	taskSpec.TaskContainerDefaults = taskContainerDefaults
	taskSpec.AgentUserGroup = agentUserGroup
	taskSpec.Owner = userModel
	return taskSpec, nil
}

func fillTaskConfig(slots int, taskSpec tasks.TaskSpec, environment *model.Environment) {
	taskContainerPodSpec := taskSpec.TaskContainerDefaults.GPUPodSpec
	if slots == 0 {
		taskContainerPodSpec = taskSpec.TaskContainerDefaults.CPUPodSpec
	}
	environment.PodSpec = (*k8sV1.Pod)(schemas.Merge(
		(*expconf.PodSpec)(environment.PodSpec),
		(*expconf.PodSpec)(taskContainerPodSpec),
	))
}

func fillContextDir(
	configWorkDir *string,
	defaultWorkDir *string,
	contextDirectory []*utilv1.File,
) (*string, []byte, error) {
	var contextDirectoryBytes []byte
	if len(contextDirectory) > 0 {
		userFiles := filesToArchive(contextDirectory)

		workdirSetInReq := configWorkDir != nil &&
			(defaultWorkDir == nil || *defaultWorkDir != *configWorkDir)
		if workdirSetInReq {
			return nil, nil, status.Errorf(codes.InvalidArgument,
				"cannot set work_dir and context directory at the same time")
		}

		var err error
		contextDirectoryBytes, err = archive.ToTarGz(userFiles)
		if err != nil {
			return nil, nil, status.Errorf(codes.InvalidArgument,
				fmt.Errorf("compressing files context files: %w", err).Error())
		}
		return nil, contextDirectoryBytes, nil
	}
	return configWorkDir, contextDirectoryBytes, nil
}

func getTaskSessionToken(ctx context.Context, userModel *model.User) (string, error) {
	var token string
	var err error
	if config.GetMasterConfig().InternalConfig.ExternalSessions.Enabled() {
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

func getGenericTaskOnAllocationExit(
	ctx context.Context,
	taskID model.TaskID,
	jobID model.JobID,
	logCtx logger.Context,
) func(ae *task.AllocationExited) {
	return func(ae *task.AllocationExited) {
		syslog := logrus.WithField("component", "genericTask").WithFields(logCtx.Fields())
		if ae.Err != nil {
			err := db.SetErrorState(taskID, time.Now().UTC())
			if err != nil {
				syslog.WithError(err).Error("setting task to error state")
			}
			if err := tasklist.GroupPriorityChangeRegistry.Delete(jobID); err != nil {
				syslog.WithError(err).Error("deleting group priority change registry")
			}
			return
		}
		isPaused, err := db.IsPaused(ctx, taskID)
		if err != nil {
			syslog.WithError(err).Error("checking if a task is paused")
		}
		if isPaused {
			err = db.SetPausedState(taskID, time.Now().UTC())
			if err != nil {
				syslog.WithError(err).Error("setting task to paused state")
			}
			return
		}
		if err := db.CompleteGenericTask(taskID, time.Now().UTC()); err != nil {
			syslog.WithError(err).Error("marking generic task complete")
		}
		if err := tasklist.GroupPriorityChangeRegistry.Delete(jobID); err != nil {
			syslog.WithError(err).Error("deleting group priority change registry")
		}
	}
}
