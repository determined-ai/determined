package aproto

import (
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	// AgentReconnectAttempts is the max attempts an agent has to reconnect.
	AgentReconnectAttempts = 5
	// AgentReconnectBackoffValue in seconds.
	AgentReconnectBackoffValue = 5
	// AgentReconnectBackoff is the time between attempts, with the exception of the first.
	AgentReconnectBackoff = AgentReconnectBackoffValue * time.Second
	// AgentReconnectWait is the max time the master should wait for an agent before considering
	// it dead. The agent waits (AgentReconnectWait - AgentReconnectBackoff) before stopping
	// attempts and AgentReconnectWait before crashing.
	AgentReconnectWait = AgentReconnectAttempts * AgentReconnectBackoff
)

// GetRPConfig is a request from agent to RP actor for some config options.
type GetRPConfig struct{}

// GetRPResponse is a response to the previous request.
type GetRPResponse struct {
	AgentReconnectWait    model.Duration
	AgentReattachEnabled  bool
	MaxZeroSlotContainers int
}
