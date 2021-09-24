package agent

import (
	"syscall"
	"time"

	"github.com/pkg/errors"

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
	MasterInfo     MasterInfo
	LoggingOptions model.LoggingConfig
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

const (
	// AgentReconnectAttempts is the max attempts an agent has to reconnect.
	AgentReconnectAttempts = 5
	// AgentReconnectBackoff is the time between attempts, with the exception of the first.
	AgentReconnectBackoff = 5 * time.Second
	// AgentReconnectWait is the max time the master should wait for an agent before considering
	// it dead. The agent waits (AgentReconnectWait - AgentReconnectBackoff) before stopping
	// attempts and AgentReconnectWait before crashing.
	AgentReconnectWait = AgentReconnectAttempts * AgentReconnectBackoff
)

// ErrAgentMustReconnect is the error returned by the master when the agent must exit and reconnect.
var ErrAgentMustReconnect = errors.New("agent is past reconnect period, it must restart")
