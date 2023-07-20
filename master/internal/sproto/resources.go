package sproto

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// All the From... methods expose the more abstract representation returned by resource managers
// (since they can't all necessary know resources at the granularity of a container). Note that
// these are here rather than on cproto.ID as ToResourcesID to prevent circular imports; sproto
// should be able to import cproto but not vice versa.

// ResourcesID is the ID of some set of resources.
type ResourcesID string

// FromContainerID converts a cproto.ID to a ResourcesID.
func FromContainerID(cID cproto.ID) ResourcesID {
	return ResourcesID(cID)
}

// ResourcesType is the type of some set of resources. This should be purely informational.
type ResourcesType string

// ResourcesState is the state of some set of resources.
type ResourcesState string

func (s ResourcesState) String() string {
	return string(s)
}

const (
	// Assigned state means that the resources have been assigned.
	Assigned ResourcesState = "ASSIGNED"
	// Pulling state means that the resources are pulling container images.
	Pulling ResourcesState = "PULLING"
	// Starting state means the service running on the resources is being started.
	Starting ResourcesState = "STARTING"
	// Running state means that the service on the resources is running.
	Running ResourcesState = "RUNNING"
	// Terminated state means that the resources have exited or has been aborted.
	Terminated ResourcesState = "TERMINATED"
	// Unknown state is a null value.
	Unknown ResourcesState = ""
)

// FromContainerState converts a cproto.State to ResourcesState. This may shortly become much less
// granular (not a one to one mapping).
func FromContainerState(state cproto.State) ResourcesState {
	switch state {
	case cproto.Assigned:
		return Assigned
	case cproto.Pulling:
		return Pulling
	case cproto.Starting:
		return Starting
	case cproto.Running:
		return Running
	case cproto.Terminated:
		return Terminated
	case cproto.Unknown:
		return Unknown
	default:
		return Unknown
	}
}

// ResourcesStarted contains the information needed by tasks from container started.
type ResourcesStarted struct {
	Addresses []cproto.Address
	// NativeResourcesID is the native Docker hex container ID of the Determined container.
	NativeResourcesID string
}

// Proto returns the proto representation of ResourcesStarted.
func (r *ResourcesStarted) Proto() *taskv1.ResourcesStarted {
	if r == nil {
		return nil
	}

	pbAddresses := []*taskv1.Address{}

	for _, address := range r.Addresses {
		pbAddresses = append(pbAddresses, address.Proto())
	}

	return &taskv1.ResourcesStarted{
		Addresses:         pbAddresses,
		NativeResourcesId: r.NativeResourcesID,
	}
}

// FromContainerStarted converts an aproto.ContainerStarted message to ResourcesStarted.
func FromContainerStarted(cs *aproto.ContainerStarted) *ResourcesStarted {
	if cs == nil {
		return nil
	}

	return &ResourcesStarted{
		Addresses:         cs.Addresses(),
		NativeResourcesID: cs.ContainerInfo.ID,
	}
}

// ResourcesStopped contains the information needed by tasks from container stopped.
type ResourcesStopped struct {
	Failure *ResourcesFailure
}

// Proto returns the proto representation of ResourcesStopped.
func (r *ResourcesStopped) Proto() *taskv1.ResourcesStopped {
	if r == nil {
		return nil
	}
	return &taskv1.ResourcesStopped{
		Failure: r.Failure.Proto(),
	}
}

// FromContainerStopped converts an aproto.ContainerStopped message to ResourcesStopped.
func FromContainerStopped(cs *aproto.ContainerStopped) *ResourcesStopped {
	if cs == nil {
		return nil
	}

	rs := &ResourcesStopped{}
	if f := cs.Failure; f != nil {
		rs.Failure = &ResourcesFailure{
			FailureType: FromContainerFailureType(f.FailureType),
			ErrMsg:      f.ErrMsg,
			ExitCode:    FromContainerExitCode(f.ExitCode),
		}
	}
	return rs
}

// ResourcesError returns a resources stopped message wrapping the provided error. If the error is
// nil, a stack trace is provided instead.
func ResourcesError(failureType FailureType, err error) ResourcesStopped {
	if err == nil {
		return ResourcesStopped{
			Failure: &ResourcesFailure{
				FailureType: failureType,
				ErrMsg:      errors.WithStack(errors.Errorf("unknown error occurred")).Error(),
			},
		}
	}
	return ResourcesStopped{
		Failure: &ResourcesFailure{
			FailureType: failureType,
			ErrMsg:      err.Error(),
		},
	}
}

func (r ResourcesStopped) String() string {
	if r.Failure == nil {
		return "resources exited successfully with a zero exit code"
	}
	return r.Failure.Error()
}

// ResourcesFailure contains information about restored resources' failure.
type ResourcesFailure struct {
	FailureType FailureType
	ErrMsg      string
	ExitCode    *ExitCode
}

// Proto returns the proto representation of ResourcesFailure.
func (f *ResourcesFailure) Proto() *taskv1.ResourcesFailure {
	if f == nil {
		return nil
	}

	pbResourcesFailure := taskv1.ResourcesFailure{
		FailureType: f.FailureType.Proto(),
		ErrMsg:      f.ErrMsg,
	}

	if f.ExitCode != nil {
		exitCode := int32(*f.ExitCode)
		pbResourcesFailure.ExitCode = &exitCode
	}

	return &pbResourcesFailure
}

// NewResourcesFailure returns a resources failure message wrapping the type, msg and exit code.
func NewResourcesFailure(
	failureType FailureType, msg string, code *ExitCode,
) *ResourcesFailure {
	return &ResourcesFailure{
		FailureType: failureType,
		ErrMsg:      msg,
		ExitCode:    code,
	}
}

func (f ResourcesFailure) Error() string {
	if f.ExitCode == nil {
		if len(f.ErrMsg) > 0 {
			return fmt.Sprintf("%s: %s", f.FailureType, f.ErrMsg)
		}
		return fmt.Sprintf("%s", f.FailureType)
	}
	return fmt.Sprintf("%s: %s (exit code %d)", f.FailureType, f.ErrMsg, *f.ExitCode)
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
	// ResourcesFailed denotes that the container ran but failed with a non-zero exit code.
	ResourcesFailed FailureType = "resources failed with non-zero exit code"

	// ResourcesAborted denotes the container was canceled before it was started.
	ResourcesAborted FailureType = "resources was aborted before it started"

	// ResourcesMissing denotes the resources were missing when the master asked about it.
	ResourcesMissing FailureType = "request for action on unknown resources"

	// TaskAborted denotes that the task was canceled before it was started.
	TaskAborted FailureType = "task was aborted before the task was started"

	// TaskError denotes that the task failed without an associated exit code.
	TaskError FailureType = "task failed without an associated exit code"

	// AgentFailed denotes that the agent failed while the container was running.
	AgentFailed FailureType = "agent failed while the container was running"

	// AgentError denotes that the agent failed to launch the container.
	AgentError FailureType = "agent failed to launch the container"

	// RestoreError denotes a failure to restore a running allocation on master blip.
	RestoreError FailureType = "RM failed to restore the allocation"

	// UnknownError denotes an internal error that did not map to a know failure type.
	UnknownError = "unknown agent failure: %s"
)

// Proto returns the proto representation of the device type.
func (f FailureType) Proto() taskv1.FailureType {
	switch f {
	case ResourcesFailed:
		return taskv1.FailureType_FAILURE_TYPE_RESOURCES_FAILED
	case ResourcesAborted:
		return taskv1.FailureType_FAILURE_TYPE_RESOURCES_ABORTED
	case ResourcesMissing:
		return taskv1.FailureType_FAILURE_TYPE_RESOURCES_MISSING
	case TaskAborted:
		return taskv1.FailureType_FAILURE_TYPE_TASK_ABORTED
	case TaskError:
		return taskv1.FailureType_FAILURE_TYPE_TASK_ERROR
	case AgentFailed:
		return taskv1.FailureType_FAILURE_TYPE_AGENT_FAILED
	case AgentError:
		return taskv1.FailureType_FAILURE_TYPE_AGENT_ERROR
	case RestoreError:
		return taskv1.FailureType_FAILURE_TYPE_RESTORE_ERROR
	case UnknownError:
		return taskv1.FailureType_FAILURE_TYPE_UNKNOWN_ERROR
	default:
		return taskv1.FailureType_FAILURE_TYPE_UNSPECIFIED
	}
}

// FromContainerFailureType converts an aproto.FailureType to a FailureType. This mapping is not
// guaranteed to remain one to one; this conversion may do some level of interpretation.
func FromContainerFailureType(t aproto.FailureType) FailureType {
	switch t {
	case aproto.ContainerFailed:
		return ResourcesFailed
	case aproto.ContainerAborted:
		return ResourcesAborted
	case aproto.ContainerMissing:
		return ResourcesMissing
	case aproto.TaskAborted:
		return TaskAborted
	case aproto.TaskError:
		return TaskError
	case aproto.AgentFailed:
		return AgentFailed
	case aproto.AgentError:
		return AgentError
	case aproto.RestoreError:
		return RestoreError
	default:
		return FailureType(fmt.Sprintf(UnknownError, t))
	}
}

// InvalidResourcesRequestError is an unrecoverable validation error from the underlying RM.
type InvalidResourcesRequestError struct {
	Cause error
}

func (e InvalidResourcesRequestError) Error() string {
	return fmt.Sprintf("invalid resources request: %s", e.Cause.Error())
}

// IsUnrecoverableSystemError checks if the error is absolutely unrecoverable.
func IsUnrecoverableSystemError(err error) bool {
	switch err.(type) {
	case InvalidResourcesRequestError:
		return true
	default:
		return false
	}
}

// IsTransientSystemError checks if the error is caused by the system and
// shouldn't count against `max_restarts`.
func IsTransientSystemError(err error) bool {
	switch err := err.(type) {
	case ResourcesFailure:
		switch err.FailureType {
		case ResourcesFailed, TaskError:
			return false
		// Questionable, could be considered failures, but for now we don't.
		case AgentError, AgentFailed, RestoreError:
			return true
		// Definitely not a failure.
		case TaskAborted, ResourcesAborted:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// ResourcesStateChanged notifies that the task actor container state has been transitioned.
// It is used by the resource managers to communicate with the task handlers.
type ResourcesStateChanged struct {
	ResourcesID    ResourcesID
	ResourcesState ResourcesState

	ResourcesStarted *ResourcesStarted
	ResourcesStopped *ResourcesStopped

	// More granular information about specific resource types.
	// TODO(!!!): This can be removed now.
	Container *cproto.Container
}

// FromContainerStateChanged converts an aproto.ContainerStateChanged message to
// ResourcesStateChanged.
func FromContainerStateChanged(sc aproto.ContainerStateChanged) *ResourcesStateChanged {
	return &ResourcesStateChanged{
		ResourcesID:      FromContainerID(sc.Container.ID),
		ResourcesState:   FromContainerState(sc.Container.State),
		ResourcesStarted: FromContainerStarted(sc.ContainerStarted),
		ResourcesStopped: FromContainerStopped(sc.ContainerStopped),
		Container:        &sc.Container,
	}
}

// ContainerIDStr returns the associated container ID str if there is one or nil.
func (r *ResourcesStateChanged) ContainerIDStr() *string {
	if r.Container == nil {
		return nil
	}

	return (*string)(&r.Container.ID)
}
