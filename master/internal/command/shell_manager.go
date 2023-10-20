package command

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

func (cs *commandService) LaunchShell(ctx context.Context, req *CreateGeneric) (*shellv1.Shell, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cmd, err := cs.createGenericCommand(ctx, model.TaskTypeShell, model.JobTypeShell, req)
	if err != nil {
		return nil, err
	}

	return cmd.toShell(), nil
}

func (cs *commandService) GetShells(req *apiv1.GetShellsRequest) (*apiv1.GetShellsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetShellsResponse{}
	cmds, users, userIds := cs.listByType(req.Users, req.UserIds, model.TaskTypeShell)
	for _, c := range cmds {
		s := c.toShell()
		// skip if it doesn't match the requested workspaceID if any.
		if req.WorkspaceId != 0 && req.WorkspaceId != s.WorkspaceId {
			continue
		}
		if (len(users) == 0 && len(userIds) == 0) || users[s.Username] || userIds[s.UserId] {
			resp.Shells = append(resp.Shells, s)
		}
	}
	return resp, nil
}

func (cs *commandService) GetShell(req *apiv1.GetShellRequest) (*apiv1.GetShellResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.ShellId), model.TaskTypeShell)
	if err != nil {
		return nil, api.NotFoundErrs("shell", req.ShellId, true)
	}

	return &apiv1.GetShellResponse{
		Shell:  c.toShell(),
		Config: protoutils.ToStruct(c.Config),
	}, nil
}

func (cs *commandService) KillShell(req *apiv1.KillShellRequest) (*apiv1.KillShellResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.ShellId), model.TaskTypeShell)
	if err != nil {
		return nil, err
	}

	err = task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
	if err != nil {
		return nil, fmt.Errorf("failed to kill allocation: %w", err)
	}
	return &apiv1.KillShellResponse{Shell: c.toShell()}, nil
}

func (cs *commandService) SetShellPriority(
	req *apiv1.SetShellPriorityRequest,
) (*apiv1.SetShellPriorityResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.ShellId), model.TaskTypeShell)
	if err != nil {
		return nil, err
	}

	err = c.setNTSCPriority(int(req.Priority), true)
	if err != nil {
		return nil, err
	}
	return &apiv1.SetShellPriorityResponse{Shell: c.toShell()}, nil
}

func (c *command) toShell() *shellv1.Shell {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &shellv1.Shell{
		Id:             c.stringID(),
		State:          enrichState(allo.State),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		PrivateKey:     *c.Metadata.PrivateKey,
		PublicKey:      *c.Metadata.PublicKey,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		Addresses:      toProto(allo.SingleContainerAddresses()),
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}
