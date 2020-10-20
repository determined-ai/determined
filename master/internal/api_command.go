package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	pstruct "github.com/golang/protobuf/ptypes/struct"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

var commandsAddr = actor.Addr("commands")

type protoCommandParams struct {
	TemplateName string
	Config       *pstruct.Struct
	Files        []*utilv1.File
	Data         []byte
}

// prepareLaunchParams prepares command launch parameters.
func (a *apiServer) prepareLaunchParams(ctx context.Context, req *protoCommandParams) (
	*command.CommandParams, *model.User, error,
) {
	user, _, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	cmdParams := command.CommandParams{UserFiles: filesToArchive(req.Files)}
	if req.TemplateName != "" {
		cmdParams.Template = &req.TemplateName
	}
	if req.Config != nil {
		configBytes, err := protojson.Marshal(req.Config)
		if err != nil {
			return nil, nil, err
		}
		cmdParams.ConfigBytes = configBytes
	}
	if len(req.Data) != 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return nil, nil, err
		}
		cmdParams.Data = data
	}
	return &cmdParams, user, nil
}

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
	cmdParams, user, err := a.prepareLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		Data:         req.Data,
	})
	if err != nil {
		return nil, err
	}

	commandLaunchReq := command.CommandLaunchRequest{
		CommandParams: cmdParams,
		User:          user,
	}
	commandIDFut := a.m.system.AskAt(commandsAddr, commandLaunchReq)
	if err = api.ProcessActorResponseError(&commandIDFut); err != nil {
		return nil, err
	}

	commandID := commandIDFut.Get().(resourcemanagers.TaskID)
	command := a.m.system.AskAt(commandsAddr.Child(commandID), &commandv1.Command{})
	if err = api.ProcessActorResponseError(&command); err != nil {
		return nil, err
	}

	return &apiv1.LaunchCommandResponse{
		Command: command.Get().(*commandv1.Command),
	}, nil
}
