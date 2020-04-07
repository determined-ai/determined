package agent

import (
	"syscall"

	"github.com/determined-ai/determined/master/pkg/container"
)

// AgentMessage is a union type for all messages sent to agents.
type AgentMessage struct {
	StartContainer  *StartContainer
	SignalContainer *SignalContainer
}

// StartContainer notifies the agent to start a container with the provided spec.
type StartContainer struct {
	Container container.Container
	Spec      container.Spec
}

// SignalContainer notifies the agent to send the requested signal to the container.
type SignalContainer struct {
	ContainerID container.ID
	Signal      syscall.Signal
}
