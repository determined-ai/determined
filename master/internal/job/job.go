package job

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// Jobs manage jobs.
type Jobs struct {
	rm        rm.ResourceManager
	actorByID map[model.JobID]*actor.Ref
}

// NewJobs creates a new jobs actor.
func NewJobs(rm rm.ResourceManager) *Jobs {
	return &Jobs{
		rm:        rm,
		actorByID: make(map[model.JobID]*actor.Ref),
	}
}

func (j *Jobs) parseV1JobMsgs(
	msgs map[*actor.Ref]actor.Message,
) (map[model.JobID]*jobv1.Job, error) {
	jobs := make(map[model.JobID]*jobv1.Job, len(msgs))
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

// jobQSnapshot asks for a fresh consistent snapshot of the job queue from the RM.
func (j *Jobs) jobQSnapshot(ctx *actor.Context, resourcePool string) (sproto.AQueue, error) {
	resp, err := j.rm.GetJobQ(ctx, sproto.GetJobQ{ResourcePool: resourcePool})
	if err != nil {
		ctx.Log().WithError(err).Error("getting job queue info from RM")
		return nil, err
	}

	return resp, nil
}

func (j *Jobs) getJobs(
	ctx *actor.Context,
	resourcePool string,
	desc bool,
	states []jobv1.State,
) ([]*jobv1.Job, error) {
	jobQ, err := j.jobQSnapshot(ctx, resourcePool)
	if err != nil {
		return nil, err
	}

	// Get jobs from the job actors.
	jobRefs := make([]*actor.Ref, 0, len(jobQ))
	for jID := range jobQ {
		jobRef, ok := j.actorByID[jID]
		if ok {
			jobRefs = append(jobRefs, jobRef)
		}
	}
	jobs, err := j.parseV1JobMsgs(ctx.AskAll(sproto.GetJob{}, jobRefs...).GetAll())
	if err != nil {
		ctx.Log().WithError(err).Error("parsing responses from job actors")
		return nil, err
	}

	// Merge the results.
	jobsInRM := make([]*jobv1.Job, 0, len(jobQ))
	for jID, jRMInfo := range jobQ {
		v1Job, ok := jobs[jID]
		if ok {
			// interesting that the update is a side effect
			// of the getJobs function. I am guessing that
			// I should leave it, regardless of filters?
			UpdateJobQInfo(v1Job, jRMInfo)

			if states == nil || slices.Contains(states, v1Job.Summary.State) {
				jobsInRM = append(jobsInRM, v1Job)
			}
		}
	}

	// order by jobsAhead first and JobId second.
	sort.SliceStable(jobsInRM, func(i, j int) bool {
		if desc {
			i, j = j, i
		}
		if jobsInRM[i].Summary == nil || jobsInRM[j].Summary == nil {
			return false
		}
		if jobsInRM[i].Summary.JobsAhead < jobsInRM[j].Summary.JobsAhead {
			return true
		}
		if jobsInRM[i].Summary.JobsAhead > jobsInRM[j].Summary.JobsAhead {
			return false
		}
		return jobsInRM[i].JobId < jobsInRM[j].JobId
	})

	return jobsInRM, nil
}

func (j *Jobs) setJobPriority(ctx *actor.Context, jobID model.JobID, priority int) error {
	if priority < 1 || priority > 99 {
		return errors.New("priority must be between 1 and 99")
	}
	jobActor := j.actorByID[jobID]
	resp := ctx.Ask(jobActor, sproto.SetGroupPriority{
		Priority: priority,
	})
	return resp.Error()
}

// Receive implements the actor.Actor interface.
func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case sproto.RegisterJob:
		j.actorByID[msg.JobID] = msg.JobActor

	case sproto.UnregisterJob:
		delete(j.actorByID, msg.JobID)

	case *apiv1.GetJobsRequest:
		jobs, err := j.getJobs(
			ctx,
			msg.ResourcePool,
			msg.OrderBy == apiv1.OrderBy_ORDER_BY_DESC,
			msg.States)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		ctx.Respond(jobs)

	case sproto.GetJobSummary:
		jobs, err := j.jobQSnapshot(ctx, msg.ResourcePool)
		if err != nil {
			ctx.Respond(err)
			return nil
		}
		jobInfo, ok := jobs[msg.JobID]
		if !ok || jobInfo == nil {
			// job is not active.
			ctx.Respond(sproto.ErrJobNotFound(msg.JobID))
			return nil
		}
		summary := jobv1.JobSummary{
			State:     jobInfo.State.Proto(),
			JobsAhead: int32(jobInfo.JobsAhead),
		}
		ctx.Respond(&summary)

	case *apiv1.UpdateJobQueueRequest:
		errors := make([]string, 0)
		for _, update := range msg.Updates {
			jobID := model.JobID(update.JobId)
			jobActor := j.actorByID[jobID]
			if jobActor == nil {
				ctx.Respond(sproto.ErrJobNotFound(jobID))
				return nil
			}
			switch action := update.GetAction().(type) {
			case *jobv1.QueueControl_Priority:
				priority := int(action.Priority)
				if err := j.setJobPriority(ctx, jobID, priority); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_Weight:
				if action.Weight <= 0 {
					errors = append(errors, "weight must be greater than 0")
					continue
				}
				resp := ctx.Ask(jobActor, sproto.SetGroupWeight{
					Weight: float64(action.Weight),
				})
				if err := resp.Error(); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_ResourcePool:
				if action.ResourcePool == "" {
					errors = append(errors, "resource pool must be set")
					continue
				}
				resp := ctx.Ask(jobActor, sproto.SetResourcePool{
					ResourcePool: action.ResourcePool,
				})
				if err := resp.Error(); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_AheadOf:
				if err := j.rm.MoveJob(ctx, sproto.MoveJob{
					ID:     jobID,
					Anchor: model.JobID(action.AheadOf),
					Ahead:  true,
				}); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_BehindOf:
				if err := j.rm.MoveJob(ctx, sproto.MoveJob{
					ID:     jobID,
					Anchor: model.JobID(action.BehindOf),
					Ahead:  false,
				}); err != nil {
					errors = append(errors, err.Error())
				}
			default:
				ctx.Respond(fmt.Errorf("unexpected action: %v", action))
				return nil
			}
		}
		if len(errors) == 1 {
			ctx.Respond(fmt.Errorf(errors[0]))
		} else if len(errors) > 1 {
			ctx.Respond(fmt.Errorf("encountered the following errors: %s", strings.Join(errors, ", ")))
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// UpdateJobQInfo updates the job with the RMJobInfo.
func UpdateJobQInfo(job *jobv1.Job, rmInfo *sproto.RMJobInfo) {
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
