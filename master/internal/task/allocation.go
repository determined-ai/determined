package task

import (
	"fmt"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type (
	// Allocation encapsulates all the state of a single allocation
	Allocation struct {
		// System dependencies.
		db db.DB

		// The persisted representation.
		model model.Allocation

		// The spec used to start reservations.
		spec TaskSpecMaker
		// The keys for SSH access to the task.
		keys *ssh.PrivateAndPublicKeys
		// The existence of allocations signifies the trial has been allocated.
		reservations map[cproto.ID]sproto.Reservation
		// The following fields tracks containers and their states.
		containers map[cproto.ID]cproto.Container
		// Tracks the initial container exit, unless we caused the failure by killed the trial.
		terminatedFirst      *cproto.ID
		terminatedContainers map[cproto.ID]sproto.TaskContainerStopped
		// Encapsulates the preemption state of the currently allocated task.
		// If there is no current task, or it is unallocated, it is nil.
		preemption Preemption
		// Encapsulates logic of rendezvousing containers of the currently
		// allocated task. If there is no current task, or it is unallocated, it is nil.
		rendezvous Rendezvous
		// Marks that we intentionally killed the allocation so we can know to
		// ignore any errors from containers dying. Not set when we kill an already
		// terminating trial.
		killedWhileRunning bool
		// We send a kill when we terminate a task forcibly. we terminate forcibly when a container
		// exits non zero. we don't need to send all these kills, so this exists.
		killCooldown *time.Time
	}

	// AllocationExited summarizes the exit status of an allocation.
	AllocationExited struct {
		// userRequestedStop is when a container unexpectedly exits with 0.
		UserRequestedStop bool
		Err               error
	}
)

const killCooldown = 30 * time.Second

// NewAllocation returns a new allocation, which tracks allocation state in a fairly generic way.
func NewAllocation(
	ctx *actor.Context, taskID model.TaskID, req sproto.AllocateRequest,
	reservations []sproto.Reservation, spec TaskSpecMaker, keys *ssh.PrivateAndPublicKeys, db db.DB,
) (*Allocation, error) {
	containerIDToReservation := map[cproto.ID]sproto.Reservation{}
	for _, a := range reservations {
		containerIDToReservation[a.Summary().ID] = a
	}
	a := &Allocation{
		db: db,

		model: model.Allocation{
			TaskID:       taskID,
			AllocationID: req.AllocationID,
			ResourcePool: req.ResourcePool,
			StartTime:    time.Now().UTC(),
		},

		reservations:         containerIDToReservation,
		spec:                 spec,
		keys:                 keys,
		preemption:           NewPreemption(req.AllocationID),
		rendezvous:           NewRendezvous(req.AllocationID, ranksFromReservations(reservations)),
		containers:           make(map[cproto.ID]cproto.Container),
		terminatedContainers: make(map[cproto.ID]sproto.TaskContainerStopped),
	}

	if err := a.db.AddAllocation(&a.model); err != nil {
		return nil, errors.Wrap(err, "saving trial allocation")
	}
	token, err := a.db.StartAllocationSession(a.model.AllocationID)
	if err != nil {
		return nil, errors.Wrap(err, "starting a new allocation session")
	}

	for cID, r := range a.reservations {
		r.Start(ctx, a.spec.ToTaskSpec(a.keys, token), a.rendezvous.ranks[cID])
	}
	actors.NotifyAfter(ctx, RendezvousTimeoutDuration, RendezvousTimeout{
		AllocationID: a.model.AllocationID,
	})

	return a, nil
}

// TaskSpecMaker an interface for anything that creates task specs.
type TaskSpecMaker interface {
	ToTaskSpec(keys *ssh.PrivateAndPublicKeys, allocationToken string) tasks.TaskSpec
}

// HandleEvent receives a message for this allocation.
func (a *Allocation) HandleEvent(ctx *actor.Context) (*AllocationExited, error) {
	switch msg := ctx.Message().(type) {
	case sproto.TaskContainerStateChanged:
		return a.processContainerMessage(ctx, msg)
	case sproto.ReleaseResources:
		return a.Terminate(ctx, Graceful)
	case WatchRendezvousInfo, UnwatchRendezvousInfo, RendezvousTimeout:
		switch err := a.rendezvous.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			ctx.Tell(ctx.Self(), model.TrialLog{Message: err.Error()})
		case nil:
		default:
			return nil, errors.Wrap(err, "processing rendezvous")
		}
		return nil, nil
	case WatchPreemption, UnwatchPreemption, PreemptionTimeout, AckPreemption:
		switch err := a.preemption.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			ctx.Log().WithError(err).Errorf("forcibly terminating trial")
			return a.Terminate(ctx, Kill)
		case nil:
		default:
			return nil, errors.Wrap(err, "processing preemption")
		}
		return nil, nil
	default:
		return nil, actor.ErrUnexpectedMessage(ctx)
	}
}

// Close tears down an allocation.
func (a *Allocation) Close(ctx *actor.Context) error {
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		return errors.Wrap(err, "error delete allocation session")
	}
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		return errors.Wrap(err, "failed to mark allocation completed")
	}
	return nil
}

func (a *Allocation) processContainerMessage(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) (*AllocationExited, error) {
	if _, ok := a.reservations[msg.Container.ID]; !ok {
		return nil, ErrStaleContainer{ID: msg.Container.ID}
	}

	a.containers[msg.Container.ID] = msg.Container
	rank := a.rendezvous.rank(msg.Container.ID)
	ctx.Log().Infof("container %s (rank %d) is %s", msg.Container.ID, rank, msg.Container.State)
	switch msg.Container.State {
	case cproto.Running:
		a.rendezvous.containerStarted(msg.Container.ID, msg.ContainerStarted.Addresses)
		if a.rendezvous.ready() {
			ctx.Log().Info("all containers are connected successfully (task container state changed)")
		}
	case cproto.Terminated:
		ctx.Tell(ctx.Self(), model.TrialLog{
			Message:     msg.ContainerStopped.String(),
			ContainerID: ptrs.StringPtr(string(msg.Container.ID)),
		})

		a.terminatedContainers[msg.Container.ID] = *msg.ContainerStopped
		a.rendezvous.containerTerminated(msg.Container.ID)
		if a.terminatedFirst == nil {
			a.terminatedFirst = &msg.Container.ID
		}

		switch {
		case msg.ContainerStopped.Failure != nil:
			return a.Terminate(ctx, Kill)
		default:
			return a.Terminate(ctx, Noop)
		}
	}
	return nil, nil
}

// TerminationType controls the way in which an allocation is terminated.
type TerminationType string

const (
	// Kill is used to forcibly halt a trial. calling this will kill existing allocations
	// and exit. terminate is re-entered after a kill when all containers have stopped.
	Kill TerminationType = "kill"
	// Graceful is used to gracefully halt a trial. calling this will (usually, with the exception
	// of unready trials) send a preemption signal to all watchers and begin a timeout after which
	// we forcibly kill the trial.
	Graceful TerminationType = "graceful"
	// Noop is used to try to move a trial to a terminal state while taking no direct action on it.
	// e.g., if the searcher tells us it's done, we either should exit right away if we're unallocated,
	// or just chill and wait for the active task to exit.
	Noop TerminationType = "noop"
)

// Terminate encapsulates all termination logic for an allocation.
//
// It just exists to translate caller desires "kill this allocation, preempt this allocation" to
// the corresponding action to actually take based on our state, instead of each caller needing
// to be aware of how to take certain actions in certain states.
func (a *Allocation) Terminate(ctx *actor.Context, tt TerminationType) (*AllocationExited, error) {
	switch {
	case len(a.reservations) == len(a.terminatedContainers):
		ctx.Log().Info("decided to close allocation because all containers have exited")
		return a.terminated(ctx), nil
	case tt == Noop, tt == Graceful && len(a.terminatedContainers) > 0:
		// Working on it.
	case tt == Graceful && a.rendezvous.ready():
		ctx.Log().Info("decided to gracefully terminate allocation")
		a.preemption.Preempt()
		ctx.Tell(ctx.Self(), PreemptionTimeout{a.model.AllocationID})
	default:
		if a.killCooldown != nil && time.Now().UTC().Before(*a.killCooldown) {
			ctx.Log().Debug("still inside of kill cooldown")
			return nil, nil
		}

		ctx.Log().Info("decided to kill allocation")
		if a.terminatedFirst == nil {
			a.killedWhileRunning = true
		}
		a.killCooldown = ptrs.TimePtr(time.Now().UTC().Add(killCooldown))
		for _, reservation := range a.reservations {
			reservation.Kill(ctx)
		}
	}
	return nil, nil
}

// terminated decides what action to take to close or restart a trial's task. This is only
// called once the current task is cleaned up and we're ready to move on.
func (a *Allocation) terminated(ctx *actor.Context) *AllocationExited {
	defer a.preemption.Close()
	defer a.rendezvous.close()

	switch {
	case a.killedWhileRunning:
		ctx.Log().Info("allocation successfully killed")
		return &AllocationExited{}
	case a.preemption.Acknowledged():
		ctx.Log().Info("allocated successfully preempted")
		return &AllocationExited{}
	case a.terminatedFirst != nil:
		err := a.terminatedContainers[*a.terminatedFirst].Failure
		if err == nil {
			// This is true because searcher and preemption exits both ack preemption.
			return &AllocationExited{
				UserRequestedStop: true,
			}
		}

		switch err.FailureType {
		case aproto.ContainerFailed, aproto.TaskError:
			ctx.Log().WithError(err).Infof("allocation exited with failure (%s)", err.FailureType)
			return &AllocationExited{Err: err}
		case aproto.AgentError, aproto.AgentFailed:
			// Questionable, could be considered failures.
			ctx.Log().WithError(err).Warnf("allocation exited due to agent (%s)", err.FailureType)
			return &AllocationExited{}
		case aproto.TaskAborted:
			// Definitely not a failure.
			ctx.Log().WithError(err).Debugf("allocation aborted (%s)", err.FailureType)
			return &AllocationExited{}
		default:
			panic(errors.Wrapf(err, "unexpected allocation failure (%s)", err.FailureType))
		}
	default:
		panic("allocation exited without being killed, preempted or having a container exit")
	}
}

// ErrTimeoutExceeded is return, with a bit of detail, when a timeout is exceeded.
type ErrTimeoutExceeded struct {
	Message string
}

func (e ErrTimeoutExceeded) Error() string {
	return fmt.Sprintf("timeout exceeded: %s", e.Message)
}

// ErrNoAllocation is returned an operation is tried without an active allocation.
type ErrNoAllocation struct {
	Action string
}

func (e ErrNoAllocation) Error() string {
	return fmt.Sprintf("%s not valid without active allocation", e.Action)
}

// ErrStaleTask is returned when an operation was attempted by a stale task.
type ErrStaleTask struct {
	Received, Actual model.AllocationID
}

func (e ErrStaleTask) Error() string {
	return fmt.Sprintf("stale task %s != %s (received != actual)", e.Received, e.Actual)
}

// ErrStaleContainer is returned when an operation was attempted by a stale container.
type ErrStaleContainer struct {
	ID cproto.ID
}

func (e ErrStaleContainer) Error() string {
	return fmt.Sprintf("stale container %s", e.ID)
}
