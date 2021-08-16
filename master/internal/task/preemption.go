package task

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
)

var (
	preemptionTimeoutDuration = time.Hour
	errNoPreemptionStatus     = errors.New("no preemption status available for unallocated task")
)

type (
	// WatchPreemption begins watching if the task has been preempted.
	// The task responds to this message with a channel of bools, where sends of true
	// indicate to preempt and sends of false are used to synchronize (e.g. you want to
	// block until you receive _something_ but not until the first preemption).
	WatchPreemption struct {
		AllocationID model.AllocationID
		ID           uuid.UUID
	}
	// PreemptionWatcher contains a channel which can be polled for a preemption signal.
	PreemptionWatcher struct{ C <-chan struct{} }
	// UnwatchPreemption removes a preemption watcher.
	UnwatchPreemption struct{ ID uuid.UUID }
	// AckPreemption acknowledges the receipt of a preemption signal.
	AckPreemption struct{ AllocationID model.AllocationID }
	// PreemptionTimeout is the time after which we forcibly terminate a trial that has no
	// preempted.
	PreemptionTimeout struct{ allocationID model.AllocationID }

	// Preemption represents the preemption status of an allocation. An alllocation is assumed to be
	// preempted exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption. This is nice until we move to trial has many task actors / generic task actor, where
	// the lifetime of a "preemption" is equivalent to the lifetime of allocation and they can be
	// initialized together.
	Preemption struct {
		allocationID model.AllocationID
		preempted    bool
		acked        bool
		preemptedAt  time.Time
		// Map of watcher AllocationID to a bool indicating if the trial should preempt.
		watchers map[uuid.UUID]chan<- struct{}
	}
)

// NewPreemption returns a new preemption struct.
func NewPreemption(allocationID model.AllocationID) Preemption {
	return Preemption{
		allocationID: allocationID,
		preempted:    false,
		acked:        false,
		watchers:     map[uuid.UUID]chan<- struct{}{},
	}
}

// Receive implements actor.Actor.
func (p *Preemption) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case WatchPreemption:
		if w, err := p.Watch(msg.AllocationID, msg.ID); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(w)
		}
	case UnwatchPreemption:
		p.Unwatch(msg.ID)
	case PreemptionTimeout:
		if err := p.CheckTimeout(msg.allocationID); err != nil {
			return ErrTimeoutExceeded{
				Message: fmt.Sprintf("preemption did not complete in %s", preemptionTimeoutDuration)}
		}
	case AckPreemption:
		if err := p.Acknowledge(msg.AllocationID); err != nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(err)
			}
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// Watch sets a watcher up to listen for preemption signals and returns it.
func (p *Preemption) Watch(
	allocationID model.AllocationID, id uuid.UUID) (PreemptionWatcher, error) {
	if p.allocationID != allocationID {
		return PreemptionWatcher{}, ErrStaleAllocation{Received: allocationID, Actual: p.allocationID}
	}

	// Size 1; at most a single message can be sent and we don't want to block.
	w := make(chan struct{}, 1)
	p.watchers[id] = w

	if p.preempted {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}

	return PreemptionWatcher{C: w}, nil
}

// Unwatch unregisters a preemption watcher.
func (p *Preemption) Unwatch(id uuid.UUID) {
	if p == nil {
		return
	}
	delete(p.watchers, id)
}

// Preempt preempts all watchers and sets the allocation as preempted for all future.
func (p *Preemption) Preempt() {
	if p == nil {
		return
	}
	p.preempted = true
	p.preemptedAt = time.Now()
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
}

// Acknowledge acknowledges preemption.
func (p *Preemption) Acknowledge(taskID model.AllocationID) error {
	if p == nil {
		return errNoPreemptionStatus
	}
	if p.allocationID != taskID {
		return ErrStaleAllocation{Received: taskID, Actual: p.allocationID}
	}

	p.acked = true
	return nil
}

// Acknowledged returns if preemption has been acknowledged.
func (p *Preemption) Acknowledged() bool {
	if p == nil {
		return false
	}

	return p.acked
}

// CheckTimeout checks the preemption deadline and returns an error if exceeded.
func (p *Preemption) CheckTimeout(taskID model.AllocationID) error {
	if p == nil {
		return nil
	}
	if p.allocationID != taskID {
		return nil
	}

	if time.Now().After(p.preemptedAt.Add(preemptionTimeoutDuration)) {
		return errors.New("preemption timeout out")
	}
	return nil
}

// Close closes the preemption object.
func (p *Preemption) Close() {
	if p == nil {
		return
	}
	p.Preempt()
}
