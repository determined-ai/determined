package agent

import (
	"syscall"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/container"
)

// AgentMessage is a union type for all messages sent to agents.
type AgentMessage struct {
	MasterSetAgentOptions *MasterSetAgentOptions
	StartContainer        *StartContainer
	SignalContainer       *SignalContainer
}

// MasterSetAgentOptions is the first message sent to an agent by the master. It lets
// the agent know to update its configuration with the options in this message.
// This is generally useful for configurations that are not _agent_ specific but
// cluster-wide.
type MasterSetAgentOptions struct {
	MasterInfo       MasterInfo
	LogDriverOptions model.LogDriverOptions
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
