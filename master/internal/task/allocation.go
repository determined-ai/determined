package task

import (
	"context"
	"fmt"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/portregistry"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/idle"
	"github.com/determined-ai/determined/master/internal/task/preemptible"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	detLogger "github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const killCooldown = 15 * time.Second

// AllocationSignal is an interface for signals that can be sent to an allocation.
type AllocationSignal string

const (
	// KillAllocation is the signal to kill an allocation; analogous to SIGKILL.
	KillAllocation AllocationSignal = "kill"
	// TerminateAllocation is the signal to kill an allocation; analogous to SIGTERM.
	TerminateAllocation AllocationSignal = "terminate"
)

// AllocationState requests allocation state. A copy is filled and returned.
type AllocationState struct {
	State     model.AllocationState
	Resources map[sproto.ResourcesID]sproto.ResourcesSummary
	Ready     bool

	Addresses  map[sproto.ResourcesID][]cproto.Address
	Containers map[sproto.ResourcesID][]cproto.Container
}

// SingleContainer returns a single random container from the allocation state.
func (a AllocationState) SingleContainer() *cproto.Container {
	for _, cs := range a.Containers {
		for _, c := range cs {
			return &c
		}
	}
	return nil
}

// SingleContainerAddresses returns a single random container's addresses from the allocation state.
func (a AllocationState) SingleContainerAddresses() []cproto.Address {
	for _, ca := range a.Addresses {
		return ca
	}
	return nil
}

// AllocationExited summarizes the exit status of an allocation.
type AllocationExited struct {
	// userRequestedStop is when a container unexpectedly exits with 0.
	UserRequestedStop bool
	Err               error
	FinalState        AllocationState
}

func (a *AllocationExited) String() string {
	switch {
	case a == nil:
		return ""
	case a.Err != nil:
		return a.Err.Error()
	default:
		return "allocation exited successfully"
	}
}

// allocation encapsulates all the state of a single allocation.
type allocation struct {
	mu sync.Mutex

	// System dependencies.
	db db.DB
	rm rm.ResourceManager

	syslog *logrus.Entry
	system *actor.System
	wg     waitgroupx.Group

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
	exited *AllocationExited

	// State for specific sub-behaviors of an allocation.
	// Encapsulates logic of rendezvousing containers of the currently
	// allocated task. If there is no current task, or it is unallocated, it is nil.
	rendezvous *rendezvous
	// proxy state
	proxies []string

	logCtx          detLogger.Context
	restored        bool
	portsRegistered bool

	closers []func()
}

// newAllocation returns a new allocation, which tracks allocation state in a fairly generic way.
func newAllocation(
	logCtx detLogger.Context, req sproto.AllocateRequest, db db.DB, rm rm.ResourceManager,
	specifier tasks.TaskSpecifier, system *actor.System,
) (*allocation, error) {
	req.LogContext = detLogger.MergeContexts(logCtx, detLogger.Context{
		"allocation-id": req.AllocationID,
	})

	if req.RequestTime.IsZero() {
		req.RequestTime = time.Now().UTC()
	}

	a := &allocation{
		db: db,
		rm: rm,

		system: system,
		wg:     waitgroupx.WithContext(context.Background()),
		syslog: logrus.WithFields(logCtx.Fields()),

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

	rmEvents, err := a.requestResources()
	if err != nil {
		return nil, fmt.Errorf("requesting resources: %w", err)
	}
	a.wg.Go(func(ctx context.Context) { a.run(ctx, rmEvents) })
	return a, nil
}

// Receive implements actor.Actor for the allocation.
// The normal flow of an allocation is to:
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
func (a *allocation) run(ctx context.Context, sub *sproto.ResourcesSubscription) {
	defer a.recover()
	defer sub.Close()
	defer a.wg.Cancel() // Important if we panic, so awaitTermination can unblock.
	for {
		event := sub.Get()
		if event == (sproto.ResourcesReleasedEvent{}) {
			return
		}
		a.HandleRMEvent(event)
	}
}

// HandleRMEvent handles downstream events from the resource manager.
func (a *allocation) HandleRMEvent(msg sproto.ResourcesEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch msg := msg.(type) {
	case *sproto.ResourcesAllocated:
		if err := a.resourcesAllocated(msg); err != nil {
			a.crash(err)
		}
	case *sproto.ResourcesStateChanged:
		a.resourcesStateChanged(msg)
	case *sproto.ResourcesFailure:
		a.restoreResourceFailure(msg)
	case *sproto.ReleaseResources:
		a.releaseResources(msg)
	case *sproto.ContainerLog:
		a.sendTaskLog(msg.ToTaskLog())
	case *sproto.InvalidResourcesRequestError:
		a.crash(msg.Cause)
	default:
		panic(fmt.Errorf("unexpected RM event"))
	}
}

// State returns a copy of the current State of the allocation.
func (a *allocation) State() AllocationState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state()
}

// IsRestoring returns if the allocation has been restored by the resource manager.
func (a *allocation) IsRestoring() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.req.Restore && !a.restored
}

// waitForRestore waits until the allocation has been restored by the resource manager or a minute
// has passed. If a minute passes, an error is returned. The allocation must exist otherwise this
// will return a not found error.
func (a *allocation) waitForRestore(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	t := time.NewTicker(time.Second)
	defer t.Stop()

	for {
		if !a.IsRestoring() {
			return nil
		}

		select {
		case <-t.C:
		case <-ctx.Done():
			return fmt.Errorf("allocation stuck restoring: %w", ctx.Err())
		}
	}
}

// Signal handles an external Signal to kill or terminate the allocation.
func (a *allocation) Signal(sig AllocationSignal, reason string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch sig {
	case KillAllocation:
		a.tryExitOrKill(reason)
	case TerminateAllocation:
		a.tryExitOrTerminate(reason, false)
	}
}

// SetProxyAddress sets the proxy address of the allocation and sets up proxies for any services
// it provides.
func (a *allocation) SetProxyAddress(_ context.Context, address string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.req.ProxyPorts) == 0 {
		a.syslog.Debug("No ports to proxy. Skipping proxy registration.")
		return nil
	}
	a.model.ProxyAddress = &address
	if err := a.db.UpdateAllocationProxyAddress(a.model); err != nil {
		a.crash(err)
		return err
	}
	a.registerProxies(a.containerProxyAddresses())
	a.closers = append(a.closers, a.unregisterProxies)
	return nil
}

// SendContainerLog sends a container log, enriched with metadata from the allocation.
func (a *allocation) SendContainerLog(log *sproto.ContainerLog) {
	a.sendTaskLog(log.ToTaskLog())
}

// SetWaiting moves the allocation to the waiting state if it has not progressed past it yet.
func (a *allocation) SetWaiting(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.setMostProgressedModelState(model.AllocationStateWaiting)
	if err := a.db.UpdateAllocationState(a.model); err != nil {
		a.crash(err)
		return err
	}
	return nil
}

// SetReady sets the ready bit and moves the allocation to the running state if it has not
// progressed past it already.
func (a *allocation) SetReady(_ context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// AllocationReady only comes from the running container, so to
	// avoid a race condition with the slower transition to running state
	// which comes via polling for dispatcher RM, move the state to running now.
	a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("Service of %s is available", a.req.Name)})
	a.setMostProgressedModelState(model.AllocationStateRunning)
	a.model.IsReady = ptrs.Ptr(true)
	if err := a.db.UpdateAllocationState(a.model); err != nil {
		a.crash(err)
		return err
	}
	return nil
}

// SetResourcesAsDaemon marks the resources as daemons. If all non-daemon resources exit, the
// allocation will kill the remaining daemon resources.
func (a *allocation) SetResourcesAsDaemon(_ context.Context, rID sproto.ResourcesID) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.resources[rID]; !ok {
		return ErrStaleResources{ID: rID}
	} else if len(a.resources) <= 1 {
		// Ignoring request to daemonize resources within an allocation for an allocation
		// 	with only one manageable set of resources, because this would just kill it. This is
		// 	expected when using the HPC launcher.
		a.syslog.Debug(`ignoring request to daemonize resources`)
		return nil
	}

	a.syslog.Debugf("setting resources as daemon %s", rID)
	a.resources[rID].Daemon = true
	if err := a.resources[rID].Persist(); err != nil {
		a.crash(err)
		return err
	}

	if len(a.resources.daemons()) == len(a.resources) {
		a.syslog.Warnf("all resources were marked as daemon, exiting")
		a.tryExitOrKill("all resources were marked as daemon")
	}
	return nil
}

// WatchRendezvous returns a watcher for the caller to wait for rendezvous to complete. When a
// process from each resource in the allocation connects and the resource manager sends each
// resource's state, each watcher will receive a copy of the rendezvous info for communicating
// with its peers.
func (a *allocation) WatchRendezvous(rID sproto.ResourcesID) (RendezvousWatcher, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.validateRendezvous()
	if err != nil {
		return RendezvousWatcher{}, err
	}

	if a.rendezvous == nil {
		a.rendezvous = newRendezvous(a.model.AllocationID, a.resources, rendezvousTimeoutDuration)
		a.closers = append(a.closers, a.rendezvous.close)
		a.wg.Go(func(ctx context.Context) {
			t := time.NewTimer(rendezvousTimeoutDuration)
			defer t.Stop()

			select {
			case <-t.C:
				a.RendezvousTimeout()
			case <-ctx.Done():
			}
		})
	}

	return a.rendezvous.watch(rID)
}

// UnwatchRendezvous removes a rendezvous watcher.
func (a *allocation) UnwatchRendezvous(rID sproto.ResourcesID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rendezvous.unwatch(rID)
}

func (a *allocation) RendezvousTimeout() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.rendezvous.checkTimeout(); err != nil {
		a.sendTaskLog(&model.TaskLog{Log: err.Error()})
	}
}

func (a *allocation) validateRendezvous() error {
	if a.rendezvous != nil {
		return nil
	}

	if len(a.resources) == 0 {
		return ErrAllocationUnfulfilled{Action: "rendezvous"}
	}

	switch a.resources.first().Summary().ResourcesType {
	case sproto.ResourcesTypeDockerContainer, sproto.ResourcesTypeK8sPod:
		break
	default:
		return ErrBehaviorUnsupported{Behavior: "rendezvous"}
	}

	return nil
}

// awaitTermination waits for the allocation and any goroutines associated with to exit.
func (a *allocation) awaitTermination() *AllocationExited {
	a.wg.Wait()
	return a.exited
}

// requestResources sets up the allocation.
func (a *allocation) requestResources() (*sproto.ResourcesSubscription, error) {
	if a.req.Restore {
		// Load allocation.
		a.syslog.Debug("requestResources load allocation")
		err := db.Bun().NewSelect().Model(&a.model).
			Where("allocation_id = ?", a.model.AllocationID).
			Scan(context.TODO())
		if err != nil {
			return nil, errors.Wrap(err, "loading trial allocation")
		}
	} else {
		// Insert new allocation.
		a.syslog.Debug("requestResources add allocation")

		a.setModelState(model.AllocationStatePending)
		if err := a.db.AddAllocation(&a.model); err != nil {
			return nil, errors.Wrap(err, "saving trial allocation")
		}
	}

	sub, err := a.rm.Allocate(a.req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to request allocation")
	}
	a.sendTaskLog(&model.TaskLog{
		Log: fmt.Sprintf("Scheduling %s (id: %s)", a.req.Name, a.req.AllocationID),
	})
	return sub, nil
}

// Cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
// This function should look _very_ similar to a.terminated.
func (a *allocation) Cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	// FYI, if we haven't exited something went terribly wrong (it is bug).
	if a.exited != nil {
		return
	}

	if a.exitErr == nil {
		a.exitErr = errors.New("unknown error occurred")
	}
	exitReason := a.exitErr.Error()
	a.SetExitStatus(exitReason, a.exitErr, ptrs.Ptr(int32(-1)))

	a.finalize(exitReason, false, logrus.ErrorLevel, a.exitErr)
}

func (a *allocation) finalize(
	exitReason string,
	userRequestedStop bool,
	severity logrus.Level,
	exitErr error,
) {
	defer a.rm.Release(sproto.ResourcesReleased{AllocationID: a.req.AllocationID})
	for _, cl := range a.closers {
		defer cl()
	}

	a.setMostProgressedModelState(model.AllocationStateTerminated)
	if err := a.db.UpdateAllocationState(a.model); err != nil {
		a.syslog.WithError(err).Error("failed to set allocation state to terminated")
	}
	a.purgeRestorableResources()
	a.markResourcesReleased()

	a.exited = &AllocationExited{UserRequestedStop: userRequestedStop, Err: exitErr, FinalState: a.state()}
	a.SetExitStatus(exitReason, exitErr, nil)
	log := fmt.Sprintf("%s was terminated: %s", a.req.Name, exitReason)
	a.syslog.Log(severity, log)
	a.sendTaskLog(&model.TaskLog{Level: ptrs.Ptr(model.TaskLogLevelFromLogrus(severity)), Log: log})
}

// resourcesAllocated handles receiving resources from the resource manager. Note: it makes a single
// ask to the parent to build its task spec.. this is mostly a hack to defer lots of computationally
// heavy stuff unless it is necessarily (which also works to spread occurrences of the same work
// out). Eventually, Allocations should just be started with their TaskSpec.
func (a *allocation) resourcesAllocated(msg *sproto.ResourcesAllocated) error {
	a.syslog.WithField("restore", a.req.Restore).Infof("%d resources allocated", len(msg.Resources))
	if !a.req.Restore {
		if a.getModelState() != model.AllocationStatePending {
			// If we have moved on from the pending state, these must be stale (and we must have
			// already released them, just the scheduler hasn't gotten word yet).
			return ErrStaleResourcesReceived{}
		}
		a.setModelState(model.AllocationStateAssigned)
	} else {
		a.syslog.Debugf("resourcesAllocated restored state: %s", a.getModelState())
	}

	a.setMostProgressedModelState(model.AllocationStateAssigned)
	err := a.resources.append(msg.Resources)
	if err != nil {
		return errors.Wrapf(err, "appending resources")
	}
	a.closers = append(a.closers, func() {
		for _, r := range a.resources {
			if r.Exited == nil {
				a.syslog.Infof("allocation exited with unterminated resources: %v", r.Summary())
				r.Kill(a.system, a.logCtx)
			}
		}
	})

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		return errors.Wrap(err, "updating allocation state")
	}

	now := time.Now().UTC()
	err = a.db.RecordTaskStats(&model.TaskStats{
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
		a.closers = append(a.closers, func() {
			preemptible.Unregister(a.req.AllocationID.String())
		})
	}

	if cfg := a.req.IdleTimeout; cfg != nil {
		idle.Register(*cfg, func(ctx context.Context, err error) {
			a.syslog.WithError(err).Infof("killing %s due to inactivity", a.req.Name)
			a.Signal(TerminateAllocation, err.Error())
		})
		a.closers = append(a.closers, func() {
			idle.Unregister(cfg.ServiceID)
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
						a.registerProxies(r.Started.Addresses)
						a.closers = append(a.closers, a.unregisterProxies)
					case a.model.ProxyAddress != nil:
						a.registerProxies(a.containerProxyAddresses())
						a.closers = append(a.closers, a.unregisterProxies)
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

		a.model.Ports, err = a.getPorts(spec.UniqueExposedPortRequests)
		if err != nil {
			return errors.Wrap(err, "getting ports")
		}
		a.closers = append(a.closers, func() {
			for _, port := range a.model.Ports {
				portregistry.ReleasePort(port)
			}
		})

		err = db.UpdateAllocationPorts(a.model)
		if err != nil {
			return fmt.Errorf("updating allocation db")
		}

		for portName, port := range a.model.Ports {
			spec.Environment.RawPorts[portName] = port
			spec.ExtraEnvVars[portName] = strconv.Itoa(port)
		}

		for cID, r := range a.resources {
			if err := r.Start(a.system, a.logCtx, spec, sproto.ResourcesRuntimeInfo{
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

// resourcesStateChanged handles changes in container states. It can move us to ready,
// kill us or close us normally depending on the changes, among other things.
func (a *allocation) resourcesStateChanged(msg *sproto.ResourcesStateChanged) {
	if _, ok := a.resources[msg.ResourcesID]; !ok {
		a.syslog.
			WithField("container", msg.Container).
			WithError(ErrStaleResources{ID: msg.ResourcesID}).Warnf("old state change")
		return
	}

	a.resources[msg.ResourcesID].Container = msg.Container
	a.syslog.Debugf("resources state changed: %+v", msg)
	switch msg.ResourcesState {
	case sproto.Pulling:
		a.setMostProgressedModelState(model.AllocationStatePulling)
		if a.model.StartTime == nil {
			a.markResourcesStarted()
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
			a.markResourcesStarted()
		}

		a.resources[msg.ResourcesID].Started = msg.ResourcesStarted
		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.crash(err)
			return
		}

		if a.rendezvous != nil && a.rendezvous.try() {
			a.syslog.
				Info("all containers are connected successfully (task container state changed)")
		}
		if len(a.req.ProxyPorts) > 0 && msg.ResourcesStarted.Addresses != nil &&
			a.resources[msg.ResourcesID].Rank == 0 {
			a.registerProxies(msg.ResourcesStarted.Addresses)
			a.closers = append(a.closers, a.unregisterProxies)
		}

		containerID := coalesceString(msg.ContainerIDStr(), "")
		a.sendTaskLog(&model.TaskLog{
			ContainerID: &containerID,
			Log:         fmt.Sprintf("Resources for %s have started", a.req.Name),
		})

		prom.AssociateAllocationTask(a.req.AllocationID, a.req.TaskID, a.req.Name, a.req.JobID)
		prom.AddAllocationResources(a.resources[msg.ResourcesID].Summary(), msg.ResourcesStarted)

	case sproto.Terminated:
		if a.resources[msg.ResourcesID].Exited != nil {
			// If we have already received the exit for this container, we only recognize the first.
			// If there are multiples, it's likely due to one being resent after a kill signal was
			// repeated. Agents always re-ack termination to ensure it is received in the event
			// of network failures and they always re-ack the same exit, anyway.
			return
		}

		a.syslog.Infof("resources terminated %s: %s", msg.ResourcesID, msg.ResourcesStopped.String())

		a.setMostProgressedModelState(model.AllocationStateTerminating)

		a.resources[msg.ResourcesID].Exited = msg.ResourcesStopped

		a.syslog.Infof("releasing resources %s", msg.ResourcesID)
		a.rm.Release(sproto.ResourcesReleased{
			AllocationID: a.req.AllocationID,
			ResourcesID:  &msg.ResourcesID,
		})

		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.crash(err)
			return
		}

		switch {
		case a.killedWhileRunning:
			a.sendTaskLog(&model.TaskLog{
				ContainerID: msg.ContainerIDStr(),
				Log:         fmt.Sprintf("killed: %s", msg.ResourcesStopped.String()),
			})
			a.tryExit("resources were killed")
		case msg.ResourcesStopped.Failure != nil:
			// Avoid erroring out if we have killed our daemons gracefully.
			// This occurs in the case of an early stop in dtrain. One resource
			// will exit with a 0 exit code and kill the rest of the resources sending
			// failed messages for these resources.
			if a.killedDaemonsGracefully {
				a.sendTaskLog(&model.TaskLog{
					ContainerID: msg.ContainerIDStr(),
					Log:         fmt.Sprintf("daemon killed: %s", msg.ResourcesStopped.String()),
				})
				a.tryExit("remaining resources terminated")
			} else {
				a.sendTaskLog(&model.TaskLog{
					ContainerID: msg.ContainerIDStr(),
					Log:         fmt.Sprintf("crashed: %s", msg.ResourcesStopped.String()),
					Level:       ptrs.Ptr(model.LogLevelError),
				})
				a.crash(*msg.ResourcesStopped.Failure)
			}
		default:
			a.sendTaskLog(&model.TaskLog{
				ContainerID: msg.ContainerIDStr(),
				Log:         msg.ResourcesStopped.String(),
				Level:       ptrs.Ptr(model.LogLevelInfo),
			})
			a.tryExit(msg.ResourcesStopped.String())
		}

		for cID := range a.resources {
			prom.DisassociateAllocationTask(a.req.AllocationID, a.req.TaskID, a.req.Name, a.req.JobID)
			prom.RemoveAllocationResources(a.resources[cID].Summary())
		}
	}

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		a.syslog.Error(err)
	}
}

// restoreResourceFailure handles the restored resource failures.
func (a *allocation) restoreResourceFailure(msg *sproto.ResourcesFailure) {
	a.syslog.Debugf("allocation resource failure")
	a.setMostProgressedModelState(model.AllocationStateTerminating)

	if err := a.db.UpdateAllocationState(a.model); err != nil {
		a.syslog.Error(err)
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
		a.syslog.WithError(err).Error("failed to mark allocation completed")
	}

	a.crash(msg)
}

// releaseResources prompts the allocate to release resources.
func (a *allocation) releaseResources(msg *sproto.ReleaseResources) {
	if msg.ForceKill {
		a.tryExitOrKill(msg.Reason)
	} else {
		a.tryExitOrTerminate(msg.Reason, msg.ForcePreemption)
	}
}

// recover recovers a crash and stops the allocation.
func (a *allocation) recover() {
	if rec := recover(); rec != nil {
		a.syslog.Error(rec)
		a.syslog.Error(string(debug.Stack()))
		if a.exitErr == nil {
			a.exitErr = errors.Errorf("unexpected panic: %v", rec)
		}
	}
}

// crash closes the allocation due to an error, beginning the kill flow.
func (a *allocation) crash(err error) {
	a.syslog.WithError(err).Errorf("allocation encountered fatal error")
	if a.exitErr == nil {
		a.exitErr = err
	}
	a.tryExitOrKill(err.Error())
}

// tryExitOrKill attempts to close an allocation by killing it.
func (a *allocation) tryExitOrKill(reason string) {
	if exited := a.tryExit(reason); exited {
		return
	}
	a.kill(reason)
}

// tryExitOrTerminate attempts to close an allocation by gracefully stopping it.
func (a *allocation) tryExitOrTerminate(reason string, forcePreemption bool) {
	if exited := a.tryExit(reason); exited {
		return
	}

	switch {
	case a.req.Preemptible && a.ready() || forcePreemption:
		a.preempt(reason)
	default:
		a.kill(reason)
	}
}

// tryExit attempts to exit an allocation while not killing or preempting it.
func (a *allocation) tryExit(reason string) (exited bool) {
	switch {
	case !a.resourcesStarted:
		a.terminated(fmt.Sprintf("exit before start: %s", reason))
		return true
	case len(a.resources.exited()) == len(a.resources):
		a.terminated(fmt.Sprintf("all resources exited: %s", reason))
		return true
	case a.allNonDaemonsExited():
		a.killedDaemons = true
		if a.exitedWithoutErr() {
			a.killedDaemonsGracefully = true
		}
		a.kill(fmt.Sprintf("all non-daemons exited: %s", reason))
	case len(a.resources.failed()) > 0:
		a.kill(fmt.Sprintf("some resources failed: %s", reason))
	}
	return false
}

func (a *allocation) preempt(reason string) {
	a.syslog.WithField("reason", reason).Info("decided to gracefully terminate allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"gracefully terminating allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	preemptible.Preempt(a.req.AllocationID.String(), func(ctx context.Context, err error) {
		a.Signal(KillAllocation, err.Error())
	})
}

func (a *allocation) kill(reason string) {
	if a.killCooldown != nil && time.Now().Before(*a.killCooldown) {
		a.syslog.Debug("still inside of kill cooldown")
		return
	}

	a.syslog.WithField("reason", reason).Info("decided to kill allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"forcibly killing allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	for _, r := range a.resources.active() {
		r.Kill(a.system, a.logCtx)
	}

	if len(a.resources.exited()) == 0 {
		a.syslog.Debugf("setting killed while running: %d", len(a.resources.exited()))
		a.killedWhileRunning = true
	}

	// Once a job has been killed, resend the kill every 30s, in the event it is lost (has
	// happened before due to network failures).
	a.killCooldown = ptrs.Ptr(time.Now().Add(killCooldown))
	a.wg.Go(func(ctx context.Context) {
		t := time.NewTimer(killCooldown * 2)
		defer t.Stop()

		select {
		case <-t.C:
			a.Signal(KillAllocation, "killing again after 30s without all container exits")
		case <-ctx.Done():
			return
		}
	})
}

func (a *allocation) allNonDaemonsExited() bool {
	for id := range a.resources {
		_, terminated := a.resources.exited()[id]
		_, daemon := a.resources.daemons()[id]
		if !(terminated || daemon) {
			return false
		}
	}
	return true
}

func (a *allocation) exitedWithoutErr() bool {
	for _, r := range a.resources.failed() {
		code := r.Exited.Failure.ExitCode
		if code != nil && *code != 0 {
			return false
		}
	}
	return true
}

func (a *allocation) SetExitStatus(exitReason string, exitErr error, statusCode *int32) {
	switch err := exitErr.(type) {
	case sproto.ResourcesFailure:
		a.model.ExitErr = ptrs.Ptr(err.Error())
		if err.ExitCode != nil {
			a.model.StatusCode = ptrs.Ptr(int32(*err.ExitCode))
		}
	case nil:
		a.model.ExitErr = nil
	default:
		a.model.ExitErr = ptrs.Ptr(err.Error())
	}
	a.model.ExitReason = &exitReason

	if statusCode != nil {
		a.model.StatusCode = statusCode
	}

	if err := db.AddAllocationExitStatus(context.TODO(), &a.model); err != nil {
		a.syslog.WithError(err).Error("failed to add allocation exit status to db")
	}
}

func (a *allocation) registerProxies(addresses []cproto.Address) {
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
		urlScheme := "http"
		if a.req.ProxyTLS {
			urlScheme = "https"
		}
		proxy.DefaultProxy.Register(pcfg.ServiceID, &url.URL{
			Scheme: urlScheme,
			Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
		}, pcfg.ProxyTCP, pcfg.Unauthenticated)
		a.syslog.Debugf("registered proxy id: %s, tcp: %v\n", pcfg.ServiceID, pcfg.ProxyTCP)
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

func (a *allocation) unregisterProxies() {
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
func (a *allocation) containerProxyAddresses() []cproto.Address {
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

func (a *allocation) terminated(reason string) {
	if a.exited != nil {
		// Never exit twice. If this were allowed, a trial could receive two task.AllocationExited
		// messages. On receipt of the first message, the trial awaits our exit. Once we exit, it
		// reschedules a new allocation, receives the second message and erroneously awaits the new
		// allocation's stop. Once the new allocation asks the trial to build its task spec, they
		// deadlock.
		// This occurred when an allocation completed and was preempted in quick succession.
		return
	}

	a.finalize(a.calculateExitStatus(reason))
}

func (a *allocation) calculateExitStatus(reason string) (
	exitReason string,
	userRequestedStop bool,
	severity logrus.Level,
	exitErr error,
) {
	switch {
	case a.killedWhileRunning:
		return fmt.Sprintf("allocation killed after %s", reason), false, logrus.InfoLevel, nil
	case a.req.Preemptible && preemptible.Acknowledged(a.req.AllocationID.String()):
		return fmt.Sprintf("allocation preempted after %s", reason), false, logrus.InfoLevel, nil
	case a.exitErr == nil && len(a.resources.exited()) > 0:
		return fmt.Sprintf("allocation stopped early after %s", reason), true, logrus.InfoLevel, nil
	case a.exitErr != nil:
		switch err := a.exitErr.(type) {
		case sproto.ResourcesFailure:
			switch err.FailureType {
			case sproto.ResourcesFailed, sproto.TaskError:
				if a.killedDaemonsGracefully {
					return "allocation terminated daemon processes as part of normal exit", false, logrus.InfoLevel, nil
				}
				return fmt.Sprintf("allocation failed: %s", err), false, logrus.ErrorLevel, err
			case sproto.AgentError, sproto.AgentFailed:
				return fmt.Sprintf("allocation failed due to agent failure: %s", err), false, logrus.ErrorLevel, err
			case sproto.TaskAborted, sproto.ResourcesAborted:
				return fmt.Sprintf("allocation aborted: %s", err.FailureType), false, logrus.InfoLevel, err
			case sproto.RestoreError:
				return fmt.Sprintf("allocation failed due to restore error: %s", err), false, logrus.ErrorLevel, err
			default:
				panic(fmt.Errorf("unexpected allocation failure: %w", err))
			}
		default:
			return fmt.Sprintf("allocation handler crashed due to error: %s", err), false, logrus.ErrorLevel, err
		}
	case len(a.resources) == 0:
		return fmt.Sprintf("allocation aborted after %s", reason), false, logrus.InfoLevel, nil
	default:
		// If we ever exit without a reason and we have no exited resources, something has gone wrong.
		panic("allocation exited early without a valid reason")
	}
}

// markResourcesStarted persists start information.
func (a *allocation) markResourcesStarted() {
	a.model.StartTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	if a.restored {
		a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("%s was recovered on an agent", a.req.Name)})
	} else {
		a.sendTaskLog(&model.TaskLog{Log: fmt.Sprintf("%s was assigned to an agent", a.req.Name)})
	}
	if err := a.db.UpdateAllocationStartTime(a.model); err != nil {
		a.syslog.
			WithError(err).
			Errorf("allocation will not be properly accounted for")
	}
}

// markResourcesReleased persists completion information.
func (a *allocation) markResourcesReleased() {
	if err := a.db.DeleteAllocationSession(a.model.AllocationID); err != nil {
		a.syslog.WithError(err).Error("error deleting allocation session")
	}
	if a.model.StartTime == nil {
		return
	}
	a.model.EndTime = ptrs.Ptr(time.Now().UTC())
	if err := a.db.CompleteAllocation(&a.model); err != nil {
		a.syslog.WithError(err).Error("failed to mark allocation completed")
	}

	telemetry.ReportAllocationTerminal(a.db, a.model, a.resources.firstDevice())
}

func (a *allocation) purgeRestorableResources() {
	_, err := db.Bun().NewDelete().Model((*taskmodel.ResourcesWithState)(nil)).
		Where("allocation_id = ?", a.model.AllocationID).
		Exec(context.TODO())
	if err != nil {
		a.syslog.WithError(err).Error("failed to purge restorable resources")
	}
}

const killedLogSubstr = "exit code 137"

func (a *allocation) enrichLog(log *model.TaskLog) *model.TaskLog {
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

// sendTaskLog is called without a lock.
func (a *allocation) sendTaskLog(log *model.TaskLog) {
	tasklogger.Insert(a.enrichLog(log))
}

func (a *allocation) state() AllocationState {
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

func (a *allocation) setModelState(v model.AllocationState) {
	a.model.State = &v
}

func (a *allocation) setMostProgressedModelState(v model.AllocationState) {
	a.setModelState(model.MostProgressedAllocationState(a.getModelState(), v))
}

func (a *allocation) getModelState() model.AllocationState {
	if a.model.State == nil {
		return model.AllocationStatePending
	}
	return *a.model.State
}

func (a *allocation) ready() bool {
	// Most trials use `a.rendezvous` and the normal rendezvous APIs, and go through this path.
	return (a.rendezvous != nil && a.rendezvous.ready()) ||
		// And finally, of course, if the task explicitly called `AllocationReady` it is ready.
		coalesceBool(a.model.IsReady, false)
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

func (a *allocation) getPorts(exposedPorts map[string]int) (map[string]int, error) {
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
		a.syslog.Debugf("%v port : %v", portName, port)
	}

	return ports, nil
}
