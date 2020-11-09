package sproto

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/container"
)

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
	// TaskContainerStarted contains the information needed by tasks from container started.
	TaskContainerStarted struct {
		Addresses []container.Address
	}
	// TaskContainerStopped contains the information needed by tasks from container stopped.
	TaskContainerStopped struct {
		agent.ContainerStopped
	}
	// TaskContainerStateChanged notifies that the task actor container state has been transitioned.
	// It is used by the resource managers to communicate with the task handlers.
	TaskContainerStateChanged struct {
		Container        container.Container
		ContainerStarted *TaskContainerStarted
		ContainerStopped *TaskContainerStopped
	}

	// SetGroupMaxSlots sets the maximum number of slots that a group can consume in the cluster.
	SetGroupMaxSlots struct {
		MaxSlots     *int
		ResourcePool string
		Handler      *actor.Ref
	}
	// SetGroupWeight sets the weight of a group in the fair share scheduler.
	SetGroupWeight struct {
		Weight       float64
		ResourcePool string
		Handler      *actor.Ref
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
	return fmt.Sprintf("[%s] %s || %s", timestamp, shortID, msg)
}
