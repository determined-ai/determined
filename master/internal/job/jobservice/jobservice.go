package jobservice

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

// Service is the interface for a `job.Manager` defined in our parent
// package.
type Service interface {
	RegisterJob(model.JobID, *actor.Ref)
	UnregisterJob(model.JobID)

	GetJobs(resourcePool string, desc bool, states []jobv1.State) ([]*jobv1.Job, error)
	GetJobSummary(id model.JobID, resourcePool string) (*jobv1.JobSummary, error)
	UpdateJobQueue(updates []*jobv1.QueueControl) error
}

// Default prevents an import cycle between jobs and commands.
// Commands import jobs to register themselves. Jobs import commands to
// do authz checks. Eventually, commands probably shouldn't import jobs
// to register themselves since they are lower in the class hierarchy
// that is slowly forming.
var Default Service

// SetDefaultService sets the package-level Default in
// this package and `jobmanager`.
func SetDefaultService(m Service) {
	Default = m
}
