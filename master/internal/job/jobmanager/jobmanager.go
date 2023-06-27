package jobmanager

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

// JobManager is the interface for a `job.Manager` defined in our parent
// package.
type JobManager interface {
	RegisterJob(model.JobID, *actor.Ref)
	UnregisterJob(model.JobID)
}

// DefaultJobManager prevents an import cycle between jobs and commands.
// Commands import jobs to register themselves. Jobs import commands to
// do authz checks. Eventually, commands probably shouldn't import jobs
// to register themselves since they are lower in the class hierarchy
// that is slowly forming.
var DefaultJobManager JobManager
