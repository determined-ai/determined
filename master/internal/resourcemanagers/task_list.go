package resourcemanagers

import (
	"strings"

	"github.com/emirpasic/gods/sets/treeset"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
)

// taskList maintains all tasks in time order.
type taskList struct {
	taskByTime    *treeset.Set
	taskByHandler map[*actor.Ref]*sproto.AllocateRequest
	taskByID      map[sproto.TaskID]*sproto.AllocateRequest
	allocations   map[*actor.Ref]*sproto.ResourcesAllocated
}

func newTaskList() *taskList {
	return &taskList{
		taskByTime:    treeset.NewWith(taskComparator),
		taskByHandler: make(map[*actor.Ref]*sproto.AllocateRequest),
		taskByID:      make(map[sproto.TaskID]*sproto.AllocateRequest),
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

func (l *taskList) GetTaskByID(id sproto.TaskID) (*sproto.AllocateRequest, bool) {
	req, ok := l.taskByID[id]
	return req, ok
}

func (l *taskList) AddTask(req *sproto.AllocateRequest) bool {
	if _, ok := l.GetTaskByHandler(req.TaskActor); ok {
		return false
	}

	l.taskByTime.Add(req)
	l.taskByHandler[req.TaskActor] = req
	l.taskByID[req.ID] = req
	return true
}

func (l *taskList) RemoveTaskByHandler(handler *actor.Ref) *sproto.AllocateRequest {
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

func (l *taskList) GetAllocations(handler *actor.Ref) *sproto.ResourcesAllocated {
	return l.allocations[handler]
}

func (l *taskList) SetAllocations(handler *actor.Ref, assigned *sproto.ResourcesAllocated) {
	l.allocations[handler] = assigned
}

func (l *taskList) RemoveAllocations(handler *actor.Ref) {
	delete(l.allocations, handler)
}

type taskIterator struct{ it treeset.Iterator }

func (i *taskIterator) next() bool {
	return i.it.Next()
}
func (i *taskIterator) value() *sproto.AllocateRequest {
	return i.it.Value().(*sproto.AllocateRequest)
}

func taskComparator(a interface{}, b interface{}) int {
	t1, t2 := a.(*sproto.AllocateRequest), b.(*sproto.AllocateRequest)
	if t1.TaskActor.RegisteredTime().Equal(t2.TaskActor.RegisteredTime()) {
		return strings.Compare(string(t1.ID), string(t2.ID))
	}
	if t1.TaskActor.RegisteredTime().Before(t2.TaskActor.RegisteredTime()) {
		return -1
	}
	return 1
}
