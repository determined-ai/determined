package sproto

import (
	"fmt"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/internal/job"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type (
	// ContainerLog notifies the task actor that a new log message is available for the container.
	// It is used by the resource providers to communicate internally and with the task handlers.
	ContainerLog struct {
		Container cproto.Container
		Timestamp time.Time

		PullMessage *string
		RunMessage  *aproto.RunMessage
		AuxMessage  *string

		// Level is typically unset, but set by parts of the system that know a log shouldn't
		// look as scary as is it. For example, it is set when an Allocation is killed intentionally
		// on the Killed logs.
		Level *string
	}
	// TaskContainerStarted contains the information needed by tasks from container started.
	TaskContainerStarted struct {
		Addresses []cproto.Address
		// NativeReservationID is the native Docker hex container ID of the Determined container.
		NativeReservationID string
	}
	// TaskContainerStopped contains the information needed by tasks from container stopped.
	TaskContainerStopped struct {
		aproto.ContainerStopped
	}
	// TaskContainerStateChanged notifies that the task actor container state has been transitioned.
	// It is used by the resource managers to communicate with the task handlers.
	TaskContainerStateChanged struct {
		Container        cproto.Container
		ContainerStarted *TaskContainerStarted
		ContainerStopped *TaskContainerStopped
	}

	// GetTaskContainerState requests cproto.Container state.
	GetTaskContainerState struct {
		ContainerID cproto.ID
	}
	// UpdatePodStatus notifies the resource manager of job state changes.
	UpdatePodStatus struct {
		ContainerID string
		State       job.SchedulingState
	}

	// SetGroupMaxSlots sets the maximum number of slots that a group can consume in the cluster.
	SetGroupMaxSlots struct {
		MaxSlots     *int
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
		return *c.PullMessage
	default:
		panic("unknown log message received")
	}
}

func (c ContainerLog) String() string {
	var shortID string
	if len(c.Container.ID) >= 8 {
		shortID = c.Container.ID[:8].String()
	}
	timestamp := c.Timestamp.UTC().Format(time.RFC3339Nano)
	return fmt.Sprintf("[%s] %s || %s", timestamp, shortID, c.Message())
}

// ToEvent converts a container log to a container event.
func (c ContainerLog) ToEvent() Event {
	return Event{
		State:       string(c.Container.State),
		ContainerID: string(c.Container.ID),
		Time:        c.Timestamp.UTC(),
		LogEvent:    ptrs.StringPtr(c.Message()),
	}
}

// ToTaskLog converts a container log to a task log.
func (c ContainerLog) ToTaskLog() model.TaskLog {
	return model.TaskLog{
		ContainerID: ptrs.StringPtr(string(c.Container.ID)),
		Level:       c.Level,
		Timestamp:   ptrs.TimePtr(c.Timestamp.UTC()),
		Log:         c.Message(),
	}
}
