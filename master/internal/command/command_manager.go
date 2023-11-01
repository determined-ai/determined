package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
)

// LaunchCommand launches *commandv1.Command.
func (cs *commandService) LaunchCommand(req *CreateGeneric) (*commandv1.Command, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cmd, err := cs.createGenericCommand(model.TaskTypeCommand, model.JobTypeCommand, req)
	if err != nil {
		return nil, err
	}

	return cmd.toCommand(), nil
}

// GetCommands returns all commands in the command service registry matching the workspace ID.
func (cs *commandService) GetCommands(req *apiv1.GetCommandsRequest) (*apiv1.GetCommandsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetCommandsResponse{}
	cmds, users, userIDs := cs.listByType(req.Users, req.UserIds, model.TaskTypeCommand)
	for _, c := range cmds {
		cmd := c.toCommand()
		// skip if it doesn't match the requested workspaceID if any.
		if req.WorkspaceId != 0 && req.WorkspaceId != cmd.WorkspaceId {
			continue
		}
		if (len(users) == 0 && len(userIDs) == 0) || users[cmd.Username] || userIDs[cmd.UserId] {
			resp.Commands = append(resp.Commands, cmd)
		}
	}
	return resp, nil
}

// GetCommand returns the command matching the commandID.
func (cs *commandService) GetCommand(req *apiv1.GetCommandRequest) (*apiv1.GetCommandResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.CommandId), model.TaskTypeCommand)
	if err != nil {
		return nil, api.NotFoundErrs("command", req.CommandId, true)
	}

	return &apiv1.GetCommandResponse{
		Command: c.toCommand(),
		Config:  protoutils.ToStruct(c.Config),
	}, nil
}

// KillCommand marks the command's allocation as killed.
func (cs *commandService) KillCommand(req *apiv1.KillCommandRequest) (*apiv1.KillCommandResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.CommandId), model.TaskTypeCommand)
	if err != nil {
		return nil, err
	}

	err = task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
	if err != nil {
		return nil, fmt.Errorf("failed to kill allocation: %w", err)
	}
	return &apiv1.KillCommandResponse{Command: c.toCommand()}, nil
}

// SetCommandPriority sets the command's resource manager group priority.
func (cs *commandService) SetCommandPriority(
	req *apiv1.SetCommandPriorityRequest,
) (*apiv1.SetCommandPriorityResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.CommandId), model.TaskTypeCommand)
	if err != nil {
		return nil, err
	}

	err = c.setNTSCPriority(int(req.Priority), true)
	if err != nil {
		return nil, err
	}
	return &apiv1.SetCommandPriorityResponse{Command: c.toCommand()}, nil
}

// toCommand() takes a *command from the command service registry & returns a *commandv1.Command.
func (c *command) toCommand() *commandv1.Command {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &commandv1.Command{
		Id:           c.stringID(),
		State:        enrichState(allo.State),
		Description:  c.Config.Description,
		Container:    allo.SingleContainer().ToProto(),
		StartTime:    protoutils.ToTimestamp(c.registeredTime),
		Username:     c.Base.Owner.Username,
		UserId:       int32(c.Base.Owner.ID),
		DisplayName:  c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   c.exitStatus.String(),
		JobId:        c.jobID.String(),
		WorkspaceId:  int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}
