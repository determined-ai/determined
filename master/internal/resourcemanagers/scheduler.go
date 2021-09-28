package resourcemanagers

import (
	"fmt"

	"github.com/goombaio/orderedset"
	"github.com/labstack/gommon/log"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
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

// WARN we don't merge requests.
// merging the states is not needed if we assume the allocaterequest in front of the queue has the dominant (desired)
// state for the job
func filterAllocateRequests(reqs AllocReqs) AllocReqs {
	isAdded := make(map[model.JobID]bool)
	rv := make(AllocReqs, 0)
	for _, req := range reqs {
		job := req.Job
		if job == nil {
			continue
		} else if _, ok := isAdded[job.JobID]; ok {
			continue
		}
		isAdded[job.JobID] = true
		rv = append(rv, req)
	}
	return rv
}

func allocReqsToJobSummaries(reqs AllocReqs) (summaries []*sproto.JobSummary) {
	for _, req := range filterAllocateRequests(reqs) {
		summaries = append(summaries, req.Job)
	}
	return summaries
}

func fillApiJob( // TODO rename me
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
			State: req.Job.State.Proto(), // look at AllocationState
		},
		EntityId:       req.Job.EntityID,
		Type:           req.Job.JobType.Proto(),
		IsPreemptible:  req.Preemptible,
		ResourcePool:   req.ResourcePool,
		User:           "hamid", // TODO
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

func doit( // TODO rename
	rp *ResourcePool,
) []*jobv1.Job {
	allocateRequests := rp.scheduler.OrderedAllocations(rp)
	v1Jobs := make([]*jobv1.Job, 0)
	for _, req := range filterAllocateRequests(allocateRequests) {
		v1Jobs = append(v1Jobs, fillApiJob(rp, req))
	}
	return v1Jobs
}

// // WIP not needed if we assume the allocaterequest in front of the queue has the dominant (desired)
// // state for the job
// func allocReqsToJobSummariesV2(q AllocReqs) (summaries []*sproto.JobSummary) {
// 	isAdded := make(map[model.JobID]*sproto.JobSummary)
// 	summaries = make([]*sproto.JobSummary, 0)
// 	for _, req := range q {
// 		curJob := req.Job
// 		if curJob == nil {
// 			continue
// 		} else if addedJob, ok := isAdded[curJob.JobID]; ok && addedJob.State >= curJob.State {
// 			continue
// 		}
// 		isAdded[curJob.JobID] = curJob
// 		summaries = append(summaries, curJob)
// 	}
// 	return summaries
// 	// jobSet := orderedset.NewOrderedSet() // TODO stop using this (same as allocReqsToJobSummaries
// 	// for _, qt := range queue {
// 	// 	if qt.AReq.Job == nil {
// 	// 		continue
// 	// 	}
// 	// 	jobSet.Add(qt.AReq.Job.JobID)
// 	// 	jobSet
// 	// }
// 	// return jobSet
// }

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
