package job

import (
	"errors"
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// TODO move sproto/job.go here

var JobsActorAddr = actor.Addr("jobs")

type RMJobInfo struct {
	JobsAhead      int
	State          sproto.SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

type Job struct {
	JobType        model.JobType
	Id             model.JobID
	SubmissionTime time.Time
	User           model.User // username?
	IsPreemptible  bool
	SubEntityRef   *actor.Ref // TODO rename
	RPRef          *actor.Ref
	RMInfo         RMJobInfo
}

// // GetJobOrder requests a list of *jobv1.Job.
// // Expected response: []*jobv1.Job.
// type GetJobOrder struct{}

// // GetJobSummary requests a JobSummary.
// // Expected response: jobv1.JobSummary.
// type GetJobSummary struct { // CHECK should these use the same type as response instead of a new msg
// 	JobID model.JobID
// }

// // GetJobQStats requests stats for a queue.
// // Expected response: jobv1.QueueStats.
// type GetJobQStats struct{}

// TODO register this job with the jobs group
func RegisterJob(system *actor.System, jobId model.JobID, aActor actor.Actor) (*actor.Ref, bool) {
	return system.ActorOf(JobsActorAddr.Child(jobId.String()), aActor)
	// system.TellAt(JobsActorAddr, actors.NewChild{
	// 	ID:    string(jobId),
	// 	Actor: aActor,
	// })
}

type Jobs struct{}

func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetJobsRequest:
		jobs := make([]*jobv1.Job, 0)
		for _, job := range ctx.AskAll(msg, ctx.Children()...).GetAll() {
			typed, ok := job.(*jobv1.Job)
			if !ok {
				return errors.New("unexpected response type")
			}
			if typed != nil {
				jobs = append(jobs, typed)
			}
		}
		// TODO do pagination here as well?
		ctx.Respond(jobs)

	// case GetJobOrder:
	// 	ctx.Respond(getV1Jobs(rp))
	// case GetJobSummary:
	// 	// for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
	// 	// 	if typed := tensorboard.(*tensorboardv1.Tensorboard); len(users) == 0 || users[typed.Username] {
	// 	// 		resp.Tensorboards = append(resp.Tensorboards, typed)
	// 	// 	}
	// 	// }
	// 	// ctx.Self().System().AskAll()
	// 	// ctx.ActorOf(job.JobsActorAddr)
	// 	ctx.Respond(resp)
	// case GetJobQStats:
	// 	ctx.Respond(*jobStats(rp))

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// func NewJob(jobType model.JobType, id model.JobID, user *model.User, isPreemptible bool, subEntityRef *actor.Ref, rpRef *actor.Ref) *Job {
// 	job := Job{
// 		JobType:        jobType,
// 		Id:             id,
// 		SubmissionTime: time.Now(),
// 		User:           user,
// 		IsPreemptible:  isPreemptible,
// 		SubEntityRef:   subEntityRef,
// 		RPRef:          rpRef,
// 	}

// 	registerJob(subEntityRef.System(), &job)
// 	return &job
// }

// TODO a jobs actor

// TODO helper for ENTbCS to register as children to this actor
// func (j *Job) RegisterChild(ctx *actor.Context, child *actor.Ref) {
// }

// setup receive method to implement actor interface
// func (j *Job) Receive(ctx *actor.Context) error {
// 	switch msg := ctx.Message().(type) {
// 	default:
// 		return actor.ErrUnhandledMessage(msg)
// 	}
// 	return nil
// }
