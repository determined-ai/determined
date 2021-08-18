package task

import (
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/internal/db"
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
	// Allocation encapsulates all the state of a single allocation. Eventually, the goal is
	// to reuse it for all allocation types.
	Allocation struct {
		// System dependencies.
		db db.DB
		rm *actor.Ref

		req sproto.AllocateRequest

		// The persisted representation.
		model model.Allocation

		// The existence of allocations signifies the trial has been allocated.
		reservations map[cproto.ID]sproto.Reservation
		// The daemon reservations within the allocation.
		daemonReservations map[cproto.ID]bool
		// The following fields tracks containers and their states.
		containers map[cproto.ID]cproto.Container
		// Tracks the initial container exit, unless we caused the failure by killed the trial.
		firstContainerExited *cproto.ID
		exitedContainers     map[cproto.ID]sproto.TaskContainerStopped
		// Encapsulates the preemption state of the currently allocated task.
		// If there is no current task, or it is unallocated, it is nil.
		preemption *Preemption
		// Encapsulates logic of rendezvousing containers of the currently
		// allocated task. If there is no current task, or it is unallocated, it is nil.
		rendezvous *Rendezvous
		// Marks that we intentionally killed the allocation so we can know to
		// ignore any errors from containers dying. Not set when we kill an already
		// terminating trial.
		killedWhileRunning bool
		// Marks that the trial exited successfully, but we killed some daemon containers.
		killedDaemons bool
		// We send a kill when we terminate a task forcibly. we terminate forcibly when a container
		// exits non zero. we don't need to send all these kills, so this exists.
		killCooldown *time.Time
		// tracks if we have finished termination.
		exited bool
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
	// BuildTaskSpec is a message to request the task spec from the parent task. This
	// is just a hack since building a task spec cant be semi-costly and we want to defer it
	// until it is needed (we save stuff to the DB and make SSH keys, doing this for 10k trials
	// at once is real bad.
	BuildTaskSpec struct{}
	// AllocationSignal is an interface for signals that can be sent to an allocation.
	AllocationSignal string
)

const (
	// Kill is the signal to kill an allocation; analogous to in SIGKILL.
	Kill AllocationSignal = "kill"
	// Terminate is the signal to kill an allocation; analogous to in SIGTERM.
	Terminate AllocationSignal = "terminate"
)

const killCooldown = 30 * time.Second

// NewAllocation returns a new allocation, which tracks allocation state in a fairly generic way.
func NewAllocation(req sproto.AllocateRequest, db db.DB, rm *actor.Ref) actor.Actor {
	return &Allocation{
		db: db,
		rm: rm,

		req: req,
		model: model.Allocation{
			TaskID:       req.TaskID,
			AllocationID: req.AllocationID,
			ResourcePool: req.ResourcePool,
			StartTime:    time.Now().UTC(),
		},

		daemonReservations: map[cproto.ID]bool{},
		reservations:       map[cproto.ID]sproto.Reservation{},
		containers:         make(map[cproto.ID]cproto.Container),
		exitedContainers:   make(map[cproto.ID]sproto.TaskContainerStopped),
	}
}

// Receive implements actor.Actor.
func (a *Allocation) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		return a.RequestResources(ctx)
	case actor.PostStop:
		a.Cleanup(ctx)
	case sproto.ResourcesAllocated:
		return a.ResourcesAllocated(ctx, msg)
	case sproto.TaskContainerStateChanged:
		return a.TaskContainerStateChanged(ctx, msg)
	case sproto.ReleaseResources:
		a.Terminate(ctx)
	case MarkReservationDaemon:
		a.SetReservationAsDaemon(ctx, msg.AllocationID, msg.ContainerID)
	case AllocationSignal:
		a.HandleSignal(ctx, msg)
	case WatchRendezvousInfo, UnwatchRendezvousInfo, RendezvousTimeout:
		switch err := a.rendezvous.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			ctx.Tell(ctx.Self(), model.TrialLog{Message: err.Error()})
		case nil:
		default:
			return err
		}
	case WatchPreemption, UnwatchPreemption, PreemptionTimeout, AckPreemption:
		switch err := a.preemption.Receive(ctx).(type) {
		case ErrTimeoutExceeded:
			a.Kill(ctx)
		case nil:
		default:
			return err
		}
	case sproto.ContainerLog:
		ctx.Tell(ctx.Self().Parent(), a.enrichLog(msg))
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

// RequestResources sets up the allocation.
func (a *Allocation) RequestResources(ctx *actor.Context) error {
	a.req.TaskActor = ctx.Self()
	if err := ctx.Ask(a.rm, a.req).Error(); err != nil {
		return errors.Wrap(err, "failed to request allocation")
	}
	return nil
}

// Cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
func (a *Allocation) Cleanup(ctx *actor.Context) {
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		ctx.Log().WithError(err).Error("error delete allocation session")
	}
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		ctx.Log().WithError(err).Error("failed to mark allocation completed")
	}
	// Just in-case code.
	if !a.exited {
		ctx.Log().Info("exit did not run properly")
		for cID, r := range a.reservations {
			if _, ok := a.exitedContainers[cID]; !ok {
				ctx.Log().Infof("allocation exited with unterminated reservation: %v", r.Summary())
				r.Kill(ctx)
			}
		}
		ctx.Tell(a.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
	}
}

// ResourcesAllocated handles receiving resources from the resource manager. Note: it makes a single
// ask to the parent to build its task spec.. this is mostly a hack to defer lots of computationally
// heavy stuff unless it is necessarily (which also works to spread occurrences of the same work
// out). Eventually, Allocations should just be started with their TaskSpec.
func (a *Allocation) ResourcesAllocated(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	// Get the task spec first, so the trial/task table is populated before allocations.
	resp := ctx.Ask(ctx.Self().Parent(), BuildTaskSpec{})
	if err := resp.Error(); err != nil {
		return errors.Wrapf(err, "could not get task spec")
	}
	spec := *resp.Get().(*tasks.TaskSpec)

	for _, r := range msg.Reservations {
		a.reservations[r.Summary().ID] = r
	}

	if err := a.db.AddAllocation(&a.model); err != nil {
		return errors.Wrap(err, "saving trial allocation")
	}

	token, err := a.db.StartAllocationSession(a.model.AllocationID)
	if err != nil {
		return errors.Wrap(err, "starting a new allocation session")
	}

	a.preemption = NewPreemption(a.model.AllocationID)
	a.rendezvous = NewRendezvous(a.model.AllocationID, ranksFromReservations(msg.Reservations))
	for cID, r := range a.reservations {
		r.Start(ctx, spec, sproto.ReservationRuntimeInfo{
			Token:        token,
			AgentRank:    a.rendezvous.rank(cID),
			IsMultiAgent: len(a.reservations) > 1,
		})
	}
	actors.NotifyAfter(ctx, RendezvousTimeoutDuration, RendezvousTimeout{
		AllocationID: a.model.AllocationID,
	})
	return nil
}

// SetReservationAsDaemon sets the reservation as a daemon reservation. This means we won't wait for
// it to exit in errorless exits and instead will kill the forcibly.
func (a *Allocation) SetReservationAsDaemon(
	ctx *actor.Context, aID model.AllocationID, cID cproto.ID,
) {
	var err error
	if aID != a.model.AllocationID {
		err = ErrStaleAllocation{aID, a.model.AllocationID}
	} else if _, ok := a.reservations[cID]; !ok {
		err = ErrStaleContainer{ID: cID}
	}
	if err != nil {
		ctx.Respond(err)
		return
	}

	a.daemonReservations[cID] = true
}

// HandleSignal handles an external signal to kill or terminate the allocation.
func (a *Allocation) HandleSignal(ctx *actor.Context, msg actor.Message) {
	switch msg {
	case Kill:
		a.Kill(ctx)
	case Terminate:
		a.Terminate(ctx)
	}
}

// TaskContainerStateChanged handles changes in container states. It can move us to ready,
// kill us or close us normally depending on the changes, among other things.
func (a *Allocation) TaskContainerStateChanged(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	if _, ok := a.reservations[msg.Container.ID]; !ok {
		return ErrStaleContainer{ID: msg.Container.ID}
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
		ctx.Tell(ctx.Self().Parent(), model.TrialLog{
			Message:     msg.ContainerStopped.String(),
			ContainerID: ptrs.StringPtr(string(msg.Container.ID)),
		})

		a.exitedContainers[msg.Container.ID] = *msg.ContainerStopped
		a.rendezvous.containerTerminated(msg.Container.ID)
		if a.firstContainerExited == nil {
			a.firstContainerExited = &msg.Container.ID
		}

		switch {
		case msg.ContainerStopped.Failure != nil:
			a.Kill(ctx)
		default:
			a.Exit(ctx)
		}
	}
	return nil
}

// Exit attempts to exit an allocation while not killing or preempting it.
func (a *Allocation) Exit(ctx *actor.Context) (exited bool) {
	switch {
	case len(a.reservations) == len(a.exitedContainers):
		a.terminated(ctx)
		return true
	case a.allNonDaemonsExited():
		a.killedDaemons = true
		a.kill(ctx)
	}
	return false
}

// Terminate attempts to close an allocation by gracefully stopping it (though a kill are possible).
func (a *Allocation) Terminate(ctx *actor.Context) {
	if exited := a.Exit(ctx); exited {
		return
	}
	switch {
	case a.rendezvous.ready():
		a.preempt(ctx)
	default:
		a.kill(ctx)
	}
}

// Kill attempts to close an allocation by killing it.
func (a *Allocation) Kill(ctx *actor.Context) {
	if exited := a.Exit(ctx); exited {
		return
	}
	a.kill(ctx)
}

func (a *Allocation) allNonDaemonsExited() bool {
	for id := range a.reservations {
		_, terminated := a.exitedContainers[id]
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
	actors.NotifyAfter(ctx, preemptionTimeoutDuration, PreemptionTimeout{a.model.AllocationID})
}

func (a *Allocation) kill(ctx *actor.Context) {
	if a.killCooldown != nil && time.Now().UTC().Before(*a.killCooldown) {
		ctx.Log().Debug("still inside of kill cooldown")
		return
	}

	ctx.Log().Info("decided to kill allocation")
	if a.firstContainerExited == nil {
		a.killedWhileRunning = true
	}
	a.killCooldown = ptrs.TimePtr(time.Now().UTC().Add(killCooldown))
	for _, reservation := range a.reservations {
		reservation.Kill(ctx)
	}
}

func (a *Allocation) terminated(ctx *actor.Context) {
	exit := &AllocationExited{}
	a.exited = true
	defer ctx.Tell(ctx.Self().Parent(), exit)
	defer ctx.Tell(a.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
	defer ctx.Self().Stop()
	if len(a.reservations) == 0 {
		return
	}

	defer a.preemption.Close()
	defer a.rendezvous.Close()
	switch {
	case a.killedWhileRunning:
		ctx.Log().Info("allocation successfully killed")
		return
	case a.preemption.Acknowledged():
		ctx.Log().Info("allocated successfully preempted")
		return
	case a.firstContainerExited != nil:
		err := a.exitedContainers[*a.firstContainerExited].Failure
		if err == nil {
			// This is true because searcher and preemption exits both ack preemption.
			exit.UserRequestedStop = true
			return
		}

		switch err.FailureType {
		case aproto.ContainerFailed, aproto.TaskError:
			ctx.Log().WithError(err).Infof("allocation exited with failure (%s)", err.FailureType)
			exit.Err = err
			return
		case aproto.AgentError, aproto.AgentFailed:
			// Questionable, could be considered failures, but for now we don't.
			ctx.Log().WithError(err).Warnf("allocation exited due to agent (%s)", err.FailureType)
			return
		case aproto.TaskAborted:
			// Definitely not a failure.
			ctx.Log().WithError(err).Debugf("allocation aborted (%s)", err.FailureType)
			return
		default:
			panic(errors.Wrapf(err, "unexpected allocation failure (%s)", err.FailureType))
		}
	default:
		panic("allocation exited without being killed, preempted or having a container exit")
	}
}

const killedLogSubstr = "exit code 137"

func (a *Allocation) enrichLog(l sproto.ContainerLog) sproto.ContainerLog {
	killLog := l.RunMessage != nil && strings.Contains(l.RunMessage.Value, killedLogSubstr)
	if a.killedDaemons && killLog {
		l.Level = ptrs.StringPtr("DEBUG")
	}
	return l
}
