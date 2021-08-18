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

		// Level is typically unset, but set by parts of the system that know a log shouldn't
		// look as scary as is it. For example, it is set when an Allocation is killed intentionally
		// on the Killed logs.
		Level *string
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
	// SetGroupPriority sets the priority of the group in the priority scheduler.
	SetGroupPriority struct {
		Priority     *int
		ResourcePool string
		Handler      *actor.Ref
	}
)

// Message returns the textual content of this log message.
func (c ContainerLog) Message() string {
	switch {
	case c.AuxMessage != nil:
		return *c.AuxMessage
	case c.RunMessage != nil:
		return strings.TrimSuffix(c.RunMessage.Value, "\n")
	case c.PullMessage != nil:
		buf := new(bytes.Buffer)
		if err := c.PullMessage.Display(buf, false); err != nil {
			return err.Error()
		}
		msg := buf.String()
		// Docker disables printing the progress bar in non-terminal mode.
		if msg == "" && c.PullMessage.Progress != nil {
			msg = c.PullMessage.Progress.String()
		}
		return strings.TrimSpace(msg)
	default:
		panic("unknown log message received")
	}
}

func (c ContainerLog) String() string {
	shortID := c.Container.ID[:8]
	timestamp := c.Timestamp.UTC().Format(time.RFC3339)
	return fmt.Sprintf("[%s] %s || %s", timestamp, shortID, c.Message())
}
