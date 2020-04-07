package container

import (
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/check"
)

// State represents the current state of the container.
type State string

func (s State) String() string {
	return string(s)
}

const (
	// Assigned state denotes that the container has been assigned to an agent but has not started
	// yet.
	Assigned State = "ASSIGNED"
	// Pulling state denotes that the container's base image is being pulled from the Docker registry.
	Pulling State = "PULLING"
	// Starting state denotes that the image has been built and the container is being started, but the
	// service in the container is not ready yet.
	Starting State = "STARTING"
	// Running state denotes that the service in the container is able to accept requests.
	Running State = "RUNNING"
	// Terminated state denotes that the container has completely exited or the container has been
	// aborted prior to getting assigned.
	Terminated State = "TERMINATED"
)

var validTransitions = map[State]map[State]bool{
	Assigned:   {Pulling: true, Terminated: true},
	Pulling:    {Starting: true, Terminated: true},
	Starting:   {Running: true, Terminated: true},
	Running:    {Terminated: true},
	Terminated: {},
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
		return errors.Errorf("invalid container state: %s", parsed)
	}
	*s = parsed
	return nil
}
