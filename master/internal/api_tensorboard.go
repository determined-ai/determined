package internal

import (
	"context"
	"fmt"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetTensorboards(
	_ context.Context, req *apiv1.GetTensorboardsRequest,
) (resp *apiv1.GetTensorboardsResponse, err error) {
	err = a.actorRequest("/tensorboards", req, &resp)
	if err != nil {
		return nil, err
	}
	a.sort(resp.Tensorboards, req.OrderBy, req.SortBy, apiv1.GetTensorboardsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Tensorboards, req.Offset, req.Limit)
}

func (a *apiServer) GetTensorboard(
	_ context.Context, req *apiv1.GetTensorboardRequest,
) (resp *apiv1.GetTensorboardResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/tensorboards/%s", req.TensorboardId), req, &resp)
}

func (a *apiServer) KillTensorboard(
	_ context.Context, req *apiv1.KillTensorboardRequest,
) (resp *apiv1.KillTensorboardResponse, err error) {
	return resp, a.actorRequest(fmt.Sprintf("/tensorboards/%s", req.TensorboardId), req, &resp)
}
