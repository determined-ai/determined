package tasklogger

import "github.com/determined-ai/determined/master/pkg/model"

// Writer stores task logs in a backend.
type Writer interface {
	AddTaskLogs([]*model.TaskLog) error
}
