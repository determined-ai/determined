package internal

import (
	"fmt"
	"sort"
	"time"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/pkg/actor/actors"

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
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// trial is an actor which is responsible for handling:
//  - messages from the scheduler,
//  - messages from the experiment,
//  - messages from the trial container(s), and
//  - keeping the trial table of the database up-to-date.
type trial struct {
	id           int
	idSet        bool
	experimentID int

	// System dependencies.
	rm     *actor.Ref
	logger *actor.Ref
	db     db.DB

	// Fields that are essentially configuration for the trial.
	config              expconf.ExperimentConfig
	taskSpec            *tasks.TaskSpec
	modelDefinition     archive.Archive
	warmStartCheckpoint *model.Checkpoint
	generatedKeys       *ssh.PrivateAndPublicKeys

	// targetState is the state we're aiming for. It's patched by experiment changes and kill trial.
	targetState model.State
	// searcher encapsulates the searcher state of the trial.
	searcher TrialSearcherState
	// restarts is a failure count, it increments when the trial fails and we retry it.
	restarts int
	// runID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When RunID increments,
	// it effectively invalidates many outstanding messages associated with the previous run.
	runID int
	// stopped marks that ctx.Self().Stop() has been called and we are in the process
	// of stopping the trial. This is helpful to guarantee the condition to reschedule
	// a task is mutually exclusive with the trial closing.
	stopped bool
	// finalState records the termination state of a closing trial.
	finalState model.State

	// The following fields tracks the interaction with the resource providers.
	// The existence of task signifies the trial has requested to be allocated.
	task *sproto.AllocateRequest
	// The existence of allocations signifies the trial has been allocated.
	allocations map[cproto.ID]sproto.Allocation
	// The following fields tracks containers and their states.
	containers           map[cproto.ID]cproto.Container
	terminatedFirst      *cproto.ID
	terminatedContainers map[cproto.ID]sproto.TaskContainerStopped
	// preemption encapsulates the preemption state of the currently allocated task.
	// If there is no current task, or it is unallocated, it is nil.
	preemption *preemption
	// rendezvous encapsulates logic of rendezvousing containers of the currently
	// allocated task. If there is no current task, or it is unallocated, it is nil.
	rendezvous *rendezvous
	// we send a kill when we terminate a task forcibly. we terminate forcibly when a container
	// exits non zero. we don't need to send all these kills, so this exists.
	killCooldown *time.Time
}

const killCooldown = 30 * time.Second

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	experimentID int,
	initialState model.State,
	searcher TrialSearcherState,
	rm, logger *actor.Ref,
	db db.DB,
	config expconf.ExperimentConfig,
	warmStartCheckpoint *model.Checkpoint,
	taskSpec *tasks.TaskSpec,
	modelDefinition archive.Archive,
) *trial {
	return &trial{
		experimentID: experimentID,
		targetState:  initialState,
		searcher:     searcher,

		rm:     rm,
		logger: logger,
		db:     db,

		config:              config,
		taskSpec:            taskSpec,
		modelDefinition:     modelDefinition,
		warmStartCheckpoint: warmStartCheckpoint,

		containers:           make(map[cproto.ID]cproto.Container),
		terminatedContainers: make(map[cproto.ID]sproto.TaskContainerStopped),
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		return t.prestart(ctx)
	case actor.PostStop:
		return t.close()

	case model.State:
		t.targetState = msg
		switch {
		case t.targetState == model.ActiveState:
			return t.maybeAllocate(ctx)
		case t.targetState == model.PausedState:
			return t.terminate(ctx, preempt)
		case model.StoppingStates[t.targetState]:
			return t.terminate(ctx, kill)
		}
	case TrialSearcherState:
		t.searcher = msg
		switch {
		case !t.searcher.Complete:
			return t.maybeAllocate(ctx)
		case t.searcher.Finished():
			return t.terminate(ctx, noop)
		}

	case sproto.ResourcesAllocated, sproto.TaskContainerStateChanged,
		sproto.ReleaseResources, sproto.ContainerLog:
		return t.processTask(ctx)
	case watchRendezvousInfo, unwatchRendezvousInfo, rendezvousTimeout:
		return t.processRendezvous(ctx)
	case watchPreemption, unwatchPreemption, preemptionTimeout, ackPreemption:
		return t.processPreemption(ctx)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (t *trial) prestart(ctx *actor.Context) error {
	ctx.AddLabel("experiment-id", t.experimentID)
	if t.idSet {
		ctx.AddLabel("trial-id", t.id)
		if err := t.recover(); err != nil {
			return err
		}
		ctx.AddLabel("task-run-id", t.runID)
	}
	return nil
}

func (t *trial) processTask(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated:
		return t.processAllocated(ctx, msg)
	case sproto.TaskContainerStateChanged:
		return t.processContainerMessage(ctx, msg)
	case sproto.ReleaseResources:
		return t.terminate(ctx, preempt)
	case sproto.ContainerLog:
		t.insertLog(ctx, &msg.Container.ID, msg.Message())
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processRendezvous(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case watchRendezvousInfo:
		if w, err := t.rendezvous.watch(msg.taskID, msg.id); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(w)
		}
	case unwatchRendezvousInfo:
		t.rendezvous.unwatch(msg.id)
	case rendezvousTimeout:
		if err := t.rendezvous.checkTimeout(msg.taskID); err != nil {
			ctx.Tell(t.logger, model.TrialLog{TrialID: t.id, Message: err.Error()})
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processPreemption(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case watchPreemption:
		if w, err := t.preemption.watch(msg.taskID, msg.id); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(w)
		}
	case unwatchPreemption:
		t.preemption.unwatch(msg.id)
	case preemptionTimeout:
		if err := t.preemption.checkTimeout(msg.taskID); err != nil {
			ctx.Log().WithError(err).Info("forcibly terminating trial")
			return t.terminate(ctx, kill)
		}
	case ackPreemption:
		if err := t.preemption.acknowledge(msg.taskID); err != nil {
			if ctx.ExpectingResponse() {
				ctx.Respond(err)
			}
		}
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processContainerMessage(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	if _, ok := t.allocations[msg.Container.ID]; !ok {
		return errStaleContainer{id: msg.Container.ID}
	}

	t.containers[msg.Container.ID] = msg.Container
	rank := t.rendezvous.rank(msg.Container.ID)
	ctx.Log().Infof("container %s (rank %d) is %s", msg.Container.ID, rank, msg.Container.State)
	switch msg.Container.State {
	case cproto.Running:
		t.processContainerRunning(ctx, msg)
	case cproto.Terminated:
		return t.processContainerTerminated(ctx, msg)
	}
	return nil
}

func (t *trial) maybeAllocate(ctx *actor.Context) error {
	if !(t.task == nil &&
		!t.searcher.Complete &&
		t.targetState == model.ActiveState &&
		!t.stopped) {
		return nil
	}

	var name string
	if t.idSet {
		name = fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID)
	} else {
		name = fmt.Sprintf("Trial (Experiment %d)", t.experimentID)
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
		return errors.Wrap(err, "failed to request allocation")
	}
	return nil
}

func (t *trial) recover() error {
	runID, restarts, err := t.db.TrialRunIDAndRestarts(t.id)
	if err != nil {
		return errors.Wrap(err, "restoring old trial state")
	}
	t.runID = runID + 1
	t.restarts = restarts
	return nil
}

func (t *trial) close() error {
	if !t.idSet {
		return nil
	}

	if !t.stopped {
		t.finalState = model.ErrorState
	}

	if err := t.db.EndTasks(model.JobTypeTrial, t.id); err != nil {
		return errors.Wrap(err, "ensuring all trial tasks final on exit")
	}

	if err := t.db.UpdateTrial(t.id, t.finalState); err != nil {
		return errors.Wrap(err, "updating trial with end state")
	}

	return nil
}

func (t *trial) setID(id int) {
	t.id = id
	t.idSet = true
}

func (t *trial) processAllocated(ctx *actor.Context, msg sproto.ResourcesAllocated) error {
	// Ignore this message if it is from the last run of the trial.
	if t.task == nil || msg.ID != t.task.ID {
		ctx.Log().Infof("ignoring and stale allocation %v (task = %v)", msg, t.task)
		return nil
	}

	t.allocations = map[cproto.ID]sproto.Allocation{}
	for _, a := range msg.Allocations {
		t.allocations[a.Summary().ID] = a
	}

	if t.generatedKeys == nil {
		generatedKeys, err := ssh.GenerateKey(nil)
		if err != nil {
			return errors.Wrap(err, "failed to generate keys for trial")
		}
		t.generatedKeys = &generatedKeys
	}

	if !t.idSet {
		modelTrial := model.NewTrial(
			t.searcher.Create.RequestID,
			t.experimentID,
			model.JSONObj(t.searcher.Create.Hparams),
			t.warmStartCheckpoint,
			int64(t.searcher.Create.TrialSeed))
		if err := t.db.AddTrial(modelTrial); err != nil {
			return errors.Wrap(err, "failed to save trial to database")
		}
		t.setID(modelTrial.ID)
		ctx.AddLabel("trial-id", t.id)
		ctx.Tell(t.rm, sproto.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experimentID),
			TaskHandler: ctx.Self(),
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{requestID: t.searcher.Create.RequestID, trialID: t.id})
	}

	t.runID++
	if err := t.db.UpdateTrialRunID(t.id, t.runID); err != nil {
		return errors.Wrap(err, "failed to save trial run ID")
	}
	ctx.AddLabel("trial-run-id", t.runID)

	if err := t.db.AddTask(model.JobTypeTrial, t.id, t.task.ID); err != nil {
		return errors.Wrap(err, "failed to save trial task")
	}

	ctx.Log().Infof("starting trial container")

	taskToken, err := t.db.StartTaskSession(string(t.task.ID))
	if err != nil {
		return errors.Wrap(err, "cannot start a new task session for a trial")
	}
	t.preemption = newPreemption(t.task.ID)
	t.rendezvous = newRendezvous(t.task.ID, ranksFromAllocations(msg.Allocations))
	actors.NotifyAfter(ctx, rendezvousTimeoutDuration, rendezvousTimeout{taskID: t.task.ID})

	var latestBatch int
	latestCheckpoint, err := t.db.LatestCheckpointForTrial(t.id)
	switch {
	case err != nil:
		return errors.Wrapf(err, "failed to query latest checkpoint for trial")
	case latestCheckpoint == nil:
		latestCheckpoint = t.warmStartCheckpoint
	default:
		latestBatch = latestCheckpoint.TotalBatches
	}

	trialSpec := &tasks.TrialSpec{
		Base: *t.taskSpec,

		ExperimentID:     t.experimentID,
		TrialID:          t.id,
		TrialRunID:       t.runID,
		ExperimentConfig: schemas.Copy(t.config).(expconf.ExperimentConfig),
		ModelDefinition:  t.modelDefinition,
		HParams:          t.searcher.Create.Hparams,
		TrialSeed:        t.searcher.Create.TrialSeed,
		LatestBatch:      latestBatch,
		LatestCheckpoint: latestCheckpoint,
		IsMultiAgent:     len(t.allocations) > 1,
	}

	for rank, a := range msg.Allocations {
		a.Start(ctx, trialSpec.ToTaskSpec(t.generatedKeys, taskToken), rank)
	}

	return nil
}

func (t *trial) processContainerRunning(ctx *actor.Context, msg sproto.TaskContainerStateChanged) {
	t.rendezvous.containerStarted(msg.Container.ID, msg.ContainerStarted.Addresses)
	if t.rendezvous.ready() {
		ctx.Log().Info("all containers are connected successfully (task container state changed)")
	}
}

func (t *trial) processContainerTerminated(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	t.insertLog(ctx, &msg.Container.ID, msg.ContainerStopped.String())

	t.terminatedContainers[msg.Container.ID] = *msg.ContainerStopped
	t.rendezvous.containerTerminated(msg.Container.ID)
	if t.terminatedFirst == nil {
		t.terminatedFirst = &msg.Container.ID
	}

	switch {
	case msg.ContainerStopped.Failure != nil:
		return t.terminate(ctx, kill)
	default:
		return t.terminate(ctx, noop)
	}
}

type terminationType string

const (
	// kill is used to forcibly halt a trial. calling this will kill existing allocations
	// and exit. terminate is re-entered after a kill when all containers have stopped.
	kill terminationType = "kill"
	// preempt is used to gracefully halt a trial. calling this will (usually, with the exception
	// of unready trials) send a preemption signal to all watchers and begin a timeout after which
	// we forcibly kill the trial.
	preempt terminationType = "preempt"
	// noop is used to try to move a trial to a terminal state while taking no direct action on it.
	// e.g., if the searcher tells us it's done, we either should exit right away if we're unallocated,
	// or just chill and wait for the active task to exit.
	noop terminationType = "noop"
)

// terminate encapsulates all termination logic for the trial. All _controlled_ termination paths
// MUST go through this function, though exception paths (panics, DB errors, network calls, etc)
// can terminate by just returning an error and letting the resource manager cleanup after the actor
// dies.
//
// It just exists to translate caller desires "kill this trial, preempt this trial" to the
// corresponding action to actually take based on our state, instead of each caller needing
// to be aware of how to take certain actions in certain states.
func (t *trial) terminate(ctx *actor.Context, tt terminationType) error {
	switch {
	case t.task == nil:
		ctx.Log().Info("terminating trial before resources are requested")
		return t.terminated(ctx)
	case len(t.allocations) == 0:
		ctx.Log().Info("terminating trial before resources are allocated")
		return t.terminated(ctx)
	case len(t.allocations) == len(t.terminatedContainers):
		ctx.Log().Info("terminating trial because all containers have exited")
		return t.terminated(ctx)
	case tt == noop, tt == preempt && len(t.terminatedContainers) > 0:
		// Working on it.
	case tt == preempt && t.rendezvous.ready():
		ctx.Log().Info("gracefully terminating trial")
		t.preemption.preempt()
		ctx.Tell(ctx.Self(), preemptionTimeout{t.task.ID})
	case tt == preempt:
		t.preemption.markUnacknowledgeable()
		fallthrough
	default:
		if t.killCooldown != nil && time.Now().UTC().Before(*t.killCooldown) {
			ctx.Log().Debug("still inside of kill cooldown")
			return nil
		}

		ctx.Log().Info("forcibly terminating trial")
		t.killCooldown = ptrs.TimePtr(time.Now().UTC().Add(killCooldown))
		for _, allocation := range t.allocations {
			allocation.Kill(ctx)
		}
	}
	return nil
}

// terminated decides what action to take to close or restart a trial. This is only
// called once the current task is cleaned up and we're ready to move on.
func (t *trial) terminated(ctx *actor.Context) error {
	switch status := t.taskExitStatus(); {
	case t.searcher.Finished():
		t.stop(ctx, model.CompletedState)
	case model.StoppingStates[t.targetState]:
		t.stop(ctx, model.StoppingToTerminalStates[t.targetState])
	case status.Failure != nil:
		switch status.Failure.FailureType {
		case aproto.ContainerFailed, aproto.TaskError:
			ctx.Log().
				WithError(status.Failure).
				Errorf("trial failed (restart %d/%d)", t.restarts, t.config.MaxRestarts())
			t.restarts++
			if err := t.db.UpdateTrialRestarts(t.id, t.restarts); err != nil {
				return errors.Wrap(err, "saving restart count")
			}
			if t.restarts > t.config.MaxRestarts() {
				t.stop(ctx, model.ErrorState)
				ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
					trialID: t.id,
					reason:  workload.Errored,
				})
			}
		case aproto.AgentError, aproto.AgentFailed:
			// Questionable, could be considered failures.
		case aproto.TaskAborted:
			// Definitely not a failure.
		}
	case t.preemption.acknowledged(), t.preemption.unacknowledgeable():
	default:
		t.stop(ctx, model.CompletedState)
		ctx.Tell(ctx.Self().Parent(), trialReportEarlyExit{
			trialID: t.id,
			reason:  workload.UserCanceled,
		})
	}
	ctx.Log().
		WithField("search_finished", t.searcher.Finished()).
		WithField("state", t.targetState).
		WithField("preempted", t.preemption.acknowledged() || t.preemption.unacknowledgeable()).
		Info("trial terminated")

	if err := t.resetTask(ctx); err != nil {
		return errors.Wrap(err, "failed to reset task")
	}
	if err := t.maybeAllocate(ctx); err != nil {
		return errors.Wrap(err, "failed to reschedule trial")
	}
	return nil
}

func (t *trial) stop(ctx *actor.Context, state model.State) {
	if t.stopped {
		return
	}

	t.stopped = true
	t.finalState = state
	ctx.Self().Stop()
}

func (t *trial) resetTask(ctx *actor.Context) error {
	var mErr *multierror.Error

	ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})

	if t.task != nil {
		if err := t.db.DeleteTaskSessionByTaskID(string(t.task.ID)); err != nil {
			mErr = multierror.Append(mErr, errors.Wrap(err, "error delete task session for a trial"))
		}
	}

	if t.task != nil && len(t.allocations) != 0 {
		if err := t.db.CompleteTask(model.JobTypeTrial, t.id, t.task.ID); err != nil {
			mErr = multierror.Append(mErr, errors.Wrap(err, "failed to mark trial run completed"))
		}
	}

	t.preemption.close()
	t.preemption = nil
	t.rendezvous.close()
	t.rendezvous = nil
	t.task = nil
	t.allocations = nil
	t.terminatedFirst = nil
	t.terminatedContainers = make(map[cproto.ID]sproto.TaskContainerStopped)
	t.killCooldown = nil

	return mErr.ErrorOrNil()
}

func (t *trial) taskExitStatus() aproto.ContainerStopped {
	anyStarted := func(cs map[cproto.ID]cproto.Container) bool {
		for _, c := range cs {
			if c.State != cproto.Assigned {
				return true
			}
		}
		return false
	}

	if !anyStarted(t.containers) {
		return aproto.ContainerError(aproto.TaskAborted, errors.New("task aborted"))
	}
	if t.terminatedFirst != nil {
		return t.terminatedContainers[*t.terminatedFirst].ContainerStopped
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

const (
	// MinLocalRendezvousPort is the smallest port to use (from the container's point of view;
	// it will be mapped to some arbitrary port on the host) for communication across containers.
	MinLocalRendezvousPort = 1734

	// MaxLocalRendezvousPort is the largest port to use for communication across containers.
	// Each distributed trial can take up to 2 host based ports and we assume a maximum.
	// of 16 slot per agent. MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1.
	MaxLocalRendezvousPort = MinLocalRendezvousPort + 2*16 - 1
)

var rendezvousTimeoutDuration = 10 * time.Minute

type (
	// watchRendezvousInfo begins watching for rendezvous info.
	// When all the containers are ready, the trial will send all the
	// peer addresses on the channel in the response.
	watchRendezvousInfo struct {
		taskID sproto.TaskID
		id     cproto.ID
	}
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
	rendezvousTimeout struct{ taskID sproto.TaskID }

	// rendezvous encapsulates the rendezvous state of a trial.
	rendezvous struct {
		taskID            sproto.TaskID
		watchers          map[cproto.ID]chan<- rendezvousInfoOrError
		ranks             map[cproto.ID]int
		addresses         map[cproto.ID][]cproto.Address
		lastWatchTime     time.Time
		allReadySucceeded bool
	}
)

func newRendezvous(taskID sproto.TaskID, ranks map[cproto.ID]int) *rendezvous {
	return &rendezvous{
		taskID:    taskID,
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

func (r *rendezvous) watch(taskID sproto.TaskID, id cproto.ID) (rendezvousWatcher, error) {
	if r == nil {
		err := errNoTask{action: "watch rendezvous"}
		return rendezvousWatcher{}, apiutils.AsValidationError(err.Error())
	} else if r.taskID != taskID {
		err := errStaleTask{received: taskID, actual: r.taskID}
		return rendezvousWatcher{}, apiutils.AsValidationError(err.Error())
	} else if _, ok := r.ranks[id]; !ok {
		err := errStaleContainer{id: id}
		return rendezvousWatcher{}, apiutils.AsValidationError(err.Error())
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
func (r *rendezvous) checkTimeout(taskID sproto.TaskID) error {
	if r == nil {
		return nil
	}

	if r.taskID == taskID && time.Now().After(r.lastWatchTime.Add(rendezvousTimeoutDuration)) {
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
	if r == nil {
		return
	}

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
	errNoPreemptionStatus     = errors.New("no preemption status available for unallocated task")
)

type (
	// watchPreemption begins watching if the task has been preempted.
	// The task responds to this message with a channel of bools, where sends of true
	// indicate to preempt and sends of false are used to synchronize (e.g. you want to
	// block until you receive _something_ but not until the first preemption).
	watchPreemption struct {
		taskID sproto.TaskID
		id     uuid.UUID
	}
	preemptionWatcher struct{ C <-chan struct{} }
	unwatchPreemption struct{ id uuid.UUID }
	ackPreemption     struct{ taskID sproto.TaskID }
	// preemptionTimeout is the time after which we forcibly terminate a trial that has no
	// preempted.
	preemptionTimeout struct{ taskID sproto.TaskID }

	// preemption represents the preemption status of a task. A task is assumed to be preempted
	// exactly one time. The object is "nil safe" - it'll gracefully handle calls on a nil
	// preemption. This is nice until we move to trial has many task actors / generic task actor,
	// where the lifetime of a "preemption" is equivalent to the lifetime of task and they can be
	// initialized together.
	preemption struct {
		taskID      sproto.TaskID
		preempted   bool
		acked       bool
		unackable   bool
		preemptedAt time.Time
		// Map of watcher ID to a bool indicating if the trial should preempt.
		watchers map[uuid.UUID]chan<- struct{}
	}
)

func newPreemption(taskID sproto.TaskID) *preemption {
	return &preemption{
		taskID:    taskID,
		preempted: false,
		acked:     false,
		watchers:  map[uuid.UUID]chan<- struct{}{},
	}
}

func (p *preemption) watch(taskID sproto.TaskID, id uuid.UUID) (preemptionWatcher, error) {
	if p == nil {
		return preemptionWatcher{}, errNoPreemptionStatus
	}
	if p.taskID != taskID {
		return preemptionWatcher{}, errStaleTask{received: taskID, actual: p.taskID}
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

func (p *preemption) acknowledge(taskID sproto.TaskID) error {
	if p == nil {
		return errNoPreemptionStatus
	}
	if p.taskID != taskID {
		return errStaleTask{received: taskID, actual: p.taskID}
	}

	p.acked = true
	return nil
}

func (p *preemption) acknowledged() bool {
	if p == nil {
		return false
	}

	return p.acked
}

// markUnacknowledgeable marks that we _were_ going to preempt a trial but
// decided instead to kill it, so it's all gravy if it throws some errors.
func (p *preemption) markUnacknowledgeable() bool {
	if p == nil {
		return false
	}

	return p.unackable
}

func (p *preemption) unacknowledgeable() bool {
	if p == nil {
		return false
	}

	return p.unackable
}

func (p *preemption) checkTimeout(taskID sproto.TaskID) error {
	if p == nil {
		return nil
	}
	if p.taskID != taskID {
		return nil
	}

	if time.Now().After(p.preemptedAt.Add(preemptionTimeoutDuration)) {
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

type errNoTask struct {
	action string
}

func (e errNoTask) Error() string {
	return fmt.Sprintf("%s not valid without active task", e.action)
}

type errStaleTask struct {
	received, actual sproto.TaskID
}

func (e errStaleTask) Error() string {
	return fmt.Sprintf("stale task %s != %s (received != actual)", e.received, e.actual)
}

type errStaleContainer struct {
	id cproto.ID
}

func (e errStaleContainer) Error() string {
	return fmt.Sprintf("stale container %s", e.id)
}
