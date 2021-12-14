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
		return *c.PullMessage
	default:
		panic("unknown log message received")
	}
}

func (c ContainerLog) String() string {
	shortID := c.Container.ID[:8]
	timestamp := c.Timestamp.UTC().Format(time.RFC3339)
	return fmt.Sprintf("[%s] %s || %s", timestamp, shortID, c.Message())
}

// ToTaskLog converts a container log to a task log.
func (c ContainerLog) ToTaskLog() model.TaskLog {
	return model.TaskLog{
		ContainerID: ptrs.StringPtr(string(c.Container.ID)),
		Log:         c.Message(),
		Level:       c.Level,
		Timestamp:   ptrs.TimePtr(c.Timestamp.UTC()),
	}
}
