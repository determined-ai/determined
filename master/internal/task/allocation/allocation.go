package allocation

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/cluster"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/portregistry"
	"github.com/determined-ai/determined/master/internal/prom"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/allocationmap"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task/tasklogger"
	"github.com/determined-ai/determined/master/internal/task/taskmodel"
	"github.com/determined-ai/determined/master/internal/telemetry"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const (
	killCooldown       = 15 * time.Second
	okExitMessage      = "allocation exited successfully"
	missingExitMessage = ""
)

type AllocationHandle struct {
	mu sync.RWMutex

	// Configuration.
	req           sproto.AllocateRequest
	buildTaskSpec func() (tasks.TaskSpec, error)
	parent        *actor.Ref // TODO(mar): fill, potentially
	logFields     logrus.Fields

	// System dependencies.
	log     *logrus.Entry
	system  *actor.System
	taskLog *tasklogger.Logger
	rm      rm.ResourceManager

	// Mutable internal state.
	wg                sync.WaitGroup
	model             model.Allocation
	resources         resourcesList
	exitErr           error
	resourcesWatchers []*sproto.Watcher[sproto.ResourcesStateChanged]

	// Separates the existence of resources from us having started them.
	resourcesStarted bool
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
	exited  bool
	exitedC chan struct{}

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
	// active all gather state
	allGather *allGather
	// records whether the allocation has completed any all gathers.
	allGatherFinished bool

	restored        bool
	portsRegistered bool

	// Return value.
	exit *AllocationExited
}

func Start(
	ctx context.Context,
	logFields logrus.Fields,
	req sproto.AllocateRequest,
	rm rm.ResourceManager,
	taskLog *tasklogger.Logger,
	buildTaskSpec func() (tasks.TaskSpec, error),
) (*AllocationHandle, error) {
	a := &AllocationHandle{
		req:       req,
		logFields: logFields,
		rm:        rm,
		taskLog:   taskLog,
		log:       logrus.WithFields(logFields),
		model: model.Allocation{
			AllocationID: req.AllocationID,
			TaskID:       req.TaskID,
			Slots:        req.SlotsNeeded,
			ResourcePool: req.ResourcePool,
			Ports:        map[string]int{},
		},
		buildTaskSpec: buildTaskSpec,
	}

	a.log.WithField("restore", req.Restore).Debug("starting allocation")
	if a.req.Restore {
		err := db.Bun().NewSelect().Model(&a.model).
			Where("allocation_id = ?", a.model.AllocationID).
			Scan(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "loading trial allocation")
		}
	} else {
		a.model.State = ptrs.Ptr(model.AllocationStatePending)
		if err := db.SingleDB().AddAllocation(&a.model); err != nil { // TODO(mar): context
			return nil, errors.Wrap(err, "saving trial allocation")
		}
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.run()
	}()

	return a, nil
}

// IsAllocationRestoring
func (a *AllocationHandle) Restoring() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.req.Restore && !a.restored
}

// GetResourcesContainerState requests cproto.Container state for a given clump of resources.
// If the resources aren't a container, this request returns a failure.
func (a *AllocationHandle) GetResourcesContainerState(rID sproto.ResourcesID) (
	*cproto.Container,
	error,
) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if v, ok := a.resources[rID]; ok {
		if v.Container == nil {
			return nil, fmt.Errorf("no container associated with %s", rID)
		} else {
			return v.Container, nil
		}
	} else {
		// TODO(mar): better errors.
		return nil, fmt.Errorf("unknown resources %s", rID)
	}
}

// TODO(mar): how many APIs for this do we need ?!?!
func (a *AllocationHandle) ReleaseResources(forceGracefulPreemption bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.exitOrTerminate("allocation being preempted by the scheduler", forceGracefulPreemption)
}

// TODO(mar): termination is still tricky; how does the inner loop know?
func (a *AllocationHandle) ChangeRP() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.exitOrTerminate("allocation resource pool changed", false)
}

func (a *AllocationHandle) SendContainerLog(log sproto.ContainerLog) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	a.sendEvent(log.ToEvent())
}

// These messages allow users (and sometimes an orchestrator, such as HP search)
// to interact with the allocation. The usually trace back to API calls.
func (a *AllocationHandle) SetReady() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// AllocationReady only comes from the running container, so to
	// avoid a race condition with the slower transition to running state
	// which comes via polling for dispatcher RM, move the state to running now.
	a.setMostProgressedModelState(model.AllocationStateRunning)
	a.model.IsReady = ptrs.Ptr(true)
	if err := db.SingleDB().UpdateAllocationState(a.model); err != nil {
		a.crashOrKill(err)
		return err
	}
	a.sendEvent(sproto.Event{ServiceReadyEvent: ptrs.Ptr(true)})
	return nil
}

func (a *AllocationHandle) SetWaiting() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.setMostProgressedModelState(model.AllocationStateWaiting)
	if err := db.SingleDB().UpdateAllocationState(a.model); err != nil {
		a.crashOrKill(err)
		return err
	}
	return nil
}

func (a *AllocationHandle) MarkResourcesDaemon(id sproto.ResourcesID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.SetResourcesAsDaemon(id); err != nil {
		a.crashOrKill(err)
	}
}

func (a *AllocationHandle) HandleSignalWithoutReason(msg sproto.AllocationSignal) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.HandleSignal(sproto.AllocationSignalWithReason{AllocationSignal: msg})
}

// HandleSignal handles an external signal to kill or terminate the allocation.
func (a *AllocationHandle) HandleSignal(msg sproto.AllocationSignalWithReason) {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch msg.AllocationSignal {
	case sproto.KillAllocation:
		a.exitOrKill(msg.InformationalReason)
	case sproto.TerminateAllocation:
		a.exitOrTerminate(msg.InformationalReason, false)
	}
}

// State returns a deepcopy of our state.
func (a *AllocationHandle) State() AllocationState {
	a.mu.RLock()
	defer a.mu.RUnlock()

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

func (a *AllocationHandle) SetProxyAddress(proxyAddress string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.req.ProxyPorts) == 0 {
		return ErrBehaviorUnsupported{Behavior: "proxy"}
	}
	a.model.ProxyAddress = &proxyAddress
	if err := db.SingleDB().UpdateAllocationProxyAddress(a.model); err != nil {
		a.crashOrKill(err)
		return err
	}
	a.registerProxies(a.containerProxyAddresses())
	return nil
}

func (a *AllocationHandle) WatchPreemption(id uuid.UUID) PreemptionWatcher {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.preemption.Watch(id)
}

func (a *AllocationHandle) UnwatchPreemption(id uuid.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.preemption.Unwatch(id)
}

func (a *AllocationHandle) AckPreemption() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.preemption.Acknowledge()
}

func (a *AllocationHandle) IdleWatcherNoteActivity(instant time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.idleTimeoutWatcher.RecordActivity(instant)
}

func (a *AllocationHandle) WatchRendezvousInfo(id sproto.ResourcesID) (w RendezvousWatcher, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.rendezvous == nil {
		if err := a.canRendezvous(); err != nil {
			return w, err
		}

		// TODO(mar): this really doesnt need to be lazily initialized
		a.rendezvous = newRendezvous(a.resources, func(err error) {
			a.sendTaskLog(&model.TaskLog{Log: err.Error()})
		})
	}

	return a.rendezvous.watch(id)
}

func (a *AllocationHandle) UnwatchRendezvousInfo(id sproto.ResourcesID) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.rendezvous == nil {
		if err := a.canRendezvous(); err != nil {
			return err
		}
		return ErrRendezvousBadRequest
	}

	a.rendezvous.unwatch(id)
	return nil
}

func (a *AllocationHandle) canRendezvous() error {
	if len(a.resources) == 0 {
		return ErrAllocationUnfulfilled{Action: "rendezvous"}
	}

	switch a.resources.first().Summary().ResourcesType {
	case sproto.ResourcesTypeDockerContainer, sproto.ResourcesTypeK8sPod:
		return nil
	default:
		// TODO(mar): better errors.
		return ErrBehaviorUnsupported{Behavior: "rendezvous"}
	}
}

func (a *AllocationHandle) WatchAllGather(msg WatchAllGather) AllGatherWatcher {
	if a.allGather == nil {
		a.allGather = newAllGather(func(err error) {
			a.sendTaskLog(&model.TaskLog{Log: err.Error()})
			a.log.WithError(err).Error("performing all gather through master")
		})
	}

	w := a.allGather.watch(msg)

	if a.allGather.done() {
		a.allGather = nil
		a.allGatherFinished = true
	}

	return w
}

func (a *AllocationHandle) UnwatchAllGather(msg UnwatchAllGather) {
	a.allGather.unwatch(msg)
}

// TODO(mar): task.AllocationExited should just be sentinel errors.
func (a *AllocationHandle) Wait() *AllocationExited {
	a.wg.Wait()
	return a.exit
}

func (a *AllocationHandle) run() {
	defer a.cleanup()

	a.req.RegisteredTime = time.Now()
	w := a.rm.Allocate(a.system, a.req)
	a.sendEvent(sproto.Event{ScheduledEvent: &a.model.AllocationID})

	// TODO(mar): kill while waiting for resources
	var res sproto.AllocateResponse
	select {
	case res = <-w.C:
	case <-a.exitedC:
		return
	}

	if err := res.Error; err != nil {
		// The only way you don't get allocated is apparently a restore failure.
		a.RestoreResourceFailure(*res.Error)
		return
	}

	// TODO(mar): canceled while launching. concurrency in general.
	err := a.ResourcesAllocated(res.Resources)
	if err != nil {
		a.exitErr = err
		a.terminated(err.Error())
		return
	}

	resourcesWatcher := sproto.MergeWatchers(a.resourcesWatchers...)
	for rsc := range resourcesWatcher.C {
		a.resourcesStateChanged(rsc)
		if a.exited {
			return
		}
	}
}

func (a *AllocationHandle) terminated(reason string) {
	a.model.State = ptrs.Ptr(model.AllocationStateTerminated)
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
	defer close(a.exitedC)
	exitReason := fmt.Sprintf("allocation terminated after %s", reason)
	defer a.system.Tell(a.parent, exit)
	defer a.rm.Release(a.system, sproto.ResourcesReleased{AllocationID: a.req.AllocationID})
	defer a.unregisterProxies()

	level := ptrs.Ptr(model.LogLevelInfo)
	if a.exitErr != nil {
		level = ptrs.Ptr(model.LogLevelError)
	}
	defer a.sendEvent(sproto.Event{Level: level, ExitedEvent: &exitReason})
	if err := a.purgeRestorableResources(); err != nil {
		a.log.WithError(err).Error("failed to purge restorable resources")
	}

	if len(a.resources) == 0 {
		return
	}
	defer a.markResourcesReleased()

	if a.req.Preemptible {
		defer a.preemption.Close()
	}
	if a.rendezvous != nil {
		defer a.rendezvous.close()
	}
	if a.idleTimeoutWatcher != nil {
		defer a.idleTimeoutWatcher.Close()
	}
	switch {
	case a.killedWhileRunning:
		exitReason = fmt.Sprintf("allocation stopped after %s", reason)
		a.log.Info(exitReason)
		return
	case a.req.Preemptible && a.preemption.Acknowledged():
		exitReason = fmt.Sprintf("allocation stopped after %s", reason)
		a.log.Info(exitReason)
		return
	case a.exitErr == nil && len(a.resources.exited()) > 0:
		// This is true because searcher and preemption exits both ack preemption.
		exit.UserRequestedStop = true
		exitReason = fmt.Sprintf("allocation stopped early after %s", reason)
		a.log.Info(exitReason)
		return
	case a.exitErr != nil:
		switch err := a.exitErr.(type) {
		case sproto.ResourcesFailure:
			switch err.FailureType {
			case sproto.ResourcesFailed, sproto.TaskError:
				if a.killedDaemonsGracefully {
					exitReason = fmt.Sprint("allocation terminated daemon processes as part of normal exit")
					a.log.Info(exitReason)
					return
				}
				exitReason = fmt.Sprintf("allocation failed: %s", err)
				a.log.Info(exitReason)
				exit.Err = err
				return
			case sproto.AgentError, sproto.AgentFailed:
				exitReason = fmt.Sprintf("allocation failed due to agent failure: %s", err)
				a.log.Warn(exitReason)
				exit.Err = err
				return
			case sproto.TaskAborted, sproto.ResourcesAborted:
				exitReason = fmt.Sprintf("allocation aborted: %s", err.FailureType)
				a.log.Debug(exitReason)
				exit.Err = err
				return
			case sproto.RestoreError:
				exitReason = fmt.Sprintf("allocation failed due to restore error: %s", err)
				a.log.Warn(exitReason)
				exit.Err = err
				return

			default:
				panic(fmt.Errorf("unexpected allocation failure: %w", err))
			}
		default:
			exitReason = fmt.Sprintf("allocation handler crashed due to error: %s", err)
			a.log.Error(exitReason)
			exit.Err = err
			return
		}
	default:
		// If we ever exit without a reason and we have no exited resources, something has gone
		// wrong.
		panic("allocation exited early without a valid reason")
	}
}

func (a *AllocationHandle) sendTaskLog(log *model.TaskLog) {
	a.taskLog.Insert(a.enrichLog(log))
}

func (a *AllocationHandle) sendEvent(ev sproto.Event) {
	ev = a.enrichEvent(ev)
	a.sendTaskLog(ev.ToTaskLog())
}

func (a *AllocationHandle) enrichEvent(ev sproto.Event) sproto.Event {
	ev.Description = a.req.Name

	ev.IsReady = false
	if a.model.IsReady != nil {
		ev.IsReady = *a.model.IsReady
	}

	if ev.Time.IsZero() {
		ev.Time = time.Now().UTC()
	}
	return ev
}

const killedLogSubstr = "exit code 137"

func (a *AllocationHandle) enrichLog(log *model.TaskLog) *model.TaskLog {
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

// cleanup ensures an allocation is properly closed. It tries to do everything before failing and
// ensures we don't leave any resources running.
func (a *AllocationHandle) cleanup() {
	// Just in-case code.
	if !a.exited {
		a.log.Info("exit did not run properly")
		for _, r := range a.resources {
			if r.Exited == nil {
				a.log.Infof("allocation exited with unterminated reservation: %v", r.Summary())
				r.Kill(a.system, logger.Context(a.logFields))
			}
		}
		if a.resourcesStarted {
			a.markResourcesReleased()
		}

		if err := a.purgeRestorableResources(); err != nil {
			a.log.WithError(err).Error("failed to purge restorable resources")
		}

		a.sendEvent(sproto.Event{ExitedEvent: ptrs.Ptr("allocation did not exit correctly")})
		a.rm.Release(a.system, sproto.ResourcesReleased{AllocationID: a.model.AllocationID})
	}

	// a.portsRegistered  is set to true right after ports are registered.
	// This variable ensures to release ports even if there's a failure after restoring ports.
	if a.portsRegistered {
		for _, port := range a.model.Ports {
			portregistry.ReleasePort(port)
		}
	}
	allocationmap.UnregisterAllocation(a.model.AllocationID)
}

// ResourcesAllocated handles receiving resources from the resource manager. Note: it makes a single
// ask to the parent to build its task spec.. this is mostly a hack to defer lots of computationally
// heavy stuff unless it is necessarily (which also works to spread occurrences of the same work
// out). Eventually, Allocations should just be started with their TaskSpec.
func (a *AllocationHandle) ResourcesAllocated(msg *sproto.ResourcesAllocated) error {
	if !a.req.Restore {
		if a.getModelState() != model.AllocationStatePending {
			// If we have moved on from the pending state, these must be stale (and we must have
			// already released them, just the scheduler hasn't gotten word yet).
			return ErrStaleResourcesReceived{}
		}

		a.setModelState(model.AllocationStateAssigned)
	} else {
		a.log.Debugf("ResourcesAllocated restored state: %s", a.getModelState())
	}

	a.setMostProgressedModelState(model.AllocationStateAssigned)
	if err := a.resources.append(msg.Resources); err != nil {
		return errors.Wrapf(err, "appending resources")
	}

	// Get the task spec first, so the trial/task table is populated before allocations.
	spec, err := a.buildTaskSpec()
	if err != nil {
		return errors.Wrapf(err, "could not get task spec")
	}

	if err := db.SingleDB().UpdateAllocationState(a.model); err != nil {
		return errors.Wrap(err, "updating allocation state")
	}

	now := time.Now().UTC()
	err = db.SingleDB().RecordTaskStats(&model.TaskStats{
		AllocationID: msg.ID,
		EventType:    "QUEUED",
		StartTime:    &msg.JobSubmissionTime,
		EndTime:      &now,
	})
	if err != nil {
		return errors.Wrap(err, "recording task queued stats")
	}

	if a.req.Preemptible {
		a.preemption = NewPreemption(a.model.AllocationID)
	}

	if cfg := a.req.IdleTimeout; cfg != nil {
		a.idleTimeoutWatcher = NewIdleTimeoutWatcher(a.req.Name, cfg, a.system, func() {
			a.log.Infof("killing %s due to inactivity", a.req.Name)
			a.HandleSignal(sproto.AllocationSignalWithReason{
				AllocationSignal: sproto.TerminateAllocation,
				InformationalReason: fmt.Sprintf(
					"inactivity for more than %s",
					cfg.TimeoutDuration.Round(time.Second)),
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
						a.registerProxies(r.Started.Addresses)
					case a.model.ProxyAddress != nil:
						a.registerProxies(a.containerProxyAddresses())
					}
				}
			}
		}
	} else {
		token, err := db.SingleDB().StartAllocationSession(a.model.AllocationID, spec.Owner)
		if err != nil {
			return errors.Wrap(err, "starting a new allocation session")
		}

		a.model.Ports, err = a.getPorts(spec.UniqueExposedPortRequests)
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
			w, err := r.Start(a.system, logger.Context(a.logFields), spec, sproto.ResourcesRuntimeInfo{
				Token:        token,
				AgentRank:    a.resources[cID].Rank,
				IsMultiAgent: len(a.resources) > 1,
			})
			if err != nil {
				return fmt.Errorf("starting resources (%v): %w", r, err)
			}
			// TODO(mar): just return the watchers!?
			a.resourcesWatchers = append(a.resourcesWatchers, w)
		}
	}

	a.restored = a.req.Restore
	a.resourcesStarted = true
	return nil
}

// SetResourcesAsDaemon sets the reservation as a daemon reservation. This means we won't wait for
// it to exit in errorless exits and instead will kill the forcibly.
func (a *AllocationHandle) SetResourcesAsDaemon(id sproto.ResourcesID) error {
	if _, ok := a.resources[id]; !ok {
		return ErrStaleResources{ID: id}
	} else if len(a.resources) <= 1 {
		a.sendTaskLog(&model.TaskLog{
			Log: `Ignoring request to daemonize resources within an allocation for an allocation
			with only one manageable set of resources, because this would just kill it. This is
			expected in when using the HPC launcher.`,
			Level: ptrs.Ptr(model.LogLevelInfo),
		})
		return nil
	}

	a.resources[id].Daemon = true
	if err := a.resources[id].Persist(); err != nil {
		return err
	}

	if len(a.resources.daemons()) == len(a.resources) {
		a.log.Warnf("all resources were marked as daemon, exiting")
		a.exitOrKill("all resources were marked as daemon")
	}

	return nil
}

// resourcesStateChanged handles changes in container states. It can move us to ready,
// kill us or close us normally depending on the changes, among other things.
func (a *AllocationHandle) resourcesStateChanged(msg sproto.ResourcesStateChanged) {
	if _, ok := a.resources[msg.ResourcesID]; !ok {
		a.log.
			WithField("container", msg.Container).
			WithError(ErrStaleResources{ID: msg.ResourcesID}).Warnf("old state change")
		return
	}

	a.resources[msg.ResourcesID].Container = msg.Container
	a.log.Debugf("resources state changed: %+v", msg)
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
			a.crashOrKill(err)
			return
		}

		if a.rendezvous != nil && a.rendezvous.try() {
			a.log.Info("all containers are connected successfully (task container state changed)")
		}
		if len(a.req.ProxyPorts) > 0 && msg.ResourcesStarted.Addresses != nil &&
			a.resources[msg.ResourcesID].Rank == 0 {
			a.registerProxies(msg.ResourcesStarted.Addresses)
		}

		a.sendEvent(sproto.Event{
			ContainerID:           coalesceString(msg.ContainerIDStr(), ""),
			ResourcesStartedEvent: msg.ResourcesStarted,
		})

		prom.AssociateAllocationTask(a.req.AllocationID, a.req.TaskID, a.req.JobID)
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

		a.rm.Release(a.system, sproto.ResourcesReleased{
			AllocationID: a.model.AllocationID,
			ResourcesID:  &msg.ResourcesID,
		})

		if err := a.resources[msg.ResourcesID].Persist(); err != nil {
			a.crashOrKill(err)
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
			a.tryExit("resources were killed")
		case msg.ResourcesStopped.Failure != nil:
			// Avoid erroring out if we have killed our daemons gracefully.
			// This occurs in the case of an early stop in dtrain. One resource
			// will exit with a 0 exit code and kill the rest of the resources sending
			// failed messages for these resources.
			if a.killedDaemonsGracefully {
				a.tryExit("remaining resources terminated")
			} else {
				a.crashOrKill(*msg.ResourcesStopped.Failure)
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
			prom.DisassociateAllocationTask(a.req.AllocationID, a.req.TaskID, a.req.JobID)
			prom.RemoveAllocationResources(a.resources[cID].Summary())
		}
	}

	if err := db.SingleDB().UpdateAllocationState(a.model); err != nil {
		a.log.Error(err)
	}
	return
}

// RestoreResourceFailure handles the restored resource failures.
func (a *AllocationHandle) RestoreResourceFailure(msg sproto.ResourcesFailure) {
	a.log.Debugf("allocation resource failure")
	a.setMostProgressedModelState(model.AllocationStateTerminating)

	if err := db.SingleDB().UpdateAllocationState(a.model); err != nil {
		a.log.Error(err)
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

	if err := db.SingleDB().CompleteAllocation(&a.model); err != nil {
		a.log.WithError(err).Error("failed to mark allocation completed")
	}

	a.crashOrKill(msg)
}

// tryExit attempts to exit an allocation while not killing or preempting it.
func (a *AllocationHandle) tryExit(reason string) (exited bool) {
	switch {
	case !a.resourcesStarted:
		a.terminated(reason)
		return true
	case len(a.resources.exited()) == len(a.resources):
		a.terminated(reason)
		return true
	case a.allNonDaemonsExited():
		a.killedDaemons = true
		if a.exitedWithoutErr() {
			a.killedDaemonsGracefully = true
		}
		a.kill(reason)
	case len(a.resources.failed()) > 0:
		a.kill(reason)
	}
	return false
}

// exitOrTerminate attempts to close an allocation by gracefully stopping it (though a kill are possible).
func (a *AllocationHandle) exitOrTerminate(reason string, forcePreemption bool) {
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

// exitOrKill attempts to close an allocation by killing it.
func (a *AllocationHandle) exitOrKill(reason string) {
	if exited := a.tryExit(reason); exited {
		return
	}
	a.kill(reason)
}

// crashOrKill closes the allocation due to an error, beginning the kill flow.
func (a *AllocationHandle) crashOrKill(err error) {
	a.log.WithError(err).Errorf("allocation encountered fatal error")
	if a.exitErr == nil {
		a.exitErr = err
	}
	a.exitOrKill(err.Error())
}

func (a *AllocationHandle) allNonDaemonsExited() bool {
	for id := range a.resources {
		_, terminated := a.resources.exited()[id]
		_, daemon := a.resources.daemons()[id]
		if !(terminated || daemon) {
			return false
		}
	}
	return true
}

func (a *AllocationHandle) exitedWithoutErr() bool {
	for _, r := range a.resources.failed() {
		code := r.Exited.Failure.ExitCode
		if code != nil && *code != 0 {
			return false
		}
	}
	return true
}

func (a *AllocationHandle) preempt(reason string) {
	a.log.WithField("reason", reason).Info("decided to gracefully terminate allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"gracefully terminating allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	a.preemption.Preempt(func(err error) {
		a.sendTaskLog(&model.TaskLog{Log: err.Error()})
		a.crashOrKill(err)
	})
}

func (a *AllocationHandle) kill(reason string) {
	if a.killCooldown != nil && time.Now().Before(*a.killCooldown) {
		a.log.Debug("still inside of kill cooldown")
		return
	}

	a.log.WithField("reason", reason).Info("decided to kill allocation")
	a.sendTaskLog(&model.TaskLog{
		Level: ptrs.Ptr(model.LogLevelInfo),
		Log: fmt.Sprintf(
			"forcibly killing allocation's remaining resources (reason: %s)",
			reason,
		),
	})

	for _, r := range a.resources.active() {
		r.Kill(a.system, logger.Context(a.logFields))
	}

	if len(a.resources.exited()) == 0 {
		a.killedWhileRunning = true
	}

	// Once a job has been killed, resend the kill every 30s, in the event it is lost (has
	// happened before due to network failures).
	a.killCooldown = ptrs.Ptr(time.Now().Add(killCooldown))

	go func() {
		if a.exited {
			return
		}

		time.Sleep(killCooldown * 2)
		a.HandleSignal(sproto.AllocationSignalWithReason{
			AllocationSignal:    sproto.KillAllocation,
			InformationalReason: "killing again after 30s without all container exits",
		})
	}()
}

func (a *AllocationHandle) registerProxies(addresses []cproto.Address) {
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
		a.system.AskAt(actor.Addr("proxy"), proxy.Register{
			ServiceID: pcfg.ServiceID,
			URL: &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%d", address.HostIP, address.HostPort),
			},
			ProxyTCP:        pcfg.ProxyTCP,
			Unauthenticated: pcfg.Unauthenticated,
		})
		a.log.Debugf("registered proxy id: %s, tcp: %v\n", pcfg.ServiceID, pcfg.ProxyTCP)
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

func (a *AllocationHandle) unregisterProxies() {
	if len(a.req.ProxyPorts) == 0 {
		return
	}

	if len(a.resources) > 1 {
		// Can't proxy more than one reservation, so we never would've made them.
		return
	}

	for _, serviceID := range a.proxies {
		a.system.Tell(a.system.Get(actor.Addr("proxy")), proxy.Unregister{
			ServiceID: serviceID,
		})
	}
}

// containerProxyAddresses forms the container address _only_ when proxyAddress is given.
func (a *AllocationHandle) containerProxyAddresses() []cproto.Address {
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

// markResourcesStarted persists start information.
func (a *AllocationHandle) markResourcesStarted() {
	a.model.StartTime = ptrs.Ptr(time.Now().UTC().Truncate(time.Millisecond))
	a.sendEvent(sproto.Event{AssignedEvent: &sproto.AllocatedEvent{Recovered: a.restored}})
	if err := db.SingleDB().UpdateAllocationStartTime(a.model); err != nil {
		a.log.
			WithError(err).
			Errorf("allocation will not be properly accounted for")
	}
}

// markResourcesReleased persists completion information.
func (a *AllocationHandle) markResourcesReleased() {
	a.model.EndTime = ptrs.Ptr(time.Now().UTC())
	if err := db.SingleDB().DeleteAllocationSession(a.model.AllocationID); err != nil {
		a.log.WithError(err).Error("error deleting allocation session")
	}
	if err := db.SingleDB().CompleteAllocation(&a.model); err != nil {
		a.log.WithError(err).Error("failed to mark allocation completed")
	}

	telemetry.ReportAllocationTerminal(a.system, db.SingleDB(), a.model, a.resources.firstDevice())
}

func (a *AllocationHandle) purgeRestorableResources() error {
	_, err := db.Bun().NewDelete().Model((*taskmodel.ResourcesWithState)(nil)).
		Where("allocation_id = ?", a.model.AllocationID).
		Exec(context.TODO())

	return err
}

func (a *AllocationHandle) getModelState() model.AllocationState {
	if a.model.State == nil {
		return model.AllocationStatePending
	}
	return *a.model.State
}

func (a *AllocationHandle) setModelState(v model.AllocationState) {
	a.model.State = &v
}

func (a *AllocationHandle) setMostProgressedModelState(v model.AllocationState) {
	a.setModelState(model.MostProgressedAllocationState(a.getModelState(), v))
}

func (a *AllocationHandle) ready() bool {
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

func (a *AllocationHandle) getPorts(exposedPorts map[string]int) (map[string]int, error) {
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
		a.log.Debugf("%v port : %v", portName, port)
	}

	return ports, nil
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
