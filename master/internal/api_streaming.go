package internal

import (
	"fmt"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
	"time"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

func (a *apiServer) ResourcePoolStates(
	req *apiv1.ResourcePoolStatesRequest,
	resp apiv1.Determined_ResourcePoolStatesServer,
) error {



	return a.m.system.MustActorOf(
		actor.Addr("resourcepool-api-state-stream-"+uuid.New().String()),
		NewResourcePoolAPIStatesStreamActor(resp),
	).AwaitTermination()
}

const tickInterval = 1 * time.Second

type ResourcePoolAPIStatesStreamActor struct {
	resp         apiv1.Determined_ResourcePoolStatesServer
	tickInterval time.Duration
	lastStates   []*resourcepoolv1.ResourcePoolState
}

// NewLogStoreProcessor creates a new LogStoreProcessor.
func NewResourcePoolAPIStatesStreamActor(
	server apiv1.Determined_ResourcePoolStatesServer,
) *ResourcePoolAPIStatesStreamActor {
	return &ResourcePoolAPIStatesStreamActor{
		resp:         server,
		tickInterval: tickInterval,
		lastStates:   nil,
	}
}


// Launch an actor
// Receive TICK



// Receive implements the actor.Actor interface.
func (a *ResourcePoolAPIStatesStreamActor) Receive(ctx *actor.Context) error {
	type tick struct{}
	switch ctx.Message().(type) {
	case actor.PreStart:
		resp := ctx.Ask(sproto.GetCurrentRM(ctx.Self().System()), resourcemanagers.GetResourceSummaries{}).Get()
		summaries := resp.(resourcemanagers.ResourceSummaries).Summaries

		states := make([]*resourcepoolv1.ResourcePoolState, 0, len(summaries))
		for _, summary := range summaries {
			state := resourcepoolv1.ResourcePoolState{
				Name:                  summary.Name,
				NumAgents:              int32(summary.NumAgents),
				NumTotalSlots:          int32(summary.NumTotalSlots),
				NumActiveSlots:         int32(summary.NumActiveSlots),
				MaxCpuContainers:       int32(summary.MaxNumCpuContainers),
				NumActiveCpuContainers: int32(summary.NumActiveCpuContainers),
			}
			states = append(states, &state)
		}
		a.lastStates = states

		err := a.resp.Send(&apiv1.ResourcePoolStatesResponse{
			States: &resourcepoolv1.ResourcePoolStates{
				ResourcePools: states,
			},
		})
		if err != nil {
			return fmt.Errorf("failed while processing batch: %w", err)
		}

		actors.NotifyAfter(ctx, a.tickInterval, tick{})

	case tick:
		// TODO: Is checking for Err() the same as checking for Done()?
		if a.resp.Context().Err() != nil {
			// TODO: Add more informative logging here
			ctx.Self().Stop()
			return nil
		}

		// Check done channel
			// if done, shut down (return nil)


		// Get the full state from the resource pools
		resp := ctx.Ask(sproto.GetCurrentRM(ctx.Self().System()), resourcemanagers.GetResourceSummaries{}).Get()
		summaries := resp.(resourcemanagers.ResourceSummaries).Summaries

		currentStates := make([]*resourcepoolv1.ResourcePoolState, 0, len(summaries))
		for _, summary := range summaries {
			state := resourcepoolv1.ResourcePoolState{
				Name:                  summary.Name,
				NumAgents:              int32(summary.NumAgents),
				NumTotalSlots:          int32(summary.NumTotalSlots),
				NumActiveSlots:         int32(summary.NumActiveSlots),
				MaxCpuContainers:       int32(summary.MaxNumCpuContainers),
				NumActiveCpuContainers: int32(summary.NumActiveCpuContainers),
			}
			currentStates = append(currentStates, &state)
		}

		updatesToSend := calculateUpdates(a.lastStates, currentStates)
		a.lastStates = currentStates

		err := a.resp.Send(&apiv1.ResourcePoolStatesResponse{
			States: &resourcepoolv1.ResourcePoolStates{
				ResourcePools: updatesToSend,
			},
		})
		if err != nil {
			return fmt.Errorf("failed while processing batch: %w", err)
		}

		actors.NotifyAfter(ctx, a.tickInterval, tick{})

	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func calculateUpdates(previous, current []*resourcepoolv1.ResourcePoolState) []*resourcepoolv1.ResourcePoolState {
	// We assume that the number of resource pools is fixed.
	if len(previous) != len(current) {
		panic("The length of resource pools has changed, which should never happen")
	}

	updates := make([]*resourcepoolv1.ResourcePoolState, 0, len(current))
	previousMap := make(map[string]*resourcepoolv1.ResourcePoolState)
	for i := range previous {
		poolName := previous[i].Name
		previousMap[poolName] = previous[i]
	}


	for i := range current {
		newState := current[i]
		oldState := previousMap[newState.Name]
		var update *resourcepoolv1.ResourcePoolState = nil
		if newState.NumAgents != oldState.NumAgents ||
			newState.NumTotalSlots != oldState.NumTotalSlots ||
			newState.NumActiveSlots != oldState.NumActiveSlots ||
			newState.MaxCpuContainers != oldState.MaxCpuContainers ||
			newState.NumActiveCpuContainers != oldState.NumActiveCpuContainers {
			update = newState
		}

		if update != nil {
			updates = append(updates, update)
		}
	}
	return updates

}
