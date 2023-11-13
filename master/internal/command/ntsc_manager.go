// Package command provides utilities for commands.
package command

import (
	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// GetCommands returns all commands in the command service registry matching the workspace ID.
func (cs *CommandService) GetCommands(req *apiv1.GetCommandsRequest) (*apiv1.GetCommandsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetCommandsResponse{}
	cmds := cs.listByType(req.Users, req.UserIds, model.TaskTypeCommand, req.WorkspaceId)
	for _, c := range cmds {
		resp.Commands = append(resp.Commands, c.ToV1Command())
	}
	return resp, nil
}

// GetCommand looks up a command by ID returns a summary of the its state and configuration.
func (cs *CommandService) GetCommand(req *apiv1.GetCommandRequest) (*apiv1.GetCommandResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.CommandId), model.TaskTypeCommand)
	if err != nil {
		return nil, api.NotFoundErrs("command", req.CommandId, true)
	}

	return &apiv1.GetCommandResponse{
		Command: c.ToV1Command(),
		Config:  protoutils.ToStruct(c.Config),
	}, nil
}

// GetNotebooks returns all notebooks in the command service registry matching the workspace ID.
func (cs *CommandService) GetNotebooks(req *apiv1.GetNotebooksRequest) (*apiv1.GetNotebooksResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetNotebooksResponse{}
	cmds := cs.listByType(req.Users, req.UserIds, model.TaskTypeNotebook, req.WorkspaceId)
	for _, c := range cmds {
		resp.Notebooks = append(resp.Notebooks, c.ToV1Notebook())
	}
	return resp, nil
}

// GetNotebook looks up a notebook by ID returns a summary of the its state and configuration.
func (cs *CommandService) GetNotebook(req *apiv1.GetNotebookRequest) (*apiv1.GetNotebookResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.NotebookId), model.TaskTypeNotebook)
	if err != nil {
		return nil, api.NotFoundErrs("notebook", req.NotebookId, true)
	}

	return &apiv1.GetNotebookResponse{
		Notebook: c.ToV1Notebook(),
		Config:   protoutils.ToStruct(c.Config),
	}, nil
}

// GetShells returns all shells in the command service registry matching the workspace ID.
func (cs *CommandService) GetShells(req *apiv1.GetShellsRequest) (*apiv1.GetShellsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetShellsResponse{}
	cmds := cs.listByType(req.Users, req.UserIds, model.TaskTypeShell, req.WorkspaceId)
	for _, c := range cmds {
		resp.Shells = append(resp.Shells, c.ToV1Shell())
	}
	return resp, nil
}

// GetShell looks up a shell by ID returns a summary of the its state and configuration.
func (cs *CommandService) GetShell(req *apiv1.GetShellRequest) (*apiv1.GetShellResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(req.ShellId), model.TaskTypeShell)
	if err != nil {
		return nil, api.NotFoundErrs("shell", req.ShellId, true)
	}

	return &apiv1.GetShellResponse{
		Shell:  c.ToV1Shell(),
		Config: protoutils.ToStruct(c.Config),
	}, nil
}

// GetTensorboards returns all tbs in the command service registry matching the workspace ID.
func (cs *CommandService) GetTensorboards(req *apiv1.GetTensorboardsRequest) (*apiv1.GetTensorboardsResponse, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	resp := &apiv1.GetTensorboardsResponse{}
	cmds := cs.listByType(req.Users, req.UserIds, model.TaskTypeTensorboard, req.WorkspaceId)
	for _, c := range cmds {
		resp.Tensorboards = append(resp.Tensorboards, c.ToV1Tensorboard())
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
		Tensorboard: c.ToV1Tensorboard(),
		Config:      protoutils.ToStruct(c.Config),
	}, nil
}
