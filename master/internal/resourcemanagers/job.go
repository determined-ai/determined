package resourcemanagers

import (
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
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

func jobStats(taskList *taskList) *jobv1.QueueStats {
	stats := &jobv1.QueueStats{}
	reqs := make(AllocReqs, 0)
	for it := taskList.iterator(); it.next(); {
		req := it.value()
		if req.JobID == nil {
			continue
		}
		reqs = append(reqs, req)
	}
	jobsMap, _ := mergeToJobQInfo(reqs)
	for _, jobInfo := range jobsMap {
		if jobInfo.State == job.SchedulingStateQueued {
			stats.QueuedCount++
		} else {
			stats.ScheduledCount++
		}
	}
	return stats
}

func updateAllocateReqState(req *sproto.AllocateRequest, taskList *taskList) {
	allocations := taskList.GetAllocations(req.TaskActor)
	if allocations == nil || len(allocations.Reservations) == 0 {
		req.State = job.SchedulingStateQueued
	} else {
		req.State = job.SchedulingStateScheduled
	}
}
