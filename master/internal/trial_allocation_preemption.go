package internal

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
	// watchPreemption begins watching if the task has been preempted.
	// The task responds to this message with a channel of bools, where sends of true
	// indicate to preempt and sends of false are used to synchronize (e.g. you want to
	// block until you receive _something_ but not until the first preemption).
	watchPreemption struct {
		allocationID model.AllocationID
		id           uuid.UUID
	}
	preemptionWatcher struct{ C <-chan struct{} }
	unwatchPreemption struct{ id uuid.UUID }
	ackPreemption     struct{ allocationID model.AllocationID }
	// preemptionTimeout is the time after which we forcibly terminate a trial that has no
	// preempted.
	preemptionTimeout struct{ allocationID model.AllocationID }

	// preemption represents the preemption status of a task. A task is assumed to be preempted
	// exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption. This is nice until we move to trial has many task actors / generic task actor,
	// where the lifetime of a "preemption" is equivalent to the lifetime of task and they can be
	// initialized together.
	preemption struct {
		allocationID model.AllocationID
		preempted    bool
		acked        bool
		preemptedAt  time.Time
		// Map of watcher AllocationID to a bool indicating if the trial should preempt.
		watchers map[uuid.UUID]chan<- struct{}
	}
)

func newPreemption(taskID model.AllocationID) preemption {
	return preemption{
		allocationID: taskID,
		preempted:    false,
		acked:        false,
		watchers:     map[uuid.UUID]chan<- struct{}{},
	}
}

func (p *preemption) process(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case watchPreemption:
		if w, err := p.watch(msg.allocationID, msg.id); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(w)
		}
	case unwatchPreemption:
		p.unwatch(msg.id)
	case preemptionTimeout:
		if err := p.checkTimeout(msg.allocationID); err != nil {
			return errTimeoutExceeded{
				message: fmt.Sprintf("preemption did not complete in %s", preemptionTimeoutDuration)}
		}
	case ackPreemption:
		if err := p.acknowledge(msg.allocationID); err != nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(err)
			}
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (p *preemption) watch(taskID model.AllocationID, id uuid.UUID) (preemptionWatcher, error) {
	if p.allocationID != taskID {
		return preemptionWatcher{}, errStaleTask{received: taskID, actual: p.allocationID}
	}

	// Size 1; at most a single message can be sent and we don't want to block.
	w := make(chan struct{}, 1)
	p.watchers[id] = w

	if p.preempted {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}

	return preemptionWatcher{C: w}, nil
}

func (p *preemption) unwatch(id uuid.UUID) {
	if p == nil {
		return
	}
	delete(p.watchers, id)
}

func (p *preemption) preempt() {
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

func (p *preemption) acknowledge(taskID model.AllocationID) error {
	if p == nil {
		return errNoPreemptionStatus
	}
	if p.allocationID != taskID {
		return errStaleTask{received: taskID, actual: p.allocationID}
	}

	p.acked = true
	return nil
}

func (p *preemption) acknowledged() bool {
	if p == nil {
		return false
	}

	return p.acked
}

func (p *preemption) checkTimeout(taskID model.AllocationID) error {
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

func (p *preemption) close() {
	if p == nil {
		return
	}
	p.preempt()
}
