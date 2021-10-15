package task

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/device"

	"github.com/determined-ai/determined/master/internal/proxy"

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
	// Allocation encapsulates all the state of a single allocation.
	Allocation struct {
		// System dependencies.
		db db.DB
		rm *actor.Ref

		// The request to create the allocation, essentially our configuration.
		req sproto.AllocateRequest
		// The persisted representation.
		model model.Allocation

		// State for the primary behavior of an allocation.
		// The state of the allocation, just informational.
		state model.AllocationState
		// State of all our reservations
		reservations reservations
		// Tracks the initial container exit, unless we caused the failure by killed the trial.
		exitReason error
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

		// State for specific sub-behaviors of an allocation.
		// Encapsulates the preemption state of the currently allocated task.
		// If there is no current task, or it is unallocated, it is nil.
		preemption *Preemption
		// Encapsulates logic of rendezvousing containers of the currently
		// allocated task. If there is no current task, or it is unallocated, it is nil.
		rendezvous *Rendezvous
		// Encapsulates the logic of watching for idle timeouts.
		idleTimeoutWatcher *IdleTimeoutWatcher
		// proxy state
		proxies []string
		// log-based readiness state
		logBasedReadinessPassed bool
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
		FinalState        AllocationState
	}
	// BuildTaskSpec is a message to request the task spec from the parent task. This
	// is just a hack since building a task spec cant be semi-costly and we want to defer it
	// until it is needed (we save stuff to the DB and make SSH keys, doing this for 10k trials
	// at once is real bad.
	BuildTaskSpec struct{}
	// AllocationSignal is an interface for signals that can be sent to an allocation.
	AllocationSignal string
	// AllocationState requests allocation state. A copy is filled and returned.
	AllocationState struct {
		State      model.AllocationState
		Containers map[cproto.ID]cproto.Container
		Addresses  map[cproto.ID][]cproto.Address
		Ready      bool
	}
)

const (
	// Kill is the signal to kill an allocation; analogous to in SIGKILL.
	Kill AllocationSignal = "kill"
	// Terminate is the signal to kill an allocation; analogous to in SIGTERM.
	Terminate AllocationSignal = "terminate"
)

const (
	killCooldown       = 30 * time.Second
	okExitMessage      = "command exited successfully"
	missingExitMessage = ""
)

// NewAllocation returns a new allocation, which tracks allocation state in a fairly generic way.
func NewAllocation(req sproto.AllocateRequest, db db.DB, rm *actor.Ref) actor.Actor {
	return &Allocation{
		db: db,
		rm: rm,

		req: req,
		model: model.Allocation{
			AllocationID: req.AllocationID,
			TaskID:       req.TaskID,
			Slots:        req.SlotsNeeded,
			AgentLabel:   req.Name,
			ResourcePool: req.ResourcePool,
			StartTime:    time.Now().UTC(),
		},

		reservations: reservations{},
	}
}

// Receive implements actor.Actor for the allocation.
// The normal flow of an Allocation is to:
//	(1) request resources,
// 	(2) receive resources,
//	(3) start the given task on the resources and
//	(4) monitor the task as it runs and handle releasing it's resources.
//
// Additionally, there are secondary flows that force exits, such as a
// reservation dying or the scheduler requesting us to stop, or being killed
// by the user; and there are user interactions driven by APIs, along the way,
// such as watching preemption, watching rendezvous, marking reservations as
// 'daemon' reservations, etc.
//
// An important note is error handling; the allocation cannot suddenly exit -
// it must clean up its resources. If an error occurs that should not force a
// stop, just return the error to the initiator (ctx.Respond for APIs) or log it
// and move on. If an error occurs that should force a stop, it is imperative
// the error is never returned by Receive, and that a.Error(ctx, err) is called,
// that way the allocation can cleanup properly.
func (a *Allocation) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	// These messages handle interaction with the resource manager. The generally
	// handle the primary allocation lifecycle/functionality.
	case actor.PreStart:
		if err := a.RequestResources(ctx); err != nil {
			a.Error(ctx, err)
		}
	case sproto.ResourcesAllocated:
		if err := a.ResourcesAllocated(ctx, msg); err != nil {
			a.Error(ctx, err)
		}
	case sproto.TaskContainerStateChanged:
		a.TaskContainerStateChanged(ctx, msg)
	case sproto.ReleaseResources:
		a.Terminate(ctx)
	case actor.PostStop:
		a.Cleanup(ctx)
	case sproto.ContainerLog:
		if a.req.StreamEvents != nil {
			ctx.Tell(a.req.StreamEvents.To, sproto.Event{
				State:    a.state.String(),
				IsReady:  a.logBasedReadinessPassed,
				LogEvent: ptrs.StringPtr(msg.String()),
			})
			if rc := a.req.LogBasedReady; rc != nil && !a.logBasedReadinessPassed {
				if rc.Pattern.MatchString(msg.String()) {
					a.logBasedReadinessPassed = true
					ctx.Tell(a.req.StreamEvents.To, sproto.Event{
						State:             a.state.String(),
						IsReady:           a.logBasedReadinessPassed,
						ServiceReadyEvent: &msg,
					})
				}
			}
		}
		ctx.Tell(ctx.Self().Parent(), a.enrichLog(msg))

	// These messages allow users (and sometimes an orchestrator, such as HP search)
	// to interact with the allocation. The usually trace back to API calls.
	case MarkReservationDaemon:
		a.SetReservationAsDaemon(ctx, msg.AllocationID, msg.ContainerID)
	case AllocationSignal:
		a.HandleSignal(ctx, msg)
	case AllocationState:
		if ctx.ExpectingResponse() {
			ctx.Respond(a.State())
		}
	case WatchRendezvousInfo, UnwatchRendezvousInfo, RendezvousTimeout:
		if !a.req.DoRendezvous {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorDisabled{Behavior: rendezvous})
			}
			return nil
		}
		if err := a.rendezvous.ReceiveMsg(ctx); err != nil {
			ctx.Tell(ctx.Self(), sproto.ContainerLog{AuxMessage: ptrs.StringPtr(err.Error())})
			a.Error(ctx, err)
		}
	case WatchPreemption, UnwatchPreemption, PreemptionTimeout, AckPreemption:
		if !a.req.Preemptible {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorDisabled{preemption})
			}
			return nil
		}
		if err := a.preemption.ReceiveMsg(ctx); err != nil {
			ctx.Tell(ctx.Self(), sproto.ContainerLog{AuxMessage: ptrs.StringPtr(err.Error())})
			a.Error(ctx, err)
		}
	case IdleTimeoutWatcherTick, IdleWatcherNoteActivity:
		if a.req.IdleTimeout == nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorDisabled{idleWatcher})
			}
			return nil
		}
		if err := a.idleTimeoutWatcher.ReceiveMsg(ctx); err != nil {
			a.Error(ctx, err)
		}

	default:
		a.Error(ctx, actor.ErrUnexpectedMessage(ctx))
	}
	return nil
}

// RequestResources sets up the allocation.
func (a *Allocation) RequestResources(ctx *actor.Context) error {
	a.state = model.AllocationStatePending
	a.req.TaskActor = ctx.Self()
	if err := ctx.Ask(a.rm, a.req).Error(); err != nil {
		return errors.Wrap(err, "failed to request allocation")
	}
	if a.req.StreamEvents != nil {
		ctx.Tell(a.req.StreamEvents.To, sproto.Event{
			State:          a.state.String(),
			IsReady:        a.logBasedReadinessPassed,
			ScheduledEvent: &a.model.AllocationID,
		})
	}
	return nil
}

// Cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
func (a *Allocation) Cleanup(ctx *actor.Context) {
	if a.req.StreamEvents != nil {
		// This message must be sent when the actor is closing since it closes all
		// websockets listening for these events.
		exitReason := okExitMessage
		if a.exitReason != nil {
			exitReason = a.exitReason.Error()
		}
		ctx.Tell(a.req.StreamEvents.To, sproto.Event{
			State:       a.state.String(),
			IsReady:     a.logBasedReadinessPassed,
			ExitedEvent: &exitReason,
		})
	}
	// Just in-case code.
	if !a.exited {
		ctx.Log().Info("exit did not run properly")
		for _, r := range a.reservations {
			if r.exit == nil {
				ctx.Log().Infof("allocation exited with unterminated reservation: %v", r.Summary())
				r.Kill(ctx)
			}
		}
		if len(a.reservations) > 0 {
			a.markResourcesReleased(ctx)
		}
		ctx.Tell(a.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
	}
}

// ResourcesAllocated handles receiving resources from the resource manager. Note: it makes a single
// ask to the parent to build its task spec.. this is mostly a hack to defer lots of computationally
// heavy stuff unless it is necessarily (which also works to spread occurrences of the same work
// out). Eventually, Allocations should just be started with their TaskSpec.
func (a *Allocation) ResourcesAllocated(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	a.state = model.AllocationStateAssigned
	a.reservations.append(msg.Reservations)

	// Get the task spec first, so the trial/task table is populated before allocations.
	resp := ctx.Ask(ctx.Self().Parent(), BuildTaskSpec{})
	if err := resp.Error(); err != nil {
		return errors.Wrapf(err, "could not get task spec")
	}
	spec := resp.Get().(tasks.TaskSpec)

	if err := a.db.AddAllocation(&a.model); err != nil {
		return errors.Wrap(err, "saving trial allocation")
	}

	token, err := a.db.StartAllocationSession(a.model.AllocationID)
	if err != nil {
		return errors.Wrap(err, "starting a new allocation session")
	}

	if a.req.Preemptible {
		a.preemption = NewPreemption(a.model.AllocationID)
	}

	if a.req.DoRendezvous {
		a.rendezvous = NewRendezvous(a.model.AllocationID, a.reservations)
		a.rendezvous.PreStart(ctx)
	}

	if cfg := a.req.IdleTimeout; cfg != nil {
		a.idleTimeoutWatcher = NewIdleTimeoutWatcher(a.req.Name, cfg)
		a.idleTimeoutWatcher.PreStart(ctx)
	}

	for cID, r := range a.reservations {
		r.Start(ctx, spec, sproto.ReservationRuntimeInfo{
			Token:        token,
			AgentRank:    a.reservations[cID].rank,
			IsMultiAgent: len(a.reservations) > 1,
		})
	}
	if a.req.StreamEvents != nil {
		ctx.Tell(a.req.StreamEvents.To, sproto.Event{
			State:         a.state.String(),
			IsReady:       a.logBasedReadinessPassed,
			AssignedEvent: &msg,
		})
	}
	return nil
}

// SetReservationAsDaemon sets the reservation as a daemon reservation. This means we won't wait for
// it to exit in errorless exits and instead will kill the forcibly.
func (a *Allocation) SetReservationAsDaemon(
	ctx *actor.Context, aID model.AllocationID, cID cproto.ID,
) {
	if aID != a.model.AllocationID {
		ctx.Respond(ErrStaleAllocation{aID, a.model.AllocationID})
		return
	} else if _, ok := a.reservations[cID]; !ok {
		ctx.Respond(ErrStaleContainer{ID: cID})
		return
	}

	a.reservations[cID].daemon = true

	if len(a.reservations.daemons()) == len(a.reservations) {
		ctx.Log().Warnf("all reservations were marked as daemon, exiting")
		a.Exit(ctx)
	}
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
) {
	if _, ok := a.reservations[msg.Container.ID]; !ok {
		ctx.Log().WithError(ErrStaleContainer{ID: msg.Container.ID}).Warnf("old state change")
		return
	}

	a.reservations[msg.Container.ID].container = &msg.Container
	ctx.Log().Debugf("container %s (rank %d) is %s",
		msg.Container.ID, a.reservations[msg.Container.ID].rank, msg.Container.State,
	)
	switch msg.Container.State {
	case cproto.Pulling:
		a.state = model.MostProgressedAllocationState(a.state, model.AllocationStatePulling)
	case cproto.Starting:
		a.state = model.MostProgressedAllocationState(a.state, model.AllocationStateStarting)
	case cproto.Running:
		a.state = model.MostProgressedAllocationState(a.state, model.AllocationStateRunning)
		a.reservations[msg.Container.ID].start = msg.ContainerStarted
		if a.req.DoRendezvous && a.rendezvous.try() {
			ctx.Log().Info("all containers are connected successfully (task container state changed)")
		}
		if a.req.ProxyPort != nil {
			a.registerProxies(ctx, msg)
		}
		if a.req.StreamEvents != nil {
			ctx.Tell(a.req.StreamEvents.To, sproto.Event{
				State:                 a.state.String(),
				IsReady:               a.logBasedReadinessPassed,
				ContainerStartedEvent: msg.ContainerStarted,
			})
		}
	case cproto.Terminated:
		a.state = model.MostProgressedAllocationState(a.state, model.AllocationStateTerminating)
		a.reservations[msg.Container.ID].exit = msg.ContainerStopped
		ctx.Tell(ctx.Self(), sproto.ContainerLog{
			AuxMessage: ptrs.StringPtr(msg.ContainerStopped.String()),
			Container:  msg.Container,
		})
		switch {
		case msg.ContainerStopped.Failure != nil:
			a.Error(ctx, *msg.ContainerStopped.Failure)
		default:
			a.Exit(ctx)
		}
	}
}

// Exit attempts to exit an allocation while not killing or preempting it.
func (a *Allocation) Exit(ctx *actor.Context) (exited bool) {
	switch {
	case len(a.reservations) == len(a.reservations.exited()):
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
	if msg, ok := ctx.Message().(sproto.ReleaseResources); ok && a.req.StreamEvents != nil {
		ctx.Tell(a.req.StreamEvents.To, sproto.Event{
			State:                 a.state.String(),
			IsReady:               a.logBasedReadinessPassed,
			TerminateRequestEvent: &msg,
		})
	}

	if exited := a.Exit(ctx); exited {
		return
	}
	switch {
	case a.req.Preemptible && a.req.DoRendezvous && a.rendezvous.ready():
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

// Error closes the allocation due to an error, beginning the kill flow.
func (a *Allocation) Error(ctx *actor.Context, err error) {
	if a.exitReason == nil {
		a.exitReason = err
	}
	a.Kill(ctx)
}

func (a *Allocation) allNonDaemonsExited() bool {
	for id := range a.reservations {
		_, terminated := a.reservations.exited()[id]
		_, daemon := a.reservations.daemons()[id]
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
	if len(a.reservations.exited()) == 0 {
		a.killedWhileRunning = true
	}
	a.killCooldown = ptrs.TimePtr(time.Now().UTC().Add(killCooldown))
	for _, reservation := range a.reservations {
		reservation.Kill(ctx)
	}
}

func (a *Allocation) registerProxies(ctx *actor.Context, msg sproto.TaskContainerStateChanged) {
	cfg := a.req.ProxyPort
	if cfg == nil {
		return
	}

	if len(a.reservations) > 1 {
		// We don't support proxying multi-reservation allocations.
		ctx.Log().Warnf("proxy for multi-reservation allocation aborted")
		return
	}

	for _, address := range msg.ContainerStarted.Addresses {
		// Only proxy the port we expect to proxy. If a dockerfile uses an EXPOSE command,
		// additional addresses will appear her, but currently we only proxy one uuid to one
		// port, so it doesn't make sense to send multiple proxy.Register messages for a
		// single ServiceID (only the last one would work).
		if address.ContainerPort != cfg.Port {
			continue
		}

		// We are keying on allocation id instead of container id. Revisit this when we need to
		// proxy multi-container tasks or when containers are created prior to being
		// assigned to an agent.
		ctx.Ask(ctx.Self().System().Get(actor.Addr("proxy")), proxy.Register{
			ServiceID: cfg.ServiceID,
			URL: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
			},
			ProxyTCP: cfg.ProxyTCP,
		})
		a.proxies = append(a.proxies, cfg.ServiceID)
	}

	if len(a.proxies) != 1 {
		ctx.Log().Errorf("proxied more than expected %v", len(a.proxies))
	}
}

func (a *Allocation) unregisterProxies(ctx *actor.Context) {
	cfg := a.req.ProxyPort
	if cfg == nil {
		return
	}

	if len(a.reservations) > 1 {
		// Can't proxy more than one reservation, so we never would've made them.
		return
	}

	for _, serviceID := range a.proxies {
		ctx.Tell(ctx.Self().System().Get(actor.Addr("proxy")), proxy.Unregister{
			ServiceID: serviceID,
		})
	}
}

func (a *Allocation) terminated(ctx *actor.Context) {
	a.state = model.MostProgressedAllocationState(a.state, model.AllocationStateTerminated)
	exit := &AllocationExited{FinalState: a.State()}
	a.exited = true
	defer ctx.Tell(ctx.Self().Parent(), exit)
	defer ctx.Tell(a.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
	defer a.unregisterProxies(ctx)
	defer ctx.Self().Stop()
	if len(a.reservations) == 0 {
		return
	}
	defer a.markResourcesReleased(ctx)

	if a.req.Preemptible {
		defer a.preemption.Close()
	}
	if a.req.DoRendezvous {
		defer a.rendezvous.Close()
	}
	switch {
	case a.killedWhileRunning:
		ctx.Log().Info("allocation successfully killed")
		return
	case a.req.Preemptible && a.preemption.Acknowledged():
		ctx.Log().Info("allocation successfully stopped")
		return
	case len(a.reservations.exited()) > 0:
		if a.exitReason == nil {
			// This is true because searcher and preemption exits both ack preemption.
			exit.UserRequestedStop = true
			ctx.Log().Info("allocation successfully stopped early")
			return
		}

		switch err := a.exitReason.(type) {
		case aproto.ContainerFailure:
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
				panic(errors.Wrapf(err, "unexpected allocation failure (%s)", err.Error()))
			}
		default:
			ctx.Log().WithError(err).Error("allocation handler crashed")
			exit.Err = err
			return
		}

	default:
		panic("allocation exited without being killed, preempted or having a container exit")
	}
}

// markResourcesReleased persists completion information.
func (a *Allocation) markResourcesReleased(ctx *actor.Context) {
	a.model.EndTime = ptrs.TimePtr(time.Now().UTC())
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		ctx.Log().WithError(err).Error("error delete allocation session")
	}
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		ctx.Log().WithError(err).Error("failed to mark allocation completed")
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

// State returns a deepcopy of our state.
func (a *Allocation) State() AllocationState {
	containers := map[cproto.ID]cproto.Container{}
	for id, r := range a.reservations {
		if r.container == nil {
			continue
		}

		c := r.container
		nd := make([]device.Device, len(c.Devices))
		copy(nd, c.Devices)
		containers[id] = cproto.Container{
			Parent:  c.Parent,
			ID:      c.ID,
			State:   c.State,
			Devices: nd,
		}
	}

	addresses := map[cproto.ID][]cproto.Address{}
	for id, r := range a.reservations {
		if r.start == nil {
			continue
		}

		a := r.start.Addresses
		na := make([]cproto.Address, len(a))
		copy(na, a)
		addresses[id] = na
	}

	return AllocationState{
		State:      a.state,
		Containers: containers,
		Addresses:  addresses,
		Ready:      a.req.DoRendezvous && a.rendezvous.ready() || a.logBasedReadinessPassed,
	}
}

func (a *AllocationExited) String() string {
	switch {
	case a == nil:
		return missingExitMessage
	case a.Err != nil:
		return a.Err.Error()
	default:
		return okExitMessage
	}
}

// FirstContainer returns the first container in the allocation state.
func (a *AllocationState) FirstContainer() *cproto.Container {
	for _, c := range a.Containers {
		return &c
	}
	return nil
}

// FirstContainerAddresses returns the first container's addresses in the allocation state.
func (a *AllocationState) FirstContainerAddresses() []cproto.Address {
	for _, ca := range a.Addresses {
		return ca
	}
	return nil
}
