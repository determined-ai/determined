package scheduler

import (
	"strings"

	"github.com/emirpasic/gods/sets/treeset"
)

// A taskList maintains tasks in time order.
type taskList struct {
	tasksByTime *treeset.Set
}

func newTaskList() *taskList {
	return &taskList{
		tasksByTime: treeset.NewWith(taskComparator),
	}
}

func (l *taskList) iterator() *taskIterator {
	return &taskIterator{it: l.tasksByTime.Iterator()}
}

type taskIterator struct{ it treeset.Iterator }

func (i *taskIterator) next() bool   { return i.it.Next() }
func (i *taskIterator) value() *Task { return i.it.Value().(*Task) }

func (l *taskList) Add(task *Task) {
	l.tasksByTime.Add(task)
}

func (l *taskList) Remove(task *Task) {
	l.tasksByTime.Remove(task)
}

func (l *taskList) TaskSummaries() map[TaskID]TaskSummary {
	ret := make(map[TaskID]TaskSummary)
	for it := l.iterator(); it.next(); {
		task := it.value()
		ret[task.ID] = newTaskSummary(task)
	}
	return ret
}

func taskComparator(a interface{}, b interface{}) int {
	t1, t2 := a.(*Task), b.(*Task)
	if t1.handler.RegisteredTime().Equal(t2.handler.RegisteredTime()) {
		return strings.Compare(string(t1.ID), string(t2.ID))
	}
	if t1.handler.RegisteredTime().Before(t2.handler.RegisteredTime()) {
		return -1
	}
	return 1
}
