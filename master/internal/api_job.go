package internal

import (
	"context"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/job/jobservice"

	"github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GetJobs retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobs(
	ctx context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	jobs, err := jobservice.Default.GetJobs(
		req.ResourcePool,
		req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC,
		req.States,
	)
	if err != nil {
		return nil, err
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

	return resp, api.Paginate(&resp.Pagination, &resp.Jobs, req.Offset, req.Limit)
}

// GetJobsV2 retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobsV2(
	ctx context.Context, req *apiv1.GetJobsV2Request,
) (resp *apiv1.GetJobsV2Response, err error) {
	jobs, err := jobservice.Default.GetJobs(
		req.ResourcePool,
		req.OrderBy == apiv1.OrderBy_ORDER_BY_DESC,
		req.States,
	)
	if err != nil {
		return nil, err
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

	return resp, api.Paginate(&resp.Pagination, &resp.Jobs, req.Offset, req.Limit)
}

// GetJobQueueStats retrieves job queue stats for a set of resource pools.
func (a *apiServer) GetJobQueueStats(
	_ context.Context, req *apiv1.GetJobQueueStatsRequest,
) (*apiv1.GetJobQueueStatsResponse, error) {
	resp, err := a.m.rm.GetJobQueueStatsRequest(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// UpdateJobQueue forwards the job queue message to the relevant resource pool.
func (a *apiServer) UpdateJobQueue(
	ctx context.Context, req *apiv1.UpdateJobQueueRequest,
) (*apiv1.UpdateJobQueueResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, err
	}
	permErr, err := job.AuthZProvider.Get().CanControlJobQueue(ctx, curUser)
	if err != nil {
		return nil, err
	}
	if permErr != nil {
		return nil, permErr
	}
	err = jobservice.Default.UpdateJobQueue(req.Updates)
	if err != nil {
		return nil, err
	}
	return &apiv1.UpdateJobQueueResponse{}, nil
}
