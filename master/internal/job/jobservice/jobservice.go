package jobservice

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// Job is the interface for commands/experiments types that implement
// the job service.
type Job interface {
	ToV1Job() (*jobv1.Job, error)
	SetJobPriority(priority int) error
	SetWeight(weight float64) error
	SetResourcePool(resourcePool string) error
	ResourcePool() string
}

// Service manages the job service.
type Service struct {
	mu      sync.Mutex
	rm      rm.ResourceManager
	jobByID map[model.JobID]Job
	syslog  *logrus.Entry
}

// DefaultService is the global singleton job service.
var DefaultService *Service

// SetDefaultService sets the package-level Default in
// this package and `jobmanager`.
func SetDefaultService(rm rm.ResourceManager) {
	if DefaultService != nil {
		logrus.Warn(
			"detected re-initialization of Job that should never occur outside of tests",
		)
	}
	DefaultService = &Service{
		rm:      rm,
		jobByID: make(map[model.JobID]Job),
		syslog:  logrus.WithField("component", "jobs"),
	}
}

// RegisterJob takes an experiment/command (of interface type Service)
// and registers it with the job manager's jobByID map.
func (s *Service) RegisterJob(jobID model.JobID, j Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobByID[jobID] = j
}

// UnregisterJob deletes a job from the jobByID map.
func (s *Service) UnregisterJob(jobID model.JobID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.jobByID, jobID)
}

func (s *Service) jobQRefs(jobQ map[model.JobID]*sproto.RMJobInfo) (map[model.JobID]*jobv1.Job, error) {
	jobRefs := map[model.JobID]*jobv1.Job{}
	for jID := range jobQ {
		jobRef, ok := s.jobByID[jID]
		if ok {
			ref, err := jobRef.ToV1Job()
			if err != nil {
				// FIXME: DET-9563 workspace and/or project is deleted.
				if errors.Is(err, sql.ErrNoRows) {
					continue
				}
				return nil, err
			}
			jobRefs[jID] = ref
		}
	}
	return jobRefs, nil
}

// GetJobs returns a list of jobs for a resource pool.
func (s *Service) GetJobs(
	resourcePool rm.ResourcePoolName,
	desc bool,
	states []jobv1.State,
) ([]*jobv1.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobQ, err := s.rm.GetJobQ(resourcePool)
	if err != nil {
		s.syslog.WithError(err).Error("getting job queue info from RM")
		return nil, err
	}
	jobs, err := s.jobQRefs(jobQ)
	if err != nil {
		return nil, err
	}

	// Try to fetch External jobs, if supported by the Resource Manager (RM).
	// If the GetExternalJobs call is supported, RM returns a list of external jobs or
	// an error if there is any problem. Otherwise, RM returns rmerrors.ErrNotSupported
	// error. In this case, continue without the External jobs.
	externalJobs, err := s.rm.GetExternalJobs(resourcePool)
	if err != nil {
		// If the error is not 'ErrNotSupported' error, propagate the error upwards.
		if err != rmerrors.ErrNotSupported {
			return nil, err
		}
	}

	// Merge the results.
	jobsInRM := make([]*jobv1.Job, 0, len(jobQ)+len(externalJobs))
	for jID, jRMInfo := range jobQ {
		if v1Job, ok := jobs[jID]; ok {
			// interesting that the update is a side effect
			// of the getJobs function. I am guessing that
			// I should leave it, regardless of filters?
			updateJobQInfo(v1Job, jRMInfo)

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

	// Append any External jobs to the bottom of the list.
	jobsInRM = append(jobsInRM, externalJobs...)

	return jobsInRM, nil
}

// GetJobSummary returns a summary of the job given an id and resource pool.
func (s *Service) GetJobSummary(id model.JobID, resourcePool rm.ResourcePoolName) (*jobv1.JobSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobQ, err := s.rm.GetJobQ(resourcePool)
	if err != nil {
		s.syslog.WithError(err).Error("getting job queue info from RM")
		return nil, err
	}
	jobInfo, ok := jobQ[id]
	if !ok || jobInfo == nil {
		// job is not active.
		return nil, sproto.ErrJobNotFound(id)
	}
	return &jobv1.JobSummary{
		State:     jobInfo.State.Proto(),
		JobsAhead: int32(jobInfo.JobsAhead),
	}, nil
}

func (s *Service) applyUpdate(update *jobv1.QueueControl) error {
	jobID := model.JobID(update.JobId)
	j := s.jobByID[jobID]
	if j == nil {
		return sproto.ErrJobNotFound(jobID)
	}

	switch action := update.GetAction().(type) {
	case *jobv1.QueueControl_Priority:
		priority := int(action.Priority)
		return j.SetJobPriority(priority)
	case *jobv1.QueueControl_Weight:
		if action.Weight <= 0 {
			s.syslog.Error("weight must be greater than 0")
		}
		err := j.SetWeight(float64(action.Weight))
		if err != nil {
			s.syslog.WithError(err).Info("setting command job weight")
			return err
		}
	case *jobv1.QueueControl_ResourcePool:
		if action.ResourcePool == "" {
			s.syslog.Error("resource pool must be set")
		}
		return j.SetResourcePool(action.ResourcePool)
	default:
		return fmt.Errorf("unexpected action: %v", action)
	}
	return nil
}

// UpdateJobQueue sends queue control updates to specific jobs.
func (s *Service) UpdateJobQueue(updates []*jobv1.QueueControl) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	errs := make([]string, 0)

	for _, update := range updates {
		if err := s.applyUpdate(update); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) == 1 {
		return fmt.Errorf(errs[0])
	} else if len(errs) > 1 {
		return fmt.Errorf("encountered the following errors: %s", strings.Join(errs, ", "))
	}

	return nil
}

// updateJobQInfo updates the job with the RMJobInfo.
func updateJobQInfo(job *jobv1.Job, rmInfo *sproto.RMJobInfo) {
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
