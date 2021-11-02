package job

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/model"
)

// TODO move sproto/job.go here

var JobsActorAddr = actor.Addr("jobs")

type Job struct {
	JobType        model.JobType
	Id             model.JobID
	SubmissionTime time.Time
	User           *model.User // username?
	IsPreemptible  bool
	SubEntityRef   *actor.Ref // TODO rename
	RPRef          *actor.Ref
}

// TODO register this job with the jobs group
func RegisterJob(system *actor.System, jobId model.JobID, aActor actor.Actor) {
	system.TellAt(JobsActorAddr, actors.NewChild{
		ID:    string(jobId),
		Actor: aActor,
	})
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
