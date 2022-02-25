package sproto

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

// All the From... methods expose the more abstract representation returned by resource managers
// (since they can't all necessary know resources at the granularity of a container). Note that
// these are here rather than on cproto.ID as ToResourcesID to prevent circular imports; sproto
// should be able to import cproto but not vice versa.

// FromContainerID converts a cproto.ID to a ResourcesID.
func FromContainerID(cID cproto.ID) ResourcesID {
	return ResourcesID(cID)
}

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
		return "container exited successfully with a zero exit code"
	}
	return r.Failure.Error()
}

// ResourcesFailure contains information about resources' failure.
type ResourcesFailure struct {
	FailureType FailureType
	ErrMsg      string
	ExitCode    *ExitCode
}

// NewResourcesFailure returns a resources failure message wrapping the type, msg and exit code.
func NewResourcesFailure(failureType FailureType, msg string, code ExitCode) *ResourcesFailure {
	return &ResourcesFailure{
		FailureType: failureType,
		ErrMsg:      msg,
		ExitCode:    &code,
	}
}

func (f ResourcesFailure) Error() string {
	if f.ExitCode == nil {
		return fmt.Sprintf("%s: %s", f.FailureType, f.ErrMsg)
	}
	return fmt.Sprintf("%s: %s (exit code %d)", f.FailureType, f.ErrMsg, *f.ExitCode)
}

// ResourcesStateChanged notifies that the task actor container state has been transitioned.
// It is used by the resource managers to communicate with the task handlers.
type ResourcesStateChanged struct {
	ResourcesID    ResourcesID
	ResourcesState ResourcesState

	ResourcesStarted *ResourcesStarted
	ResourcesStopped *ResourcesStopped

	// More granular information about specific resource types.
	Container *cproto.Container
}

// FromContainerStateChanged converts an aproto.ContainerStateChanged message to
// ResourcesStateChanged.
func FromContainerStateChanged(sc aproto.ContainerStateChanged) ResourcesStateChanged {
	return ResourcesStateChanged{
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
