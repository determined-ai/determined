package job

import (
	"errors"
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// TODO move sproto/job.go here

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
type RMJobInfo struct {
	JobsAhead      int
	State          sproto.SchedulingState
	RequestedSlots int
	AllocatedSlots int
	IsPreemptible  bool
	// should preemptible status come from RM? internal/experiments
	// order: wherever we save it, job config, rm config,
}

// type Job struct { // probably not needed? can we merged into ENTbCS but could be used to unify the two
// 	JobType model.JobType
// 	Id      model.JobID // TODO is already merged in to the job actors
// 	// SubmissionTime time.Time
// 	// User           model.User // username?
// 	// IsPreemptible  bool
// 	// RPRef          *actor.Ref
// 	RMInfo RMJobInfo
// }

type GetJobSummary struct {
}

// Jobs manage jobs.
type Jobs struct {
	RMRef *actor.Ref
}

func (j *Jobs) askJobActors(ctx *actor.Context, msg actor.Message) map[*actor.Ref]actor.Message {
	children := getJobRefs(ctx.Self().System())
	fmt.Printf("children count %d \n", len(children))
	// jobs := make([]*jobv1.Job, 0)
	return ctx.AskAll(msg, children...).GetAll()

	// IMPROVE. look up reflect
	// for _, val := range ctx.AskAll(msg, children...).GetAll() {
	// 	rType := reflect.TypeOf(responses).Elem()
	// 	typed, ok := val.(rType)
	// 	if !ok {
	// 		return errors.New("unexpected response type")
	// 	}
	// 	if typed != nil {
	// 		responses = append(responses, typed)
	// 	}

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

func (j *Jobs) getV1Jobs(ctx *actor.Context, msg actor.Message) ([]*jobv1.Job, error) {
	return j.parseV1JobResposnes(j.askJobActors(ctx, msg))
}

func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetJobsRequest:
		fmt.Printf("GetJobsRequest %v \n", *msg)

		jobs, err := j.getV1Jobs(ctx, msg)
		if err != nil {
			return err
		}
		// TODO do pagination here as well?
		ctx.Respond(jobs)

	// case *apiv1.GetJobQueueStatsRequest:
	// 	jobs, err := j.getV1Jobs(ctx, msg) // TODO specialize to returning just stats.
	// 	if err != nil {
	// 		return err
	// 	}
	// 	ctx.Respond(QueueStatsFromJobs(jobs))
	// TODO sync with RMInfo from RM

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func FillInRmJobInfo(job *jobv1.Job, rmInfo *RMJobInfo) {
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
