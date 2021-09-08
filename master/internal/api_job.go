package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	switch {
	case sproto.UseAgentRM(a.m.system):
		err = a.actorRequest(sproto.JobsAddr, req, &resp)
	case sproto.UseK8sRM(a.m.system):
		err = a.actorRequest(sproto.PodsAddr, req, &resp)
	default:
		err = status.Error(codes.NotFound, "cannot find jobs or pods actor")
	}
	if err != nil {
		return nil, err
	}
	a.filter(&resp.Jobs, func(i int) bool {
		v := resp.Jobs[i]
		return req.Label == "" || v.Label == req.Label
	})
	a.sort(resp.Jobs, req.OrderBy, req.SortBy, apiv1.GetJobsRequest_SORT_BY_ID)
	return resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Offset, req.Limit)
}
