package resourcemanagers

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// GetJobOrder requests a list of *jobv1.Job.
// FIXME haven't decided if resource manager actor should be responsible for this or not
// we don't want a separate actor do we? could be useful for streaming job endpoints.
type GetJobOrder struct{}

// SetJobOrder conveys a job queue change for a specific jobID to the resource pool.
type SetJobOrder struct {
	QPosition float64
	Weight    float64
	Priority  *int
	JobID     model.JobID
}

/* filterAllocateRequests
1. filters allocations that are not associated with a job
2. merge/filter multilpe allocations representing a single job. If a job has many allocReqs this
would only keep the one that's most representative of the final job state.
Input: a list of allocateRequests sorted by expected order of execution from the scheduler.
*/
func filterAllocateRequests(reqs AllocReqs) AllocReqs {
	isAdded := make(map[model.JobID]sproto.SchedulingState)
	filteredReqs := make(AllocReqs, 0)
	for _, req := range reqs {
		job := req.Job
		if job == nil {
			continue
		} else if state, ok := isAdded[job.JobID]; ok {
			if state < job.State {
				isAdded[job.JobID] = job.State
			}
			continue
		}
		isAdded[job.JobID] = req.Job.State
		filteredReqs = append(filteredReqs, req)
	}
	for _, req := range filteredReqs {
		req.Job.State = isAdded[req.Job.JobID]
	}
	return filteredReqs
}

// allocReqsToJobOrder converts sorted allocation requests to job order.
func allocReqsToJobOrder(reqs []*sproto.AllocateRequest) (jobIds []string) {
	for _, req := range filterAllocateRequests(reqs) {
		jobIds = append(jobIds, string(req.Job.JobID))
	}
	return jobIds
}

func allocateReqToV1Job(
	rp *ResourcePool,
	req *sproto.AllocateRequest,
) (job *jobv1.Job) {
	if req.Job == nil {
		return job
	}
	group := rp.groups[req.Group]
	job = &jobv1.Job{
		Summary: &jobv1.JobSummary{
			JobId: string(req.Job.JobID),
			State: req.Job.State.Proto(),
		},
		EntityId:       req.Job.EntityID,
		Type:           req.Job.JobType.Proto(),
		IsPreemptible:  req.Preemptible,
		ResourcePool:   req.ResourcePool,
		User:           "demo-hamid", // TODO
		SubmissionTime: timestamppb.New(req.TaskActor.RegisteredTime()),
	}
	switch schdType := rp.config.Scheduler.GetType(); schdType {
	case fairShareScheduling:
		job.Weight = group.weight
	case priorityScheduling:
		job.Priority = int32(*group.priority)
	}
	return job
}

// getV1Jobs generates a list of jobv1.Job through scheduler.OrderedAllocations.
func getV1Jobs( // TODO rename
	rp *ResourcePool,
) []*jobv1.Job {
	allocateRequests := rp.scheduler.OrderedAllocations(rp)
	v1Jobs := make([]*jobv1.Job, 0)
	for _, req := range filterAllocateRequests(allocateRequests) {
		v1Jobs = append(v1Jobs, allocateReqToV1Job(rp, req))
	}
	return v1Jobs
}

func setJobState(req *sproto.AllocateRequest, state sproto.SchedulingState) {
	if req.Job == nil {
		return
	}
	req.Job.State = state
}
