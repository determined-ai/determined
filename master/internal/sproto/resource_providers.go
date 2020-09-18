package sproto

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/pkg/actor"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
)

// Outgoing messages.
type (
	// ContainerLog notifies the task actor that a new log message is available for the container.
	// It is used by the resource providers to communicate internally and with the task handlers.
	ContainerLog struct {
		Container container.Container
		Timestamp time.Time

		PullMessage *jsonmessage.JSONMessage
		RunMessage  *agent.RunMessage
		AuxMessage  *string
	}

	// ContainerStateChanged notifies that the recipient container state has been transitioned.
	// It is used by the resource providers to communicate with the task handlers.
	ContainerStateChanged struct {
		Container        container.Container
		ContainerStopped *agent.ContainerStopped
	}
)

// Incoming messages.
type (
	// ConfigureEndpoints informs the resource provider to configure the endpoints resources.
	ConfigureEndpoints struct {
		System *actor.System
		Echo   *echo.Echo
	}

	// GetEndpointActorAddress requests the name of the actor that is managing the resources.
	GetEndpointActorAddress struct{}
	// KillContainer notifies to stop a container/pod associated with the container id.
	KillContainer struct {
		ContainerID container.ID
	}
)

func (c ContainerLog) String() string {
	msg := ""
	switch {
	case c.AuxMessage != nil:
		msg = *c.AuxMessage
	case c.RunMessage != nil:
		msg = strings.TrimSuffix(c.RunMessage.Value, "\n")
	case c.PullMessage != nil:
		buf := new(bytes.Buffer)
		if err := c.PullMessage.Display(buf, false); err != nil {
			msg = err.Error()
		} else {
			msg = buf.String()
			// Docker disables printing the progress bar in non-terminal mode.
			if msg == "" && c.PullMessage.Progress != nil {
				msg = c.PullMessage.Progress.String()
			}
			msg = strings.TrimSpace(msg)
		}
	default:
		panic("unknown log message received")
	}
	shortID := c.Container.ID[:8]
	timestamp := c.Timestamp.UTC().Format(time.RFC3339)
	return fmt.Sprintf("[%s] %s [%s] || %s", timestamp, shortID, c.Container.State, msg)
}
