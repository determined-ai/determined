package scheduler

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Task-related cluster level messages.
type (
	// AddTask adds the sender of the message to the cluster as a task.
	AddTask struct {
		ID                  *TaskID
		Name                string
		Group               *actor.Ref
		SlotsNeeded         int
		CanTerminate        bool
		Label               string
		FittingRequirements FittingRequirements
	}
	// taskStopped notifies that the task actor is stopped.
	taskStopped struct {
		Ref *actor.Ref
	}
	// SetTaskName sets the name of the task handled by the sender of the message.
	SetTaskName struct{ Name string }
	// GetTaskSummary returns the summary of the specified task.
	GetTaskSummary struct{ ID *TaskID }
	// GetTaskSummaries returns the summaries of all the tasks in the cluster.
	GetTaskSummaries struct{}
	// TerminateTask attempts to terminate the task. If requesting to terminate forcibly, the task
	// will be forcibly terminated (via a SIGKILL).
	TerminateTask struct {
		TaskID   TaskID
		Forcible bool
	}
)

// Incoming task actor messages; task actors must accept these messages.
type (
	// ContainerStarted notifies the task actor that a container has been started on an agent.
	ContainerStarted struct{ Container Container }
	// TerminateRequest notifies the task actor that its task has been told to terminate by the
	// scheduler.
	TerminateRequest struct{}
	// TaskTerminated notifies the task actor that all of its containers have terminated.
	TaskTerminated struct {
		Task    TaskSummary
		Aborted bool
	}
	// TaskAborted notifies the task actor that it was terminated before being scheduled.
	TaskAborted struct{}
)

// TaskID is a unique ID assigned to tasks when added to the cluster.
type TaskID string

// NewTaskID constructs a new unique task id.
func NewTaskID() TaskID {
	return TaskID(uuid.New().String())
}

// taskState represents the current state of a task.
type taskState string

const (
	// taskPending denotes that the task has been added to the cluster but has not been assigned any
	// slots.
	taskPending taskState = "PENDING"
	// taskRunning denotes that the task has been assigned slots; this does not, however, indicate if
	// the task container has started or not.
	taskRunning taskState = "RUNNING"
	// taskTerminating denotes that the task has been requested to terminate but has not yet shut
	// down all task containers.
	taskTerminating taskState = "TERMINATING"
	// taskTerminated denotes that all containers have been terminated and the task no longer
	// occupies any slots.
	taskTerminated taskState = "TERMINATED"
)

var taskTransitions = map[taskState]map[taskState]bool{
	taskPending: {
		taskRunning:    true,
		taskTerminated: true,
	},
	taskRunning: {
		taskTerminating: true,
		taskTerminated:  true,
	},
	// A terminating task can transition to terminating again if a task is forcibly terminated.
	taskTerminating: {
		taskTerminating: true,
		taskTerminated:  true,
	},
}

func isValidTaskStateTransition(cur, next taskState) bool {
	return taskTransitions[cur][next]
}

// Task represents a single schedulable unit. A task has a lifecycle that it transitions through
// in response to scheduling directives.
type Task struct {
	ID                  TaskID
	name                string
	handler             *actor.Ref
	group               *group
	slotsNeeded         int
	canTerminate        bool
	state               taskState
	agentLabel          string
	fittingRequirements FittingRequirements
	containers          map[ContainerID]*container
}

// newTask constructs a single task in a pending state from a task template.
func newTask(task *Task) *Task {
	newT := *task

	newT.state = taskPending

	if len(newT.ID) == 0 {
		newT.ID = NewTaskID()
	}

	if newT.containers == nil {
		newT.containers = make(map[ContainerID]*container)
	}

	return &newT
}

// SlotsNeeded returns the number of slots this task needs to start.
func (t *Task) SlotsNeeded() int {
	return t.slotsNeeded
}

// mustTransition transitions a task to the next state. This function panics if the next state is an
// invalid transition from the current task state.
func (t *Task) mustTransition(next taskState) {
	if !isValidTaskStateTransition(t.state, next) {
		panic(fmt.Sprintf("invalid task transition from %v to %v", t.state, next))
	}
	t.state = next
}
