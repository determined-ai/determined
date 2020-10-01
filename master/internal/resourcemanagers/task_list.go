package resourcemanagers

import (
	"strings"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/emirpasic/gods/sets/treeset"
)

// taskList maintains all tasks in time order.
type taskList struct {
	taskByTime    *treeset.Set
	taskByHandler map[*actor.Ref]*AllocateRequest
	taskByID      map[TaskID]*AllocateRequest
	allocations   map[*actor.Ref]*ResourcesAllocated
}

func newTaskList() *taskList {
	return &taskList{
		taskByTime:    treeset.NewWith(taskComparator),
		taskByHandler: make(map[*actor.Ref]*AllocateRequest),
		taskByID:      make(map[TaskID]*AllocateRequest),
		allocations:   make(map[*actor.Ref]*ResourcesAllocated),
	}
}

func (l *taskList) iterator() *taskIterator {
	return &taskIterator{it: l.taskByTime.Iterator()}
}

func (l *taskList) len() int {
	return len(l.taskByHandler)
}

func (l *taskList) GetTaskByHandler(handler *actor.Ref) (*AllocateRequest, bool) {
	req, ok := l.taskByHandler[handler]
	return req, ok
}

func (l *taskList) GetTaskByID(id TaskID) (*AllocateRequest, bool) {
	req, ok := l.taskByID[id]
	return req, ok
}

func (l *taskList) AddTask(req *AllocateRequest) bool {
	if _, ok := l.GetTaskByHandler(req.TaskActor); ok {
		return false
	}

	l.taskByTime.Add(req)
	l.taskByHandler[req.TaskActor] = req
	l.taskByID[req.ID] = req
	return true
}

func (l *taskList) RemoveTaskByHandler(handler *actor.Ref) *AllocateRequest {
	req, ok := l.GetTaskByHandler(handler)
	if !ok {
		return nil
	}

	l.taskByTime.Remove(req)
	delete(l.taskByHandler, handler)
	delete(l.taskByID, req.ID)
	delete(l.allocations, handler)
	return req
}

func (l *taskList) GetAllocations(handler *actor.Ref) *ResourcesAllocated {
	return l.allocations[handler]
}

func (l *taskList) SetAllocations(handler *actor.Ref, assigned *ResourcesAllocated) {
	l.allocations[handler] = assigned
}

type taskIterator struct{ it treeset.Iterator }

func (i *taskIterator) next() bool              { return i.it.Next() }
func (i *taskIterator) value() *AllocateRequest { return i.it.Value().(*AllocateRequest) }

func taskComparator(a interface{}, b interface{}) int {
	t1, t2 := a.(*AllocateRequest), b.(*AllocateRequest)
	if t1.TaskActor.RegisteredTime().Equal(t2.TaskActor.RegisteredTime()) {
		return strings.Compare(string(t1.ID), string(t2.ID))
	}
	if t1.TaskActor.RegisteredTime().Before(t2.TaskActor.RegisteredTime()) {
		return -1
	}
	return 1
}
