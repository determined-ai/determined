package scheduler

import (
	"strings"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/emirpasic/gods/sets/treeset"
)

// taskList maintains all tasks in time order.
type taskList struct {
	taskByTime    *treeset.Set
	taskByHandler map[*actor.Ref]*AddTask
	taskByID      map[TaskID]*AddTask
	assignments   map[*actor.Ref]*ResourceAssigned
}

func newTaskList() *taskList {
	return &taskList{
		taskByTime:    treeset.NewWith(taskComparator),
		taskByHandler: make(map[*actor.Ref]*AddTask),
		taskByID:      make(map[TaskID]*AddTask),
		assignments:   make(map[*actor.Ref]*ResourceAssigned),
	}
}

func (l *taskList) iterator() *taskIterator {
	return &taskIterator{it: l.taskByTime.Iterator()}
}

func (l *taskList) len() int {
	return len(l.taskByHandler)
}

func (l *taskList) GetTask(handler *actor.Ref) (*AddTask, bool) {
	req, ok := l.taskByHandler[handler]
	return req, ok
}

func (l *taskList) GetTaskByID(id TaskID) (*AddTask, bool) {
	req, ok := l.taskByID[id]
	return req, ok
}

func (l *taskList) AddTask(req *AddTask) bool {
	if _, ok := l.GetTask(req.Handler); ok {
		return false
	}

	l.taskByTime.Add(req)
	l.taskByHandler[req.Handler] = req
	l.taskByID[req.ID] = req
	return true
}

func (l *taskList) RemoveTask(handler *actor.Ref) *AddTask {
	req, ok := l.GetTask(handler)
	if !ok {
		return nil
	}

	l.taskByTime.Remove(req)
	delete(l.taskByHandler, handler)
	delete(l.taskByID, req.ID)
	delete(l.assignments, handler)
	return req
}

func (l *taskList) GetAssignments(handler *actor.Ref) *ResourceAssigned {
	return l.assignments[handler]
}

func (l *taskList) SetAssignments(handler *actor.Ref, assigned *ResourceAssigned) {
	l.assignments[handler] = assigned
}

func (l *taskList) ClearAssignments(handler *actor.Ref) {
	delete(l.assignments, handler)
}

type taskIterator struct{ it treeset.Iterator }

func (i *taskIterator) next() bool      { return i.it.Next() }
func (i *taskIterator) value() *AddTask { return i.it.Value().(*AddTask) }

func taskComparator(a interface{}, b interface{}) int {
	t1, t2 := a.(*AddTask), b.(*AddTask)
	if t1.Handler.RegisteredTime().Equal(t2.Handler.RegisteredTime()) {
		return strings.Compare(string(t1.ID), string(t2.ID))
	}
	if t1.Handler.RegisteredTime().Before(t2.Handler.RegisteredTime()) {
		return -1
	}
	return 1
}
