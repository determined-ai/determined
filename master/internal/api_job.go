package internal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

//var notImplementedError = status.Error(codes.Unimplemented, "API not implemented")

// GetJobs retrieves a list of jobs for a resource pool.
func (a *apiServer) GetJobs(
	_ context.Context, req *apiv1.GetJobsRequest,
) (resp *apiv1.GetJobsResponse, err error) {
	resp = &apiv1.GetJobsResponse{
		Jobs: make([]*jobv1.Job, 0),
	}

	if req.ResourcePool == "" {
		return nil, status.Error(codes.InvalidArgument, "missing resourcePool parameter")
	}

	switch {
	case sproto.UseAgentRM(a.m.system):
		err = a.actorRequest(
			sproto.AgentRMAddr.Child(req.ResourcePool), resourcemanagers.GetJobOrder{}, &resp.Jobs,
		)
	case sproto.UseK8sRM(a.m.system):
		err = a.actorRequest(sproto.K8sRMAddr, resourcemanagers.GetJobOrder{}, &resp.Jobs)
	default:
		err = status.Error(codes.NotFound, "cannot find the appropriate resource manager")
	}
	if err != nil {
		return nil, err
	}

	if req.OrderBy == apiv1.OrderBy_ORDER_BY_ASC {
		// Reverese the list.
		for i, j := 0, len(resp.Jobs)-1; i < j; i, j = i+1, j-1 {
			resp.Jobs[i], resp.Jobs[j] = resp.Jobs[j], resp.Jobs[i]
		}
	}

	if req.Pagination == nil {
		req.Pagination = &apiv1.PaginationRequest{}
	}
	/* TODO user information
	2. persist use with job info. not all jobs are persisted.
	3. allocateReq.taskActor => a msg to get the task user/owner.
	would need to bubble up incase of eg trial actor to experiment
	*/
	return resp, a.paginate(&resp.Pagination, &resp.Jobs, req.Pagination.Offset, req.Pagination.Limit)
}

// GetJobQueueStats retrieves job queue stats for a set of resource pools.
func (a *apiServer) GetJobQueueStats(
	_ context.Context, req *apiv1.GetJobQueueStatsRequest,
) (resp *apiv1.GetJobQueueStatsResponse, err error) {
	resp = &apiv1.GetJobQueueStatsResponse{
		Results: make([]*apiv1.RPQueueStat, 0),
	}

	rmRef := sproto.GetCurrentRM(a.m.system)
	rpAddresses := make([]actor.Address, 0)
	if len(req.ResourcePools) == 0 {
		for _, ref := range rmRef.Children() {
			rpAddresses = append(rpAddresses, ref.Address())
		}
	} else {
		for _, rp := range req.ResourcePools {
			rpAddresses = append(rpAddresses, rmRef.Child(rp).Address())
		}
	}

	for _, rpAddr := range rpAddresses {
		stats := jobv1.QueueStats{}
		qStats := apiv1.RPQueueStat{ResourcePool: rpAddr.Local()}
		err = a.actorRequest(
			rpAddr, resourcemanagers.GetJobQStats{}, &stats,
		)
		if err != nil {
			return nil, err
		}
		qStats.Stats = &stats
		resp.Results = append(resp.Results, &qStats)
	}
	return resp, nil
}

// UpdateJobQueue forwards the job queue message to the relevant resource pool.
func (a *apiServer) UpdateJobQueue(
	_ context.Context, req *apiv1.UpdateJobQueueRequest,
) (resp *apiv1.UpdateJobQueueResponse, err error) {
	resp = &apiv1.UpdateJobQueueResponse{}

	for _, update := range req.Updates {
		qPosition := float64(update.GetQueuePosition())
		priority := int(update.GetPriority())
		weight := float64(update.GetWeight())
		msg := resourcemanagers.SetJobOrder{
			QPosition: qPosition,
			Priority:  &priority,
			Weight:    weight,
			JobID:     model.JobID(update.GetJobId()),
		}
		switch {
		case sproto.UseAgentRM(a.m.system):
			err = a.m.system.AskAt(sproto.AgentRMAddr.Child(update.SourceResourcePool), msg).Error()
		case sproto.UseK8sRM(a.m.system):
			err = a.m.system.AskAt(sproto.K8sRMAddr, msg).Error()
		default:
			err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
		}
		if err != nil {
			return resp, err
		}
	}
	return resp, nil
}
