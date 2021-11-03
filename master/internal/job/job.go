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

// helper to get all the childrens of job managers addresse into a list
func getJobRefs(system *actor.System) []*actor.Ref {
	jobRefs := make([]*actor.Ref, 0)
	for _, addr := range jobManagers {
		jobRefs = append(jobRefs, system.Get(addr).Children()...)
	}
	return jobRefs
}

// RMJobInfo packs information available only to the RM that updates frequently.
type RMJobInfo struct {
	JobsAhead      int
	State          sproto.SchedulingState
	RequestedSlots int
	AllocatedSlots int
}

type Job struct {
	JobType model.JobType
	Id      model.JobID // TODO is already merged in to the job actors
	// SubmissionTime time.Time
	// User           model.User // username?
	// IsPreemptible  bool
	// RPRef          *actor.Ref
	RMInfo RMJobInfo
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

// Jobs manage jobs.
type Jobs struct{}

func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetJobsRequest:
		fmt.Printf("GetJobsRequest %v \n", *msg)
		children := getJobRefs(ctx.Self().System())
		fmt.Printf("children count %d \n", len(children))
		jobs := make([]*jobv1.Job, 0)
		for _, job := range ctx.AskAll(msg, children...).GetAll() {
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
