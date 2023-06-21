package task

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

// ErrTimeoutExceeded is return, with a bit of detail, when a timeout is exceeded.
type ErrTimeoutExceeded struct {
	Message string
}

func (e ErrTimeoutExceeded) Error() string {
	return fmt.Sprintf("timeout exceeded: %s", e.Message)
}

// ErrNoAllocation is returned an operation is tried without a requested allocation.
type ErrNoAllocation struct {
	Action string
}

func (e ErrNoAllocation) Error() string {
	return fmt.Sprintf("%s not valid without requested allocation", e.Action)
}

// ErrAllocationUnfulfilled is returned an operation is tried without an active allocation.
type ErrAllocationUnfulfilled struct {
	Action string
}

func (e ErrAllocationUnfulfilled) Error() string {
	return fmt.Sprintf("%s not valid without active allocation", e.Action)
}

// ErrStaleResourcesReceived is returned the scheduler gives an allocation resources between
// when it requests them and it deciding, for some reason or another, they are not needed.
type ErrStaleResourcesReceived struct{}

func (e ErrStaleResourcesReceived) Error() string {
	return "allocation no longer needs these resources"
}

// ErrStaleAllocation is returned when an operation was attempted by a stale task.
type ErrStaleAllocation struct {
	Received, Actual model.AllocationID
}

func (e ErrStaleAllocation) Error() string {
	return fmt.Sprintf("stale task %s != %s (received != actual)", e.Received, e.Actual)
}

// ErrStaleContainer is returned when an operation was attempted by a stale container.
type ErrStaleContainer struct {
	ID cproto.ID
}

func (e ErrStaleContainer) Error() string {
	return fmt.Sprintf("stale container %s", e.ID)
}

// ErrStaleResources is returned when an operation was attempted by a stale resources.
type ErrStaleResources struct {
	ID sproto.ResourcesID
}

func (e ErrStaleResources) Error() string {
	return fmt.Sprintf("stale resources %s", e.ID)
}

// All behaviors for allocations.
const (
	idleWatcher = "idle_watcher"
)

// ErrBehaviorDisabled is returned an operation is tried without the behavior being enabled.
type ErrBehaviorDisabled struct {
	Behavior string
}

func (e ErrBehaviorDisabled) Error() string {
	return fmt.Sprintf("%s not enabled for this allocation", e.Behavior)
}

// ErrBehaviorUnsupported is returned an operation is tried without the behavior being supported.
type ErrBehaviorUnsupported struct {
	Behavior string
}

func (e ErrBehaviorUnsupported) Error() string {
	return fmt.Sprintf("%s not supported for this allocation or resource manager", e.Behavior)
}

// ErrAlreadyCancelled is returned to the allocation when it tries to take an action but has an
// unread cancellation in its inbox.
type ErrAlreadyCancelled struct{}

func (e ErrAlreadyCancelled) Error() string {
	return "the allocation was canceled while this message was waiting"
}
