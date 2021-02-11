package command

import (
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// State represents the current state of the container.
type State string

func (s State) String() string {
	return string(s)
}

const (
	// Pending state denotes that the command is awaiting allocation.
	Pending State = "PENDING"
	// Assigned state denotes that the command has been assigned to an agent but has not started
	// yet.
	Assigned State = "ASSIGNED"
	// Pulling state denotes that the command's base image is being pulled from the Docker registry.
	Pulling State = "PULLING"
	// Starting state denotes that the image has been built and the command is being started, but the
	// service in the command is not ready yet.
	Starting State = "STARTING"
	// Running state denotes that the service in the command is able to accept requests.
	Running State = "RUNNING"
	// Terminated state denotes that the command has completely exited or the command has been
	// aborted prior to getting assigned.
	Terminated State = "TERMINATED"
)

// Proto returns the proto representation of the task state.
func (s State) Proto() taskv1.State {
	switch s {
	case Pending:
		return taskv1.State_STATE_PENDING
	case Assigned:
		return taskv1.State_STATE_ASSIGNED
	case Pulling:
		return taskv1.State_STATE_PULLING
	case Starting:
		return taskv1.State_STATE_STARTING
	case Running:
		return taskv1.State_STATE_RUNNING
	case Terminated:
		return taskv1.State_STATE_TERMINATED
	default:
		return taskv1.State_STATE_UNSPECIFIED
	}
}
