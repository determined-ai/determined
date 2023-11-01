package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

// LaunchNotebook launches *notebookv1.Notebook.
func (cs *commandService) LaunchNotebook(req *CreateGeneric) (*notebookv1.Notebook, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cmd, err := cs.createGenericCommand(model.TaskTypeNotebook, model.JobTypeNotebook, req)
	if err != nil {
		return nil, err
	}

	return cmd.toNotebook(), nil
}

// GetNotebooks returns all notebooks in the command service registry matching the workspace ID.
func (cs *commandService) GetNotebooks(req *apiv1.GetNotebooksRequest) (*apiv1.GetNotebooksResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetNotebooksResponse{}
	cmds, users, userIds := cs.listByType(req.Users, req.UserIds, model.TaskTypeNotebook)
	for _, c := range cmds {
		n := c.toNotebook()
		// skip if it doesn't match the requested workspaceID if any.
		if req.WorkspaceId != 0 && req.WorkspaceId != n.WorkspaceId {
			continue
		}
		if (len(users) == 0 && len(userIds) == 0) || users[n.Username] || userIds[n.UserId] {
			resp.Notebooks = append(resp.Notebooks, n)
		}
	}
	return resp, nil
}

// GetNotebook returns the notebook matching the notebookID.
func (cs *commandService) GetNotebook(req *apiv1.GetNotebookRequest) (*apiv1.GetNotebookResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.NotebookId), model.TaskTypeNotebook)
	if err != nil {
		return nil, api.NotFoundErrs("notebook", req.NotebookId, true)
	}

	return &apiv1.GetNotebookResponse{
		Notebook: c.toNotebook(),
		Config:   protoutils.ToStruct(c.Config),
	}, nil
}

// KillNotebook marks the notebook's allocation as killed..
func (cs *commandService) KillNotebook(req *apiv1.KillNotebookRequest) (*apiv1.KillNotebookResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.NotebookId), model.TaskTypeNotebook)
	if err != nil {
		return nil, err
	}

	err = task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
	if err != nil {
		return nil, fmt.Errorf("failed to kill allocation: %w", err)
	}
	return &apiv1.KillNotebookResponse{Notebook: c.toNotebook()}, nil
}

// SetNotebookPriority sets the notebook's resource manager group priority.
func (cs *commandService) SetNotebookPriority(
	req *apiv1.SetNotebookPriorityRequest,
) (*apiv1.SetNotebookPriorityResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.NotebookId), model.TaskTypeNotebook)
	if err != nil {
		return nil, err
	}

	err = c.setNTSCPriority(int(req.Priority), true)
	if err != nil {
		return nil, err
	}
	return &apiv1.SetNotebookPriorityResponse{Notebook: c.toNotebook()}, nil
}

// toNotebook() takes a *command from the command service registry & returns a *notebookv1.Notebook.
func (c *command) toNotebook() *notebookv1.Notebook {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &notebookv1.Notebook{
		Id:             c.stringID(),
		State:          enrichState(allo.State),
		Description:    c.Config.Description,
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}
