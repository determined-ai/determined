package task

import (
	"fmt"

	cproto "github.com/determined-ai/determined/master/pkg/container"
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

// All behaviors for allocations.
const (
	rendezvous  = "rendezvous"
	preemption  = "preemption"
	idleWatcher = "idle_watcher"
)

// ErrBehaviorDisabled is returned an operation is tried without the behavior being enabled.
type ErrBehaviorDisabled struct {
	Behavior string
}

func (e ErrBehaviorDisabled) Error() string {
	return fmt.Sprintf("%s not enabled for this allocation", e.Behavior)
}
