package resourcemanagers

import (
	"time"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

const (
	// DefaultSchedulingPriority is the default resource manager priority.
	DefaultSchedulingPriority = 42

	actionCoolDown          = 500 * time.Millisecond
	defaultResourcePoolName = "default"
)

// schedulerTick periodically triggers the scheduler to act.
type schedulerTick struct{}

// ResourceManagers manages the configured resource providers.
// Currently support only one resource manager at a time.
type ResourceManagers struct {
	ref *actor.Ref
}

// Receive implements the actor.Actor interface.
func (rm *ResourceManagers) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case
		sproto.AllocateRequest, sproto.ResourcesReleased,
		sproto.SetGroupMaxSlots, job.SetGroupWeight,
		job.SetGroupPriority, job.RecoverJobPosition,
		job.MoveJob, sproto.GetTaskSummary,
		sproto.GetTaskSummaries, sproto.SetTaskName,
		sproto.GetTaskHandler:
		rm.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (rm *ResourceManagers) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rm.ref, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rm.ref, msg)
	}
}

// GetResourceManagerType returns the type of resourceManager being used.
func GetResourceManagerType(rmConfig *config.ResourceManagerConfig) string {
	switch {
	case rmConfig.AgentRM != nil:
		return "agentsRM"

	case rmConfig.KubernetesRM != nil:
		return "kubernetesRM"

	default:
		panic("no expected resource manager config is defined")
	}
}
