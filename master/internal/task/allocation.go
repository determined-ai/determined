package task

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/portregistry"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/allocationmap"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/idle"
	"github.com/determined-ai/determined/master/internal/task/preemptible"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
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
		db db.DB
		rm rm.ResourceManager

		// The request to create the allocation, essentially our configuration.
		req sproto.AllocateRequest
		// The persisted representation.
		model model.Allocation
		// The task spec to run.
		specifier tasks.TaskSpecifier

		// State of all our resources.
		resources resourcesList
		// Separates the existence of resources from us having started them.
		resourcesStarted bool
		// Tracks the initial container exit, unless we caused the failure by killed the trial.
		exitErr error
		// Marks that we intentionally killed the allocation so we can know to
		// ignore any errors from containers dying. Not set when we kill an already
		// terminating trial.
		killedWhileRunning bool
		// Marks that the trial exited successfully, but we killed some daemon containers.
		killedDaemons bool
		// Marks that we killed some daemon containers but after a zero exit.
		killedDaemonsGracefully bool
		// We send a kill when we terminate a task forcibly. we terminate forcibly when a container
		// exits non zero. we don't need to send all these kills, so this exists.
		killCooldown *time.Time
		// tracks if we have finished termination.
		exited bool

		// State for specific sub-behaviors of an allocation.
		// Encapsulates logic of rendezvousing containers of the currently
		// allocated task. If there is no current task, or it is unallocated, it is nil.
		rendezvous *rendezvous
		// proxy state
		proxies []string
		// active all gather state
		allGather *allGather
		// records whether the allocation has completed any all gathers.
		allGatherFinished bool

		logCtx          detLogger.Context
		restored        bool
		portsRegistered bool
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
	// AllocationWaiting marks an allocation as waiting.
	AllocationWaiting struct {
		Message string
	}
	// SetAllocationProxyAddress manually sets the allocation proxy address.
	SetAllocationProxyAddress struct {
		ProxyAddress string
	}
	// IsAllocationRestoring asks the allocation if it is in the middle of a restore.
	IsAllocationRestoring struct{}
)

const (
	killCooldown       = 15 * time.Second
	okExitMessage      = "allocation exited successfully"
	missingExitMessage = ""
)

// NewAllocation returns a new allocation, which tracks allocation state in a fairly generic way.
func NewAllocation(
	logCtx detLogger.Context, req sproto.AllocateRequest, db db.DB, rm rm.ResourceManager,
	specifier tasks.TaskSpecifier,
) actor.Actor {
	req.LogContext = detLogger.MergeContexts(logCtx, detLogger.Context{
		"allocation-id": req.AllocationID,
	})
	return &Allocation{
		db: db,
		rm: rm,

		req: req,
		model: model.Allocation{
			AllocationID: req.AllocationID,
			TaskID:       req.TaskID,
			Slots:        req.SlotsNeeded,
			ResourcePool: req.ResourcePool,
			Ports:        map[string]int{},
		},
		specifier: specifier,

		resources: resourcesList{},

		logCtx: req.LogContext,
	}
}

// Receive implements actor.Actor for the allocation.
// The normal flow of an Allocation is to:
//
//	(1) request resources,
//	(2) receive resources,
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
		allocationmap.RegisterAllocation(a.model.AllocationID, ctx.Self())
		ctx.AddLabels(a.logCtx)
		if err := a.RequestResources(ctx); err != nil {
			a.Error(ctx, err)
		}

	case IsAllocationRestoring:
		ctx.Respond(a.req.Restore && !a.restored)

	case sproto.ResourcesAllocated:
		if err := a.ResourcesAllocated(ctx, msg); err != nil {
			a.Error(ctx, err)
		}
	case sproto.ResourcesStateChanged:
		a.ResourcesStateChanged(ctx, msg)
	case sproto.ResourcesFailure:
		a.RestoreResourceFailure(ctx, msg)
	case sproto.GetResourcesContainerState:
		if v, ok := a.resources[msg.ResourcesID]; ok {
			if v.Container == nil {
				ctx.Respond(fmt.Errorf("no container associated with %s", msg.ResourcesID))
			} else {
				ctx.Respond(*v.Container)
			}
		} else {
			ctx.Respond(fmt.Errorf("unknown resources %s", msg.ResourcesID))
		}
	case sproto.ReleaseResources:
		a.Terminate(ctx, "allocation being preempted by the scheduler", msg.ForcePreemption)
	case sproto.ChangeRP:
		a.Terminate(ctx, "allocation resource pool changed", false)
	case actor.PostStop:
		a.Cleanup(ctx)
		// a.portsRegistered  is set to true right after ports are registered.
		// This variable ensures to release ports even if there's a failure after restoring ports.
		if a.portsRegistered {
			for _, port := range a.model.Ports {
				portregistry.ReleasePort(port)
			}
		}
		if a.req.Preemptible {
			preemptible.Unregister(a.req.AllocationID.String())
		}
		if cfg := a.req.IdleTimeout; cfg != nil {
			idle.Unregister(cfg.ServiceID)
		}
		allocationmap.UnregisterAllocation(a.model.AllocationID)
	case sproto.ContainerLog:
		a.sendTaskLog(msg.ToTaskLog())

	// These messages allow users (and sometimes an orchestrator, such as HP search)
	// to interact with the allocation. The usually trace back to API calls.
	case AllocationReady:
		// AllocationReady only comes from the running container, so to
		// avoid a race condition with the slower transition to running state
		// which comes via polling for dispatcher RM, move the state to running now.
		a.setMostProgressedModelState(model.AllocationStateRunning)
		a.model.IsReady = ptrs.Ptr(true)
		if err := a.db.UpdateAllocationState(a.model); err != nil {
			a.Error(ctx, err)
		}
		a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("Service of %s is available", a.req.Name)})
	case AllocationWaiting:
		a.setMostProgressedModelState(model.AllocationStateWaiting)
		if err := a.db.UpdateAllocationState(a.model); err != nil {
			a.Error(ctx, err)
		}
	case MarkResourcesDaemon:
		if err := a.SetResourcesAsDaemon(ctx, msg.AllocationID, msg.ResourcesID); err != nil {
			a.Error(ctx, err)
		}
	case sproto.AllocationSignal:
		a.HandleSignal(ctx, sproto.AllocationSignalWithReason{AllocationSignal: msg})
	case sproto.AllocationSignalWithReason:
		a.HandleSignal(ctx, msg)
	case AllocationState:
		if ctx.ExpectingResponse() {
			ctx.Respond(a.State())
		}
	case SetAllocationProxyAddress:
		if len(a.req.ProxyPorts) == 0 {
			if ctx.ExpectingResponse() {
				ctx.Respond(ErrBehaviorUnsupported{Behavior: fmt.Sprintf("%T", msg)})
			}
			return nil
		}
		a.model.ProxyAddress = &msg.ProxyAddress
		if err := a.db.UpdateAllocationProxyAddress(a.model); err != nil {
			a.Error(ctx, err)
			return nil
		}
		a.registerProxies(ctx, a.containerProxyAddresses())
	case WatchRendezvousInfo, UnwatchRendezvousInfo, rendezvousTimeout:
		if a.rendezvous == nil {
			if len(a.resources) == 0 {
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
				a.sendTaskLog(&model.TaskLog{Log: err.Error()})
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
				a.sendTaskLog(&model.TaskLog{Log: err.Error()})
				ctx.Log().WithError(err).Error("performing all gather through master")
			}
		default:
			return actor.ErrUnexpectedMessage(ctx)
		}

		if a.allGather.done() {
			a.allGather = nil
			a.allGatherFinished = true
		}
	case sproto.InvalidResourcesRequestError:
		ctx.Tell(a.req.AllocationRef, msg)
		a.Error(ctx, msg)

	default:
		a.Error(ctx, actor.ErrUnexpectedMessage(ctx))
	}
	return nil
}

// RequestResources sets up the allocation.
func (a *Allocation) RequestResources(ctx *actor.Context) error {
	if a.req.Restore {
		// Load allocation.
		ctx.Log().Debug("RequestResources load allocation")
		err := db.Bun().NewSelect().Model(&a.model).
			Where("allocation_id = ?", a.model.AllocationID).
			Scan(context.TODO())
		if err != nil {
			return errors.Wrap(err, "loading trial allocation")
		}
	} else {
		// Insert new allocation.
		ctx.Log().Debug("RequestResources add allocation")

		a.setModelState(model.AllocationStatePending)
		if err := a.db.AddAllocation(&a.model); err != nil {
			return errors.Wrap(err, "saving trial allocation")
		}
	}

	a.req.AllocationRef = ctx.Self()
	if err := a.rm.Allocate(ctx, a.req); err != nil {
		return errors.Wrap(err, "failed to request allocation")
	}
	a.sendTaskLog(&model.TaskLog{
		Log: fmt.Sprintf("Scheduling %s (id: %s)", a.req.Name, ctx.Self().Parent().Address().Local()),
	})
	return nil
}

// Cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
func (a *Allocation) Cleanup(ctx *actor.Context) {
	// Just in-case code.
	if !a.exited {
		ctx.Log().Info("exit did not run properly")
		for _, r := range a.resources {
			if r.Exited == nil {
				ctx.Log().Infof("allocation exited with unterminated reservation: %v", r.Summary())
				r.Kill(ctx, a.logCtx)
			}
		}
		if a.resourcesStarted {
			a.markResourcesReleased(ctx)
		}

		if err := a.purgeRestorableResources(ctx); err != nil {
			ctx.Log().WithError(err).Error("failed to purge restorable resources")
		}

		a.sendTaskLog(&model.TaskLog{
			Log: fmt.Sprintf("%s was terminated: %s", a.req.Name, "allocation did not exit correctly"),
		})
		a.rm.Release(ctx, sproto.ResourcesReleased{AllocationID: a.req.AllocationID})
	}
}

// ResourcesAllocated handles receiving resources from the resource manager. Note: it makes a single
// ask to the parent to build its task spec.. this is mostly a hack to defer lots of computationally
// heavy stuff unless it is necessarily (which also works to spread occurrences of the same work
// out). Eventually, Allocations should just be started with their TaskSpec.
func (a *Allocation) ResourcesAllocated(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	if !a.req.Restore {
		if a.getModelState() != model.AllocationStatePending {
			// If we have moved on from the pending state, these must be stale (and we must have
			// already released them, just the scheduler hasn't gotten word yet).
			return ErrStaleResourcesReceived{}
		}

		a.setModelState(model.AllocationStateAssigned)
	} else {
		ctx.Log().Debugf("ResourcesAllocated restored state: %s", a.getModelState())
	}

	a.setMostProgressedModelState(model.AllocationStateAssigned)
	if err := a.resources.append(msg.Resources); err != nil {
		return errors.Wrapf(err, "appending resources")
	}

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

	if a.req.Preemptible {
		preemptible.Register(a.req.AllocationID.String())
	}

	if cfg := a.req.IdleTimeout; cfg != nil {
		idle.Register(*cfg, func(err error) {
			ctx.Log().WithError(err).Infof("killing %s due to inactivity", a.req.Name)
			ctx.Tell(ctx.Self(), sproto.AllocationSignalWithReason{
				AllocationSignal:    sproto.TerminateAllocation,
				InformationalReason: err.Error(),
			})
		})
	}

	if a.req.Restore {
		for _, port := range a.model.Ports {
			portregistry.RestorePort(port)
		}
		a.portsRegistered = true
		if a.getModelState() == model.AllocationStateRunning {
			// Restore proxies.
			if len(a.req.ProxyPorts) > 0 {
				for _, r := range a.resources {
					switch {
					case r.Rank == 0 && r.Started != nil && r.Started.Addresses != nil:
						a.registerProxies(ctx, r.Started.Addresses)
					case a.model.ProxyAddress != nil:
						a.registerProxies(ctx, a.containerProxyAddresses())
					}
				}
			}
		}
	} else {
		spec := a.specifier.ToTaskSpec()

		token, err := a.db.StartAllocationSession(a.model.AllocationID, spec.Owner)
		if err != nil {
			return errors.Wrap(err, "starting a new allocation session")
		}

		a.model.Ports, err = a.getPorts(spec.UniqueExposedPortRequests, ctx)
		if err != nil {
			return errors.Wrap(err, "getting ports")
		}
		a.portsRegistered = true
		err = db.UpdateAllocationPorts(a.model)
		if err != nil {
			return fmt.Errorf("updating allocation db")
		}

		for portName, port := range a.model.Ports {
			spec.Environment.RawPorts[portName] = port
			spec.ExtraEnvVars[portName] = strconv.Itoa(port)
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
	}

	a.restored = a.req.Restore
	a.resourcesStarted = true
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
		a.sendTaskLog(&model.TaskLog{
			Log: `Ignoring request to daemonize resources within an allocation for an allocation
			with only one manageable set of resources, because this would just kill it. This is
			expected in when using the HPC launcher.`,
			Level: ptrs.Ptr(model.LogLevelInfo),
		})
		return nil
	}

	a.resources[rID].Daemon = true
	if err := a.resources[rID].Persist(); err != nil {
		return err
	}

	if len(a.resources.daemons()) == len(a.resources) {
		ctx.Log().Warnf("all resources were marked as daemon, exiting")
		a.Kill(ctx, "all resources were marked as daemon")
	}

	return nil
}

// HandleSignal handles an external signal to kill or terminate the allocation.
func (a *Allocation) HandleSignal(ctx *actor.Context, msg sproto.AllocationSignalWithReason) {
	switch msg.AllocationSignal {
	case sproto.KillAllocation:
		a.Kill(ctx, msg.InformationalReason)
	case sproto.TerminateAllocation:
		a.Terminate(ctx, msg.InformationalReason, false)
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

	a.resources[msg.ResourcesID].Container = msg.Container
	ctx.Log().Debugf("resources state changed: %+v", msg)
	switch msg.ResourcesState {
	case sproto.Pulling:
		a.setMostProgressedModelState(model.AllocationStatePulling)
		if a.model.StartTime == nil {
			a.markResourcesStarted(ctx)
		}
	case sproto.Starting:
		a.setMostProgressedModelState(model.AllocationStateStarting)
	case sproto.Running:
		if a.resources[msg.ResourcesID].Started != nil {
			// Only recognize the first start message for each resource, since the slurm resource
			// manager is polling based instead and sends us a message that the resources are
			// running each time it polls.
			return
		}

		a.setMostProgressedModelState(model.AllocationStateRunning)
		if a.model.StartTime == nil {
			a.markResourcesStarted(ctx)
		}

		a.resources[msg.ResourcesID].Started = msg.ResourcesStarted
		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.Error(ctx, err)
			return
		}

		if a.rendezvous != nil && a.rendezvous.try() {
			ctx.Log().
				Info("all containers are connected successfully (task container state changed)")
		}
		if len(a.req.ProxyPorts) > 0 && msg.ResourcesStarted.Addresses != nil &&
			a.resources[msg.ResourcesID].Rank == 0 {
			a.registerProxies(ctx, msg.ResourcesStarted.Addresses)
		}

		containerID := coalesceString(msg.ContainerIDStr(), "")
		a.sendTaskLog(&model.TaskLog{
			ContainerID: &containerID,
			Log:         fmt.Sprintf("Resources for %s have started", a.req.Name),
		})

		prom.AssociateAllocationTask(a.req.AllocationID,
			a.req.TaskID,
			a.req.AllocationRef.Address(),
			a.req.JobID)
		prom.AddAllocationResources(a.resources[msg.ResourcesID].Summary(), msg.ResourcesStarted)

	case sproto.Terminated:
		if a.resources[msg.ResourcesID].Exited != nil {
			// If we have already received the exit for this container, we only recognize the first.
			// If there are multiples, it's likely due to one being resent after a kill signal was
			// repeated. Agents always re-ack termination to ensure it is received in the event
			// of network failures and they always re-ack the same exit, anyway.
			return
		}

		a.setMostProgressedModelState(model.AllocationStateTerminating)

		a.resources[msg.ResourcesID].Exited = msg.ResourcesStopped

		a.rm.Release(ctx, sproto.ResourcesReleased{
			AllocationID: a.req.AllocationID,
			ResourcesID:  &msg.ResourcesID,
		})

		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.Error(ctx, err)
			return
		}

		switch {
		case a.killedWhileRunning:
			a.sendTaskLog(&model.TaskLog{
				ContainerID: msg.ContainerIDStr(),
				Log: fmt.Sprintf(
					"resources were killed: %s",
					msg.ResourcesStopped.String(),
				),
			})
			a.Exit(ctx, "resources were killed")
		case msg.ResourcesStopped.Failure != nil:
			// Avoid erroring out if we have killed our daemons gracefully.
			// This occurs in the case of an early stop in dtrain. One resource
			// will exit with a 0 exit code and kill the rest of the resources sending
			// failed messages for these resources.
			if a.killedDaemonsGracefully {
				a.Exit(ctx, "remaining resources terminated")
			} else {
				a.Error(ctx, *msg.ResourcesStopped.Failure)
			}
		default:
			a.sendTaskLog(&model.TaskLog{
				ContainerID: msg.ContainerIDStr(),
				Log:         msg.ResourcesStopped.String(),
				Level:       ptrs.Ptr(model.LogLevelInfo),
			})
			a.Exit(ctx, msg.ResourcesStopped.String())
		}

		for cID := range a.resources {
			prom.DisassociateAllocationTask(a.req.AllocationID,
				a.req.TaskID,
				a.req.AllocationRef.Address(),
				a.req.JobID)
			prom.RemoveAllocationResources(a.resources[cID].Summary())
		}
	}

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		ctx.Log().Error(err)
	}
}

// RestoreResourceFailure handles the restored resource failures.
func (a *Allocation) RestoreResourceFailure(
	ctx *actor.Context, msg sproto.ResourcesFailure,
) {
	ctx.Log().Debugf("allocation resource failure")
	a.setMostProgressedModelState(model.AllocationStateTerminating)

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		ctx.Log().Error(err)
	}

	if a.req.Restore {
		// TODO(DET-8822): This heartbeat can be nil.
		switch heartbeat := cluster.TheLastBootClusterHeartbeat(); {
		case a.model.StartTime == nil:
			break
		case heartbeat.Before(*a.model.StartTime):
			a.model.EndTime = a.model.StartTime
		default:
			a.model.EndTime = heartbeat
		}
	} else {
		a.model.EndTime = ptrs.Ptr(time.Now().UTC())
	}

	if err := a.db.CompleteAllocation(&a.model); err != nil {
		ctx.Log().WithError(err).Error("failed to mark allocation completed")
	}

	a.Error(ctx, msg)
}

// Exit attempts to exit an allocation while not killing or preempting it.
func (a *Allocation) Exit(ctx *actor.Context, reason string) (exited bool) {
	switch {
	case !a.resourcesStarted:
		a.terminated(ctx, reason)
		return true
	case len(a.resources.exited()) == len(a.resources):
		a.terminated(ctx, reason)
		return true
	case a.allNonDaemonsExited():
		a.killedDaemons = true
		if a.exitedWithoutErr() {
			a.killedDaemonsGracefully = true
		}
		a.kill(ctx, reason)
	case len(a.resources.failed()) > 0:
		a.kill(ctx, reason)
	}
	return false
}

// Terminate attempts to close an allocation by gracefully stopping it (though a kill are possible).
func (a *Allocation) Terminate(ctx *actor.Context, reason string, forcePreemption bool) {
	if exited := a.Exit(ctx, reason); exited {
		return
	}

	switch {
	case a.req.Preemptible && a.ready() || forcePreemption:
		a.preempt(ctx, reason)
	default:
		a.kill(ctx, reason)
	}
}

// Kill attempts to close an allocation by killing it.
func (a *Allocation) Kill(ctx *actor.Context, reason string) {
	if exited := a.Exit(ctx, reason); exited {
		return
	}
	a.kill(ctx, reason)
}

// Error closes the allocation due to an error, beginning the kill flow.
func (a *Allocation) Error(ctx *actor.Context, err error) {
	ctx.Log().WithError(err).Errorf("allocation encountered fatal error")
	if a.exitErr == nil {
		a.exitErr = err
	}
	a.Kill(ctx, err.Error())
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

func (a *Allocation) exitedWithoutErr() bool {
	for _, r := range a.resources.failed() {
		code := r.Exited.Failure.ExitCode
		if code != nil && *code != 0 {
			return false
		}
	}
	return true
}

func (a *Allocation) preempt(ctx *actor.Context, reason string) {
	ctx.Log().WithField("reason", reason).Info("decided to gracefully terminate allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"gracefully terminating allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	preemptible.Preempt(a.req.AllocationID.String(), func(err error) {
		ctx.Tell(ctx.Self(), sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: err.Error(),
		})
	})
}

func (a *Allocation) kill(ctx *actor.Context, reason string) {
	if a.killCooldown != nil && time.Now().Before(*a.killCooldown) {
		ctx.Log().Debug("still inside of kill cooldown")
		return
	}

	ctx.Log().WithField("reason", reason).Info("decided to kill allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"forcibly killing allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	for _, r := range a.resources.active() {
		r.Kill(ctx, a.logCtx)
	}

	if len(a.resources.exited()) == 0 {
		a.killedWhileRunning = true
	}

	// Once a job has been killed, resend the kill every 30s, in the event it is lost (has
	// happened before due to network failures).
	a.killCooldown = ptrs.Ptr(time.Now().Add(killCooldown))
	actors.NotifyAfter(ctx, killCooldown*2, sproto.AllocationSignalWithReason{
		AllocationSignal:    sproto.KillAllocation,
		InformationalReason: "killing again after 30s without all container exits",
	})
}

func (a *Allocation) registerProxies(ctx *actor.Context, addresses []cproto.Address) {
	// For multi-reservation allocations, proxies are only setup for rank=0 (i.e. the chief).
	if len(a.req.ProxyPorts) == 0 {
		return
	}

	for _, address := range addresses {
		// Only proxy the port we expect to proxy. If a dockerfile uses an EXPOSE command,
		// additional addresses will appear her, but currently we only proxy one uuid to one
		// port, so it doesn't make sense to send multiple proxy.Register messages for a
		// single ServiceID (only the last one would work).
		var pcfg *sproto.ProxyPortConfig
		for _, cfg := range a.req.ProxyPorts {
			if address.ContainerPort == cfg.Port {
				pcfg = cfg
			}
		}
		if pcfg == nil {
			continue
		}

		// We are keying on allocation id instead of container id. Revisit this when we need to
		// proxy multi-container tasks or when containers are created prior to being
		// assigned to an agent.
		proxy.DefaultProxy.Register(pcfg.ServiceID, &url.URL{
			Scheme: "http",
			Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
		}, pcfg.ProxyTCP, pcfg.Unauthenticated)
		ctx.Log().Debugf("registered proxy id: %s, tcp: %v\n", pcfg.ServiceID, pcfg.ProxyTCP)
		a.proxies = append(a.proxies, pcfg.ServiceID)
	}

	if len(a.proxies) != len(a.req.ProxyPorts) {
		a.sendTaskLog(&model.TaskLog{
			Log: fmt.Sprintf(
				"did not proxy as expected %v (found addrs %v, requested %v)",
				len(a.proxies), addresses, len(a.req.ProxyPorts)),
		})
	}
}

func (a *Allocation) unregisterProxies(ctx *actor.Context) {
	if len(a.req.ProxyPorts) == 0 {
		return
	}

	if len(a.resources) > 1 {
		// Can't proxy more than one reservation, so we never would've made them.
		return
	}

	for _, serviceID := range a.proxies {
		proxy.DefaultProxy.Unregister(serviceID)
	}
}

// containerProxyAddresses forms the container address _only_ when proxyAddress is given.
func (a *Allocation) containerProxyAddresses() []cproto.Address {
	if a.model.ProxyAddress == nil || len(a.req.ProxyPorts) == 0 {
		return []cproto.Address{}
	}

	result := []cproto.Address{}

	for _, pp := range a.req.ProxyPorts {
		result = append(result, cproto.Address{
			ContainerIP:   *a.model.ProxyAddress,
			ContainerPort: pp.Port,
			HostIP:        *a.model.ProxyAddress,
			HostPort:      pp.Port,
		})
	}

	return result
}

func (a *Allocation) terminated(ctx *actor.Context, reason string) {
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
	exitReason := fmt.Sprintf("allocation terminated after %s", reason)
	defer ctx.Tell(ctx.Self().Parent(), exit)
	defer a.rm.Release(ctx, sproto.ResourcesReleased{AllocationID: a.req.AllocationID})
	defer a.unregisterProxies(ctx)
	defer ctx.Self().Stop()

	level := ptrs.Ptr(model.LogLevelInfo)
	if a.exitErr != nil {
		level = ptrs.Ptr(model.LogLevelError)
	}
	defer func() {
		a.sendTaskLog(&model.TaskLog{
			Level: level,
			Log:   fmt.Sprintf("%s was terminated: %s", a.req.Name, exitReason),
		})
	}()

	if err := a.purgeRestorableResources(ctx); err != nil {
		ctx.Log().WithError(err).Error("failed to purge restorable resources")
	}

	defer a.markResourcesReleased(ctx)

	if a.req.Preemptible {
		defer preemptible.Unregister(a.req.AllocationID.String())
	}
	if a.rendezvous != nil {
		defer a.rendezvous.close()
	}
	if cfg := a.req.IdleTimeout; cfg != nil {
		defer idle.Unregister(cfg.ServiceID)
	}
	switch {
	case a.killedWhileRunning:
		exitReason = fmt.Sprintf("allocation stopped after %s", reason)
		ctx.Log().Info(exitReason)
		return
	case a.req.Preemptible && preemptible.Acknowledged(a.req.AllocationID.String()):
		exitReason = fmt.Sprintf("allocation stopped after %s", reason)
		ctx.Log().Info(exitReason)
		return
	case a.exitErr == nil && len(a.resources.exited()) > 0:
		// This is true because searcher and preemption exits both ack preemption.
		exit.UserRequestedStop = true
		exitReason = fmt.Sprintf("allocation stopped early after %s", reason)
		ctx.Log().Info(exitReason)
		return
	case a.exitErr != nil:
		switch err := a.exitErr.(type) {
		case sproto.ResourcesFailure:
			switch err.FailureType {
			case sproto.ResourcesFailed, sproto.TaskError:
				if a.killedDaemonsGracefully {
					exitReason = fmt.Sprint("allocation terminated daemon processes as part of normal exit")
					ctx.Log().Info(exitReason)
					return
				}
				exitReason = fmt.Sprintf("allocation failed: %s", err)
				ctx.Log().Info(exitReason)
				exit.Err = err
				return
			case sproto.AgentError, sproto.AgentFailed:
				exitReason = fmt.Sprintf("allocation failed due to agent failure: %s", err)
				ctx.Log().Warn(exitReason)
				exit.Err = err
				return
			case sproto.TaskAborted, sproto.ResourcesAborted:
				exitReason = fmt.Sprintf("allocation aborted: %s", err.FailureType)
				ctx.Log().Debug(exitReason)
				exit.Err = err
				return
			case sproto.RestoreError:
				exitReason = fmt.Sprintf("allocation failed due to restore error: %s", err)
				ctx.Log().Warn(exitReason)
				exit.Err = err
				return

			default:
				panic(fmt.Errorf("unexpected allocation failure: %w", err))
			}
		default:
			exitReason = fmt.Sprintf("allocation handler crashed due to error: %s", err)
			ctx.Log().Error(exitReason)
			exit.Err = err
			return
		}
	case len(a.resources) == 0:
		return
	default:
		// If we ever exit without a reason and we have no exited resources, something has gone
		// wrong.
		panic("allocation exited early without a valid reason")
	}
}

// markResourcesStarted persists start information.
func (a *Allocation) markResourcesStarted(ctx *actor.Context) {
	a.model.StartTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	if a.restored {
		a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("%s was recovered on an agent", a.req.Name)})
	} else {
		a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("%s was assigned to an agent", a.req.Name)})
	}
	if err := a.db.UpdateAllocationStartTime(a.model); err != nil {
		ctx.Log().
			WithError(err).
			Errorf("allocation will not be properly accounted for")
	}
}

// markResourcesReleased persists completion information.
func (a *Allocation) markResourcesReleased(ctx *actor.Context) {
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		ctx.Log().WithError(err).Error("error deleting allocation session")
	}
	if a.model.StartTime == nil {
		return
	}
	a.model.EndTime = ptrs.Ptr(time.Now().UTC())
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		ctx.Log().WithError(err).Error("failed to mark allocation completed")
	}

	telemetry.ReportAllocationTerminal(
		ctx.Self().System(), a.db, a.model, a.resources.firstDevice())
}

func (a *Allocation) purgeRestorableResources(ctx *actor.Context) error {
	_, err := db.Bun().NewDelete().Model((*taskmodel.ResourcesWithState)(nil)).
		Where("allocation_id = ?", a.model.AllocationID).
		Exec(context.TODO())

	return err
}

const killedLogSubstr = "exit code 137"

func (a *Allocation) enrichLog(log *model.TaskLog) *model.TaskLog {
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

func (a *Allocation) sendTaskLog(log *model.TaskLog) {
	tasklogger.Insert(a.enrichLog(log))
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
		case a.model.ProxyAddress != nil:
			addresses[id] = a.containerProxyAddresses()
		}

		if r.Container != nil {
			containers[id] = append(containers[id], *r.Container)
		}
	}

	return AllocationState{
		State:      a.getModelState(),
		Resources:  resources,
		Addresses:  addresses,
		Containers: containers,
		Ready:      a.ready(),
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

func (a *Allocation) ready() bool {
	// Most trials use `a.rendezvous` and the normal rendezvous APIs, and go through this path.
	return (a.rendezvous != nil && a.rendezvous.ready()) ||
		// But HPC trials don't, they don't use `a.rendezvous` at all but just do an allgather,
		// so we check if we have done at least one, which also indicates all the workers are up.
		a.allGatherFinished ||
		// And finally, of course, if the task explicitly called `AllocationReady` it is ready.
		coalesceBool(a.model.IsReady, false)
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

func (a *Allocation) getPorts(exposedPorts map[string]int,
	ctx *actor.Context,
) (map[string]int, error) {
	ports := make(map[string]int)
	var err error
	defer func() {
		if err != nil {
			for _, port := range ports {
				portregistry.ReleasePort(port)
			}
		}
	}()
	for portName, base := range exposedPorts {
		port, err := portregistry.GetPort(base)
		if err != nil {
			return nil, fmt.Errorf("getting %v port from the registry for an allocation", portName)
		}
		ports[portName] = port
		ctx.Log().Debugf("%v port : %v", portName, port)
	}

	return ports, nil
}
