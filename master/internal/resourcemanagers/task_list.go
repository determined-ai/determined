package resourcemanagers

import (
	"strings"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/emirpasic/gods/sets/treeset"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// taskList maintains all tasks in time order.
type taskList struct {
	taskByTime    *treeset.Set
	taskByHandler map[*actor.Ref]*sproto.AllocateRequest
	taskByID      map[model.AllocationID]*sproto.AllocateRequest
	allocations   map[*actor.Ref]*sproto.ResourcesAllocated
}

func newTaskList() *taskList {
	return &taskList{
		taskByTime: treeset.NewWith(func(a, b interface{}) int {
			t1, t2 := a.(*sproto.AllocateRequest), b.(*sproto.AllocateRequest)
			return aReqComparator(t1, t2)
		}),
		taskByHandler: make(map[*actor.Ref]*sproto.AllocateRequest),
		taskByID:      make(map[model.AllocationID]*sproto.AllocateRequest),
		allocations:   make(map[*actor.Ref]*sproto.ResourcesAllocated),
	}
}

func (l *taskList) iterator() *taskIterator {
	return &taskIterator{it: l.taskByTime.Iterator()}
}

func (l *taskList) len() int {
	return len(l.taskByHandler)
}

func (l *taskList) GetTaskByHandler(handler *actor.Ref) (*sproto.AllocateRequest, bool) {
	req, ok := l.taskByHandler[handler]
	return req, ok
}

func (l *taskList) GetTaskByID(id model.AllocationID) (*sproto.AllocateRequest, bool) {
	req, ok := l.taskByID[id]
	return req, ok
}

func (l *taskList) AddTask(req *sproto.AllocateRequest) bool {
	if _, ok := l.GetTaskByHandler(req.TaskActor); ok {
		return false
	}

	l.taskByTime.Add(req)
	l.taskByHandler[req.TaskActor] = req
	l.taskByID[req.AllocationID] = req
	return true
}

func (l *taskList) RemoveTaskByHandler(handler *actor.Ref) *sproto.AllocateRequest {
	req, ok := l.GetTaskByHandler(handler)
	if !ok {
		return nil
	}

	l.taskByTime.Remove(req)
	delete(l.taskByHandler, handler)
	delete(l.taskByID, req.AllocationID)
	delete(l.allocations, handler)
	return req
}

func (l *taskList) GetAllocations(handler *actor.Ref) *sproto.ResourcesAllocated {
	return l.allocations[handler]
}

func (l *taskList) SetAllocations(handler *actor.Ref, assigned *sproto.ResourcesAllocated) {
	if assignmentIsScheduled(assigned) {
		l.taskByHandler[handler].State = job.SchedulingStateScheduled
	} else {
		l.taskByHandler[handler].State = job.SchedulingStateQueued
	}
	l.SetAllocationsRaw(handler, assigned)
}

func (l *taskList) SetAllocationsRaw(handler *actor.Ref, assigned *sproto.ResourcesAllocated) {
	l.allocations[handler] = assigned
}

func (l *taskList) RemoveAllocations(handler *actor.Ref) {
	delete(l.allocations, handler)
	l.taskByHandler[handler].State = job.SchedulingStateQueued
}

func (l *taskList) ForResourcePool(name string) *taskList {
	newTaskList := newTaskList()
	for it := l.iterator(); it.next(); {
		task := it.value()
		if task.ResourcePool != name {
			continue
		}

		newTaskList.AddTask(it.value())
	}
	return newTaskList
}

type taskIterator struct{ it treeset.Iterator }

func (i *taskIterator) next() bool {
	return i.it.Next()
}

func (i *taskIterator) value() *sproto.AllocateRequest {
	return i.it.Value().(*sproto.AllocateRequest)
}

// registerTimeComparator compares AllocateRequests based on when their Allocate actor was
// registred.
func registerTimeComparator(t1 *sproto.AllocateRequest, t2 *sproto.AllocateRequest) int {
	if !t1.TaskActor.RegisteredTime().Equal(t2.TaskActor.RegisteredTime()) {
		if t1.TaskActor.RegisteredTime().Before(t2.TaskActor.RegisteredTime()) {
			return -1
		}
		return 1
	}
	return strings.Compare(string(t1.AllocationID), string(t2.AllocationID))
}

// aReqComparator compares AllocateRequests by how long their jobs have been submitted
// while falling back to when their Allocation actor was created for non-job tasks.
// a < b iff a is older than b.
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func aReqComparator(a *sproto.AllocateRequest, b *sproto.AllocateRequest) int {
	if a.JobSubmissionTime.Equal(b.JobSubmissionTime) {
		return registerTimeComparator(a, b)
	}
	if a.JobSubmissionTime.Before(b.JobSubmissionTime) {
		return -1
	}
	return 1
}
