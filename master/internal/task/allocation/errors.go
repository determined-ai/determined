package allocation

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/model"
)

var errUserRequestedStop = errors.New("user requested stop")

// ErrPreemptionTimeoutExceeded is return, with a bit of detail, when a timeout is exceeded.
var ErrPreemptionTimeoutExceeded = fmt.Errorf("preemption did not complete in %s", preemptionTimeoutDuration)

var ErrRendezvousBadRequest = fmt.Errorf("a rendezvous request was made out of order, " +
	"e.g., unwatch called before watch")

var ErrRendezvousTimeoutExceeded = errors.New("some containers are taking a long time to " +
	"connect to master; when running on kubernetes this may happen " +
	"because only some of the pods have been scheduled; it is possible " +
	"that some pods will never be scheduled without adding compute " +
	"resources or pausing / killing other experiments in the cluster")

var ErrAllGatherTimeoutExceeded = errors.New("some ranks are taking a long time to connect to master" +
	"during all gather; when running on kubernetes this may happen " +
	"because only some of the pods have been scheduled; it is possible " +
	"that some pods will never be scheduled without adding compute " +
	"resources or pausing / killing other experiments in the cluster")

var ErrAllocationNotFound = errors.New("no such allocation in the system")

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

// ErrBehaviorUnsupported is returned an operation is tried without the behavior being supported.
// TODO(mar): all these messages are garbage.
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
