package tasklogger

import "github.com/determined-ai/determined/master/pkg/model"

// Writer stores task logs in a backend.
type Writer interface {
	AddTaskLogs([]*model.TaskLog) error
}

type funcWriter func([]*model.TaskLog) error

func (f funcWriter) AddTaskLogs(tl []*model.TaskLog) error { return f(tl) }

var (
	nullWriter = funcWriter(func(tl []*model.TaskLog) error { return nil })
	nullLogger = New(nullWriter)
)
