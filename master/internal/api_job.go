package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

var notImplementedError = status.Error(codes.Unimplemented, "API not implemented")

// GetJobs TODO.
func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	// TODO pagination and filtering
	// a.sort(resp.Jobs, req.OrderBy, req.SortBy, apiv1.GetJobsRequest_SORT_BY_QUEUE_POSITION)
	// resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Pagination.Offset, req.Pagination.Limit)

	var jobs []*jobv1.Job
	resp = &apiv1.GetJobsResponse{}

	// TODO loop over all resource pools in the request
	if len(req.ResourcePools) < 1 {
		return nil, status.Error(codes.InvalidArgument, "missing resource_pools parameter") // FIXME
	}

	switch {
	case sproto.UseAgentRM(a.m.system):
		err = a.actorRequest(sproto.AgentRMAddr.Child(req.ResourcePools[0]), resourcemanagers.GetJobOrder{}, &jobs)
	case sproto.UseK8sRM(a.m.system):
		err = a.actorRequest(sproto.K8sRMAddr, resourcemanagers.GetJobOrder{}, &jobs)
	default:
		err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
	}
	if err != nil {
		return nil, err
	}
	// for _, job := range jobs {
	// 	if job == nil {
	// 		panic("received an empty job summary") // this shouldn't happen
	// 	}
	// 	resp.Jobs = append(resp.Jobs, &jobv1.Job{
	// 		Summary: &jobv1.JobSummary{
	// 			JobId: string(job.JobID),
	// 			State: job.State.Proto(), // look at AllocationState
	// 		},
	// 		EntityId: job.EntityID,
	// 		Type:     job.JobType.Proto(),
	// 	})
	// }
	resp.Jobs = jobs
	return resp, nil
}

// GetJobQueueStats TODO.
func (a *apiServer) GetJobQueueStats(
	_ context.Context, req *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	return nil, notImplementedError
}

// UpdateJobQueue TODO.
func (a *apiServer) UpdateJobQueue(
	_ context.Context, req *apiv1.UpdateJobQueueRequest,
) (resp *apiv1.UpdateJobQueueResponse, err error) {
	return nil, notImplementedError
}
