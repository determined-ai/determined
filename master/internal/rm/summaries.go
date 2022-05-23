package rm

import (
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/determined-ai/determined/master/internal/sproto"
)

func newTaskSummary(
	request *sproto.AllocateRequest,
	allocated *sproto.ResourcesAllocated,
	groups map[*actor.Ref]*group,
	schedulerType string,
) sproto.AllocationSummary {
	// Summary returns a new immutable view of the task state.
	resourcesSummaries := make([]sproto.ResourcesSummary, 0)
	if allocated != nil {
		for _, r := range allocated.Resources {
			resourcesSummaries = append(resourcesSummaries, r.Summary())
		}
	}
	summary := sproto.AllocationSummary{
		TaskID:         request.TaskID,
		AllocationID:   request.AllocationID,
		Name:           request.Name,
		RegisteredTime: request.AllocationActor.RegisteredTime(),
		ResourcePool:   request.ResourcePool,
		SlotsNeeded:    request.SlotsNeeded,
		Resources:      resourcesSummaries,
		SchedulerType:  schedulerType,
	}

	if group, ok := groups[request.Group]; ok {
		summary.Priority = group.priority
	}
	return summary
}

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *AgentState) sproto.AgentSummary {
	return sproto.AgentSummary{
		Name:   state.Handler.Address().Local(),
		IsIdle: state.Idle(),
	}
}

func getTaskHandler(
	reqList *taskList,
	id model.AllocationID,
) *actor.Ref {
	if req, ok := reqList.GetTaskByID(id); ok {
		return req.AllocationActor
	}
	return nil
}

func getTaskSummary(
	reqList *taskList,
	id model.AllocationID,
	groups map[*actor.Ref]*group,
	schedulerType string,
) *sproto.AllocationSummary {
	if req, ok := reqList.GetTaskByID(id); ok {
		summary := newTaskSummary(req, reqList.GetAllocations(req.AllocationActor), groups, schedulerType)
		return &summary
	}
	return nil
}

func getTaskSummaries(
	reqList *taskList,
	groups map[*actor.Ref]*group,
	schedulerType string,
) map[model.AllocationID]sproto.AllocationSummary {
	ret := make(map[model.AllocationID]sproto.AllocationSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.AllocationID] = newTaskSummary(
			req, reqList.GetAllocations(req.AllocationActor), groups, schedulerType)
	}
	return ret
}

// ResourceSummary is a summary of the resource available/used by a resource pool.
type ResourceSummary struct {
	numAgents              int
	numTotalSlots          int
	numActiveSlots         int
	maxNumAuxContainers    int
	numActiveAuxContainers int
	slotType               device.Type
}

func getResourceSummary(
	agentInfo map[*actor.Ref]*AgentState,
) ResourceSummary {
	summary := ResourceSummary{
		numTotalSlots:          0,
		numActiveSlots:         0,
		maxNumAuxContainers:    0,
		numActiveAuxContainers: 0,
		slotType:               device.ZeroSlot,
	}

	deviceTypeCount := make(map[device.Type]int)

	for _, agentState := range agentInfo {
		summary.numAgents++
		summary.numTotalSlots += agentState.NumSlots()
		summary.numActiveSlots += agentState.NumUsedSlots()
		summary.maxNumAuxContainers += agentState.NumZeroSlots()
		summary.numActiveAuxContainers += agentState.NumUsedZeroSlots()
		for agentDevice := range agentState.Devices {
			deviceTypeCount[agentDevice.Type]++
		}
	}

	// If we have homogenous slots, get their type. Otherwise, we default to
	// `UNSPECIFIED` aka `device.ZeroSlot`, although it may be an error/warning.
	if len(deviceTypeCount) == 1 {
		for deviceType := range deviceTypeCount {
			summary.slotType = deviceType
		}
	}

	return summary
}
