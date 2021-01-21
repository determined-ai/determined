package command

import (
	"time"

	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	// getSummary is an actor message for getting the summary of the command.
	getSummary struct {
		userFilter string
	}
)

type (
	// summary holds an immutable snapshot of the command.
	summary struct {
		RegisteredTime time.Time               `json:"registered_time"`
		Owner          commandOwner            `json:"owner"`
		ID             resourcemanagers.TaskID `json:"id"`
		Config         model.CommandConfig     `json:"config"`
		State          string                  `json:"state"`
		ServiceAddress *string                 `json:"service_address"`
		Addresses      []container.Address     `json:"addresses"`
		ExitStatus     *string                 `json:"exit_status"`
		Misc           map[string]interface{}  `json:"misc"`
		IsReady        bool                    `json:"is_ready"`
		AgentUserGroup *model.AgentUserGroup   `json:"agent_user_group"`
		ResourcePool   string                  `json:"resource_pool"`
	}
)

// newSummary returns a new summary of the command.
func newSummary(c *command) summary {
	state := "PENDING"
	switch {
	case c.container != nil:
		state = c.container.State.String()
	case c.exitStatus != nil:
		state = container.Terminated.String()
	}
	return summary{
		RegisteredTime: c.registeredTime,
		Owner:          c.owner,
		ID:             c.taskID,
		Config:         c.config,
		State:          state,
		ServiceAddress: c.serviceAddress,
		Addresses:      c.addresses,
		ExitStatus:     c.exitStatus,
		Misc:           c.metadata,
		IsReady:        c.readinessMessageSent,
		AgentUserGroup: c.agentUserGroup,
		ResourcePool:   c.config.Resources.ResourcePool,
	}
}
