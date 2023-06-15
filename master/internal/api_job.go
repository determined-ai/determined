package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GetJobs retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobs(
	ctx context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	actorResp := a.m.system.AskAt(sproto.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	jobs, ok := actorResp.Get().([]*jobv1.Job)
	if !ok {
		return nil, status.Error(codes.Internal, "unexpected response from actor")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	resp = &apiv1.GetJobsResponse{Jobs: make([]*jobv1.Job, 0)}

	jobs, err = job.AuthZProvider.Get().FilterJobs(ctx, *curUser, jobs)
	if err != nil {
		return nil, err
	}
	resp.Jobs = jobs

	if req.Limit == 0 {
		req.Limit = 100
	}

	return resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Offset, req.Limit)
}

// GetJobsV2 retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobsV2(
	ctx context.Context, req *apiv1.GetJobsV2Request,
) (resp *apiv1.GetJobsV2Response, err error) {
	actorResp := a.m.system.AskAt(sproto.JobsActorAddr, req)
	if err := actorResp.Error(); err != nil {
		return nil, err
	}
	jobs, ok := actorResp.Get().([]*jobv1.Job)
	if !ok {
		return nil, status.Error(codes.Internal, "unexpected response from actor")
	}
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	resp = &apiv1.GetJobsV2Response{Jobs: make([]*jobv1.RBACJob, 0)}

	okJobs, err := job.AuthZProvider.Get().FilterJobs(ctx, *curUser, jobs)
	if err != nil {
		return nil, err
	}
	okJobsMap := make(map[string]bool)
	for _, j := range okJobs {
		okJobsMap[j.JobId] = true
	}

	for _, job := range jobs {
		j := jobv1.RBACJob{}
		if ok := okJobsMap[job.JobId]; ok {
			j.Job = &jobv1.RBACJob_Full{
				Full: job,
			}
		} else {
			limitedJob := authz.ObfuscateJob(job)
			j.Job = &jobv1.RBACJob_Limited{
				Limited: &limitedJob,
			}
		}
		resp.Jobs = append(resp.Jobs, &j)
	}

	if req.Limit == 0 {
		req.Limit = 100
	}

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
