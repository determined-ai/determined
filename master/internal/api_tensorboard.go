package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/grpc"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/internal/status"
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

func (a *apiServer) LaunchTensorboard(
	ctx context.Context, req *apiv1.LaunchTensorboardRequest,
) (resp *apiv1.LaunchTensorboardResponse, err error) {

	experimentIds := make([]int, 0)
	trialIds := make([]int, 0)
	for _, id := range req.ExperimentIds {
		experimentIds = append(experimentIds, int(id))
	}
	for _, id := range req.TrialIds {
		trialIds = append(trialIds, int(id))
	}
	tensorboardReq := command.TensorboardRequest{
		ExperimentIDs: experimentIds,
		TrialIDs:      trialIds,
	}
	user, _, err := grpc.GetUser(ctx, a.m.db)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user")
	}

	var tensorboardID scheduler.TaskID
	err = a.actorRequest(
		tensorboardsAddr.String(),
		command.TensorboardRequestWithUser{Tensorboard: tensorboardReq, User: user},
		&tensorboardID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to launch tensorboard")
	}

	var tensorboardv1 *tensorboardv1.Tensorboard
	err = a.actorRequest(tensorboardsAddr.Child(tensorboardID).String(), tensorboardv1, &tensorboardv1)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			"failed to get the created tensorboard %s",
			tensorboardID,
		)
	}

	return &apiv1.LaunchTensorboardResponse{Tensorboard: tensorboardv1}, err
}
