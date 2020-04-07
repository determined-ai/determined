package command

import (
	"time"

	"github.com/determined-ai/determined/master/internal/scheduler"
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
		RegisteredTime time.Time              `json:"registered_time"`
		Owner          commandOwner           `json:"owner"`
		ID             scheduler.TaskID       `json:"id"`
		Config         model.CommandConfig    `json:"config"`
		State          string                 `json:"state"`
		ServiceAddress *string                `json:"service_address"`
		Addresses      []scheduler.Address    `json:"addresses"`
		ExitStatus     *string                `json:"exit_status"`
		Misc           map[string]interface{} `json:"misc"`
		IsReady        bool                   `json:"is_ready"`
	}
)

// newSummary returns a new summary of the command.
func newSummary(c *command) summary {
	state := "PENDING"
	if c.container != nil {
		state = c.container.State.String()
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
	}
}
