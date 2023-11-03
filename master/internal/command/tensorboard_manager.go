// Package command provides utilities for commands.
package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// LaunchTensorboard launches a *tensorboardv1.Tensorboard.
func (cs *CommandService) LaunchTensorboard(req *CreateGeneric) (*tensorboardv1.Tensorboard, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cmd, err := cs.createGenericCommand(model.TaskTypeTensorboard, model.JobTypeTensorboard, req)
	if err != nil {
		return nil, err
	}

	return cmd.toTensorboard(), nil
}

// GetTensorboards returns all tbs in the command service registry matching the workspace ID.
func (cs *CommandService) GetTensorboards(req *apiv1.GetTensorboardsRequest) (*apiv1.GetTensorboardsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetTensorboardsResponse{}
	cmds, users, userIds := cs.listByType(req.Users, req.UserIds, model.TaskTypeTensorboard)
	for _, c := range cmds {
		t := c.toTensorboard()
		// skip if it doesn't match the requested workspaceID if any.
		if req.WorkspaceId != 0 && req.WorkspaceId != t.WorkspaceId {
			continue
		}
		if (len(users) == 0 && len(userIds) == 0) || users[t.Username] || userIds[t.UserId] {
			resp.Tensorboards = append(resp.Tensorboards, t)
		}
	}
	return resp, nil
}

// GetTensorboard looks up a tensorboard by ID returns a summary of the its state and configuration.
func (cs *CommandService) GetTensorboard(req *apiv1.GetTensorboardRequest) (*apiv1.GetTensorboardResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.TensorboardId), model.TaskTypeTensorboard)
	if err != nil {
		return nil, api.NotFoundErrs("tensorboard", req.TensorboardId, true)
	}

	return &apiv1.GetTensorboardResponse{
		Tensorboard: c.toTensorboard(),
		Config:      protoutils.ToStruct(c.Config),
	}, nil
}

// KillTensorboard sends a kill signal to the command's allocation.
func (cs *CommandService) KillTensorboard(req *apiv1.KillTensorboardRequest) (*apiv1.KillTensorboardResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.TensorboardId), model.TaskTypeTensorboard)
	if err != nil {
		return nil, err
	}

	err = task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
	if err != nil {
		return nil, fmt.Errorf("failed to kill allocation: %w", err)
	}
	return &apiv1.KillTensorboardResponse{Tensorboard: c.toTensorboard()}, nil
}

// SetTensorboardPriority sets the tb's resource manager group priority.
func (cs *CommandService) SetTensorboardPriority(
	req *apiv1.SetTensorboardPriorityRequest,
) (*apiv1.SetTensorboardPriorityResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.TensorboardId), model.TaskTypeTensorboard)
	if err != nil {
		return nil, err
	}

	err = c.setNTSCPriority(int(req.Priority), true)
	if err != nil {
		return nil, err
	}
	return &apiv1.SetTensorboardPriorityResponse{Tensorboard: c.toTensorboard()}, nil
}

// toTensorboard() takes a *command from the command service registry & returns a *tensorboardv1.Tensorboard.
func (c *command) toTensorboard() *tensorboardv1.Tensorboard {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	state := enrichState(allo.State)
	return &tensorboardv1.Tensorboard{
		Id:             c.stringID(),
		State:          state,
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		ExperimentIds:  c.Metadata.ExperimentIDs,
		TrialIds:       c.Metadata.TrialIDs,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}
