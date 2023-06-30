package tasklist

import (
	"strings"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/emirpasic/gods/sets/treeset"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// TaskList maintains all tasks in time order, and stores their allocation actor,
// active allocations and allocate requests.
type TaskList struct {
	taskByTime  *treeset.Set
	taskByID    map[model.AllocationID]*sproto.AllocateRequest
	allocations map[model.AllocationID]*sproto.ResourcesAllocated
}

// New constructs a new TaskList.
func New() *TaskList {
	return &TaskList{
		taskByTime: treeset.NewWith(func(a, b interface{}) int {
			t1, t2 := a.(*sproto.AllocateRequest), b.(*sproto.AllocateRequest)
			return allocationRequestComparator(t1, t2)
		}),
		taskByID:    make(map[model.AllocationID]*sproto.AllocateRequest),
		allocations: make(map[model.AllocationID]*sproto.ResourcesAllocated),
	}
}

// Iterator returns a TaskIterator that traverses the TaskList by request time.
func (l *TaskList) Iterator() *TaskIterator {
	return &TaskIterator{it: l.taskByTime.Iterator()}
}

// Len gives number of tasks in the TaskList.
func (l *TaskList) Len() int {
	return len(l.taskByID)
}

// TaskByID returns the sproto.AllocateRequest for a task.
func (l *TaskList) TaskByID(id model.AllocationID) (*sproto.AllocateRequest, bool) {
	req, ok := l.taskByID[id]
	return req, ok
}

// AddTask adds a task to the TaskList.
func (l *TaskList) AddTask(req *sproto.AllocateRequest) bool {
	if _, ok := l.TaskByID(req.AllocationID); ok {
		return false
	}

	l.taskByTime.Add(req)
	l.taskByID[req.AllocationID] = req
	return true
}

// RemoveTaskByID deletes the task and its allocation, if any, from the TaskList.
func (l *TaskList) RemoveTaskByID(id model.AllocationID) *sproto.AllocateRequest {
	req, ok := l.TaskByID(id)
	if !ok {
		return nil
	}

	l.taskByTime.Remove(req)
	delete(l.taskByID, req.AllocationID)
	delete(l.allocations, req.AllocationID)
	return req
}

// Allocation returns the allocation, or nil if there is none, for the allocation actor.
func (l *TaskList) Allocation(id model.AllocationID) *sproto.ResourcesAllocated {
	return l.allocations[id]
}

// IsScheduled returns true if the allocation has resources.
func (l *TaskList) IsScheduled(id model.AllocationID) bool {
	return AssignmentIsScheduled(l.Allocation(id))
}

// AddAllocation adds an allocation for the allocation actor and updates the
// sproto.AllocateRequest's sproto.SchedulingState.
func (l *TaskList) AddAllocation(id model.AllocationID, assigned *sproto.ResourcesAllocated) {
	if AssignmentIsScheduled(assigned) {
		l.taskByID[id].State = sproto.SchedulingStateScheduled
	} else {
		l.taskByID[id].State = sproto.SchedulingStateQueued
	}
	l.AddAllocationRaw(id, assigned)
}

// AddAllocationRaw adds an allocation for the allocation actor without modifying the
// sproto.AllocateRequest's  sproto.SchedulingState.
func (l *TaskList) AddAllocationRaw(id model.AllocationID, assigned *sproto.ResourcesAllocated) {
	l.allocations[id] = assigned
}

// RemoveAllocation deletes any allocations for the allocation actor from the TaskList.
func (l *TaskList) RemoveAllocation(id model.AllocationID) {
	delete(l.allocations, id)
	l.taskByID[id].State = sproto.SchedulingStateQueued
}

// ForResourcePool returns a new TaskList filtered by resource pool.
func (l *TaskList) ForResourcePool(name string) *TaskList {
	newTaskList := New()
	for it := l.Iterator(); it.Next(); {
		task := it.Value()
		if task.ResourcePool != name {
			continue
		}

		newTaskList.AddTask(it.Value())
	}
	return newTaskList
}

// TaskSummary returns a summary for an allocation in the TaskList.
func (l *TaskList) TaskSummary(
	id model.AllocationID,
	groups map[*actor.Ref]*Group,
	schedulerType string,
) *sproto.AllocationSummary {
	if req, ok := l.TaskByID(id); ok {
		summary := newTaskSummary(
			req, l.Allocation(req.AllocationID), groups, schedulerType)
		return &summary
	}
	return nil
}

// TaskSummaries returns a summary of allocations for tasks in the TaskList.
func (l *TaskList) TaskSummaries(
	groups map[*actor.Ref]*Group,
	schedulerType string,
) map[model.AllocationID]sproto.AllocationSummary {
	ret := make(map[model.AllocationID]sproto.AllocationSummary)
	for it := l.Iterator(); it.Next(); {
		req := it.Value()
		ret[req.AllocationID] = newTaskSummary(
			req, l.Allocation(req.AllocationID), groups, schedulerType)
	}
	return ret
}

// TaskIterator is an iterator over some of AllocateRequests.
type TaskIterator struct{ it treeset.Iterator }

// Next moves the iterator forward to the next AllocateRequest.
func (i *TaskIterator) Next() bool {
	return i.it.Next()
}

// Value returns the AllocateRequest at the current position of the iterator.
func (i *TaskIterator) Value() *sproto.AllocateRequest {
	return i.it.Value().(*sproto.AllocateRequest)
}

func newTaskSummary(
	request *sproto.AllocateRequest,
	allocated *sproto.ResourcesAllocated,
	groups map[*actor.Ref]*Group,
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
		RegisteredTime: request.AllocationRef.RegisteredTime(),
		ResourcePool:   request.ResourcePool,
		SlotsNeeded:    request.SlotsNeeded,
		Resources:      resourcesSummaries,
		SchedulerType:  schedulerType,
		ProxyPorts:     request.ProxyPorts,
	}

	if group, ok := groups[request.Group]; ok {
		summary.Priority = group.Priority
	}
	return summary
}

// allocationRequestComparator compares AllocateRequests by how long their jobs have been submitted
// while falling back to when their Allocation actor was created for non-job tasks.
// a < b iff a is older than b.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func allocationRequestComparator(a *sproto.AllocateRequest, b *sproto.AllocateRequest) int {
	if a.JobSubmissionTime.Equal(b.JobSubmissionTime) {
		return registerTimeComparator(a, b)
	}
	if a.JobSubmissionTime.Before(b.JobSubmissionTime) {
		return -1
	}
	return 1
}

// registerTimeComparator compares AllocateRequests based on when their Allocate actor was
// registred.
func registerTimeComparator(t1 *sproto.AllocateRequest, t2 *sproto.AllocateRequest) int {
	if !t1.AllocationRef.RegisteredTime().Equal(t2.AllocationRef.RegisteredTime()) {
		if t1.AllocationRef.RegisteredTime().Before(t2.AllocationRef.RegisteredTime()) {
			return -1
		}
		return 1
	}
	return strings.Compare(string(t1.AllocationID), string(t2.AllocationID))
}
