package scheduler

import (
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	sproto "github.com/determined-ai/determined/master/pkg/scheduler"
)

// ResourceProvider manages the configured resource providers.
// Currently support only one resource provider at a time.
type ResourceProvider struct {
	clusterID                   string
	scheduler                   Scheduler
	fittingMethod               SoftConstraint
	proxy                       *actor.Ref
	harnessPath                 string
	taskContainerDefaults       model.TaskContainerDefaultsConfig
	provisioner                 *actor.Ref
	provisionerSlotsPerInstance int

	resourceProvider *actor.Ref
}

// NewResourceProvider creates and instance of ResourceProvider.
func NewResourceProvider(
	clusterID string,
	scheduler Scheduler,
	fittingMethod SoftConstraint,
	proxy *actor.Ref,
	harnessPath string,
	taskContainerDefaults model.TaskContainerDefaultsConfig,
	provisioner *actor.Ref,
	provisionerSlotsPerInstance int,
) *ResourceProvider {
	return &ResourceProvider{
		clusterID:                   clusterID,
		scheduler:                   scheduler,
		fittingMethod:               fittingMethod,
		proxy:                       proxy,
		harnessPath:                 harnessPath,
		taskContainerDefaults:       taskContainerDefaults,
		provisioner:                 provisioner,
		provisionerSlotsPerInstance: provisionerSlotsPerInstance,
	}
}

// Receive implements the actor.Actor interface.
func (rp *ResourceProvider) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		rp.resourceProvider, _ = ctx.ActorOf("defaultRP", NewDefaultRP(
			rp.clusterID,
			rp.scheduler,
			rp.fittingMethod,
			rp.proxy,
			rp.harnessPath,
			rp.taskContainerDefaults,
			rp.provisioner,
			rp.provisionerSlotsPerInstance,
		))

	case AddTask, StartTask, sproto.ContainerStateChanged, SetMaxSlots, SetWeight,
		SetTaskName, TerminateTask, GetTaskSummary, GetTaskSummaries:
		rp.forward(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

// ConfigureEndpoints initializes the agent endpoints.
func (rp *ResourceProvider) ConfigureEndpoints(s *actor.System, e *echo.Echo) {
	agent.Initialize(s, e, rp.resourceProvider)
}

func (rp *ResourceProvider) forward(ctx *actor.Context, msg actor.Message) {
	if ctx.ExpectingResponse() {
		response := ctx.Ask(rp.resourceProvider, msg)
		ctx.Respond(response.Get())
	} else {
		ctx.Tell(rp.resourceProvider, msg)
	}
}
