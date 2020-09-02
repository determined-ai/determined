package scheduler

import image "github.com/determined-ai/determined/master/pkg/tasks"

// Assignment is an interface that provides function for task actors
// to start tasks on assigned resources.
type Assignment interface {
	StartContainer(spec image.TaskSpec)
	KillContainer()
}
