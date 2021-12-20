package job

import (
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// JobsActorAddr is the address of the jobs actor.
var JobsActorAddr = actor.Addr("jobs")

// RMJobInfo packs information available only to the RM that updates frequently.
type RMJobInfo struct { // rename ?
	JobsAhead      int
	State          SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

// GetJobSummary requests a summary of the job.
type GetJobSummary struct{}

// GetJob requests a job representation from a job.
type GetJob struct{}

// GetJobQ is used to get all job information in one go to avoid any inconsistencies.
type GetJobQ struct {
	ResourcePool string
}

// GetJobQStats requests stats for a queue.
// Expected response: jobv1.QueueStats.
type GetJobQStats struct {
	ResourcePool string
}

type (
	// SetGroupWeight sets the weight of a group in the fair share scheduler.
	SetGroupWeight struct {
		Weight       float64
		ResourcePool string
		Handler      *actor.Ref
	}
	// SetGroupPriority sets the priority of the group in the priority scheduler.
	SetGroupPriority struct {
		Priority     *int
		ResourcePool string
		Handler      *actor.Ref
	}
	// SetGroupOrder sets the order of the group in the priority scheduler.
	SetGroupOrder struct {
		QPosition    float64
		ResourcePool string
		Handler      *actor.Ref
	}
)

// RegisterJob Registers an active job with the jobs actor.
// Used as to denote a child actor.
type RegisterJob struct {
	JobID    model.JobID
	JobActor *actor.Ref
}

// UnregisterJob removes a job from the jobs actor.
type UnregisterJob struct {
	JobID model.JobID
}

// Jobs manage jobs.
type Jobs struct {
	RMRef    *actor.Ref
	jobsByID map[model.JobID]*actor.Ref
}

// AQueue is a map of jobID to RMJobInfo.
type AQueue = map[model.JobID]*RMJobInfo

// NewJobs creates a new jobs actor.
func NewJobs(rmRef *actor.Ref) *Jobs {
	return &Jobs{
		RMRef:    rmRef,
		jobsByID: make(map[model.JobID]*actor.Ref),
	}
}

func (j *Jobs) parseV1JobMsgs(
	msgs map[*actor.Ref]actor.Message,
) (map[model.JobID]*jobv1.Job, error) {
	jobs := make(map[model.JobID]*jobv1.Job)
	for _, val := range msgs {
		if val == nil {
			continue
		}
		typed, ok := val.(*jobv1.Job)
		if !ok {
			return nil, fmt.Errorf("unexpected response type: %T", val)
		}
		jobs[model.JobID(typed.JobId)] = typed
	}
	return jobs, nil
}

// Receive implements the actor.Actor interface.
func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case RegisterJob:
		j.jobsByID[msg.JobID] = msg.JobActor

	case UnregisterJob:
		delete(j.jobsByID, msg.JobID)

	case *apiv1.GetJobsRequest:
		// Ask for a consistent snapshot of the job queue from the RM.
		aResp := ctx.Ask(j.RMRef, GetJobQ{ResourcePool: msg.ResourcePool})
		if err := aResp.Error(); err != nil {
			ctx.Log().WithError(err).Error("getting job queue info from RM")
			ctx.Respond(err)
			return nil
		}

		jobQ, ok := aResp.Get().(AQueue)
		if !ok {
			err := fmt.Errorf("unexpected response type: %T from RM", aResp.Get())
			ctx.Log().WithError(err).Error("")
			ctx.Respond(err)
			return nil
		}

		// Get jobs from the job actors.
		jobRefs := make([]*actor.Ref, 0)
		for jID := range jobQ {
			jobRef, ok := j.jobsByID[jID]
			if ok {
				jobRefs = append(jobRefs, jobRef)
			}
		}
		jobs, err := j.parseV1JobMsgs(ctx.AskAll(GetJob{}, jobRefs...).GetAll())
		if err != nil {
			ctx.Log().WithError(err).Error("parsing responses from job actors")
			ctx.Respond(err)
			return nil
		}

		// Merge the results.
		jobsInRM := make([]*jobv1.Job, 0)
		for jID, jRMInfo := range jobQ {
			v1Job, ok := jobs[jID]
			if ok {
				UpdateJobQInfo(v1Job, jRMInfo)
				jobsInRM = append(jobsInRM, v1Job)
			}
		}
		ctx.Respond(jobsInRM)

	case *apiv1.UpdateJobQueueRequest:
		for _, update := range msg.Updates {
			jobID := update.JobId
			jobActor := j.jobsByID[model.JobID(jobID)]
			if jobActor == nil {
				ctx.Respond(fmt.Errorf("job %s not found", jobID))
				return nil
			}
			switch action := update.GetAction().(type) {
			case *jobv1.QueueControl_Priority:
				priority := int(action.Priority)
				// TODO switch to Ask and check for errors
				ctx.Tell(jobActor, SetGroupPriority{
					Priority: &priority,
				})
			case *jobv1.QueueControl_Weight:
				ctx.Tell(jobActor, SetGroupWeight{
					Weight: float64(action.Weight),
				})
			case *jobv1.QueueControl_ResourcePool:
				// not supported yet
				// send an error with unsupported action
				ctx.Respond(fmt.Errorf("action not implemented: %v", action))
				return nil
			case *jobv1.QueueControl_QueuePosition:
				// REMOVEME: keep this until ahead_of and behind_of are implemented
				ctx.Tell(jobActor, SetGroupOrder{
					QPosition: float64(update.GetQueuePosition()),
				})
			case *jobv1.QueueControl_AheadOf, *jobv1.QueueControl_BehindOf:
				// TODO not supported yet
				ctx.Respond(fmt.Errorf("action not implemented: %v", action))
				return nil
			default:
				ctx.Respond(fmt.Errorf("unexpected action: %v", action))
				return nil
			}
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// UpdateJobQInfo updates the job with the RMJobInfo.
func UpdateJobQInfo(job *jobv1.Job, rmInfo *RMJobInfo) {
	if job == nil {
		panic("nil job ptr")
	}

	if rmInfo == nil {
		job.Summary = nil
		job.RequestedSlots = 0
		job.AllocatedSlots = 0
		return
	}

	job.RequestedSlots = int32(rmInfo.RequestedSlots)
	job.AllocatedSlots = int32(rmInfo.AllocatedSlots)
	if job.Summary == nil {
		job.Summary = &jobv1.JobSummary{}
	}
	job.Summary.State = rmInfo.State.Proto()
	job.Summary.JobsAhead = int32(rmInfo.JobsAhead)
}
