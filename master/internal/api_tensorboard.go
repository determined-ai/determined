package internal

import (
	"context"
	"os"
	"time"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

var tensorboardsAddr = actor.Addr("tensorboard")

func filesToArchive(files []*utilv1.File) archive.Archive {
	filesArchive := make([]archive.Item, 0)
	for _, file := range files {
		item := archive.Item{
			Content:      file.Content,
			FileMode:     os.FileMode(file.Mode),
			GroupID:      int(file.Gid),
			ModifiedTime: archive.UnixTime{Time: time.Unix(file.Mtime, 0)},
			Path:         file.Path,
			Type:         byte(file.Type),
			UserID:       int(file.Uid),
		}
		filesArchive = append(filesArchive, item)
	}
	return filesArchive
}

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

	params, err := a.prepareLaunchParams(ctx, &protoCommandParams{
		TemplateName: req.TemplateName,
		Config:       req.Config,
		Files:        req.Files,
		MustZeroSlot: true,
	})
	if err != nil {
		return nil, err
	}

	tensorboardConfig := command.TensorboardRequest{
		CommandParams: params,
		ExperimentIDs: experimentIds,
		TrialIDs:      trialIds,
	}
	tensorboardLaunchReq := command.TensorboardRequestWithUser{Tensorboard: tensorboardConfig}
	tensorboardIDFut := a.m.system.AskAt(tensorboardsAddr, tensorboardLaunchReq)
	if err = api.ProcessActorResponseError(&tensorboardIDFut); err != nil {
		return nil, err
	}

	tensorboardID := tensorboardIDFut.Get().(sproto.TaskID)
	tensorboard := a.m.system.AskAt(
		tensorboardsAddr.Child(tensorboardID),
		&tensorboardv1.Tensorboard{},
	)
	if err = api.ProcessActorResponseError(&tensorboard); err != nil {
		return nil, err
	}

	return &apiv1.LaunchTensorboardResponse{
		Tensorboard: tensorboard.Get().(*tensorboardv1.Tensorboard),
		Config:      protoutils.ToStruct(*params.FullConfig),
	}, err
}
