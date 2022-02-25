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
	// ResourcesID is the ID of some set of resources.
	ResourcesID string

	// ResourcesType is the type of some set of resources. This should be purely informational.
	ResourcesType string

	// ResourcesStarted contains the information needed by tasks from container started.
	ResourcesStarted struct {
		Addresses []cproto.Address
		// NativeResourcesID is the native Docker hex container ID of the Determined container.
		NativeResourcesID string
	}

	// GetResourcesContainerState requests cproto.Container state for a given clump of resources.
	// If the resources aren't a container, this request returns a failure.
	GetResourcesContainerState struct {
		ResourcesID ResourcesID
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

// ExitCode is the process exit code of the container.
type ExitCode int

const (
	// SuccessExitCode is the 0 zero value exit code.
	SuccessExitCode = 0
)

// FromContainerExitCode converts an aproto.ExitCode to an ExitCode. ExitCode's type is subject to
// change - it may become an enum instead where we interpret the type of exit for consumers.
func FromContainerExitCode(c *aproto.ExitCode) *ExitCode {
	if c == nil {
		return nil
	}
	ec := ExitCode(*c)
	return &ec
}

// FailureType denotes the type of failure that resulted in the container stopping.
// Each FailureType must be handled by ./internal/task/allocation.go.
type FailureType string

const (
	// ContainerFailed denotes that the container ran but failed with a non-zero exit code.
	ContainerFailed = FailureType("container failed with non-zero exit code")

	// ContainerAborted denotes the container was canceled before it was started.
	ContainerAborted = FailureType("container was aborted before it started")

	// TaskAborted denotes that the task was canceled before it was started.
	TaskAborted = FailureType("task was aborted before the task was started")

	// TaskError denotes that the task failed without an associated exit code.
	TaskError = FailureType("task failed without an associated exit code")

	// AgentFailed denotes that the agent failed while the container was running.
	AgentFailed = FailureType("agent failed while the container was running")

	// AgentError denotes that the agent failed to launch the container.
	AgentError = FailureType("agent failed to launch the container")

	// UnknownError denotes an internal error that did not map to a know failure type.
	UnknownError
)

// FromContainerFailureType converts an aproto.FailureType to a FailureType. This mapping is not
// guaranteed to remain one to one; this conversion may do some level of interpretation.
func FromContainerFailureType(t aproto.FailureType) FailureType {
	switch t {
	case aproto.ContainerFailed:
		return FailureType(t)
	case aproto.ContainerAborted:
		return FailureType(t)
	case aproto.TaskAborted:
		return FailureType(t)
	case aproto.TaskError:
		return FailureType(t)
	case aproto.AgentFailed:
		return FailureType(t)
	case aproto.AgentError:
		return FailureType(t)
	default:
		return FailureType(t)
	}
}

// IsRestartableSystemError checks if the error is caused by the system and
// shouldn't count against `max_restarts`.
func IsRestartableSystemError(err error) bool {
	switch contErr := err.(type) {
	case ResourcesFailure:
		switch contErr.FailureType {
		case ContainerFailed, TaskError:
			return false
		// Questionable, could be considered failures, but for now we don't.
		case AgentError, AgentFailed:
			return true
		// Definitely not a failure.
		case TaskAborted, ContainerAborted:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
