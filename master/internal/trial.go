package internal

import (
	"archive/tar"
	"fmt"
	"sort"
	"time"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

	"github.com/determined-ai/determined/master/pkg/searcher"

	"github.com/hashicorp/go-multierror"

	"github.com/determined-ai/determined/proto/pkg/trialv1"

	"github.com/google/uuid"

	"github.com/pkg/errors"

	apiutils "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	killTrial struct{}
)

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

	// System dependencies.
	rm                  *actor.Ref
	logger              *actor.Ref
	db                  *db.PgDB

	// Fields that are essentially configuration for the trial.
	experiment          *model.Experiment
	config              expconf.ExperimentConfig
	taskSpec       *tasks.TaskSpec
	modelDefinition     archive.Archive
	warmStartCheckpoint *model.Checkpoint
	agentUserGroup *model.AgentUserGroup

	// The state of the experiment.
	experimentState     model.State
	// searcher encapsulates the searcher state of the trial.
	searcher trialSearcher
	// restarts is essentially a failure count, it increments when the trial fails and we retry it.
	restarts int
	// RunID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates any outstanding terminateTimeout messages so that we don't
	// accidentally kill a fresh container due to the terminateTimeout message from an older
	// container.
	runID int

	// The following fields tracks the interaction with the resource providers.
	// The existence of task signifies the trial has requested to be allocated.
	task *sproto.AllocateRequest
	// The existence of allocations signifies the trial has been allocated.
	allocations []sproto.Allocation
	// The following fields tracks containers and their states.
	startedContainers    map[cproto.ID]bool
	containers           map[cproto.ID]cproto.Container // only for running containers.
	terminatedContainers map[cproto.ID]sproto.TaskContainerStopped
	// canceled marks that we are in the process of canceling or preempting the trial.
	canceled bool
	// killed marks that we are in the process of killing the trial.
	killed bool
	// stopping marks that ctx.Self().Stop() has been called and we are in the process
	// of stopping the trial. This is helpful to guarantee the condition to reschedule
	// a task is mutually exclusive with the trial closing.
	stopping            bool
	// preemption encapsulates the preemption state of the currently allocated task.
	// If there is no current task, or it is unallocated, it is nil.
	preemption *preemption
	// rendezvous encapsulates logic of rendezvousing containers of the currently
	// allocated task. If there is no current task, or it is unallocated, it is nil.
	rendezvous *rendezvous
	// The SSH keys used for distributed training.
	privateKey     []byte
	publicKey      []byte
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
		taskSpec:       exp.taskSpec,
		modelDefinition:     exp.modelDefinition,
		warmStartCheckpoint: warmStartCheckpoint,
		agentUserGroup: exp.agentUserGroup,

		startedContainers:    make(map[cproto.ID]bool),
		containers:           make(map[cproto.ID]cproto.Container),
		terminatedContainers: make(map[cproto.ID]sproto.TaskContainerStopped),

		searcher: newTrialSearcher(state),
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("experiment-id", t.experiment.ID)
		if t.idSet {
			ctx.AddLabel("trial-id", t.id)
			if err := t.recover(); err != nil {
				return err
			}
		}
	case actor.PostStop:
		return t.close(ctx)

	// Messages relaying external state changes.
	case experimentStateChanged:
		t.experimentState = msg.state
		if t.task == nil && model.StoppingStates[t.experimentState] {
			ctx.Self().Stop()
			t.stopping = true
		} else if t.task != nil && msg.state != model.ActiveState {
			t.releaseResources(ctx)
		}
	case TrialSearcherState:
		t.searcher.setState(msg)
		if t.task == nil && t.searcher.finished() {
			ctx.Self().Stop()
			t.stopping = true
		} else if t.task != nil && t.searcher.finished() {
			t.releaseResources(ctx)
		}
	case killTrial:
		if t.task == nil {
			ctx.Self().Stop()
			t.stopping = true
		} else {
			t.killed = true
			t.terminate(ctx)
		}

	// Messages only received by the active task. If there is no active task, receipt of these messages is invalid.
	case sproto.ResourcesAllocated, sproto.ReleaseResources, sproto.TaskContainerStateChanged,
		watchRendezvousInfo, unwatchRendezvousInfo, rendezvousTimeout, watchPreemption, unwatchPreemption:
		if t.task == nil {
			ctx.Log().WithError(actor.ErrUnexpectedMessage(ctx)).Warnf("message invalid without active task")
			return nil
		}
		return t.processTaskMessage(ctx)
	case sproto.ContainerLog:
		t.insertLog(ctx, &msg.Container.ID, msg.Message())

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	if t.task == nil && t.searcher.workRemaining() && t.experimentState == model.ActiveState && !t.stopping {
		if err := t.allocate(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (t *trial) allocate(ctx *actor.Context) error {
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
		SlotsNeeded:    t.config.Resources().SlotsPerTrial(),
		NonPreemptible: false,
		Label:          t.config.Resources().AgentLabel(),
		ResourcePool:   t.config.Resources().ResourcePool(),
		FittingRequirements: sproto.FittingRequirements{
			SingleAgent: false,
		},
		TaskActor: ctx.Self(),
	}
	if err := ctx.Ask(t.rm, *t.task).Error(); err != nil {
		ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
		return errors.Wrap(err, "failed to request allocation")
	}
	return nil
}

func (t *trial) recover() error {
	runID, restarts, err := t.db.TrialRunIDAndRestartCount(t.id)
	if err != nil {
		return errors.Wrap(err, "restoring old trial state")
	}
	t.runID = runID
	t.restarts = restarts
	return nil
}

func (t *trial) close(ctx *actor.Context) error {
	if !t.idSet {
		return nil
	}

	if err := t.db.EndTrialRuns(t.id); err != nil {
		return errors.Wrap(err, "failed to close trial runs on exit")
	}

	if t.restarts > t.config.MaxRestarts() {
		if err := t.db.UpdateTrial(t.id, model.ErrorState); err != nil {
			ctx.Log().Error(err)
		}
		return errors.Errorf("trial %d failed and reached maximum number of restarts", t.id)
	}

	endState := model.CompletedState
	if t.experimentState == model.StoppingCanceledState || t.killed {
		endState = model.CanceledState
	}

	if err := t.db.UpdateTrial(t.id, endState); err != nil {
		return errors.Wrap(err, "failed to update trial with end state")
	}
	return nil
}

func (t *trial) processTaskMessage(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated, sproto.ReleaseResources:
		if err := t.processSchedulerMessage(ctx); err != nil {
			return err
		}
	case sproto.TaskContainerStateChanged:
		if err := t.processContainerMessage(ctx, msg); err != nil {
			return err
		}

	case watchPreemption:
		if resp, err := t.preemption.watch(msg.id); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(resp)
		}
	case unwatchPreemption:
		t.preemption.unwatch(msg.id)
	case preemptionTimeout:
		if err := t.preemption.checkTimeout(t.runID); err != nil {
			ctx.Log().WithError(err).Info("forcibly terminating trial")
			t.terminate(ctx)
		}

	case watchRendezvousInfo:
		resp, err := t.rendezvous.watch(msg.id)
		switch {
		case err != nil:
			ctx.Respond(err)
			return nil
		case t.rendezvous.ready():
			ctx.Log().Info("all containers are connected successfully (watcher connected)")
		}
		ctx.Respond(resp)
	case unwatchRendezvousInfo:
		t.rendezvous.unwatch(msg.id)
	case rendezvousTimeout:
		if err := t.rendezvous.checkTimeout(msg.runID); err != nil {
			ctx.Tell(t.logger, model.TrialLog{TrialID: t.id, Message: err.Error()})
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processSchedulerMessage(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated:
		if err := t.processAllocated(ctx, msg); err != nil {
			return err
		}
	case sproto.ReleaseResources:
		ctx.Log().Info("releasing resources because of being preempted")
		t.releaseResources(ctx)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processContainerMessage(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	if msg.Container.State != cproto.Assigned {
		t.startedContainers[msg.Container.ID] = true
	}
	switch msg.Container.State {
	case cproto.Running:
		return t.processContainerRunning(ctx, msg)
	case cproto.Terminated:
		t.processContainerTerminated(ctx, msg)
	}
	return nil
}

func (t *trial) releaseResources(ctx *actor.Context) {
	t.canceled = true
	if !t.rendezvous.ready() {
		t.terminate(ctx)
	} else {
		t.preempt(ctx)
	}
}

func (t *trial) processID(id int) {
	t.id = id
	t.idSet = true
}

func (t *trial) processAllocated(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	// Ignore this message if it is from the last run of the trial.
	if msg.ID != t.task.ID {
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
		//
		modelTrial := model.NewTrial(
			t.searcher.requestID(),
			t.experiment.ID,
			model.JSONObj(t.searcher.hparams()),
			t.warmStartCheckpoint,
			int64(t.searcher.seed()))
		if err := t.db.AddTrial(modelTrial); err != nil {
			return errors.Wrap(err, "failed to save trial to database")
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
	t.preemption = newPreemption(t.runID)
	t.rendezvous = newRendezvous(t.runID, ranksFromAllocations(msg.Allocations))
	actors.NotifyAfter(ctx, rendezvousTimeoutDuration, rendezvousTimeout{runID: t.runID})

	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	}

	for rank, a := range msg.Allocations {
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

func (t *trial) processContainerRunning(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	ctx.Log().Infof("found container running: %s (rank %d)",
		msg.Container.ID, t.rendezvous.rank(msg.Container.ID))

	t.containers[msg.Container.ID] = msg.Container
	t.rendezvous.containerStarted(msg.Container.ID, msg.ContainerStarted.Addresses)
	if t.rendezvous.ready() {
		ctx.Log().Info("all containers are connected successfully (task container state changed)")
	}
	return nil
}

func (t *trial) processContainerTerminated(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) {
	cID := msg.Container.ID
	exit := *msg.ContainerStopped

	ctx.Log().Infof("found container terminated: %s", cID)
	t.insertLog(ctx, &msg.Container.ID, exit.String())
	t.terminatedContainers[cID] = exit

	_, ok := t.containers[cID]
	delete(t.containers, cID)
	t.rendezvous.containerTerminated(cID)

	// Terminate the task if the container never started (since this prevents the gang
	// from ever being able to start), if the leader of the gang has exited out, or if
	// one of the containers exited with a failure.
	if !ok || t.rendezvous.isLeader(cID) || exit.Failure != nil {
		t.terminate(ctx)
	}

	// If all containers are terminated, the trial is considered terminated.
	if len(t.terminatedContainers) == len(t.allocations) {
		t.terminated(ctx)
	}
}

func (t *trial) terminate(ctx *actor.Context) {
	switch {
	case len(t.allocations) == 0:
		ctx.Log().Info("aborting trial before resources are allocated in response to kill")
		t.terminated(ctx)
	default:
		ctx.Log().Info("forcibly terminating trial")
		for _, allocation := range t.allocations {
			allocation.Kill(ctx)
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
		ctx.Tell(ctx.Self(), preemptionTimeout{t.runID})
	}
}

// terminated handles errors and restarting for trials when they are failed, paused, canceled,
// or killed.
func (t *trial) terminated(ctx *actor.Context) {
	defer func() {
		ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})
		if err := t.resetTask(); err != nil {
			ctx.Log().WithError(err).Error("failed to reset task")
		}
	}()

	var final bool
	switch status := t.taskExitStatus(); {
	// Check reasons that indicate this termination should be final.
	case t.searcher.finished() && status.Failure != nil:
		ctx.Log().WithError(status.Failure).Warn("trial closed with an error")
		final = true
	case t.searcher.finished() && status.Failure == nil:
		ctx.Log().Info("trial closed")
		final = true
	case t.restarts == t.config.MaxRestarts() && status.Failure != nil:
		ctx.Log().WithError(status.Failure).Info("trial exceeded max restarts")
		t.restarts++ // needed to distinguish succeeding on the Nth restart and failing N times
		final = true
	case t.killed:
		ctx.Log().WithError(status.Failure).Info("trial was killed")
		final = true
	case model.StoppingStates[t.experimentState]:
		ctx.Log().Infof("trial closed after experiment transitioned to %s", t.experimentState)
		final = true

	// Check reasons that indicate this termination is OK or expected.
	case status.Failure == nil && !t.searcher.workRemaining():
		ctx.Log().Info("trial finished current work")
	case status.Failure == nil && t.searcher.workRemaining():
		ctx.Log().Warn("trial exited but was not finished with operations")
	case status.Failure != nil && !t.searcher.workRemaining():
		ctx.Log().WithError(status.Failure).Warn("trial failed but was finished with operations")
	case status.Failure.FailureType == aproto.TaskAborted:
		ctx.Log().WithError(status.Failure).Info("trial exited after being aborted")
	case t.experimentState != model.ActiveState:
		ctx.Log().Infof("trial exited after experiment transitioned to %s", t.experimentState)
	case t.canceled && !t.rendezvous.ready():
		ctx.Log().WithError(status.Failure).Info("trial was canceled while unready")
	case t.canceled:
		ctx.Log().Info("trial was preempted and canceled")

	// Default case, something went wrong and we should restart and count it as a failure.
	default:
		// any task termination that isn't otherwise explainable is considered a restart.
		ctx.Log().WithError(status.Failure).Errorf(
			"trial exited unexpectedly (restart %d/%d)", t.restarts, t.config.MaxRestarts(),
		)
		t.restarts++
		if err := t.db.SetTrialRestartCount(t.id, t.restarts); err != nil {
			ctx.Log().WithError(err).Error("failed to persist restart count")
		}
	}

	if final {
		ctx.Self().Stop()
		t.stopping = true
		return
	}
}

func (t *trial) resetTask() error {
	if t.task == nil {
		return errors.New("cannot reset nil task")
	}

	var mErr *multierror.Error
	if err := t.db.CompleteTrialRun(t.id, t.runID); err != nil {
		mErr = multierror.Append(mErr, errors.Wrap(err, "failed to mark trial run completed"))
	}

	if t.task != nil {
		if err := t.db.DeleteTaskSessionByTaskID(string(t.task.ID)); err != nil {
			mErr = multierror.Append(mErr, errors.Wrap(err, "error delete task session for a trial"))
		}
	}

	t.preemption.close()
	t.preemption = nil
	t.rendezvous.close()
	t.rendezvous = nil
	t.task = nil
	t.allocations = nil
	t.terminatedContainers = make(map[cproto.ID]sproto.TaskContainerStopped)
	t.startedContainers = make(map[cproto.ID]bool)
	t.canceled = false

	return mErr.ErrorOrNil()
}

func (t *trial) taskExitStatus() aproto.ContainerStopped {
	if len(t.startedContainers) == 0 {
		return aproto.ContainerError(aproto.TaskAborted, errors.New("task aborted"))
	}

	for cID, exit := range t.terminatedContainers {
		if t.rendezvous.isLeader(cID) {
			return exit.ContainerStopped
		}
	}
	return aproto.ContainerError(aproto.AgentError, errors.New("no error status provided"))
}

func (t *trial) insertLog(ctx *actor.Context, cID *cproto.ID, msg string) {
	// Log messages should never come in before the trial ID is set, since no trial runners are
	// launched until after the trial ID is set. But for futureproofing, we will log an error while
	// we protect the database.
	if !t.idSet {
		ctx.Log().Warnf("not saving log message from container without a trial ID: %s", msg)
		return
	}

	if t.logger == nil {
		// A trial created for a unit test does not have a logger.
		return
	}

	var cIDStr string
	if cID != nil {
		cIDStr = string(*cID)
	}
	now := time.Now()
	msg += "\n"
	level := "INFO"
	source := "master"
	stdType := "stdout"
	ctx.Tell(t.logger, model.TrialLog{
		TrialID: t.id,
		Log:     &msg,

		ContainerID: &cIDStr,
		Timestamp:   &now,
		Level:       &level,
		Source:      &source,
		StdType:     &stdType,
	})
}

var (
	rendezvousTimeoutDuration = 10 * time.Minute
)

type (
	// watchRendezvousInfo begins watching for rendezvous info.
	// When all the containers are ready, the trial will send all the
	// peer addresses on the channel in the response.
	watchRendezvousInfo   struct{ id cproto.ID }
	rendezvousInfoOrError struct {
		info *trialv1.RendezvousInfo
		err  error
	}
	rendezvousWatcher struct {
		C <-chan rendezvousInfoOrError
	}
	unwatchRendezvousInfo struct{ id cproto.ID }

	// It is possible that it takes very long for all containers to be connected after the first
	// container is connected. This might happen when the k8s cluster waits for new instances
	// to spin up, which might not happen at all. At the same time, taking up part of all
	// the resources and waiting is wasteful. So we need to detect this situation.
	rendezvousTimeout struct {
		runID int
	}

	// rendezvous encapsulates the rendezvous state of a trial.
	rendezvous struct {
		runID             int
		watchers          map[cproto.ID]chan<- rendezvousInfoOrError
		ranks             map[cproto.ID]int
		addresses         map[cproto.ID][]cproto.Address
		lastWatchTime     time.Time
		allReadySucceeded bool
	}
)

func newRendezvous(runID int, ranks map[cproto.ID]int) *rendezvous {
	return &rendezvous{
		runID:     runID,
		ranks:     ranks,
		addresses: map[cproto.ID][]cproto.Address{},
		watchers:  map[cproto.ID]chan<- rendezvousInfoOrError{},
	}
}

func ranksFromAllocations(allocations []sproto.Allocation) map[cproto.ID]int {
	ranks := map[cproto.ID]int{}
	for rank, a := range allocations {
		ranks[a.Summary().ID] = rank
	}
	return ranks
}

func (r *rendezvous) watch(id cproto.ID) (rendezvousWatcher, error) {
	if r == nil {
		return rendezvousWatcher{}, apiutils.AsValidationError(
			"no rendezvous for unallocated task",
		)
	} else if _, ok := r.ranks[id]; !ok {
		return rendezvousWatcher{}, apiutils.AsValidationError(
			"rendezvous request from stale container: %s", id,
		)
	} else if _, ok := r.watchers[id]; ok {
		return rendezvousWatcher{}, apiutils.AsValidationError(
			"rendezvous request from already connected container: %s", id,
		)
	}

	// Channel is size 1 since rendezvous info will only ever be sent once.
	w := make(chan rendezvousInfoOrError, 1)
	r.watchers[id] = w
	r.lastWatchTime = time.Now()
	if r.ready() {
		r.push()
	}
	return rendezvousWatcher{C: w}, nil
}

func (r *rendezvous) unwatch(id cproto.ID) {
	if r == nil {
		return
	}
	delete(r.watchers, id)
}

func (r *rendezvous) containerStarted(id cproto.ID, addresses []cproto.Address) {
	r.addresses[id] = addresses
	if r.ready() {
		r.push()
	}
}

func (r *rendezvous) containerTerminated(id cproto.ID) {
	delete(r.addresses, id)
}

func (r rendezvous) isLeader(id cproto.ID) bool {
	return r.ranks[id] == 0
}

func (r rendezvous) rank(id cproto.ID) int {
	return r.ranks[id]
}

// ready returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (r *rendezvous) ready() bool {
	// If a trial has passed allReady it can never return to a state of not ready until the
	// current containers are all terminated.
	if r.allReadySucceeded {
		return true
	}

	allAddressesArrived := len(r.addresses) == len(r.ranks)
	allWaiting := len(r.watchers) == len(r.ranks)

	r.allReadySucceeded = allAddressesArrived && allWaiting
	return r.allReadySucceeded
}

// push gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (r rendezvous) push() bool {
	if !r.ready() {
		return false
	}
	caddrs, raddrs, err := r.info()
	for _, caddr := range caddrs {
		w := r.watchers[caddr.id]
		w <- rendezvousInfoOrError{
			info: &trialv1.RendezvousInfo{
				Addresses: raddrs,
				Rank:      int32(r.ranks[caddr.id]),
			},
			err: err,
		}
		close(w)
		delete(r.watchers, caddr.id)
	}
	return true
}

// checkTimeout checks if the task should timeout waiting for rendezvous.
func (r *rendezvous) checkTimeout(runID int) error {
	if r == nil {
		return nil
	}

	if r.runID == runID && time.Now().After(r.lastWatchTime.Add(rendezvousTimeoutDuration)) {
		return errors.New("some containers are taking a long time to " +
			"connect to master; when running on kubernetes this may happen " +
			"because only some of the pods have been scheduled; it is possible " +
			"that some pods will never be scheduled without adding compute " +
			"resources or pausing / killing other experiments in the cluster",
		)
	}
	return nil
}

func (r *rendezvous) close() {
	for cID, w := range r.watchers {
		w <- rendezvousInfoOrError{err: errors.New("task terminated")}
		close(w)
		delete(r.watchers, cID)
	}
}

type cAddress struct {
	id        cproto.ID
	addresses []cproto.Address
	ordinal   int
}

func (r *rendezvous) info() ([]cAddress, []string, error) {
	var caddrs []cAddress
	for id, rank := range r.ranks {
		caddr := cAddress{
			id:        id,
			addresses: r.addresses[id],
			ordinal:   rank,
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
			return a.id < b.id
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
				len(addrs), caddr.id, addrs))
		}
	}
	return caddrs, raddrs, err.ErrorOrNil()
}

func formatAddress(p cproto.Address) string {
	return fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
}

var (
	preemptionTimeoutDuration = time.Hour
)

type (
	// watchPreemption begins watching if the task has been preempted.
	// The task responds to this message with a channel of bools, where sends of true
	// indicate to preempt and sends of false are used to synchronize (e.g. you want to
	// block until you receive _something_ but not until the first preemption).
	watchPreemption   struct{ id uuid.UUID }
	preemptionWatcher struct{ C <-chan struct{} }
	unwatchPreemption struct{ id uuid.UUID }

	// preemptionTimeout is the time after which we forcibly terminate a trial that has no
	// preempted.
	preemptionTimeout struct {
		runID int
	}

	// preemption represents the preemption status of a task. A task is assumed to be preempted
	// exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption. This is nice until we move to trial has many task actors / generic task actor,
	// where the lifetime of a "preemption" is equivalent to the lifetime of task and they can be
	// initialized together.
	preemption struct {
		runID int
		preempted bool
		preemptedAt time.Time
		// Map of watcher ID to a bool indicating if the trial should preempt.
		watchers map[uuid.UUID]chan<- struct{}
	}
)

func newPreemption(runID int) *preemption {
	return &preemption{
		runID: runID,
		preempted: false,
		watchers:  map[uuid.UUID]chan<- struct{}{},
	}
}

func (p *preemption) watch(id uuid.UUID) (preemptionWatcher, error) {
	if p == nil {
		return preemptionWatcher{}, errors.New("no preemption status available nil preemption")
	}

	// Size 1; at most a single message can be sent and we don't want to block.
	w := make(chan struct{}, 1)
	p.watchers[id] = w

	if p.preempted {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}

	return preemptionWatcher{C: w}, nil
}

func (p *preemption) unwatch(id uuid.UUID) {
	if p == nil {
		return
	}
	delete(p.watchers, id)
}

func (p *preemption) preempt() {
	if p == nil {
		return
	}
	p.preempted = true
	p.preemptedAt = time.Now()
	for id, w := range p.watchers {
		w <- struct{}{}
		close(w)
		delete(p.watchers, id)
	}
}

func (p *preemption) checkTimeout(runID int) error {
	if p == nil {
		return nil
	}

	if p.runID == runID && time.Now().After(p.preemptedAt.Add(preemptionTimeoutDuration)) {
		return errors.New("preemption timeout out")
	}
	return nil
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
