package task

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/cproto"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

type (
	// Allocation encapsulates all the state of a single allocation.
	Allocation struct {
		// System dependencies.
		db     db.DB
		rm     *actor.Ref
		logger *Logger

		// The request to create the allocation, essentially our configuration.
		req sproto.AllocateRequest
		// The persisted representation.
		model model.Allocation

		// State of all our resources.
		resources resourcesList
		// Separates the existence of resources from us having started them.
		resourcesStarted bool
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
		rendezvous *rendezvous
		// Encapsulates the logic of watching for idle timeouts.
		idleTimeoutWatcher *IdleTimeoutWatcher
		// proxy state
		proxies []string
		// proxyAddress is provided by determined.exec.prep_container if the RM doesn't provide it.
		proxyAddress *string
		// active all gather state
		allGather *allGather

		logCtx detLogger.Context
	}

	// MarkResourcesDaemon marks the given reservation as a daemon. In the event of a normal exit,
	// the allocation will not wait for it to exit on its own and instead will kill it and instead
	// await it's hopefully quick termination.
	MarkResourcesDaemon struct {
		AllocationID model.AllocationID
		ResourcesID  sproto.ResourcesID
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
		State     model.AllocationState
		Resources map[sproto.ResourcesID]sproto.ResourcesSummary
		Ready     bool

		Addresses  map[sproto.ResourcesID][]cproto.Address
		Containers map[sproto.ResourcesID][]cproto.Container
	}
	// AllocationReady marks an allocation as ready.
	AllocationReady struct {
		Message string
	}
	// SetAllocationProxyAddress manually sets the allocation proxy address.
	SetAllocationProxyAddress struct {
		ProxyAddress string
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
func NewAllocation(
	logCtx detLogger.Context, req sproto.AllocateRequest, db db.DB, rm *actor.Ref, logger *Logger,
) actor.Actor {
	return &Allocation{
		db:     db,
		rm:     rm,
		logger: logger,

		req: req,
		model: model.Allocation{
			AllocationID: req.AllocationID,
			TaskID:       req.TaskID,
			Slots:        req.SlotsNeeded,
			AgentLabel:   req.Name,
			ResourcePool: req.ResourcePool,
		},

		resources: resourcesList{},

		logCtx: detLogger.MergeContexts(logCtx, detLogger.Context{
			"allocation-id": req.AllocationID,
		}),
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
// such as watching preemption, watching rendezvous, marking resources as
// 'daemon' resources, etc.
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
		ctx.AddLabels(a.logCtx)
		if err := a.RequestResources(ctx); err != nil {
			a.Error(ctx, err)
		}
	case sproto.ResourcesAllocated:
		if err := a.ResourcesAllocated(ctx, msg); err != nil {
			a.Error(ctx, err)
		}
	case sproto.ResourcesStateChanged:
		a.ResourcesStateChanged(ctx, msg)
	case sproto.GetResourcesContainerState:
		if v, ok := a.resources[msg.ResourcesID]; ok {
			if v.container == nil {
				ctx.Respond(fmt.Errorf("no container associated with %s", msg.ResourcesID))
			} else {
				ctx.Respond(*v.container)
			}
		} else {
			ctx.Respond(fmt.Errorf("unknown resources %s", msg.ResourcesID))
		}
	case sproto.ReleaseResources, sproto.ChangeRP:
		a.Terminate(ctx)
	case actor.PostStop:
		a.Cleanup(ctx)
	case sproto.ContainerLog:
		a.sendEvent(ctx, msg.ToEvent())

	// These messages allow users (and sometimes an orchestrator, such as HP search)
	// to interact with the allocation. The usually trace back to API calls.
	case AllocationReady:
		a.model.IsReady = ptrs.Ptr(true)
		if err := a.db.UpdateAllocationState(a.model); err != nil {
			a.Error(ctx, err)
		}
		a.sendEvent(ctx, sproto.Event{ServiceReadyEvent: ptrs.Ptr(true)})
	case MarkResourcesDaemon:
		if err := a.SetResourcesAsDaemon(ctx, msg.AllocationID, msg.ResourcesID); err != nil {
			a.Error(ctx, err)
		}
	case AllocationSignal:
		a.HandleSignal(ctx, msg)
	case AllocationState:
		if ctx.ExpectingResponse() {
			ctx.Respond(a.State())
		}
	case SetAllocationProxyAddress:
		if a.req.ProxyPort == nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorUnsupported{Behavior: fmt.Sprintf("%T", msg)})
			}
			return nil
		}
		a.proxyAddress = &msg.ProxyAddress
		a.registerProxies(ctx, a.containerProxyAddresses())
	case WatchRendezvousInfo, UnwatchRendezvousInfo, rendezvousTimeout:
		if a.rendezvous == nil {
			if a.resources == nil {
				return ErrAllocationUnfulfilled{Action: fmt.Sprintf("%T", msg)}
			}

			switch a.resources.first().Summary().ResourcesType {
			case sproto.ResourcesTypeDockerContainer, sproto.ResourcesTypeK8sPod:
				break
			default:
				return ErrBehaviorUnsupported{Behavior: fmt.Sprintf("%T", msg)}
			}

			switch msg.(type) {
			case WatchRendezvousInfo:
				a.rendezvous = newRendezvous(ctx, a.model.AllocationID, a.resources)
			case UnwatchRendezvousInfo, rendezvousTimeout:
				// Ignore without active rendezvous.
				return nil
			}
		}

		switch msg := ctx.Message().(type) {
		case WatchRendezvousInfo:
			if w, err := a.rendezvous.watch(msg); err != nil {
				ctx.Respond(err)
			} else {
				ctx.Respond(w)
			}
		case UnwatchRendezvousInfo:
			a.rendezvous.unwatch(msg)
		case rendezvousTimeout:
			if err := a.rendezvous.checkTimeout(msg); err != nil {
				a.logger.Insert(ctx, a.enrichLog(model.TaskLog{Log: err.Error()}))
			}
		default:
			a.Error(ctx, actor.ErrUnexpectedMessage(ctx))
		}
	case WatchAllGather, UnwatchAllGather, allGatherTimeout:
		if a.allGather == nil {
			switch msg.(type) {
			case WatchAllGather:
				a.allGather = newAllGather(ctx)
			case UnwatchAllGather, allGatherTimeout:
				// Ignore without active all gather.
				return nil
			}
		}

		switch msg := ctx.Message().(type) {
		case WatchAllGather:
			watcher := a.allGather.watch(msg)
			ctx.Respond(watcher)
		case UnwatchAllGather:
			a.allGather.unwatch(msg)
		case allGatherTimeout:
			if err := a.allGather.checkTimeout(msg); err != nil {
				a.logger.Insert(ctx, a.enrichLog(model.TaskLog{Log: err.Error()}))
				ctx.Log().WithError(err).Error("performing all gather through master")
			}
		default:
			return actor.ErrUnexpectedMessage(ctx)
		}

		if a.allGather.done() {
			a.allGather = nil
		}
	case WatchPreemption, UnwatchPreemption, PreemptionTimeout, AckPreemption:
		if !a.req.Preemptible {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorDisabled{preemption})
			}
			return nil
		}
		if err := a.preemption.ReceiveMsg(ctx); err != nil {
			a.logger.Insert(ctx, a.enrichLog(model.TaskLog{Log: err.Error()}))
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
	a.setModelState(model.AllocationStatePending)

	if err := a.db.AddAllocation(&a.model); err != nil {
		return errors.Wrap(err, "saving trial allocation")
	}
	a.req.TaskActor = ctx.Self()
	if err := ctx.Ask(a.rm, a.req).Error(); err != nil {
		return errors.Wrap(err, "failed to request allocation")
	}
	a.sendEvent(ctx, sproto.Event{ScheduledEvent: &a.model.AllocationID})
	return nil
}

// Cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
func (a *Allocation) Cleanup(ctx *actor.Context) {
	// This message must be sent when the actor is closing since it closes all
	// websockets listening for these events.
	exitReason := okExitMessage
	if a.exitReason != nil {
		exitReason = a.exitReason.Error()
	}
	a.sendEvent(ctx, sproto.Event{ExitedEvent: &exitReason})

	// Just in-case code.
	if !a.exited {
		ctx.Log().Info("exit did not run properly")
		for _, r := range a.resources {
			if r.Exited == nil {
				ctx.Log().Infof("allocation exited with unterminated reservation: %v", r.Summary())
				r.Kill(ctx, a.logCtx)
			}
		}
		if len(a.resources) > 0 {
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
	if a.getModelState() != model.AllocationStatePending {
		// If we have moved on from the pending state, these must be stale (and we must have
		// already released them, just the scheduler hasn't gotten word yet).
		return ErrStaleResourcesReceived{}
	}

	a.setModelState(model.AllocationStateAssigned)
	if err := a.resources.append(msg.Resources); err != nil {
		return errors.Wrapf(err, "appending resources")
	}

	// Get the task spec first, so the trial/task table is populated before allocations.
	resp := ctx.Ask(ctx.Self().Parent(), BuildTaskSpec{})
	switch ok, err := resp.ErrorOrTimeout(time.Hour); {
	case err != nil:
		return errors.Wrapf(err, "could not get task spec")
	case !ok:
		return errors.Wrapf(err, "timeout getting task spec, likely a deadlock")
	}
	spec := resp.Get().(tasks.TaskSpec)

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		return errors.Wrap(err, "updating allocation state")
	}

	now := time.Now().UTC()
	err := a.db.RecordTaskStats(&model.TaskStats{
		AllocationID: msg.ID,
		EventType:    "QUEUED",
		StartTime:    &msg.JobSubmissionTime,
		EndTime:      &now,
	})
	if err != nil {
		return errors.Wrap(err, "recording task queued stats")
	}

	token, err := a.db.StartAllocationSession(a.model.AllocationID)
	if err != nil {
		return errors.Wrap(err, "starting a new allocation session")
	}

	if a.req.Preemptible {
		a.preemption = NewPreemption(a.model.AllocationID)
	}

	if cfg := a.req.IdleTimeout; cfg != nil {
		a.idleTimeoutWatcher = NewIdleTimeoutWatcher(a.req.Name, cfg)
		a.idleTimeoutWatcher.PreStart(ctx)
	}

	for cID, r := range a.resources {
		if err := r.Start(ctx, a.logCtx, spec, sproto.ResourcesRuntimeInfo{
			Token:        token,
			AgentRank:    a.resources[cID].Rank,
			IsMultiAgent: len(a.resources) > 1,
		}); err != nil {
			return fmt.Errorf("starting resources (%v): %w", r, err)
		}
	}
	a.resourcesStarted = true
	a.sendEvent(ctx, sproto.Event{AssignedEvent: &msg})
	return nil
}

// SetResourcesAsDaemon sets the reservation as a daemon reservation. This means we won't wait for
// it to exit in errorless exits and instead will kill the forcibly.
func (a *Allocation) SetResourcesAsDaemon(
	ctx *actor.Context, aID model.AllocationID, rID sproto.ResourcesID,
) error {
	if aID != a.model.AllocationID {
		ctx.Respond(ErrStaleAllocation{aID, a.model.AllocationID})
		return nil
	} else if _, ok := a.resources[rID]; !ok {
		ctx.Respond(ErrStaleResources{ID: rID})
		return nil
	} else if len(a.resources) <= 1 {
		ctx.Respond(api.AsValidationError(`ignoring set daemon request for allocation with a single
			set of resources since this would just kill the allocation`))
		return nil
	}

	a.resources[rID].Daemon = true
	if err := a.resources[rID].Persist(); err != nil {
		return err
	}

	if len(a.resources.daemons()) == len(a.resources) {
		ctx.Log().Warnf("all resources were marked as daemon, exiting")
		a.Exit(ctx)
	}

	return nil
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

// ResourcesStateChanged handles changes in container states. It can move us to ready,
// kill us or close us normally depending on the changes, among other things.
func (a *Allocation) ResourcesStateChanged(
	ctx *actor.Context, msg sproto.ResourcesStateChanged,
) {
	if _, ok := a.resources[msg.ResourcesID]; !ok {
		ctx.Log().
			WithField("container", msg.Container).
			WithError(ErrStaleResources{ID: msg.ResourcesID}).Warnf("old state change")
		return
	}

	a.resources[msg.ResourcesID].container = msg.Container
	ctx.Log().Debugf("resources %s (rank %d) is %s [container=%v]",
		msg.ResourcesID, a.resources[msg.ResourcesID].Rank, msg.ResourcesState, msg.Container,
	)
	switch msg.ResourcesState {
	case sproto.Pulling:
		a.setMostProgressedModelState(model.AllocationStatePulling)
		a.model.StartTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
		if err := a.db.UpdateAllocationStartTime(a.model); err != nil {
			ctx.Log().
				WithError(err).
				Errorf("allocation will not be properly accounted for")
		}
	case sproto.Starting:
		a.setMostProgressedModelState(model.AllocationStateStarting)
	case sproto.Running:
		a.setMostProgressedModelState(model.AllocationStateRunning)

		a.resources[msg.ResourcesID].Started = msg.ResourcesStarted
		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.Error(ctx, err)
			return
		}

		if a.rendezvous != nil && a.rendezvous.try() {
			ctx.Log().Info("all containers are connected successfully (task container state changed)")
		}
		if a.req.ProxyPort != nil && msg.ResourcesStarted.Addresses != nil {
			a.registerProxies(ctx, msg.ResourcesStarted.Addresses)
		}

		a.sendEvent(ctx, sproto.Event{
			ContainerID:           coalesceString(msg.ContainerIDStr(), ""),
			ContainerStartedEvent: msg.ResourcesStarted,
		})

		prom.AssociateAllocationTask(a.req.AllocationID,
			a.req.TaskID,
			a.req.TaskActor.Address(),
			a.req.JobID)
		prom.AddAllocationResources(a.resources[msg.ResourcesID].Summary(), msg.ResourcesStarted)

	case sproto.Terminated:
		a.setMostProgressedModelState(model.AllocationStateTerminating)

		a.resources[msg.ResourcesID].Exited = msg.ResourcesStopped

		logLevel := ptrs.Ptr(model.LogLevelInfo)
		if msg.ResourcesStopped.Failure != nil {
			logLevel = ptrs.Ptr(model.LogLevelError)
		}

		a.logger.Insert(ctx, a.enrichLog(model.TaskLog{
			ContainerID: msg.ContainerIDStr(),
			Log:         msg.ResourcesStopped.String(),
			Level:       logLevel,
		}))

		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.Error(ctx, err)
			return
		}

		switch {
		case msg.ResourcesStopped.Failure != nil:
			a.Error(ctx, *msg.ResourcesStopped.Failure)
		default:
			a.Exit(ctx)
		}

		for cID := range a.resources {
			prom.DisassociateAllocationTask(a.req.AllocationID,
				a.req.TaskID,
				a.req.TaskActor.Address(),
				a.req.JobID)
			prom.RemoveAllocationResources(a.resources[cID].Summary())
		}
	}

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		ctx.Log().Error(err)
	}
}

// Exit attempts to exit an allocation while not killing or preempting it.
func (a *Allocation) Exit(ctx *actor.Context) (exited bool) {
	switch {
	case !a.resourcesStarted:
		a.terminated(ctx)
		return true
	case len(a.resources) == len(a.resources.exited()):
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
	forcePreemption := false
	if msg, ok := ctx.Message().(sproto.ReleaseResources); ok {
		if msg.ForcePreemption {
			forcePreemption = true
		}
		a.sendEvent(ctx, sproto.Event{TerminateRequestEvent: &msg})
	}

	if exited := a.Exit(ctx); exited {
		return
	}
	switch {
	case a.req.Preemptible && (a.rendezvous != nil && a.rendezvous.ready()) || forcePreemption:
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
	ctx.Log().WithError(err).Errorf("allocation encountered fatal error")
	if a.exitReason == nil {
		a.exitReason = err
	}
	a.Kill(ctx)
}

func (a *Allocation) allNonDaemonsExited() bool {
	for id := range a.resources {
		_, terminated := a.resources.exited()[id]
		_, daemon := a.resources.daemons()[id]
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
	if len(a.resources.exited()) == 0 {
		a.killedWhileRunning = true
	}
	a.killCooldown = ptrs.Ptr(time.Now().UTC().Add(killCooldown))
	for _, r := range a.resources {
		r.Kill(ctx, a.logCtx)
	}
}

func (a *Allocation) registerProxies(ctx *actor.Context, addresses []cproto.Address) {
	cfg := a.req.ProxyPort
	if cfg == nil {
		return
	}

	if len(a.resources) > 1 {
		// We don't support proxying multi-reservation allocations.
		ctx.Log().Warnf("proxy for multi-reservation allocation aborted")
		return
	}

	for _, address := range addresses {
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
			ProxyTCP:        cfg.ProxyTCP,
			Unauthenticated: cfg.Unauthenticated,
		})
		a.proxies = append(a.proxies, cfg.ServiceID)
	}

	if len(a.proxies) != 1 {
		ctx.Log().Errorf("did not proxy as expected %v (found addrs %v)", len(a.proxies), addresses)
	}
}

func (a *Allocation) unregisterProxies(ctx *actor.Context) {
	cfg := a.req.ProxyPort
	if cfg == nil {
		return
	}

	if len(a.resources) > 1 {
		// Can't proxy more than one reservation, so we never would've made them.
		return
	}

	for _, serviceID := range a.proxies {
		ctx.Tell(ctx.Self().System().Get(actor.Addr("proxy")), proxy.Unregister{
			ServiceID: serviceID,
		})
	}
}

// containerProxyAddresses forms the container address when proxyAddress is given.
func (a *Allocation) containerProxyAddresses() []cproto.Address {
	if a.proxyAddress == nil || a.req.ProxyPort == nil {
		return []cproto.Address{}
	}
	return []cproto.Address{
		{
			ContainerIP:   *a.proxyAddress,
			ContainerPort: a.req.ProxyPort.Port,
			HostIP:        *a.proxyAddress,
			HostPort:      a.req.ProxyPort.Port,
		},
	}
}

func (a *Allocation) terminated(ctx *actor.Context) {
	a.setMostProgressedModelState(model.AllocationStateTerminated)
	exit := &AllocationExited{FinalState: a.State()}
	if a.exited {
		// Never exit twice. If this were allowed, a trial could receive two task.AllocationExited
		// messages. On receipt of the first message, the trial awaits our exit. Once we exit, it
		// reschedules a new allocation, receives the second message and erroneously awaits the new
		// allocation's stop. Once the new allocation asks the trial to build its task spec, they
		// deadlock.
		// This occurred when an allocation completed and was preempted in quick succession.
		return
	}
	a.exited = true
	defer ctx.Tell(ctx.Self().Parent(), exit)
	defer ctx.Tell(a.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
	defer a.unregisterProxies(ctx)
	defer ctx.Self().Stop()
	if len(a.resources) == 0 {
		return
	}
	defer a.markResourcesReleased(ctx)

	if a.req.Preemptible {
		defer a.preemption.Close()
	}
	if a.rendezvous != nil {
		defer a.rendezvous.close()
	}
	switch {
	case a.killedWhileRunning:
		ctx.Log().Info("allocation successfully killed")
		return
	case a.req.Preemptible && a.preemption.Acknowledged():
		ctx.Log().Info("allocation successfully stopped")
		return
	case a.exitReason == nil && len(a.resources.exited()) > 0:
		// This is true because searcher and preemption exits both ack preemption.
		exit.UserRequestedStop = true
		ctx.Log().Info("allocation successfully stopped early")
		return
	case a.exitReason != nil:
		switch err := a.exitReason.(type) {
		case sproto.ResourcesFailure:
			switch err.FailureType {
			case sproto.ContainerFailed, sproto.TaskError:
				ctx.Log().WithError(err).Infof("allocation exited with failure (%s)", err.FailureType)
				exit.Err = err
				return
			case sproto.AgentError, sproto.AgentFailed:
				ctx.Log().WithError(err).Warnf("allocation exited due to agent (%s)", err.FailureType)
				exit.Err = err
				return
			case sproto.TaskAborted, sproto.ContainerAborted:
				ctx.Log().WithError(err).Debugf("allocation aborted (%s)", err.FailureType)
				exit.Err = err
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
		// If we ever exit without a reason and we have no exited resources, something has gone
		// wrong.
		panic("allocation exited early without a valid reason")
	}
}

// markResourcesReleased persists completion information.
func (a *Allocation) markResourcesReleased(ctx *actor.Context) {
	a.model.EndTime = ptrs.Ptr(time.Now().UTC())
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		ctx.Log().WithError(err).Error("error delete allocation session")
	}
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		ctx.Log().WithError(err).Error("failed to mark allocation completed")
	}

	telemetry.ReportAllocationTerminal(
		ctx.Self().System(), a.db, a.model, a.resources.firstDevice())
}

const killedLogSubstr = "exit code 137"

func (a *Allocation) enrichLog(log model.TaskLog) model.TaskLog {
	log.TaskID = string(a.req.TaskID)

	if log.Timestamp == nil || log.Timestamp.IsZero() {
		log.Timestamp = ptrs.Ptr(time.Now().UTC())
	}

	if a.killedDaemons && strings.Contains(log.Log, killedLogSubstr) {
		log.Level = ptrs.Ptr(model.LogLevelDebug)
	} else if log.Level == nil {
		log.Level = ptrs.Ptr(model.LogLevelInfo)
	}

	if log.Source == nil {
		log.Source = ptrs.Ptr("master")
	}

	if log.StdType == nil {
		log.StdType = ptrs.Ptr("stdout")
	}

	log.Log += "\n"
	return log
}

func (a *Allocation) sendEvent(ctx *actor.Context, ev sproto.Event) {
	ev = a.enrichEvent(ctx, ev)
	a.logger.Insert(ctx, a.enrichLog(ev.ToTaskLog()))
	if a.req.StreamEvents != nil {
		ctx.Tell(a.req.StreamEvents.To, ev)
	}
}

func (a *Allocation) enrichEvent(ctx *actor.Context, ev sproto.Event) sproto.Event {
	ev.ParentID = ctx.Self().Parent().Address().Local()
	ev.Description = a.req.Name
	ev.IsReady = coalesceBool(a.model.IsReady, false)
	if ev.State == "" {
		ev.State = a.getModelState().String()
	}
	if ev.Time.IsZero() {
		ev.Time = time.Now().UTC()
	}
	return ev
}

// State returns a deepcopy of our state.
func (a *Allocation) State() AllocationState {
	addresses := map[sproto.ResourcesID][]cproto.Address{}
	containers := map[sproto.ResourcesID][]cproto.Container{}
	resources := map[sproto.ResourcesID]sproto.ResourcesSummary{}
	for id, r := range a.resources {
		resources[id] = r.Summary()

		switch {
		case r.Started != nil && r.Started.Addresses != nil:
			a := r.Started.Addresses
			na := make([]cproto.Address, len(a))
			copy(na, a)
			addresses[id] = na
		case a.proxyAddress != nil:
			addresses[id] = a.containerProxyAddresses()
		}

		if r.container != nil {
			containers[id] = append(containers[id], *r.container)
		}
	}

	return AllocationState{
		State:      a.getModelState(),
		Resources:  resources,
		Addresses:  addresses,
		Containers: containers,
		Ready: a.rendezvous != nil && a.rendezvous.ready() ||
			coalesceBool(a.model.IsReady, false),
	}
}

func (a *Allocation) setModelState(v model.AllocationState) {
	a.model.State = &v
}

func (a *Allocation) setMostProgressedModelState(v model.AllocationState) {
	a.setModelState(model.MostProgressedAllocationState(a.getModelState(), v))
}

func (a *Allocation) getModelState() model.AllocationState {
	if a.model.State == nil {
		return model.AllocationStatePending
	}
	return *a.model.State
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
func (a AllocationState) FirstContainer() *cproto.Container {
	for _, cs := range a.Containers {
		for _, c := range cs {
			return &c
		}
	}
	return nil
}

// FirstContainerAddresses returns the first container's addresses in the allocation state.
func (a AllocationState) FirstContainerAddresses() []cproto.Address {
	for _, ca := range a.Addresses {
		return ca
	}
	return nil
}

func coalesceBool(x *bool, fallback bool) bool {
	if x == nil {
		return fallback
	}
	return *x
}

func coalesceString(x *string, fallback string) string {
	if x == nil {
		return fallback
	}
	return *x
}
