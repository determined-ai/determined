package internal

import (
	"context"
	"fmt"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/google/uuid"

)

func (a *apiServer) GetResourcePools(
	_ context.Context, req *apiv1.GetResourcePoolsRequest,
) (resp *apiv1.GetResourcePoolsResponse, err error) {
	switch {
	case sproto.UseAgentRM(a.m.system):
		err = a.actorRequest(sproto.AgentRMAddr.String(), req, &resp)
	case sproto.UseK8sRM(a.m.system):
		err = a.actorRequest(sproto.K8sRMAddr.String(), req, &resp)

	default:
		err = status.Error(codes.NotFound, "cannot find appropriate resource manager")
	}
	if err != nil {
		return nil, err
	}

	return resp, a.paginate(&resp.Pagination, &resp.ResourcePools, req.Offset, req.Limit)
}






const tickInterval = 1 * time.Second

func (a *apiServer) ResourcePoolStates(
	req *apiv1.ResourcePoolStatesRequest,
	resp apiv1.Determined_ResourcePoolStatesServer,
) error {
	return a.m.system.MustActorOf(
		actor.Addr("resourcepool-api-state-stream-"+uuid.New().String()),
		NewResourcePoolAPIStatesStreamActor(resp),
	).AwaitTermination()
}

// ResourcePoolAPIStatesStreamActor is the actor that streams state updates
// for the resource pool state streaming API.
type ResourcePoolAPIStatesStreamActor struct {
	resp         apiv1.Determined_ResourcePoolStatesServer
	tickInterval time.Duration
	lastStates   []*resourcepoolv1.ResourcePoolState
}

// NewResourcePoolAPIStatesStreamActor creates a new actor that streams updates until
// the stream finishes.
func NewResourcePoolAPIStatesStreamActor(
	server apiv1.Determined_ResourcePoolStatesServer,
) *ResourcePoolAPIStatesStreamActor {
	return &ResourcePoolAPIStatesStreamActor{
		resp:         server,
		tickInterval: tickInterval,
		lastStates:   nil,
	}
}

// Receive implements the actor's "receive message and act" loop.
func (a *ResourcePoolAPIStatesStreamActor) Receive(ctx *actor.Context) error {
	type tick struct{}
	switch ctx.Message().(type) {
	case actor.PreStart:
		resp := ctx.Ask(sproto.GetCurrentRM(ctx.Self().System()),
			resourcemanagers.GetResourceSummaries{}).Get()
		summaries := resp.(resourcemanagers.ResourceSummaries).Summaries

		states := make([]*resourcepoolv1.ResourcePoolState, 0, len(summaries))
		for _, summary := range summaries {
			state := resourcepoolv1.ResourcePoolState{
				Name:                   summary.Name,
				NumAgents:              int32(summary.NumAgents),
				NumTotalSlots:          int32(summary.NumTotalSlots),
				NumActiveSlots:         int32(summary.NumActiveSlots),
				MaxCpuContainers:       int32(summary.MaxNumCPUContainers),
				NumActiveCpuContainers: int32(summary.NumActiveCPUContainers),
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
		if a.resp.Context().Err() != nil {
			ctx.Self().Stop()
			return nil
		}

		// Get the full state from the resource pools
		resp := ctx.Ask(sproto.GetCurrentRM(ctx.Self().System()),
			resourcemanagers.GetResourceSummaries{}).Get()
		summaries := resp.(resourcemanagers.ResourceSummaries).Summaries

		currentStates := make([]*resourcepoolv1.ResourcePoolState, 0, len(summaries))
		for _, summary := range summaries {
			state := resourcepoolv1.ResourcePoolState{
				Name:                   summary.Name,
				NumAgents:              int32(summary.NumAgents),
				NumTotalSlots:          int32(summary.NumTotalSlots),
				NumActiveSlots:         int32(summary.NumActiveSlots),
				MaxCpuContainers:       int32(summary.MaxNumCPUContainers),
				NumActiveCpuContainers: int32(summary.NumActiveCPUContainers),
			}
			currentStates = append(currentStates, &state)
		}

		// Convert full state into diff to send to client
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

func calculateUpdates(
	previous,
	current []*resourcepoolv1.ResourcePoolState,
) []*resourcepoolv1.ResourcePoolState {
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
