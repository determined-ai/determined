package job

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

var (
	// JobsActorAddr is the address of the jobs actor.
	JobsActorAddr = actor.Addr("jobs")
	// HeadAnchor is an internal anchor for the head of the job queue.
	HeadAnchor = model.JobID("INTERNAL-head")
	// TailAnchor is an internal anchor for the tail of the job queue.
	TailAnchor = model.JobID("INTERNAL-tail")
)

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
		ResourcePool string // TODO are we using this?
		Handler      *actor.Ref
	}
	// SetGroupPriority sets the priority of the group in the priority scheduler.
	SetGroupPriority struct {
		Priority     int
		ResourcePool string // TODO are we using this?
		Handler      *actor.Ref
	}
	// SetResourcePool switches the resource pool that the job belongs to.
	SetResourcePool struct {
		ResourcePool string
		Handler      *actor.Ref
	}
	// MoveJob requests the job to be moved within a priority queue relative to another job.
	MoveJob struct {
		ID      model.JobID
		Anchor1 model.JobID
		Anchor2 model.JobID
	}
)

// RegisterJobPosition gets sent from the resource pool to jobs.
// It notifies the job of its new position.
type RegisterJobPosition struct {
	JobID       model.JobID
	JobPosition string
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
	RMRef     *actor.Ref
	actorByID map[model.JobID]*actor.Ref
}

// AQueue is a map of jobID to RMJobInfo.
type AQueue = map[model.JobID]*RMJobInfo

func errJobNotFound(jobID model.JobID) error {
	return fmt.Errorf("job %s not found", jobID)
}

// NewJobs creates a new jobs actor.
func NewJobs(rmRef *actor.Ref) *Jobs {
	return &Jobs{
		RMRef:     rmRef,
		actorByID: make(map[model.JobID]*actor.Ref),
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

// jobQSnapshot asks for a fresh consistent snapshot of the job queue from the RM.
func (j *Jobs) jobQSnapshot(ctx *actor.Context, resourcePool string) (AQueue, error) {
	aResp := ctx.Ask(j.RMRef, GetJobQ{ResourcePool: resourcePool})
	if err := aResp.Error(); err != nil {
		ctx.Log().WithError(err).Error("getting job queue info from RM")
		return nil, err
	}

	jobQ, ok := aResp.Get().(AQueue)
	if !ok {
		err := fmt.Errorf("unexpected response type: %T from RM", aResp.Get())
		ctx.Log().WithError(err).Error("")
		return nil, err
	}
	return jobQ, nil
}

// figure out what changes are needed to move the job to the desired position.
// Later we can support cross RP moving this way as well.
func moveJobMessages(
	jobs []*jobv1.Job,
	target *jobv1.Job,
	anchor *jobv1.Job,
	anchorIdx int,
	aheadOf bool,
) (*SetGroupPriority, *MoveJob, error) {
	// validate anchorIdx
	if anchorIdx < 0 || anchorIdx > len(jobs) {
		return nil, nil, fmt.Errorf("invalid anchor index %d", anchorIdx)
	}
	// validate anchor and target
	if anchor == nil {
		return nil, nil, fmt.Errorf("missing anchor job")
	}
	if target == nil {
		return nil, nil, fmt.Errorf("missing target job")
	}
	if target.JobId == anchor.JobId {
		return nil, nil, fmt.Errorf("target and anchor jobs are the same")
	}

	// sanity check
	lastJob := int32(-1)
	for _, job := range jobs {
		check.Panic(check.GreaterThanOrEqualTo(job.Summary.JobsAhead, lastJob))
		lastJob = job.Summary.JobsAhead
	}

	var priorityMsg *SetGroupPriority
	if target.Priority != anchor.Priority {
		priorityMsg = &SetGroupPriority{
			Priority: int(anchor.Priority),
		}
	}

	var moveMsg *MoveJob
	// find the next or previous job based on aheadOf in the same priority lane
	var secondAnchor model.JobID
	if aheadOf {
		for idx := anchorIdx - 1; idx >= 0; idx-- {
			if jobs[idx].Priority == anchor.Priority {
				secondAnchor = model.JobID(jobs[idx].JobId)
				break
			}
		}
		if secondAnchor == model.JobID("") {
			secondAnchor = HeadAnchor
		}
	} else {
		for idx := anchorIdx + 1; idx < len(jobs); idx++ {
			if jobs[idx].Priority == anchor.Priority {
				secondAnchor = model.JobID(jobs[idx].JobId)
				break
			}
		}
		if secondAnchor == model.JobID("") {
			secondAnchor = TailAnchor
		}
	}

	check.Panic(check.True(secondAnchor != model.JobID("")))
	if secondAnchor.String() != target.JobId {
		moveMsg = &MoveJob{
			ID:      model.JobID(target.JobId),
			Anchor1: model.JobID(anchor.JobId),
			Anchor2: secondAnchor,
		}
	}

	return priorityMsg, moveMsg, nil
}

func (j *Jobs) moveJob(
	ctx *actor.Context, jobID model.JobID, anchorID model.JobID, aheadOf bool,
) error {
	if anchorID == jobID {
		return nil
	}
	// find the job resource pool
	aResp := ctx.Ask(j.actorByID[jobID], GetJob{})
	if err := aResp.Error(); err != nil {
		return err
	}
	targetJob, ok := aResp.Get().(*jobv1.Job)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", aResp.Get())
	}
	jobs, err := j.getJobs(ctx, targetJob.ResourcePool, false)
	if err != nil {
		return err
	}
	// WARN assuming all job rp and priority changes goes through jobsActor
	// and thus is synchoronzed here.

	// find anchorJob by matching ID
	var anchorJob *jobv1.Job
	anchorIdx := -1
	for idx, job := range jobs {
		if job.JobId == anchorID.String() {
			anchorJob = job
			anchorIdx = idx
			break
		}
	}
	if anchorJob == nil || anchorIdx == -1 {
		return errJobNotFound(anchorID)
	}

	// we might wanna limit the scope of this to just generating the moveJob message.
	prioChange, moveJob, err := moveJobMessages(
		jobs,
		targetJob,
		anchorJob,
		anchorIdx,
		aheadOf,
	)

	if prioChange != nil {
		err = j.setJobPriority(ctx, jobID, prioChange.Priority)
		if err != nil {
			return err
		}

		// FIXME after this priority change we could be in the situation
		// where the job is placed in the correct position as is. Right now
		// the RM checks and handles this case as a no op.
	}

	if moveJob != nil {
		resp := ctx.Ask(j.RMRef, *moveJob)
		err = resp.Error()
	}

	return err
}

func (j *Jobs) getJobs(ctx *actor.Context, resourcePool string, desc bool) ([]*jobv1.Job, error) {
	jobQ, err := j.jobQSnapshot(ctx, resourcePool)
	if err != nil {
		return nil, err
	}

	// Get jobs from the job actors.
	jobRefs := make([]*actor.Ref, 0)
	for jID := range jobQ {
		jobRef, ok := j.actorByID[jID]
		if ok {
			jobRefs = append(jobRefs, jobRef)
		}
	}
	jobs, err := j.parseV1JobMsgs(ctx.AskAll(GetJob{}, jobRefs...).GetAll())
	if err != nil {
		ctx.Log().WithError(err).Error("parsing responses from job actors")
		return nil, err
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
	resp := ctx.Ask(jobActor, SetGroupPriority{
		Priority: priority,
	})
	return resp.Error()
}

// Receive implements the actor.Actor interface.
func (j *Jobs) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case RegisterJob:
		j.actorByID[msg.JobID] = msg.JobActor

	case UnregisterJob:
		delete(j.actorByID, msg.JobID)

	case *apiv1.GetJobsRequest:
		jobs, err := j.getJobs(ctx, msg.ResourcePool, msg.OrderBy == apiv1.OrderBy_ORDER_BY_DESC)
		if err != nil {
			ctx.Respond(err)
		}
		ctx.Respond(jobs)

	case *apiv1.UpdateJobQueueRequest:
		errors := make([]string, 0)
		for _, update := range msg.Updates {
			jobID := model.JobID(update.JobId)
			jobActor := j.actorByID[jobID]
			if jobActor == nil {
				ctx.Respond(errJobNotFound(jobID))
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
				resp := ctx.Ask(jobActor, SetGroupWeight{
					Weight: float64(action.Weight),
				})
				if err := resp.Error(); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_ResourcePool:
				if action.ResourcePool == "" {
					errors = append(errors, "resource pool must be set")
				}
				// TODO tell whoever keeping track of the qposition for this job
				// to forget it. (depends..)
				resp := ctx.Ask(jobActor, SetResourcePool{
					ResourcePool: action.ResourcePool,
				})
				if err := resp.Error(); err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_AheadOf:
				err := j.moveJob(ctx, jobID, model.JobID(action.AheadOf), true)
				if err != nil {
					errors = append(errors, err.Error())
				}
			case *jobv1.QueueControl_BehindOf:
				err := j.moveJob(ctx, jobID, model.JobID(action.BehindOf), false)
				if err != nil {
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
