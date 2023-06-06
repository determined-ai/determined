package job

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"golang.org/x/exp/slices"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// Manager manages jobs.
type Manager struct {
	mu        sync.Mutex
	rm        rm.ResourceManager
	actorByID map[model.JobID]*actor.Ref
	system    *actor.System
	syslog    *logrus.Entry
}

// NewManager creates a new jobs manager instance.
func NewManager(rm rm.ResourceManager, system *actor.System) *Manager {
	return &Manager{
		rm:        rm,
		actorByID: make(map[model.JobID]*actor.Ref),
		system:    system,
		syslog:    logrus.WithField("component", "jobs"),
	}
}

func (j *Manager) parseV1JobMsgs(
	msgs map[*actor.Ref]actor.Message,
) (map[model.JobID]*jobv1.Job, error) {
	jobs := make(map[model.JobID]*jobv1.Job)
	for _, val := range msgs {
		switch typed := val.(type) {
		case nil:
			continue
		case error:
			return nil, typed
		case *jobv1.Job:
			if typed != nil {
				jobs[model.JobID(typed.JobId)] = typed
			}
		default:
			return nil, fmt.Errorf("unexpected response type: %T", val)
		}
	}
	return jobs, nil
}

// jobQSnapshot asks for a fresh consistent snapshot of the job queue from the RM.
func (j *Manager) jobQSnapshot(resourcePool string) (sproto.AQueue, error) {
	resp, err := j.rm.GetJobQ(j.system, sproto.GetJobQ{ResourcePool: resourcePool})
	if err != nil {
		j.syslog.WithError(err).Error("getting job queue info from RM")
		return nil, err
	}

	return resp, nil
}

func (j *Manager) jobQRefs(jobQ map[model.JobID]*sproto.RMJobInfo) []*actor.Ref {
	j.mu.Lock()
	defer j.mu.Unlock()
	// Get jobs from the job actors.
	jobRefs := make([]*actor.Ref, 0, len(jobQ))
	for jID := range jobQ {
		jobRef, ok := j.actorByID[jID]
		if ok {
			jobRefs = append(jobRefs, jobRef)
		}
	}

	return jobRefs
}

// GetJobs returns a list of jobs for a resource pool.
func (j *Manager) GetJobs(
	resourcePool string,
	desc bool,
	states []jobv1.State,
) ([]*jobv1.Job, error) {
	jobQ, err := j.jobQSnapshot(resourcePool)
	if err != nil {
		return nil, err
	}
	jobRefs := j.jobQRefs(jobQ)

	jobs, err := j.parseV1JobMsgs(j.system.AskAll(sproto.GetJob{}, jobRefs...).GetAll())
	if err != nil {
		j.syslog.WithError(err).Error("parsing responses from job actors")
		return nil, err
	}

	nonDaiJobs, _ := j.rm.GetNonDaiJobs(ctx, sproto.GetNonDaiJobs{
		ResourcePool: resourcePool,
	})

	// Merge the results.
	jobsInRM := make([]*jobv1.Job, 0, len(jobQ)+len(nonDaiJobs))
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

	jobsInRM = append(jobsInRM, nonDaiJobs...)

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

func (j *Manager) setJobPriority(ref *actor.Ref, priority int) error {
	if priority < 1 || priority > 99 {
		return errors.New("priority must be between 1 and 99")
	}
	resp := j.system.Ask(ref, sproto.SetGroupPriority{
		Priority: priority,
	})
	return resp.Error()
}

func (j *Manager) jobRef(id model.JobID) *actor.Ref {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.actorByID[id]
}

// RegisterJob registers a job actor with the job registry.
func (j *Manager) RegisterJob(id model.JobID, ref *actor.Ref) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.actorByID[id] = ref
}

// UnregisterJob removes the job from the job registry.
func (j *Manager) UnregisterJob(id model.JobID) {
	j.mu.Lock()
	defer j.mu.Unlock()

	delete(j.actorByID, id)
}

// GetJobSummary returns a summary of the job given an id and resource pool.
func (j *Manager) GetJobSummary(id model.JobID, resourcePool string) (*jobv1.JobSummary, error) {
	jobs, err := j.jobQSnapshot(resourcePool)
	if err != nil {
		return nil, err
	}
	jobInfo, ok := jobs[id]
	if !ok || jobInfo == nil {
		// job is not active.
		return nil, sproto.ErrJobNotFound(id)
	}
	return &jobv1.JobSummary{
		State:     jobInfo.State.Proto(),
		JobsAhead: int32(jobInfo.JobsAhead),
	}, nil
}

// UpdateJobQueue sends queue control updates to specific jobs.
func (j *Manager) UpdateJobQueue(updates []*jobv1.QueueControl) error {
	errors := make([]string, 0)
	for _, update := range updates {
		jobID := model.JobID(update.JobId)
		jobActor := j.jobRef(jobID)
		if jobActor == nil {
			return sproto.ErrJobNotFound(jobID)
		}
		switch action := update.GetAction().(type) {
		case *jobv1.QueueControl_Priority:
			priority := int(action.Priority)
			if err := j.setJobPriority(jobActor, priority); err != nil {
				errors = append(errors, err.Error())
			}
		case *jobv1.QueueControl_Weight:
			if action.Weight <= 0 {
				errors = append(errors, "weight must be greater than 0")
				continue
			}
			resp := j.system.Ask(jobActor, sproto.SetGroupWeight{
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
			resp := j.system.Ask(jobActor, sproto.SetResourcePool{
				ResourcePool: action.ResourcePool,
			})
			if err := resp.Error(); err != nil {
				errors = append(errors, err.Error())
			}
		case *jobv1.QueueControl_AheadOf:
			if err := j.rm.MoveJob(j.system, sproto.MoveJob{
				ID:     jobID,
				Anchor: model.JobID(action.AheadOf),
				Ahead:  true,
			}); err != nil {
				errors = append(errors, err.Error())
			}
		case *jobv1.QueueControl_BehindOf:
			if err := j.rm.MoveJob(j.system, sproto.MoveJob{
				ID:     jobID,
				Anchor: model.JobID(action.BehindOf),
				Ahead:  false,
			}); err != nil {
				errors = append(errors, err.Error())
			}
		default:
			return fmt.Errorf("unexpected action: %v", action)
		}
	}
	if len(errors) == 1 {
		return fmt.Errorf(errors[0])
	} else if len(errors) > 1 {
		return fmt.Errorf("encountered the following errors: %s", strings.Join(errors, ", "))
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
