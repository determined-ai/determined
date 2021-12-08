package resourcemanagers

import (
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

func mergeToJobQInfo(reqs AllocReqs) (map[model.JobID]*job.RMJobInfo, map[model.JobID]*actor.Ref) {
	isAdded := make(map[model.JobID]*job.RMJobInfo)
	jobActors := make(map[model.JobID]*actor.Ref)
	jobsAhead := 0
	for _, req := range reqs {
		if req.JobID == nil {
			continue
		}
		v1JobInfo, exists := isAdded[*req.JobID]
		if !exists {
			v1JobInfo = &job.RMJobInfo{
				JobsAhead: jobsAhead,
				State:     req.State,
			}
			isAdded[*req.JobID] = v1JobInfo
			jobActors[*req.JobID] = req.Group
			jobsAhead++
		}
		// Carry over the the highest state.
		if v1JobInfo.State < req.State {
			isAdded[*req.JobID].State = req.State
		}
		v1JobInfo.RequestedSlots += req.SlotsNeeded
		if job.ScheduledStates[req.State] {
			v1JobInfo.AllocatedSlots += req.SlotsNeeded
		}
	}
	return isAdded, jobActors
}

func jobStats(rp *ResourcePool) *jobv1.QueueStats {
	stats := &jobv1.QueueStats{}
	counted := make(map[model.JobID]bool)
	for it := rp.taskList.iterator(); it.next(); {
		req := it.value()
		if req.JobID == nil || counted[*req.JobID] {
			continue
		}
		counted[*req.JobID] = true

		if req.Preemptible {
			stats.PreemptibleCount++
		}
		if req.State == job.SchedulingStateQueued {
			stats.QueuedCount++
		} else {
			stats.ScheduledCount++
		}
	}
	return stats
}
