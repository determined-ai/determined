package scheduler

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// Assignment is an interface that provides function for task actors
// to start tasks on assigned resources.
type Assignment interface {
	Summary() ContainerSummary
	StartContainer(ctx *actor.Context, spec image.TaskSpec)
	KillContainer(ctx *actor.Context)
}
