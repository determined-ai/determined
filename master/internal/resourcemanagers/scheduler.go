package resourcemanagers

import (
	"fmt"

	"github.com/goombaio/orderedset"
	"github.com/labstack/gommon/log"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

type SchedulingState uint8

const (
	SchedulingStateQueued              SchedulingState = 0
	SchedulingStateScheduledBackfilled SchedulingState = 1
	SchedulingStateScheduled           SchedulingState = 2
)

type QueueThing struct { // TODO rename me
	AReq  *sproto.AllocateRequest
	State SchedulingState
}
type SchedulerQueue = []QueueThing

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

func allocReqsToJobSummaries(reqs []*sproto.AllocateRequest) (summaries []*sproto.JobSummary) {
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

// WIP
func allocReqsToJobSummariesV2(queue SchedulerQueue) (jobs []*jobv1.Job) {
	isAdded := make(map[model.JobID]SchedulingState)
	for _, qt := range queue {
		job := qt.AReq.Job
		if job == nil {
			continue
		} else if _, ok := isAdded[job.JobID]; ok {
			// TODO need to merge the scheduler states
			continue
		}
		isAdded[job.JobID] = qt.State
		// jobs = append(jobs, job)
	}
	return jobs
}

func logAllocRequests(reqs []*sproto.AllocateRequest) {
	var str string
	for _, req := range reqs {
		if req.Job == nil {
			continue
		}
		str += fmt.Sprintf(", AID %s, JID %s | ", req.AllocationID, req.Job.JobID)
		// str = fmt.Sprintf("%s, AID %s, JID %s | ", str, req.AllocationID, req.JobID)
	}
	log.Debug("allocRequests" + str)
}
