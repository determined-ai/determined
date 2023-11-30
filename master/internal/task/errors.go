package task

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
)

// TimeoutExceededError is return, with a bit of detail, when a timeout is exceeded.
type TimeoutExceededError struct {
	Message string
}

func (e TimeoutExceededError) Error() string {
	return fmt.Sprintf("timeout exceeded: %s", e.Message)
}

// NoAllocationError is returned an operation is tried without a requested allocation.
type NoAllocationError struct {
	Action string
}

func (e NoAllocationError) Error() string {
	return fmt.Sprintf("%s not valid without requested allocation", e.Action)
}

// AllocationUnfulfilledError is returned an operation is tried without an active allocation.
type AllocationUnfulfilledError struct {
	Action string
}

func (e AllocationUnfulfilledError) Error() string {
	return fmt.Sprintf("%s not valid without active allocation", e.Action)
}

// StaleResourcesReceivedError is returned the scheduler gives an allocation resources between
// when it requests them and it deciding, for some reason or another, they are not needed.
type StaleResourcesReceivedError struct{}

func (e StaleResourcesReceivedError) Error() string {
	return "allocation no longer needs these resources"
}

// StaleContainerError is returned when an operation was attempted by a stale container.
type StaleContainerError struct {
	ID cproto.ID
}

func (e StaleContainerError) Error() string {
	return fmt.Sprintf("stale container %s", e.ID)
}

// StaleResourcesError is returned when an operation was attempted by a stale resources.
type StaleResourcesError struct {
	ID sproto.ResourcesID
}

func (e StaleResourcesError) Error() string {
	return fmt.Sprintf("stale resources %s", e.ID)
}

// BehaviorDisabledError is returned an operation is tried without the behavior being enabled.
type BehaviorDisabledError struct {
	Behavior string
}

func (e BehaviorDisabledError) Error() string {
	return fmt.Sprintf("%s not enabled for this allocation", e.Behavior)
}

// BehaviorUnsupportedError is returned an operation is tried without the behavior being supported.
type BehaviorUnsupportedError struct {
	Behavior string
}

func (e BehaviorUnsupportedError) Error() string {
	return fmt.Sprintf("%s not supported for this allocation or resource manager", e.Behavior)
}

// AlreadyCancelledError is returned to the allocation when it tries to take an action but has an
// unread cancellation in its inbox.
type AlreadyCancelledError struct{}

func (e AlreadyCancelledError) Error() string {
	return "the allocation was canceled while this message was waiting"
}
