package job // jobqueue?

import (
	"errors"
	"fmt"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

var JobsActorAddr = actor.Addr("jobs")

// TODO these could be set up as jobs children.
var jobManagers = [...]actor.Address{
	actor.Addr("experiments"),
	actor.Addr("tensorboard"), // should be tensorboards
	actor.Addr("commands"),
	actor.Addr("shells"),
	actor.Addr("notebooks"),
}

// TODO attach jobs to jobs actor for direct access via id? would need alias address support form the actor system
// helper to get all the childrens of job managers addresse into a list
func getJobRefs(system *actor.System) []*actor.Ref {
	jobRefs := make([]*actor.Ref, 0)
	for _, addr := range jobManagers {
		jobRefs = append(jobRefs, system.Get(addr).Children()...)
	}
	return jobRefs
}

func JobActorAddr(jobType model.JobType, entityId string) actor.Address {
	parentAddress := ""
	switch jobType {
	case model.JobTypeExperiment:
		parentAddress = "experiments"
	case model.JobTypeTensorboard:
		parentAddress = "tensorboard"
	case model.JobTypeCommand:
		parentAddress = "commands"
	case model.JobTypeNotebook:
		parentAddress = "notebooks"
	case model.JobTypeShell:
		parentAddress = "shells"
	}
	return actor.Addr(parentAddress).Child(entityId)
}

// RMJobInfo packs information available only to the RM that updates frequently.
type RMJobInfo struct { // rename ?
	JobsAhead      int
	State          SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

type GetJobSummary struct{}

// GetJobQInfo is used to get all job information in one go to avoid any inconsistencies.
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

type AQueue = map[model.JobID]*RMJobInfo

func (j *Jobs) askJobActors(ctx *actor.Context, msg actor.Message) map[*actor.Ref]actor.Message {
	children := getJobRefs(ctx.Self().System())
	fmt.Printf("children count %d \n", len(children))
	// jobs := make([]*jobv1.Job, 0)
	return ctx.AskAll(msg, children...).GetAll()
	// IMPROVE. look up reflect
}

func (j *Jobs) parseV1JobResposnes(responses map[*actor.Ref]actor.Message) ([]*jobv1.Job, error) {
	jobs := make([]*jobv1.Job, 0)
	for _, val := range responses {
		typed, ok := val.(*jobv1.Job)
		if !ok {
			return nil, errors.New("unexpected response type")
		}
		if typed != nil {
			jobs = append(jobs, typed)
		}
	}
	return jobs, nil
}

func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetJobsRequest:
		fmt.Printf("GetJobsRequest %v \n", *msg)

		jobs, err := j.parseV1JobResposnes(j.askJobActors(ctx, msg))
		if err != nil {
			return err
		}
		// ask for a consistent snapshot of the job queue
		jobQ := ctx.Ask(j.RMRef, GetJobQ{ResourcePool: msg.ResourcePool}).Get().(AQueue)
		for _, j := range jobs {
			rmInfo, ok := jobQ[model.JobID(j.JobId)]
			if ok {
				UpdateJobQInfo(j, rmInfo)
			}
		}
		ctx.Respond(jobs)

	// case SetJobQ:
	// 	j.Queues[msg.Identifier] = msg.Queue

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func UpdateJobQInfo(job *jobv1.Job, rmInfo *RMJobInfo) {
	if job == nil {
		panic("nil job ptr")
	}
	if rmInfo == nil {
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
