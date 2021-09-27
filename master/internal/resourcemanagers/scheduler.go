package resourcemanagers

import (
	"fmt"

	"github.com/goombaio/orderedset"
	"github.com/labstack/gommon/log"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// type QueueThing struct { // TODO rename me
// 	AReq  *sproto.AllocateRequest
// 	State sproto.SchedulingState
// }
// type SchedulerQueue = []QueueThing

// Scheduler schedules tasks on agents.  Its only function Schedule is called
// to determine which pending requests can be fulfilled and which scheduled tasks
// can be terminated. Schedule is expected to ba called every time there is a change
// to the cluster status, for example, new agents being connected, devices being disabled,
// and etc,. Schedule should avoid unnecessary shuffling tasks on agents to avoid
// the overhead of restarting a preempted task.
type Scheduler interface {
	Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref)
	OrderedAllocations(rp *ResourcePool) []*sproto.AllocateRequest
	// Queue(rp *ResourcePool) SchedulerQueue
}

// MakeScheduler returns the corresponding scheduler implementation.
func MakeScheduler(config *SchedulerConfig) Scheduler {
	switch config.GetType() {
	case priorityScheduling:
		return NewPriorityScheduler(config)
	case fairShareScheduling:
		return NewFairShareScheduler()
	case roundRobinScheduling:
		return NewRoundRobinScheduler()
	default:
		panic(fmt.Sprintf("invalid scheduler: %s", config.GetType()))
	}
}

// allocReqsToJobOrder convertes sorted allocation requests to job order.
func allocReqsToJobOrder(reqs []*sproto.AllocateRequest) *orderedset.OrderedSet {
	jobSet := orderedset.NewOrderedSet() // TODO stop using this (same as allocReqsToJobSummaries
	for _, req := range reqs {
		if req.Job == nil {
			continue
		}
		jobSet.Add(req.Job.JobID)
	}
	return jobSet
}

func allocReqsToJobSummaries(reqs AllocReqs) (summaries []*sproto.JobSummary) {
	isAdded := make(map[model.JobID]bool)
	for _, req := range reqs {
		job := req.Job
		if job == nil {
			continue
		} else if _, ok := isAdded[job.JobID]; ok {
			continue
		}
		isAdded[job.JobID] = true
		summaries = append(summaries, job)
	}
	return summaries
}

// WIP not needed if we assume the allocaterequest in front of the queue has the dominant (desired)
// state for the job
func allocReqsToJobSummariesV2(q AllocReqs) (summaries []*sproto.JobSummary) {
	isAdded := make(map[model.JobID]*sproto.JobSummary)
	summaries = make([]*sproto.JobSummary, 0)
	for _, req := range q {
		curJob := req.Job
		if curJob == nil {
			continue
		} else if addedJob, ok := isAdded[curJob.JobID]; ok && addedJob.State >= curJob.State {
			continue
		}
		isAdded[curJob.JobID] = curJob
		summaries = append(summaries, curJob)
	}
	return summaries
	// jobSet := orderedset.NewOrderedSet() // TODO stop using this (same as allocReqsToJobSummaries
	// for _, qt := range queue {
	// 	if qt.AReq.Job == nil {
	// 		continue
	// 	}
	// 	jobSet.Add(qt.AReq.Job.JobID)
	// 	jobSet
	// }
	// return jobSet
}

func logAllocRequests(reqs []*sproto.AllocateRequest, prefix string) {
	var str string
	for _, req := range reqs {
		if req.Job == nil {
			continue
		}
		str += fmt.Sprintf(", AID %s, JID %s | ", req.AllocationID, req.Job.JobID)
		// str = fmt.Sprintf("%s, AID %s, JID %s | ", str, req.AllocationID, req.JobID)
	}
	log.Debug(prefix + ": " + str)
}

func setJobState(req *sproto.AllocateRequest, state sproto.SchedulingState) {
	if req.Job == nil {
		return
	}
	req.Job.State = state
}
