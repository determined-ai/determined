package resourcemanagers

import (
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// FIXME haven't decided if resource manager actor should be responsible for this or not
// we don't want a separate actor do we? could be useful for streaming job endpoints.
// CHECK do we define the following messages in sproto package?
// QUESTION should we use proto defined messages more often internally or keep them at api level

// SetJobOrder conveys a job queue change for a specific jobID to the resource pool.
type SetJobOrder struct {
	QPosition float64
	Weight    float64
	Priority  *int
	JobID     model.JobID
}

// GetJobQInfo is used to get all job information in one go to avoid any inconsistencies.
type GetJobQInfo struct{}

// GetJobQStats requests stats for a queue.
// Expected response: jobv1.QueueStats.
type GetJobQStats struct{}

func mergeToJobQInfo(reqs AllocReqs) (map[model.JobID]*job.RMJobInfo, map[model.JobID]*actor.Ref) {
	isAdded := make(map[model.JobID]*job.RMJobInfo)
	jobActors := make(map[model.JobID]*actor.Ref)
	jobsAhead := 0
	for _, req := range reqs {
		curJob := req.Job
		if curJob == nil {
			continue
		}
		v1JobInfo, exists := isAdded[curJob.JobID]
		if !exists {
			v1JobInfo = &job.RMJobInfo{
				JobsAhead:      jobsAhead,
				State:          req.Job.State,
				RequestedSlots: req.Job.RequestedSlots,
				AllocatedSlots: req.Job.AllocatedSlots,
				IsPreemptible:  req.Preemptible,
			}
			isAdded[curJob.JobID] = v1JobInfo
			jobsAhead++
			jobActors[curJob.JobID] = req.Group
		}
		// Carry over the the highest state.
		if v1JobInfo.State < curJob.State {
			isAdded[curJob.JobID].State = curJob.State
		}
		v1JobInfo.RequestedSlots += req.SlotsNeeded
		if job.ScheduledStates[req.Job.State] {
			v1JobInfo.AllocatedSlots += req.SlotsNeeded
		}
	}
	return isAdded, jobActors
}

func setJobState(req *sproto.AllocateRequest, state job.SchedulingState) {
	if req.Job == nil {
		return
	}
	req.Job.State = state
}

func jobStats(rp *ResourcePool) *jobv1.QueueStats {
	stats := &jobv1.QueueStats{}
	jobinfo := rp.scheduler.JobQInfo(rp)
	for _, j := range jobinfo {
		if j.IsPreemptible {
			stats.PreemptibleCount++
		}
		if j.State == job.SchedulingStateQueued {
			stats.QueuedCount++
		} else {
			stats.ScheduledCount++
		}
	}
	return stats
}
