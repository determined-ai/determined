package aproto

import (
	"syscall"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/pkg/cproto"
)

// AgentShutdown is an explicit message from master to agent it should shutdown itself.
type AgentShutdown struct {
	ErrMsg string
}

// AgentMessage is a union type for all messages sent to agents.
type AgentMessage struct {
	MasterSetAgentOptions *MasterSetAgentOptions
	StartContainer        *StartContainer
	SignalContainer       *SignalContainer
	AgentShutdown         *AgentShutdown
}

// MasterSetAgentOptions is the first message sent to an agent by the master. It lets
// the agent know to update its configuration with the options in this message.
// This is generally useful for configurations that are not _agent_ specific but
// cluster-wide.
type MasterSetAgentOptions struct {
	MasterInfo           MasterInfo
	LoggingOptions       model.LoggingConfig
	ContainersToReattach []ContainerReattach
}

// StartContainer notifies the agent to start a container with the provided spec.
type StartContainer struct {
	Container cproto.Container
	Spec      cproto.Spec
}

// SignalContainer notifies the agent to send the requested signal to the container.
type SignalContainer struct {
	ContainerID cproto.ID
	Signal      syscall.Signal
}

// ErrAgentMustReconnect is the error returned by the master when the agent must exit and reconnect.
var ErrAgentMustReconnect = errors.New("agent is past reconnect period, it must restart")
