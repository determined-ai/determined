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
		// The daemon reservations within the allocation.
		daemonReservations map[cproto.ID]bool
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

	// MarkReservationDaemon marks the given reservation as a daemon. In the event of a normal exit,
	// the allocation will not wait for it to exit on its own and instead will kill it and instead
	// await it's hopefully quick termination.
	MarkReservationDaemon struct {
		AllocationID model.AllocationID
		ContainerID  cproto.ID
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
		return a.Terminate(ctx), nil
	case MarkReservationDaemon:
		if err := a.processSetReservationDaemon(msg.AllocationID, msg.ContainerID); err != nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(err)
			} else {
				ctx.Log().WithError(err).Warn("setting daemon")
			}
		}

	case WatchRendezvousInfo, UnwatchRendezvousInfo, RendezvousTimeout:
		switch err := a.rendezvous.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			ctx.Tell(ctx.Self(), model.TrialLog{Message: err.Error()})
		case nil:
		default:
			return nil, errors.Wrap(err, "processing rendezvous")
		}
	case WatchPreemption, UnwatchPreemption, PreemptionTimeout, AckPreemption:
		switch err := a.preemption.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			return a.Kill(ctx), nil
		case nil:
		default:
			return nil, errors.Wrap(err, "processing preemption")
		}
	default:
		return nil, actor.ErrUnexpectedMessage(ctx)
	}
	return nil, nil
}

// ResourcesReleased tears down an allocation.
func (a *Allocation) ResourcesReleased() error {
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		return errors.Wrap(err, "error delete allocation session")
	}
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		return errors.Wrap(err, "failed to mark allocation completed")
	}
	return nil
}

func (a *Allocation) processSetReservationDaemon(aID model.AllocationID, cID cproto.ID) error {
	if aID != a.model.AllocationID {
		return ErrStaleAllocation{aID, a.model.AllocationID}
	}
	if _, ok := a.reservations[cID]; !ok {
		return ErrStaleContainer{ID: cID}
	}
	a.daemonReservations[cID] = true
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
			return a.Kill(ctx), nil
		default:
			return a.Close(ctx), nil
		}
	}
	return nil, nil
}

// AllocationSignal is an interface for signals that can be sent to an allocation.
type AllocationSignal func(ctx *actor.Context) *AllocationExited

// Close attempts to cleanup an allocation while not killing or preempting it.
func (a *Allocation) Close(ctx *actor.Context) *AllocationExited {
	switch {
	case len(a.reservations) == len(a.terminatedContainers):
		return a.terminated(ctx)
	case a.allNonDaemonsExited():
		a.kill(ctx)
	}
	return nil
}

// Terminate attempts to close an allocation by gracefully stopping it (though a kill are possible).
func (a *Allocation) Terminate(ctx *actor.Context) *AllocationExited {
	if exit := a.Close(ctx); exit != nil {
		return exit
	}

	switch {
	case a.rendezvous.ready():
		a.preempt(ctx)
	default:
		a.kill(ctx)
	}
	return nil
}

// Kill attempts to close an allocation by killing it.
func (a *Allocation) Kill(ctx *actor.Context) *AllocationExited {
	if exit := a.Close(ctx); exit != nil {
		return exit
	}

	a.kill(ctx)
	return nil
}

func (a *Allocation) allNonDaemonsExited() bool {
	for id := range a.reservations {
		_, terminated := a.terminatedContainers[id]
		_, daemon := a.daemonReservations[id]
		if !(terminated || daemon) {
			return false
		}
	}
	return true
}

func (a *Allocation) preempt(ctx *actor.Context) {
	ctx.Log().Info("decided to gracefully terminate allocation")
	a.preemption.Preempt()
	ctx.Tell(ctx.Self(), PreemptionTimeout{a.model.AllocationID})
}

func (a *Allocation) kill(ctx *actor.Context) {
	if a.killCooldown != nil && time.Now().UTC().Before(*a.killCooldown) {
		ctx.Log().Debug("still inside of kill cooldown")
		return
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
			// Questionable, could be considered failures, but for now we don't.
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
