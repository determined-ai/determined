package internal

import (
	"context"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GetJobs retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	actorResp := a.m.system.AskAt(job.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	jobs, ok := actorResp.Get().([]*jobv1.Job)
	if !ok {
		return nil, status.Error(codes.Internal, "unexpected response from actor")
	}

	// order by jobsAhead first and JobId second.
	sort.SliceStable(jobs, func(i, j int) bool {
		if req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC {
			i, j = j, i
		}
		if jobs[i].Summary == nil || jobs[j].Summary == nil {
			return false
		}
		if jobs[i].Summary.JobsAhead < jobs[j].Summary.JobsAhead {
			return true
		}
		if jobs[i].Summary.JobsAhead > jobs[j].Summary.JobsAhead {
			return false
		}
		return jobs[i].JobId < jobs[j].JobId
	})

	if req.Pagination == nil {
		req.Pagination = &apiv1.PaginationRequest{}
	}

	resp = &apiv1.GetJobsResponse{Jobs: jobs}
	return resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Pagination.Offset, req.Pagination.Limit)
}

// GetJobQueueStats retrieves job queue stats for a set of resource pools.
func (a *apiServer) GetJobQueueStats(
	_ context.Context, req *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	resp = &apiv1.GetJobQueueStatsResponse{
		Results: make([]*apiv1.RPQueueStat, 0),
	}

	err = a.ask(sproto.GetCurrentRM(a.m.system).Address(), req, &resp)
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

	actorResp := a.m.system.AskAt(job.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	return resp, nil
}
