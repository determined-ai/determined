package job

import (
	"fmt"

	"github.com/pkg/errors"

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

// GetJobQ is used to get all job information in one go to avoid any inconsistencies.
type GetJobQ struct {
	ResourcePool string
}

// GetJobQStats requests stats for a queue.
// Expected response: jobv1.QueueStats.
type GetJobQStats struct {
}

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

func (j *Jobs) askJobActors(ctx *actor.Context, msg actor.Message) map[*actor.Ref]actor.Message {
	children := make([]*actor.Ref, 0)
	for _, jobRef := range j.jobsByID {
		children = append(children, jobRef)
	}
	// children := getJobRefs(ctx.Self().System())
	fmt.Printf("children count %d \n", len(children))
	return ctx.AskAll(msg, children...).GetAll()
}

func (j *Jobs) parseV1JobResposnes(
	responses map[*actor.Ref]actor.Message,
) (map[model.JobID]*jobv1.Job, error) {
	jobs := make(map[model.JobID]*jobv1.Job)
	for _, val := range responses {
		typed, ok := val.(*jobv1.Job)
		if !ok {
			return nil, errors.New("unexpected response type")
		}
		if typed != nil {
			jobs[model.JobID(typed.JobId)] = typed
		}
	}
	return jobs, nil
}

// Receive implements the actor.Actor interface.
func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case RegisterJob:
		fmt.Printf("RegisterJob %v \n, actor ref %s", msg.JobID, msg.JobActor.Address())
		j.jobsByID[msg.JobID] = msg.JobActor

	case UnregisterJob:
		delete(j.jobsByID, msg.JobID)

	case *apiv1.GetJobsRequest:
		jobs, err := j.parseV1JobResposnes(j.askJobActors(ctx, msg))
		if err != nil {
			return err
		}
		// ask for a consistent snapshot of the job queue from the RM
		jobsInRM := make([]*jobv1.Job, 0)
		jobQ := ctx.Ask(j.RMRef, GetJobQ{ResourcePool: msg.ResourcePool}).Get().(AQueue)
		for jID, jRMInfo := range jobQ {
			v1Job, ok := jobs[jID]
			if ok {
				UpdateJobQInfo(v1Job, jRMInfo)
				jobsInRM = append(jobsInRM, v1Job)
			}
		}
		ctx.Respond(jobsInRM)

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
