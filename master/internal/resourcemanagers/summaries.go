package resourcemanagers

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/determined-ai/determined/master/internal/sproto"
)

// TaskSummary contains information about a task for external display.
type TaskSummary struct {
	TaskID         model.TaskID              `json:"task_id"`
	AllocationID   model.AllocationID        `json:"allocation_id"`
	Name           string                    `json:"name"`
	RegisteredTime time.Time                 `json:"registered_time"`
	ResourcePool   string                    `json:"resource_pool"`
	SlotsNeeded    int                       `json:"slots_needed"`
	Containers     []sproto.ContainerSummary `json:"containers"`
	SchedulerType  string                    `json:"scheduler_type"`
	Priority       *int                      `json:"priority"`
}

func newTaskSummary(
	request *sproto.AllocateRequest,
	allocated *sproto.ResourcesAllocated,
	groups map[*actor.Ref]*group,
	schedulerType string,
) TaskSummary {
	// Summary returns a new immutable view of the task state.
	containerSummaries := make([]sproto.ContainerSummary, 0)
	if allocated != nil {
		for _, c := range allocated.Reservations {
			containerSummaries = append(containerSummaries, c.Summary())
		}
	}
	summary := TaskSummary{
		TaskID:         request.TaskID,
		AllocationID:   request.AllocationID,
		Name:           request.Name,
		RegisteredTime: request.TaskActor.RegisteredTime(),
		ResourcePool:   request.ResourcePool,
		SlotsNeeded:    request.SlotsNeeded,
		Containers:     containerSummaries,
		SchedulerType:  schedulerType,
	}

	if group, ok := groups[request.Group]; ok {
		summary.Priority = group.priority
	}
	return summary
}

// newAgentSummary returns a new immutable view of the agent.
func newAgentSummary(state *agentState) sproto.AgentSummary {
	return sproto.AgentSummary{
		Name:   state.handler.Address().Local(),
		IsIdle: state.idle(),
	}
}

func getTaskHandler(
	reqList *taskList,
	id model.AllocationID,
) *actor.Ref {
	if req, ok := reqList.GetTaskByID(id); ok {
		return req.TaskActor
	}
	return nil
}

func getTaskSummary(
	reqList *taskList,
	id model.AllocationID,
	groups map[*actor.Ref]*group,
	schedulerType string,
) *TaskSummary {
	if req, ok := reqList.GetTaskByID(id); ok {
		summary := newTaskSummary(req, reqList.GetAllocations(req.TaskActor), groups, schedulerType)
		return &summary
	}
	return nil
}

func getTaskSummaries(
	reqList *taskList,
	groups map[*actor.Ref]*group,
	schedulerType string,
) map[model.AllocationID]TaskSummary {
	ret := make(map[model.AllocationID]TaskSummary)
	for it := reqList.iterator(); it.next(); {
		req := it.value()
		ret[req.AllocationID] = newTaskSummary(
			req, reqList.GetAllocations(req.TaskActor), groups, schedulerType)
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
	agentInfo map[*actor.Ref]*agentState,
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
		summary.numTotalSlots += agentState.numSlots()
		summary.numActiveSlots += agentState.numUsedSlots()
		summary.maxNumAuxContainers += agentState.numZeroSlots()
		summary.numActiveAuxContainers += agentState.numUsedZeroSlots()
		for agentDevice := range agentState.devices {
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
