package command

import (
	"time"

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
	// commandOwner describes the owner of a command.
	commandOwner struct {
		ID       model.UserID `json:"id"`
		Username string       `json:"username"`
	}

	// summary holds an immutable snapshot of the command.
	summary struct {
		RegisteredTime time.Time              `json:"registered_time"`
		Owner          commandOwner           `json:"owner"`
		ID             model.TaskID           `json:"id"`
		Config         model.CommandConfig    `json:"config"`
		State          string                 `json:"state"`
		ServiceAddress *string                `json:"service_address"`
		Addresses      []container.Address    `json:"addresses"`
		ExitStatus     *string                `json:"exit_status"`
		Misc           map[string]interface{} `json:"misc"`
		IsReady        bool                   `json:"is_ready"`
		AgentUserGroup *model.AgentUserGroup  `json:"agent_user_group"`
		ResourcePool   string                 `json:"resource_pool"`
	}
)

// newSummary returns a new summary of the command.
func newSummary(c *command) summary {
	return summary{
		RegisteredTime: c.registeredTime,
		Owner: commandOwner{
			ID:       c.Base.Owner.ID,
			Username: c.Base.Owner.Username,
		},
		ID:             c.taskID,
		Config:         c.Config,
		State:          c.State().String(),
		ServiceAddress: c.serviceAddress,
		Addresses:      c.addresses,
		ExitStatus:     c.exitStatus,
		Misc:           c.Metadata,
		IsReady:        c.readinessMessageSent,
		AgentUserGroup: c.Base.AgentUserGroup,
		ResourcePool:   c.Config.Resources.ResourcePool,
	}
}
