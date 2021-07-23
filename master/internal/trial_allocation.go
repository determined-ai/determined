package internal

import (
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type (
	// trialAllocation encapsulates all the state of a single trial allocation
	trialAllocation struct {
		// the request associated with this allocation
		req sproto.AllocateRequest
		// The existence of allocations signifies the trial has been allocated.
		reservations map[cproto.ID]sproto.Reservation
		// The following fields tracks containers and their states.
		containers           map[cproto.ID]cproto.Container
		terminatedFirst      *cproto.ID
		terminatedContainers map[cproto.ID]sproto.TaskContainerStopped
		// preemption encapsulates the preemption state of the currently allocated task.
		// If there is no current task, or it is unallocated, it is nil.
		preemption preemption
		// rendezvous encapsulates logic of rendezvousing containers of the currently
		// allocated task. If there is no current task, or it is unallocated, it is nil.
		rendezvous rendezvous
		// killed marks that we have intentionally killed the trial, so we can know to ignore
		// any errors from containers dying.
		killed bool
		// we send a kill when we terminate a task forcibly. we terminate forcibly when a container
		// exits non zero. we don't need to send all these kills, so this exists.
		killCooldown *time.Time
	}

	// AllocationExitStatus summarized the exit status of an allocation.
	AllocationExitStatus struct {
		// userRequestedStop when a container unexpectedly exits with 0.
		userRequestedStop bool
		err               error
	}
)

const killCooldown = 30 * time.Second

func newTrialAllocation(
	req sproto.AllocateRequest, reservations []sproto.Reservation,
) *trialAllocation {
	containerIDToReservation := map[cproto.ID]sproto.Reservation{}
	for _, a := range reservations {
		containerIDToReservation[a.Summary().ID] = a
	}
	return &trialAllocation{
		req:                  req,
		reservations:         containerIDToReservation,
		preemption:           newPreemption(req.AllocationID),
		rendezvous:           newRendezvous(req.AllocationID, ranksFromReservations(reservations)),
		containers:           make(map[cproto.ID]cproto.Container),
		terminatedContainers: make(map[cproto.ID]sproto.TaskContainerStopped),
	}
}

func (t *trialAllocation) process(ctx *actor.Context) (*AllocationExitStatus, error) {
	switch msg := ctx.Message().(type) {
	case sproto.TaskContainerStateChanged:
		return t.processContainerMessage(ctx, msg)
	case sproto.ReleaseResources:
		return t.terminate(ctx, preempt)
	case watchRendezvousInfo, unwatchRendezvousInfo, rendezvousTimeout:
		switch err := t.rendezvous.process(ctx).(type) {
		case errTimeoutExceeded:
			ctx.Tell(ctx.Self(), model.TrialLog{Message: err.Error()})
		case nil:
		default:
			return nil, errors.Wrap(err, "processing rendezvous")
		}
		return nil, nil
	case watchPreemption, unwatchPreemption, preemptionTimeout, ackPreemption:
		switch err := t.preemption.process(ctx).(type) {
		case errTimeoutExceeded:
			ctx.Log().WithError(err).Errorf("forcibly terminating trial")
			return t.terminate(ctx, kill)
		case nil:
		default:
			return nil, errors.Wrap(err, "processing preemption")
		}
		return nil, nil
	default:
		return nil, actor.ErrUnexpectedMessage(ctx)
	}
}

func (t *trialAllocation) processContainerMessage(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) (*AllocationExitStatus, error) {
	if _, ok := t.reservations[msg.Container.ID]; !ok {
		return nil, errStaleContainer{id: msg.Container.ID}
	}

	t.containers[msg.Container.ID] = msg.Container
	rank := t.rendezvous.rank(msg.Container.ID)
	ctx.Log().Infof("container %s (rank %d) is %s", msg.Container.ID, rank, msg.Container.State)
	switch msg.Container.State {
	case cproto.Running:
		t.rendezvous.containerStarted(msg.Container.ID, msg.ContainerStarted.Addresses)
		if t.rendezvous.ready() {
			ctx.Log().Info("all containers are connected successfully (task container state changed)")
		}
	case cproto.Terminated:
		ctx.Tell(ctx.Self(), model.TrialLog{
			Message:     msg.ContainerStopped.String(),
			ContainerID: ptrs.StringPtr(string(msg.Container.ID)),
		})

		t.terminatedContainers[msg.Container.ID] = *msg.ContainerStopped
		t.rendezvous.containerTerminated(msg.Container.ID)
		if t.terminatedFirst == nil {
			t.terminatedFirst = &msg.Container.ID
		}

		switch {
		case msg.ContainerStopped.Failure != nil:
			return t.terminate(ctx, kill)
		default:
			return t.terminate(ctx, noop)
		}
	}
	return nil, nil
}

type terminationType string

const (
	// kill is used to forcibly halt a trial. calling this will kill existing allocations
	// and exit. terminate is re-entered after a kill when all containers have stopped.
	kill terminationType = "kill"
	// preempt is used to gracefully halt a trial. calling this will (usually, with the exception
	// of unready trials) send a preemption signal to all watchers and begin a timeout after which
	// we forcibly kill the trial.
	preempt terminationType = "preempt"
	// noop is used to try to move a trial to a terminal state while taking no direct action on it.
	// e.g., if the searcher tells us it's done, we either should exit right away if we're unallocated,
	// or just chill and wait for the active task to exit.
	noop terminationType = "noop"
)

// terminate encapsulates all termination logic for the trial's allocation.
//
// It just exists to translate caller desires "kill this task, preempt this task" to the
// corresponding action to actually take based on our state, instead of each caller needing
// to be aware of how to take certain actions in certain states.
func (t *trialAllocation) terminate(
	ctx *actor.Context, tt terminationType,
) (*AllocationExitStatus, error) {
	switch {
	case len(t.reservations) == len(t.terminatedContainers):
		ctx.Log().Info("terminating trial because all containers have exited")
		exitStatus := t.terminated(ctx)
		return &exitStatus, nil
	case tt == noop, tt == preempt && len(t.terminatedContainers) > 0:
		// Working on it.
		return nil, nil
	case tt == preempt && t.rendezvous.ready():
		ctx.Log().Info("gracefully terminating trial")
		t.preemption.preempt()
		ctx.Tell(ctx.Self(), preemptionTimeout{t.req.AllocationID})
		return nil, nil
	default:
		if t.killCooldown != nil && time.Now().UTC().Before(*t.killCooldown) {
			ctx.Log().Debug("still inside of kill cooldown")
			return nil, nil
		}

		ctx.Log().Info("forcibly terminating trial")
		t.killed = true
		t.killCooldown = ptrs.TimePtr(time.Now().UTC().Add(killCooldown))
		for _, reservation := range t.reservations {
			reservation.Kill(ctx)
		}
		return nil, nil
	}
}

// terminated decides what action to take to close or restart a trial's task. This is only
// called once the current task is cleaned up and we're ready to move on.
func (t *trialAllocation) terminated(ctx *actor.Context) AllocationExitStatus {
	ctx.Log().
		WithField("preempt_ack", t.preemption.acknowledged()).
		WithField("killed", t.killed).
		Info("trial task terminated")

	defer t.preemption.close()
	defer t.rendezvous.close()

	switch status := t.ExitStatus(); {
	case t.killed:
		return AllocationExitStatus{}
	case status.Failure != nil:
		switch status.Failure.FailureType {
		case aproto.ContainerFailed, aproto.TaskError:
			return AllocationExitStatus{err: status.Failure}
		case aproto.AgentError, aproto.AgentFailed:
			// Questionable, could be considered failures.
		case aproto.TaskAborted:
			// Definitely not a failure.
		}
		return AllocationExitStatus{}
	case t.preemption.acknowledged():
		return AllocationExitStatus{}
	default:
		return AllocationExitStatus{
			userRequestedStop: true,
		}
	}
}

func (t *trialAllocation) ExitStatus() aproto.ContainerStopped {
	anyStarted := func(cs map[cproto.ID]cproto.Container) bool {
		for _, c := range cs {
			if c.State != cproto.Assigned {
				return true
			}
		}
		return false
	}

	if !anyStarted(t.containers) {
		return aproto.ContainerError(aproto.TaskAborted, errors.New("task aborted"))
	}
	if t.terminatedFirst != nil {
		return t.terminatedContainers[*t.terminatedFirst].ContainerStopped
	}
	return aproto.ContainerError(aproto.AgentError, errors.New("no error status provided"))
}

type errTimeoutExceeded struct {
	message string
}

func (e errTimeoutExceeded) Error() string {
	return fmt.Sprintf("timeout exceeded: %s", e.message)
}

type errNoAllocation struct {
	action string
}

func (e errNoAllocation) Error() string {
	return fmt.Sprintf("%s not valid without active allocation", e.action)
}

type errStaleTask struct {
	received, actual model.AllocationID
}

func (e errStaleTask) Error() string {
	return fmt.Sprintf("stale task %s != %s (received != actual)", e.received, e.actual)
}

type errStaleContainer struct {
	id cproto.ID
}

func (e errStaleContainer) Error() string {
	return fmt.Sprintf("stale container %s", e.id)
}
