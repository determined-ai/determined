package job // jobqueue?

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

// TODO these could be set up as jobs children.
var jobManagers = [...]actor.Address{
	actor.Addr("experiments"),
	actor.Addr("tensorboard"), // should be tensorboards
	actor.Addr("commands"),
	actor.Addr("shells"),
	actor.Addr("notebooks"),
}

// TODO attach jobs to jobs actor for direct access via id?
// would need alias address support form the actor system
// helper to get all the childrens of job managers addresse into a list.
func getJobRefs(system *actor.System) []*actor.Ref {
	jobRefs := make([]*actor.Ref, 0)
	for _, addr := range jobManagers {
		jobRefs = append(jobRefs, system.Get(addr).Children()...)
	}
	return jobRefs
}

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
	ResourcePool string
}

// SetJobOrder conveys a job queue change for a specific jobID to the resource pool.
type SetJobOrder struct {
	ResourcePool string
	QPosition    float64
	Weight       float64
	Priority     *int
	JobID        model.JobID
}

// type SetJobQ struct {
// 	Identifier string
// 	Queue      map[model.JobID]*RMJobInfo
// }

// Jobs manage jobs.
type Jobs struct {
	RMRef *actor.Ref
	// Queues map[string]map[model.JobID]*RMJobInfo
}

// AQueue is a map of jobID to RMJobInfo.
type AQueue = map[model.JobID]*RMJobInfo

func (j *Jobs) askJobActors(ctx *actor.Context, msg actor.Message) map[*actor.Ref]actor.Message {
	children := getJobRefs(ctx.Self().System())
	fmt.Printf("children count %d \n", len(children))
	// jobs := make([]*jobv1.Job, 0)
	return ctx.AskAll(msg, children...).GetAll()
	// IMPROVE. look up reflect
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

	// case SetJobQ:
	// 	j.Queues[msg.Identifier] = msg.Queue

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
