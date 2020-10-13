package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetCommands(
	_ context.Context, req *apiv1.GetCommandsRequest,
) (resp *apiv1.GetCommandsResponse, err error) {
	err = a.actorRequest("/commands", req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Commands, req.OrderBy, req.SortBy, apiv1.GetCommandsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Commands, req.Offset, req.Limit)
}

func (a *apiServer) GetCommand(
	_ context.Context, req *apiv1.GetCommandRequest) (resp *apiv1.GetCommandResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/commands/%s", req.CommandId), req, &resp)
}

func (a *apiServer) KillCommand(
	_ context.Context, req *apiv1.KillCommandRequest) (resp *apiv1.KillCommandResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/commands/%s", req.CommandId), req, &resp)
}

func (a *apiServer) LaunchCommand(
	ctx context.Context, req *apiv1.LaunchCommandRequest,
) (*apiv1.LaunchCommandResponse, error) {

	// commandConfig := command.CommandRequest{
	// 	ExperimentIDs: experimentIds,
	// 	TrialIDs:      trialIds,
	// }
	// user, _, err := grpc.GetUser(ctx, a.m.db)
	// if err != nil {
	// 	return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	// }

	// commandLaunchReq := command.CommandRequestWithUser{
	// 	Command: commandConfig,
	// 	User:    user,
	// }
	// actorResp := a.m.system.AskAt(commandsAddr, commandLaunchReq)
	// if err = api.ProcessActorResponseError(&actorResp); err != nil {
	// 	return nil, err
	// }

	// commandID := actorResp.Get().(resourcemanagers.TaskID)
	// commandReq := commandv1.Command{}
	// actorResp = a.m.system.AskAt(commandsAddr.Child(commandID), &commandReq)
	// if err = api.ProcessActorResponseError(&actorResp); err != nil {
	// 	return nil, err
	// }

	return &apiv1.LaunchCommandResponse{
		// Command: actorResp.Get().(*commandv1.Command),
	}, nil
}
