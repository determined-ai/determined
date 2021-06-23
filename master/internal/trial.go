package internal

import (
	"archive/tar"
	"fmt"
	"sort"
	"time"

	"github.com/determined-ai/determined/master/pkg/searcher"

	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	apiutils "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

const (
	allReadyTimeoutPeriod = 10 * time.Minute
)

const (
	// MinLocalRendezvousPort is the smallest port to use (from the container's point of view;
	// it will be mapped to some arbitrary port on the host) for communication across containers.
	MinLocalRendezvousPort = 1734

	// MaxLocalRendezvousPort is the largest port to use for communication across containers.
	// Each distributed trial can take up to 2 host based ports and we assume a maximum.
	// of 16 slot per agent. MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1.
	MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1

	trialEntrypointFile = "/run/determined/train/entrypoint.sh"
	trialEntrypointMode = 0744

	// Put as many ssh-related files in /run/determined as possible. In particular, it is very
	// important that we don't overwrite the user's host $HOME/.ssh/id_rsa, if the user happens to
	// mount their host $HOME into the container's $HOME. Since we control the invocation of sshd,
	// we can keep our sshd_config in a location not likely to be mounted by users.
	trialAuthorizedKeysFile = "/run/determined/ssh/authorized_keys"
	trialAuthorizedKeysMode = 0600
	trialRSAPublicKeyFile   = "/run/determined/ssh/id_rsa.pub"
	trialRSAPublicKeyMode   = 0600
	trialRSAPrivateKeyFile  = "/run/determined/ssh/id_rsa"
	trialRSAPrivateKeyMode  = 0600
	trialSSHDConfigFile     = "/run/determined/ssh/sshd_config"
	trialSSHDConfigMode     = 0600
	trialSSHDir             = "/run/determined/ssh"
	trialSSHDirMode         = 0700

	// horovodrun controls how ssh is invoked, and we are force to overwrite a default ssh
	// configuration file.
	trialSSHConfigFile = "/etc/ssh/ssh_config"
	trialSSHConfigMode = 0644
)

// Trial-specific actor messages.
type (
	killTrial    struct{}

	// It is possible that it takes very long for all containers to be connected after the first
	// container is connected. This might happen when the k8s cluster waits for new instances
	// to spin up, which might not happen at all. At the same time, taking up part of all
	// the resources and waiting is wasteful. So we need to detect this situation.
	allReadyTimeout struct {
		runID int
	}

	// trialWatchRendezvousInfoReq begins watching for rendezvous info.
	// When all the containers are ready, the trial will send all the
	// peer addresses on the channel in the response.
	watchRendezvousInfo   struct{ containerID cproto.ID }
	rendezvousInfoOrError struct {
		info *trialv1.RendezvousInfo
		err  error
	}
	rendezvousWatcher struct {
		C <-chan rendezvousInfoOrError
	}
	unwatchRendezvousInfo struct{ containerID cproto.ID }
)

// terminatedContainerWithState records the terminatedContainer message with some state about the
// trial at the time termination was received. That information is analyzed when determining if a
// trial should be considered to have errored or not.
type terminatedContainerWithState struct {
	exitStatus sproto.TaskContainerStopped
	isLeader   bool
}

// trial is an actor which is responsible for handling:
//  - messages from the scheduler,
//  - messages from the experiment,
//  - messages from the trial container(s), and
//  - keeping the trial table of the database up-to-date.
//
// It is not responsible for maintaining the current state of the task running in the trial
// container, or the desired state as described by searcher operations; that is offloaded onto the
// workloadSequencer.
type trial struct {
	id    int
	idSet bool

	rm                  *actor.Ref
	logger              *actor.Ref
	db                  *db.PgDB
	experimentState     model.State
	experiment          *model.Experiment
	config              expconf.ExperimentConfig
	modelDefinition     archive.Archive
	warmStartCheckpoint *model.Checkpoint

	// The following fields tracks the interaction with the resource providers.
	// The existence of task signifies the trial has requested to be allocated.
	task *sproto.AllocateRequest
	// The existence of allocations signifies the trial has been allocated.
	allocations []sproto.Allocation

	// The following fields tracks containers and their states.
	lastContainerConnectedTime time.Time
	startedContainers          map[cproto.ID]bool
	containers                 map[cproto.ID]cproto.Container // only for running containers.
	containerRanks             map[cproto.ID]int              // only for launched containers.
	containerAddresses         map[cproto.ID][]cproto.Address // only for running containers.
	terminatedContainers       map[cproto.ID]terminatedContainerWithState
	// tracks if allReady check has passed successfully.
	allReadySucceeded bool

	agentUserGroup *model.AgentUserGroup
	taskSpec       *tasks.TaskSpec
	privateKey     []byte
	publicKey      []byte

	// searcher encapsulates the searcher state of the trial.
	searcher trialSearcher
	// preemption encapsulates the preemption state of the current allocated task.
	// If there is no current task, or it is unallocated, it is nil.
	preemption *preemption
	// Map of container ID to watcher ID a rendezvous info listener.
	rendezvousWatchers map[cproto.ID]chan<- rendezvousInfoOrError

	// restarts is essentially a failure count, it increments when the trial fails and we retry it.
	restarts int

	// RunID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates any outstanding terminateTimeout messages so that we don't
	// accidentally kill a fresh container due to the terminateTimeout message from an older
	// container.
	runID int

	canceledBeforeReady bool
	killed              bool
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	exp *experiment,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	state TrialSearcherState,
) *trial {
	return &trial{
		rm:                  exp.rm,
		logger:              exp.trialLogger,
		db:                  exp.db,
		experimentState:     exp.State,
		experiment:          exp.Experiment,
		config:              config,
		modelDefinition:     exp.modelDefinition,
		warmStartCheckpoint: warmStartCheckpoint,

		startedContainers:    make(map[cproto.ID]bool),
		containers:           make(map[cproto.ID]cproto.Container),
		containerRanks:       make(map[cproto.ID]int),
		containerAddresses:   make(map[cproto.ID][]cproto.Address),
		terminatedContainers: make(map[cproto.ID]terminatedContainerWithState),

		agentUserGroup: exp.agentUserGroup,
		taskSpec:       exp.taskSpec,

		rendezvousWatchers: make(map[cproto.ID]chan<- rendezvousInfoOrError),

		searcher: newTrialSearcher(state),
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("experiment-id", t.experiment.ID)
		if t.idSet {
			// t.idSet in actor.PreStart indicates we are in a restart.
			ctx.AddLabel("trial-id", t.id)
			runID, restarts, err := t.db.TrialRunIDAndRestartCount(t.id)
			if err != nil {
				return errors.Wrap(err, "restoring old trial state")
			}
			t.runID = runID
			t.restarts = restarts
		}

	case model.State:
		t.experimentState = msg

	case sproto.ContainerLog:
		t.insertLog(ctx, msg.Container, msg.Message())

	case TrialSearcherState:
		t.searcher.setState(msg)

	case watchPreemption:
		if resp, err := t.preemption.watch(msg); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(resp)
		}
	case unwatchPreemption:
		t.preemption.unwatch(msg)

	case watchRendezvousInfo:
		if resp, err := t.registerRendezvousWatcher(ctx, msg); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(resp)
		}
	case unwatchRendezvousInfo:
		delete(t.rendezvousWatchers, msg.containerID)

	case actor.PostStop:
		if !t.idSet {
			return nil
		}

		if err := t.db.EndTrialRuns(t.id); err != nil {
			ctx.Log().WithError(err).Error(`
				failed to close trial runs on exit, if this was an unexpected exit
				then manual intervention may be needed to correct resource allocation accounting`)
		}

		if t.restarts > t.config.MaxRestarts() {
			if err := t.db.UpdateTrial(t.id, model.ErrorState); err != nil {
				ctx.Log().Error(err)
			}
			return errors.Errorf("trial %d failed and reached maximum number of restarts", t.id)
		}
		ctx.Log().Info("trial stopped successfully")
		endState := model.CompletedState
		if t.experimentState == model.StoppingCanceledState || t.killed {
			endState = model.CanceledState
		}
		if err := t.db.UpdateTrial(t.id, endState); err != nil {
			ctx.Log().Error(err)
		}
		return nil
	default:
		if t.task != nil {
			if err := t.runningReceive(ctx); err != nil {
				return err
			}
		}
	}

	if t.experimentState != model.ActiveState {
		_ = t.releaseResource(ctx)
	}

	if t.task == nil && t.searcher.workRemaining() && t.experimentState == model.ActiveState {
		slotsNeeded := t.config.Resources().SlotsPerTrial()
		label := t.config.Resources().AgentLabel()
		resourcePool := t.config.Resources().ResourcePool()
		var name string
		if t.idSet {
			name = fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experiment.ID)
		} else {
			name = fmt.Sprintf("Trial (Experiment %d)", t.experiment.ID)
		}

		t.runID++
		if err := t.db.AddTrialRun(t.id, t.runID); err != nil {
			return errors.Wrap(err, "failed to save trial run")
		}

		t.task = &sproto.AllocateRequest{
			ID:             sproto.NewTaskID(),
			Name:           name,
			Group:          ctx.Self().Parent(),
			SlotsNeeded:    slotsNeeded,
			NonPreemptible: false,
			Label:          label,
			ResourcePool:   resourcePool,
			FittingRequirements: sproto.FittingRequirements{
				SingleAgent: false,
			},
			TaskActor: ctx.Self(),
		}
		if err := ctx.Ask(t.rm, *t.task).Error(); err != nil {
			ctx.Log().Error(err)
			t.terminated(ctx)
		}
	}

	return nil
}

func (t *trial) registerRendezvousWatcher(
	ctx *actor.Context, msg watchRendezvousInfo,
) (rendezvousWatcher, error) {
	// Validate this watch request is unique and not stale.
	if _, ok := t.containerRanks[msg.containerID]; !ok {
		return rendezvousWatcher{}, apiutils.AsValidationError(
			"rendezvous request from stale container: %s", msg.containerID,
		)
	} else if _, ok := t.rendezvousWatchers[msg.containerID]; ok {
		return rendezvousWatcher{}, apiutils.AsValidationError(
			"rendezvous request from already connected container: %s", msg.containerID,
		)
	}

	// Channel is size 1 since rendezvous info will only ever be sent once.
	w := make(chan rendezvousInfoOrError, 1)
	t.rendezvousWatchers[msg.containerID] = w

	t.lastContainerConnectedTime = time.Now()
	if !t.allReady() {
		actors.NotifyAfter(ctx, allReadyTimeoutPeriod, allReadyTimeout{runID: t.runID})
		ctx.Log().Debug(
			"not sending rendezvous information because not all trial containers are connected",
		)
	} else {
		t.pushRendezvous(ctx)
	}

	return rendezvousWatcher{C: w}, nil
}

func (t *trial) runningReceive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated, sproto.ReleaseResources:
		return t.processSchedulerMsg(ctx)

	case sproto.TaskContainerStateChanged:
		if msg.Container.State != cproto.Assigned {
			t.startedContainers[msg.Container.ID] = true
		}
		switch msg.Container.State {
		case cproto.Running:
			return t.processContainerRunning(ctx, msg)
		case cproto.Terminated:
			t.processContainerTerminated(ctx, msg)
		}

	case actor.ChildFailed:
		ctx.Log().Info("found child actor failed, terminating forcibly")
		t.terminate(ctx)

	case killTrial:
		ctx.Log().Info("received API request to kill trial")
		t.killed = true
		t.terminate(ctx)

	case allReadyTimeout:
		if msg.runID == t.runID &&
			time.Now().After(t.lastContainerConnectedTime.Add(allReadyTimeoutPeriod)) {
			ctx.Tell(t.logger, model.TrialLog{
				TrialID: t.id, Message: "some containers are taking a long time to " +
					"connect to master; when running on kubernetes this may happen " +
					"because only some of the pods have been scheduled; it is possible " +
					"that some pods will never be scheduled without adding compute " +
					"resources or pausing / killing other experiments in the cluster",
			})
		}

	case actor.ChildStopped:

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (t *trial) processSchedulerMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated:
		if err := t.processAllocated(ctx, msg); err != nil {
			return err
		}

	case sproto.ReleaseResources:
		ctx.Log().Info("releasing resources because of being preempted")
		return t.releaseResource(ctx)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) releaseResource(ctx *actor.Context) error {
	if !t.allReady() {
		t.canceledBeforeReady = true
		t.terminate(ctx)
	} else {
		t.preempt(ctx)
	}
	return nil
}

func (t *trial) processID(id int) {
	t.id = id
	t.idSet = true
}

func (t *trial) processAllocated(
	ctx *actor.Context, msg sproto.ResourcesAllocated,
) error {
	// Ignore this message if the resources are already released or
	// it is from the last run of the trial.
	if t.task == nil {
		ctx.Log().Info("ignoring resource allocation since the resources are already released.")
		return nil
	} else if msg.ID != t.task.ID {
		ctx.Log().Info("ignoring resource allocation since it is from the last run of the trial.")
		return nil
	}

	t.allocations = msg.Allocations

	if len(t.privateKey) == 0 {
		generatedKeys, err := ssh.GenerateKey(nil)
		if err != nil {
			ctx.Respond(err)
			return err
		}
		t.privateKey = generatedKeys.PrivateKey
		t.publicKey = generatedKeys.PublicKey
	}
	if !t.idSet {
		modelTrial := model.NewTrial(
			t.searcher.requestID(),
			t.experiment.ID,
			model.JSONObj(t.searcher.hparams()),
			t.warmStartCheckpoint,
			int64(t.searcher.seed()))
		if err := t.db.AddTrial(modelTrial); err != nil {
			ctx.Log().WithError(err).Error("failed to save trial to database")
			t.terminate(ctx)
			return err
		}
		t.processID(modelTrial.ID)
		ctx.AddLabel("trial-id", t.id)
		ctx.Tell(t.rm, sproto.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experiment.ID),
			TaskHandler: ctx.Self(),
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.requestID(), trialID: t.id})
	}

	ctx.Log().Infof("starting trial container")

	additionalFiles := archive.Archive{
		t.agentUserGroup.OwnedArchiveItem(
			trialEntrypointFile,
			etc.MustStaticFile(etc.TrialEntrypointScriptResource),
			trialEntrypointMode,
			tar.TypeReg,
		),

		t.agentUserGroup.OwnedArchiveItem(trialSSHDir, nil, trialSSHDirMode, tar.TypeDir),
		t.agentUserGroup.OwnedArchiveItem(trialAuthorizedKeysFile,
			t.publicKey,
			trialAuthorizedKeysMode,
			tar.TypeReg,
		),
		t.agentUserGroup.OwnedArchiveItem(
			trialRSAPublicKeyFile, t.publicKey, trialRSAPublicKeyMode, tar.TypeReg,
		),
		t.agentUserGroup.OwnedArchiveItem(
			trialRSAPrivateKeyFile, t.privateKey, trialRSAPrivateKeyMode, tar.TypeReg,
		),
		t.agentUserGroup.OwnedArchiveItem(trialSSHDConfigFile,
			etc.MustStaticFile(etc.SSHDConfigResource),
			trialSSHDConfigMode,
			tar.TypeReg,
		),

		archive.RootItem(
			trialSSHConfigFile,
			etc.MustStaticFile(etc.SSHConfigResource),
			trialSSHConfigMode,
			tar.TypeReg,
		),
	}
	taskToken, err := t.db.StartTaskSession(string(t.task.ID))
	if err != nil {
		return errors.Wrap(err, "cannot start a new task session for a trial")
	}
	t.preemption = newPreemption()

	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	}

	for rank, a := range msg.Allocations {
		t.containerRanks[a.Summary().ID] = rank
		taskSpec := *t.taskSpec
		taskSpec.AgentUserGroup = t.agentUserGroup
		taskSpec.TaskToken = taskToken
		taskSpec.SetInner(&tasks.StartTrial{
			ExperimentID:     t.experiment.ID,
			TrialID:          t.id,
			ExperimentConfig: schemas.Copy(t.config).(expconf.ExperimentConfig),
			TaskRunID:        t.runID,
			ModelDefinition:  t.modelDefinition,
			HParams:          t.searcher.hparams(),
			TrialSeed:        t.searcher.seed(),
			LatestCheckpoint: latestCheckpoint,
			AdditionalFiles:  additionalFiles,
			IsMultiAgent:     len(t.allocations) > 1,
			Rank:             rank,
		})
		a.Start(ctx, taskSpec)
	}

	return nil
}

func formatAddress(p cproto.Address) string {
	return fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
}

// allReady returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (t *trial) allReady() bool {
	// If a trial has passed allReady it can never return to a state of not ready until the
	// current containers are all terminated.
	if t.allReadySucceeded {
		return true
	}

	allAddressesArrived := len(t.containerAddresses) == len(t.allocations)
	allWaiting := len(t.rendezvousWatchers) == len(t.allocations)

	t.allReadySucceeded = allAddressesArrived && allWaiting
	return t.allReadySucceeded
}

// pushRendezvous gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (t *trial) pushRendezvous(ctx *actor.Context) {
	caddrs, raddrs, err := t.rendezvousInfo(ctx)
	for _, caddr := range caddrs {
		c := caddr.container
		w := t.rendezvousWatchers[c.ID]
		if err != nil {
			w <- rendezvousInfoOrError{err: err}
		} else {
			w <- rendezvousInfoOrError{
				info: &trialv1.RendezvousInfo{
					Addresses: raddrs,
					Rank:      int32(t.containerRanks[c.ID]),
				},
			}
		}
		close(w)
		delete(t.rendezvousWatchers, c.ID)
	}
}

func (t *trial) closeRendezvous() {
	for cID, w := range t.rendezvousWatchers {
		w <- rendezvousInfoOrError{err: errors.New("task terminated")}
		close(w)
		delete(t.rendezvousWatchers, cID)
	}
}

type cAddress struct {
	container cproto.Container
	addresses []cproto.Address
	ordinal   int
}

func (t *trial) rendezvousInfo(ctx *actor.Context) ([]cAddress, []string, error) {
	ctx.Log().Info("found all containers are connected successfully")

	var caddrs []cAddress
	for k, v := range t.containers {
		caddr := cAddress{
			container: v,
			addresses: t.containerAddresses[k],
			ordinal:   t.containerRanks[k],
		}
		caddrs = append(caddrs, caddr)

		sort.Slice(caddr.addresses, func(i, j int) bool {
			a := caddr.addresses[i]
			b := caddr.addresses[j]

			return a.ContainerPort < b.ContainerPort
		})
	}

	sort.Slice(caddrs, func(i, j int) bool {
		a := caddrs[i]
		b := caddrs[j]
		switch {
		case a.ordinal == 0 && b.ordinal != 0:
			return true
		case a.ordinal != 0 && b.ordinal == 0:
			return false
		default:
			return a.container.ID < b.container.ID
		}
	})

	var raddrs []string
	var err *multierror.Error
	for _, caddr := range caddrs {
		var addrs []cproto.Address
		for _, addr := range caddr.addresses {
			if MinLocalRendezvousPort <= addr.ContainerPort &&
				addr.ContainerPort <= MaxLocalRendezvousPort {
				addrs = append(addrs, addr)
			}
		}

		if len(addrs) == 1 {
			raddrs = append(raddrs, formatAddress(addrs[0]))
		} else {
			err = multierror.Append(err, fmt.Errorf(
				"found %d rendezvous addresses instead of 1 for container %s; dropping rendezvous addresses %v",
				len(addrs), caddr.container.ID, addrs))
		}
	}
	return caddrs, raddrs, err.ErrorOrNil()
}

func (t *trial) processContainerRunning(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	ctx.Log().Infof("found container running: %s (rank %d)",
		msg.Container.ID, t.containerRanks[msg.Container.ID])

	t.containers[msg.Container.ID] = msg.Container
	t.containerAddresses[msg.Container.ID] = msg.ContainerStarted.Addresses
	if !t.allReady() {
		ctx.Log().Info("found not all containers are connected")
	} else {
		t.pushRendezvous(ctx)
	}
	return nil
}

func (t *trial) processContainerTerminated(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) {
	ctx.Log().Infof("found container terminated: %s", msg.Container.ID)
	t.terminatedContainers[msg.Container.ID] = terminatedContainerWithState{
		exitStatus: *msg.ContainerStopped,
		isLeader:   t.containerRanks[msg.Container.ID] == 0,
	}

	_, ok := t.containers[msg.Container.ID]
	delete(t.containers, msg.Container.ID)
	delete(t.containerAddresses, msg.Container.ID)

	exitMsg := msg.ContainerStopped.String()
	t.insertLog(ctx, msg.Container, exitMsg)

	// Terminate the task if the container never started (since this prevents the gang
	// from ever being able to start), if the leader of the gang has exited out, or if
	// one of the containers exited with a failure.
	if !ok || t.containerRanks[msg.Container.ID] == 0 || msg.ContainerStopped.Failure != nil {
		t.terminate(ctx)
	}

	// If all containers are terminated, the trial is considered terminated.
	if len(t.terminatedContainers) == len(t.allocations) {
		t.terminated(ctx)
	}
}

func (t *trial) canLog(ctx *actor.Context, msg string) bool {
	// Log messages should never come in before the trial ID is set, since no trial runners are
	// launched until after the trial ID is set. But for futureproofing, we will log an error while
	// we protect the database.
	if !t.idSet {
		ctx.Log().Warnf("not saving log message from container without a trial ID: %s", msg)
		return false
	}

	if t.logger == nil {
		// A trial created for a unit test does not have a logger.
		return false
	}
	return true
}

func (t *trial) insertLog(ctx *actor.Context, container cproto.Container, msg string) {
	if !t.canLog(ctx, msg) {
		return
	}

	cid := string(container.ID)
	now := time.Now()
	msg += "\n"
	level := "INFO"
	source := "master"
	stdType := "stdout"
	ctx.Tell(t.logger, model.TrialLog{
		TrialID: t.id,
		Log:     &msg,

		ContainerID: &cid,
		Timestamp:   &now,
		Level:       &level,
		Source:      &source,
		StdType:     &stdType,
	})
}

func (t *trial) terminate(ctx *actor.Context) {
	switch {
	case len(t.allocations) == 0:
		ctx.Log().Info("aborting trial before resources are allocated in response to kill")
		t.terminated(ctx)
	default:
		ctx.Log().Info("forcibly terminating trial")
		if t.task != nil && t.allocations != nil {
			for _, allocation := range t.allocations {
				allocation.Kill(ctx)
			}
		}
	}
}

func (t *trial) preempt(ctx *actor.Context) {
	switch {
	case len(t.allocations) == 0:
		ctx.Log().Info("aborting trial before resources are allocated in response to preemption")
		t.terminated(ctx)
	default:
		ctx.Log().Info("gracefully terminating trial")
		t.preemption.preempt()
	}
}

// terminated handles errors and restarting for trials when they are failed, paused, canceled,
// or killed.
func (t *trial) terminated(ctx *actor.Context) {
	status := t.taskExitStatus()
	if err := t.db.CompleteTrialRun(t.id, t.runID); err != nil {
		ctx.Log().WithError(err).Error("failed to mark trial run completed")
	}

	if t.task != nil {
		if err := t.db.DeleteTaskSessionByTaskID(string(t.task.ID)); err != nil {
			ctx.Log().WithError(err).Error("error delete task session for a trial")
		}
	}

	t.preemption.close()
	t.closeRendezvous()

	t.task = nil
	t.allocations = nil
	t.containerRanks = make(map[cproto.ID]int)
	ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})

	t.allReadySucceeded = false
	t.terminatedContainers = make(map[cproto.ID]terminatedContainerWithState)
	t.startedContainers = make(map[cproto.ID]bool)

	// Check reasons that indicate this termination should be final.
	switch {
	case t.searcher.finished():
		ctx.Log().Info("trial is finished")
		ctx.Self().Stop()
	case t.restarts == t.config.MaxRestarts():
		ctx.Log().WithField("failure", status.Failure).Info("trial exceeded max restarts")
		ctx.Self().Stop()
	case t.killed:
		ctx.Log().WithField("failure", status.Failure).Info("trial was killed")
		ctx.Self().Stop()
	case model.StoppingStates[t.experimentState]:
		ctx.Log().Info("trial's experiment is stopping")
		ctx.Self().Stop()
	// Check reasons that indicate this termination is OK or expected.
	case status.Failure == nil && !t.searcher.workRemaining():
		ctx.Log().Info("trial runner exited successfully after finishing work")
	case status.Failure.FailureType == aproto.TaskAborted:
		ctx.Log().WithField("failure", status.Failure).Info("trial runner aborted")
	case t.canceledBeforeReady:
		// This is just a special case to catch a hard kill that should look/be treated
		// like a hard kill.
		ctx.Log().Info("trial runner exited after preempted while unready")
		t.canceledBeforeReady = false
	// Default, something went wrong and we should restart and count it as a failure.
	default:
		// any task termination that isn't otherwise explainable is considered a restart.
		ctx.Log().WithField("failure", status.Failure).Errorf(
			"trial exited with failure (restart %d/%d)", t.restarts, t.config.MaxRestarts(),
		)
		t.restarts++
		if err := t.db.SetTrialRestartCount(t.id, t.restarts); err != nil {
			ctx.Log().WithError(err).Error("failed to set restart ID")
		}
	}
}

func (t *trial) taskExitStatus() aproto.ContainerStopped {
	switch leaderState, ok := t.getLeaderTerminatedState(); {
	case len(t.startedContainers) == 0:
		// If there are no containers started executing, consider as aborted.
		// The trial state will be restart.
		return aproto.ContainerError(aproto.TaskAborted, errors.New("task aborted"))
	case ok:
		return leaderState.exitStatus.ContainerStopped
	default:
		return aproto.ContainerError(aproto.AgentError, errors.New("no error status provided"))
	}
}

func (t trial) getLeaderTerminatedState() (terminatedContainerWithState, bool) {
	for _, c := range t.terminatedContainers {
		if c.isLeader {
			return c, true
		}
	}
	return terminatedContainerWithState{}, false
}

type (
	// watchPreemption begins watching if the task has been preempted.
	// The task responds to this message with a channel of bools, where sends of true
	// indicate to preempt and sends of false are used to synchronize (e.g. you want to
	// block until you receive _something_ but not until the first preemption).
	watchPreemption   struct{ id uuid.UUID }
	preemptionWatcher struct{ C <-chan struct{} }
	unwatchPreemption struct{ id uuid.UUID }

	// preemption represents the preemption status of a task. A task is assumed to be preempted
	// exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption. This is nice until we move to trial has many task actors / generic task actor,
	// where the lifetime of a "preemption" is equivalent to the lifetime of task and they can be
	// initialized together.
	preemption struct {
		preempted bool
		// Map of watcher ID to a bool indicating if the trial should preempt.
		watchers map[uuid.UUID]chan<- struct{}
	}
)

func newPreemption() *preemption {
	return &preemption{
		preempted: false,
		watchers:  map[uuid.UUID]chan<- struct{}{},
	}
}

func (p *preemption) watch(msg watchPreemption) (preemptionWatcher, error) {
	if p == nil {
		return preemptionWatcher{}, errors.New("no preemption status available nil preemption")
	}

	// Size 1; at most a single message can be sent and we don't want to block.
	w := make(chan struct{}, 1)
	p.watchers[msg.id] = w

	if p.preempted {
		w <- struct{}{}
		close(w)
		delete(p.watchers, msg.id)
	}

	return preemptionWatcher{C: w}, nil
}

func (p *preemption) unwatch(msg unwatchPreemption) {
	if p == nil {
		return
	}
	delete(p.watchers, msg.id)
}

func (p *preemption) preempt() {
	if p == nil {
		return
	}
	p.preempted = true
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
}

func (p *preemption) close() {
	if p == nil {
		return
	}
	p.preempt()
}

// trialSearcher manages all searcher-related logic for a trial.
type trialSearcher struct {
	state TrialSearcherState
}

func newTrialSearcher(state TrialSearcherState) trialSearcher {
	return trialSearcher{
		state: state,
	}
}

func (s *trialSearcher) setState(state TrialSearcherState) {
	s.state = state
}

func (s trialSearcher) workRemaining() bool {
	return !s.state.Complete
}

func (s trialSearcher) finished() bool {
	return s.state.Complete && s.state.Closed
}

func (s trialSearcher) requestID() model.RequestID {
	return s.state.Create.RequestID
}

func (s trialSearcher) seed() uint32 {
	return s.state.Create.TrialSeed
}

func (s trialSearcher) hparams() searcher.HParamSample {
	return s.state.Create.Hparams
}
