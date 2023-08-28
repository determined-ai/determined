package aproto

import (
	"fmt"

	"github.com/pkg/errors"
)

// ContainerFailure holds the reason why a container did not complete successfully.
type ContainerFailure struct {
	FailureType FailureType
	ErrMsg      string
	ExitCode    *ExitCode
}

func (c ContainerFailure) Error() string {
	if c.ExitCode == nil {
		return fmt.Sprintf("%s: %s", c.FailureType, c.ErrMsg)
	}
	return fmt.Sprintf("%s: %s (exit code %d)", c.FailureType, c.ErrMsg, *c.ExitCode)
}

// ExitCode is the process exit code of the container.
type ExitCode int

const (
	// SuccessExitCode is the 0 zero value exit code.
	SuccessExitCode = 0
)

// ContainerError returns a container failure wrapping the provided error. If the error is nil,
// a stack trace is provided instead.
func ContainerError(failureType FailureType, err error) ContainerStopped {
	if err == nil {
		return ContainerStopped{
			Failure: &ContainerFailure{
				FailureType: failureType,
				ErrMsg:      errors.WithStack(errors.Errorf("unknown error occurred")).Error(),
			},
		}
	}
	return ContainerStopped{
		Failure: &ContainerFailure{
			FailureType: failureType,
			ErrMsg:      err.Error(),
		},
	}
}

// NewContainerFailure returns a container failure wrapping the provided error. If the error is nil,
// a stack trace is provided instead.
func NewContainerFailure(failureType FailureType, err error) *ContainerFailure {
	if err == nil {
		return &ContainerFailure{
			FailureType: failureType,
			ErrMsg:      errors.WithStack(errors.Errorf("unknown error occurred")).Error(),
		}
	}
	return &ContainerFailure{
		FailureType: failureType,
		ErrMsg:      err.Error(),
	}
}

// NewContainerExit returns a container failure with the encoded exit code. If the exit code is a
// the zero value, no failure is returned.
func NewContainerExit(code ExitCode) *ContainerFailure {
	if code == SuccessExitCode {
		return nil
	}
	return &ContainerFailure{
		FailureType: ContainerFailed,
		ErrMsg:      errors.Errorf("%s: %s", ContainerFailed, code).Error(),
		ExitCode:    &code,
	}
}

// ContainerExited returns a container failure with the encoded exit code. If the exit code is a
// the zero value, no failure is returned.
func ContainerExited(code ExitCode) ContainerStopped {
	if code == SuccessExitCode {
		return ContainerStopped{}
	}
	return ContainerStopped{
		&ContainerFailure{
			FailureType: ContainerFailed,
			ErrMsg:      errors.Errorf("%s: %d", ContainerFailed, code).Error(),
			ExitCode:    &code,
		},
	}
}

// FailureType denotes the type of failure that resulted in the container stopping.
// Each FailureType must be handled by ./internal/task/allocation.go.
type FailureType string

const (
	// ContainerFailed denotes that the container ran but failed with a non-zero exit code.
	ContainerFailed FailureType = "container failed with non-zero exit code"

	// ContainerAborted denotes the container was canceled before it was started.
	ContainerAborted FailureType = "container was aborted before it started"

	// ContainerMissing denotes the container was missing when the master asked about it.
	ContainerMissing FailureType = "request for action on unknown container"

	// TaskAborted denotes that the task was canceled before it was started.
	TaskAborted FailureType = "task was aborted before the task was started"

	// TaskError denotes that the task failed without an associated exit code.
	TaskError FailureType = "task failed without an associated exit code"

	// AgentFailed denotes that the agent failed while the container was running.
	AgentFailed FailureType = "agent failed while the container was running"

	// RestoreError denotes that we failed to restore the container after some agent failure.
	RestoreError FailureType = "container failed to restore after agent failure"

	// AgentError denotes that the agent failed to launch the container.
	AgentError FailureType = "agent failed to launch the container"
)
