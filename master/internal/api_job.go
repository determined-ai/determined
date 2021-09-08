package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

var notImplementedError = status.Error(codes.Unimplemented, "API not implemented")

// GetJobs TODO.
func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	// a.sort(resp.Jobs, req.OrderBy, req.SortBy, apiv1.GetJobsRequest_SORT_BY_QUEUE_POSITION)
	// resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Pagination.Offset, req.Pagination.Limit)
	return nil, notImplementedError
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
