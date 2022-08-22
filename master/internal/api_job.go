package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GetJobs retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	actorResp := a.m.system.AskAt(sproto.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	jobs, ok := actorResp.Get().([]*jobv1.Job)
	if !ok {
		return nil, status.Error(codes.Internal, "unexpected response from actor")
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

	resp = &apiv1.GetJobsResponse{Jobs: jobs}
	return resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Offset, req.Limit)
}

// GetJobQueueStats retrieves job queue stats for a set of resource pools.
func (a *apiServer) GetJobQueueStats(
	_ context.Context, req *apiv1.GetJobQueueStatsRequest,
) (*apiv1.GetJobQueueStatsResponse, error) {
	resp, err := a.m.rm.GetJobQueueStatsRequest(a.m.system, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateJobQueue forwards the job queue message to the relevant resource pool.
func (a *apiServer) UpdateJobQueue(
	_ context.Context, req *apiv1.UpdateJobQueueRequest,
) (resp *apiv1.UpdateJobQueueResponse, err error) {
	resp = &apiv1.UpdateJobQueueResponse{}

	actorResp := a.m.system.AskAt(sproto.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	return resp, nil
}
