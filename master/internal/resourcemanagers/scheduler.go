package resourcemanagers

import (
	"fmt"

	"github.com/goombaio/orderedset"
	"github.com/labstack/gommon/log"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// Scheduler schedules tasks on agents.  Its only function Schedule is called
// to determine which pending requests can be fulfilled and which scheduled tasks
// can be terminated. Schedule is expected to ba called every time there is a change
// to the cluster status, for example, new agents being connected, devices being disabled,
// and etc,. Schedule should avoid unnecessary shuffling tasks on agents to avoid
// the overhead of restarting a preempted task.
type Scheduler interface {
	Schedule(rp *ResourcePool) ([]*sproto.AllocateRequest, []*actor.Ref)
	OrderedAllocations(rp *ResourcePool) []*sproto.AllocateRequest
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
	jobSet := orderedset.NewOrderedSet()
	for _, req := range reqs {
		if req.JobID == "" {
			continue
		}
		jobSet.Add(req.JobID)
	}
	return jobSet
}

func allocReqsToJobSummaries(reqs []*sproto.AllocateRequest) *orderedset.OrderedSet {
	jobSet := orderedset.NewOrderedSet()
	for _, req := range reqs {
		if req.JobID == "" {
			continue
		}
		jobSet.Add(JobSummary{
			JobID:    req.JobID,
			JobType:  model.JobTypeTensorboard, // getJobType(req.TaskActor)
			EntityID: "TODO.ID",                // getEntityId(req.TaskActor)
		})
	}
	return jobSet
}

// TODO get final entity id and type

func logAllocRequests(reqs []*sproto.AllocateRequest) {
	var str string
	for _, req := range reqs {
		str += fmt.Sprintf(", AID %s, JID %s | ", req.AllocationID, req.JobID)
		// str = fmt.Sprintf("%s, AID %s, JID %s | ", str, req.AllocationID, req.JobID)
	}
	log.Debug("allocRequests" + str)
}
