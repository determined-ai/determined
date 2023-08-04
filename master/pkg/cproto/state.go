package cproto

import (
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/containerv1"
)

// State represents the current state of the container.
type State string

func (s State) String() string {
	return string(s)
}

// Before returns if our state comes before or is equal to another. Callers have an implicit
// assumption that states always transition in order.
func (s State) Before(other State) bool {
	ordering := []State{
		Assigned,
		Pulling,
		Starting,
		Running,
		Terminated,
	}

	selfPos := slices.Index(ordering, s)
	otherPos := slices.Index(ordering, other)

	return selfPos <= otherPos
}

const (
	// Assigned state means that the container has been assigned to an agent but has not started
	// yet.
	Assigned State = "ASSIGNED"
	// Pulling state means that the container's base image is being pulled from the Docker registry.
	Pulling State = "PULLING"
	// Starting state means that the image has been pulled and the container is being started, but
	// the container is not ready yet.
	Starting State = "STARTING"
	// Running state means that the service in the container is running.
	Running State = "RUNNING"
	// Terminated state means that the container has exited or has been aborted.
	Terminated State = "TERMINATED"
	// Unknown state is a null value.
	Unknown State = ""
)

var validTransitions = map[State]map[State]bool{
	Assigned:   {Pulling: true, Terminated: true},
	Pulling:    {Starting: true, Terminated: true},
	Starting:   {Running: true, Terminated: true},
	Running:    {Terminated: true},
	Terminated: {},
	Unknown:    {},
}

func (s State) checkTransition(new State) error {
	valid, ok := validTransitions[s][new]
	return check.True(valid && ok,
		"cannot transition from %s to %s", s, new)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (s State) MarshalText() (text []byte, err error) {
	return []byte(s), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (s *State) UnmarshalText(text []byte) error {
	parsed := State(text)
	if _, ok := validTransitions[parsed]; !ok {
		return errors.Errorf("invalid container state: %s", text)
	}
	*s = parsed
	return nil
}

// Proto returns the proto representation of the container state.
func (s State) Proto() containerv1.State {
	switch s {
	case Assigned:
		return containerv1.State_STATE_ASSIGNED
	case Pulling:
		return containerv1.State_STATE_PULLING
	case Starting:
		return containerv1.State_STATE_STARTING
	case Running:
		return containerv1.State_STATE_RUNNING
	case Terminated:
		return containerv1.State_STATE_TERMINATED
	default:
		return containerv1.State_STATE_UNSPECIFIED
	}
}

// ParseStateFromDocker parses raw docker state into our state.
func ParseStateFromDocker(cont types.Container) (State, error) {
	switch cont.State {
	case "created", "restarting":
		return Starting, nil
	case "paused", "exited":
		return Terminated, nil
	case "running":
		return Running, nil
	default:
		return Unknown, fmt.Errorf("unknown container state: %s", cont.State)
	}
}
