package resourcemanagers

import (
	"github.com/determined-ai/determined/master/pkg/actor"
	image "github.com/determined-ai/determined/master/pkg/tasks"
)

// Allocation is an interface that provides function for task actors
// to start tasks on assigned resources.
type Allocation interface {
	Summary() ContainerSummary
	Start(ctx *actor.Context, spec image.TaskSpec)
	Kill(ctx *actor.Context)
}
