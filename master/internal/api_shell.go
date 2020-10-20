package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

var shellsAddr = actor.Addr("shells")

func (a *apiServer) GetShells(
	_ context.Context, req *apiv1.GetShellsRequest,
) (resp *apiv1.GetShellsResponse, err error) {
	err = a.actorRequest("/shells", req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Shells, req.OrderBy, req.SortBy, apiv1.GetShellsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Shells, req.Offset, req.Limit)
}

func (a *apiServer) GetShell(
	_ context.Context, req *apiv1.GetShellRequest) (resp *apiv1.GetShellResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/shells/%s", req.ShellId), req, &resp)
}

func (a *apiServer) KillShell(
	_ context.Context, req *apiv1.KillShellRequest) (resp *apiv1.KillShellResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/shells/%s", req.ShellId), req, &resp)
}

func (a *apiServer) LaunchShell(
	ctx context.Context, req *apiv1.LaunchShellRequest,
) (*apiv1.LaunchShellResponse, error) {
	cmdParams, user, err := a.prepareLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		Data:         req.Data,
	})
	if err != nil {
		return nil, err
	}

	shellLaunchReq := command.ShellLaunchRequest{
		CommandParams: cmdParams,
		User:          user,
	}
	shellIDFut := a.m.system.AskAt(shellsAddr, shellLaunchReq)
	if err = api.ProcessActorResponseError(&shellIDFut); err != nil {
		return nil, err
	}

	shellID := shellIDFut.Get().(resourcemanagers.TaskID)
	shell := a.m.system.AskAt(shellsAddr.Child(shellID), &shellv1.Shell{})
	if err = api.ProcessActorResponseError(&shell); err != nil {
		return nil, err
	}

	return &apiv1.LaunchShellResponse{
		Shell: shell.Get().(*shellv1.Shell),
	}, nil
}
