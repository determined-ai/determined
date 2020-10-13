package internal

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

var tensorboardsAddr = actor.Addr("tensorboard")

func (a *apiServer) GetTensorboards(
	_ context.Context, req *apiv1.GetTensorboardsRequest,
) (resp *apiv1.GetTensorboardsResponse, err error) {
	err = a.actorRequest(tensorboardsAddr.String(), req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Tensorboards, req.OrderBy, req.SortBy, apiv1.GetTensorboardsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Tensorboards, req.Offset, req.Limit)
}

func (a *apiServer) GetTensorboard(
	_ context.Context, req *apiv1.GetTensorboardRequest,
) (resp *apiv1.GetTensorboardResponse, err error) {
	return resp, a.actorRequest(tensorboardsAddr.Child(req.TensorboardId).String(), req, &resp)
}

func (a *apiServer) KillTensorboard(
	_ context.Context, req *apiv1.KillTensorboardRequest,
) (resp *apiv1.KillTensorboardResponse, err error) {
	return resp, a.actorRequest(tensorboardsAddr.Child(req.TensorboardId).String(), req, &resp)
}

func filesToArchive(files []*utilv1.File) archive.Archive {
	filesArchive := make([]archive.Item, 0)
	for _, file := range files {
		item := archive.Item{
			Path:     file.Path,
			Type:     byte(file.Type),
			FileMode: os.FileMode(file.Mode),
			Content:  file.Content,
			UserID:   int(file.Uid),
			GroupID:  int(file.Gid),
		}
		item.ModifiedTime = archive.UnixTime{Time: time.Unix(file.Mtime, 0)}
		filesArchive = append(filesArchive, item)
	}
	return filesArchive
}

func apiCmdParamsToCommandParams(apiCmdParams *apiv1.CommandParams) (*command.CommandParams, error) {
	commandParams := command.CommandParams{
		ConfigBytes: apiCmdParams.Config,
		UserFiles:   filesToArchive(apiCmdParams.UserFiles),
	}
	if len(apiCmdParams.Data) != 0 {
		var data map[string]interface{}
		if err := json.Unmarshal(apiCmdParams.Data, &data); err != nil {
			return nil, err
		}
		commandParams.Data = data
	}
	if apiCmdParams.TemplateName != "" {
		commandParams.Template = &apiCmdParams.TemplateName
	}
	return &commandParams, nil
}

func (a *apiServer) LaunchTensorboard(
	ctx context.Context, req *apiv1.LaunchTensorboardRequest,
) (*apiv1.LaunchTensorboardResponse, error) {
	experimentIds := make([]int, 0)
	trialIds := make([]int, 0)
	for _, id := range req.ExperimentIds {
		experimentIds = append(experimentIds, int(id))
	}
	for _, id := range req.TrialIds {
		trialIds = append(trialIds, int(id))
	}
	cmdParams, err := apiCmdParamsToCommandParams(req.CommandParams)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to parse command parameters: %s", err)
	}
	tensorboardConfig := command.TensorboardRequest{
		CommandParams: *cmdParams,
		ExperimentIDs: experimentIds,
		TrialIDs:      trialIds,
	}
	user, _, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	tensorboardLaunchReq := command.TensorboardRequestWithUser{
		Tensorboard: tensorboardConfig,
		User:        user,
	}
	actorResp := a.m.system.AskAt(tensorboardsAddr, tensorboardLaunchReq)
	if err = api.ProcessActorResponseError(&actorResp); err != nil {
		return nil, err
	}

	tensorboardID := actorResp.Get().(resourcemanagers.TaskID)
	tensorboardReq := tensorboardv1.Tensorboard{}
	actorResp = a.m.system.AskAt(tensorboardsAddr.Child(tensorboardID), &tensorboardReq)
	if err = api.ProcessActorResponseError(&actorResp); err != nil {
		return nil, err
	}

	return &apiv1.LaunchTensorboardResponse{
		Tensorboard: actorResp.Get().(*tensorboardv1.Tensorboard),
	}, err
}
