package internal

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/workload"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	cproto "github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/searcher"
	"github.com/determined-ai/determined/master/pkg/ssh"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/master/pkg/union"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	allReadyTimeoutPeriod  = 10 * time.Minute
	terminateTimeoutPeriod = time.Minute
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
	trialAborted struct{}

	// This message is used to synchronize the trial workload sequencer with the searcher. It allows
	// the searcher to get more operations to the trial workload sequencer as a result of the trial
	// completing a searcher operation before the trial decides to tell the scheduler it is
	// done, since stopping and restarting trials has relatively high overhead.
	sendNextWorkload struct {
		runID int
	}

	// It is possible that it takes very long for all containers to be connected after the first
	// container is connected. This might happen when the k8s cluster waits for new instances
	// to spin up, which might not happen at all. At the same time, taking up part of all
	// the resources and waiting is wasteful. So we need to detect this situation.
	allReadyTimeout struct {
		runID int
	}

	// When we issue a TERMINATE workload, we send a delayed terminateTimeout message with a record
	// of the number of RunID that the trial had at the time we issued the TERMINATE. If we
	// receive the terminateTimeout message and t.RunID has not changed, we forcibly kill the
	// running containers.
	terminateTimeout struct{ runID int }

	containerConnected struct {
		ContainerID cproto.ID
		socket      *websocket.Conn
	}
)

// Trial-specific external messages.
type trialMessage struct {
	RendezvousInfo *rendezvousInfoMessage `union:"type,RENDEZVOUS_INFO" json:"-"`
	RunWorkload    *runWorkload           `union:"type,RUN_WORKLOAD" json:"-"`
}

func (m trialMessage) MarshalJSON() ([]byte, error) {
	return union.Marshal(m)
}

func (m *trialMessage) UnmarshalJSON(data []byte) error {
	if err := union.Unmarshal(data, m); err != nil {
		return err
	}

	type DefaultParser *trialMessage

	return errors.Wrap(json.Unmarshal(data, DefaultParser(m)), "failed to parse trial message")
}

type rendezvousInfoMessage struct {
	// Addrs is deprecated in favor of Containers.
	Addrs []string `json:"addrs"`
	// Addrs2 is deprecated in favor of Containers.
	Addrs2 []string `json:"addrs2"`

	Rank int `json:"rank"`

	// Containers contains rendezvous information for each container.
	Containers []*rendezvousContainer `json:"containers"`
}

type rendezvousContainer struct {
	Addresses []*rendezvousAddress `json:"addresses"`
}

type rendezvousAddress struct {
	ContainerPort int    `json:"container_port"`
	ContainerIP   string `json:"container_ip"`
	HostPort      int    `json:"host_port"`
	HostIP        string `json:"host_ip"`
}

type runWorkload struct {
	Workload workload.Workload `json:"workload"`
}

// terminatedContainerWithState records the terminatedContainer message with some state about the
// trial at the time termination was received. That information is analyzed when determining if a
// trial should be considered to have errored or not.
type terminatedContainerWithState struct {
	exitStatus                 sproto.TaskContainerStopped
	isLeader                   bool
	pendingGracefulTermination bool
	needsCheckpoint            bool
}

type (
	// trialState keeps all the state of the trial that we persist between failures.
	trialState struct {
		TrialWorkloadSequencerState json.RawMessage `json:"trial_workload_sequencer_state"`

		// restarts is essentially a failure count, it increments when the trial fails and we retry it.
		Restarts int `json:"restarts"`

		// RunID is a count of how many times the task container(s) have stopped and restarted, which
		// could be due to a failure or due to normal pausing and continuing. When RunID increments,
		// it effectively invalidates any outstanding terminateTimeout messages so that we don't
		// accidentally kill a fresh container due to the terminateTimeout message from an older
		// container.
		RunID int `json:"run_id"`

		// The following fields tracks the reasons for termination.
		EarlyExit                  bool `json:"early_exit"`
		PendingGracefulTermination bool `json:"pending_graceful_termination"`
		TerminationSent            bool `json:"termination_sent"`
		CancelUnready              bool `json:"cancel_unready"`
		Killed                     bool `json:"killed"`
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
	trial struct {
		trialState
		id           int
		idSet        bool
		replayCreate bool

		rm              *actor.Ref
		logger          *actor.Ref
		db              *db.PgDB
		experimentState model.State
		experiment      *model.Experiment
		modelDefinition archive.Archive

		warmStartCheckpointID *int

		create searcher.Create
		close  *searcher.Close

		sequencer *trialWorkloadSequencer

		// The following fields tracks the interaction with the resource providers.
		task        *sproto.AllocateRequest
		allocations []sproto.Allocation

		// The following fields tracks containers and their states.
		lastContainerConnectedTime time.Time
		startedContainers          map[cproto.ID]bool
		containers                 map[cproto.ID]cproto.Container // only for running containers.
		containerRanks             map[cproto.ID]int              // only for launched containers.
		containerAddresses         map[cproto.ID][]cproto.Address // only for running containers.
		containerSockets           map[cproto.ID]*actor.Ref       // only for running containers.
		terminatedContainers       map[cproto.ID]terminatedContainerWithState
		// tracks if allReady check has passed successfully.
		allReadySucceeded bool

		agentUserGroup *model.AgentUserGroup
		taskSpec       *tasks.TaskSpec
		privateKey     []byte
		publicKey      []byte
	}
)

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
// It must only error when a snapshot is passed.
func newTrial(
	exp *experiment,
	create searcher.Create,
	firstCheckpoint *model.Checkpoint,
) *trial {
	var warmStartCheckpointID *int
	if firstCheckpoint != nil {
		checkpointID := firstCheckpoint.ID
		warmStartCheckpointID = &checkpointID
	}
	return &trial{
		rm:                    exp.rm,
		logger:                exp.trialLogger,
		db:                    exp.db,
		experimentState:       exp.State,
		experiment:            exp.Experiment,
		modelDefinition:       exp.modelDefinition,
		warmStartCheckpointID: warmStartCheckpointID,

		sequencer: newTrialWorkloadSequencer(exp.Experiment, create, firstCheckpoint),

		create: create,

		startedContainers:    make(map[cproto.ID]bool),
		containers:           make(map[cproto.ID]cproto.Container),
		containerRanks:       make(map[cproto.ID]int),
		containerAddresses:   make(map[cproto.ID][]cproto.Address),
		containerSockets:     make(map[cproto.ID]*actor.Ref),
		terminatedContainers: make(map[cproto.ID]terminatedContainerWithState),

		agentUserGroup: exp.agentUserGroup,
		taskSpec:       exp.taskSpec,
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("experiment-id", t.experiment.ID)
		if t.idSet {
			ctx.AddLabel("trial-id", t.id)
		}
		if t.replayCreate {
			if err := t.tellWithSnapshot(ctx, ctx.Self().Parent(), func(s trialSnapshot) interface{} {
				return trialCreated{create: t.create, trialSnapshot: s}
			}); err != nil {
				ctx.Log().WithError(err).Warn("failed to snapshot replay create (trial may fail to restore)")
			}
		}

	case model.State:
		t.experimentState = msg
	case []searcher.Operation:
		t.processOperations(msg)

	case sproto.ContainerLog:
		t.insertLog(ctx, msg.Container, msg.Message())

	case trialAborted:
		// This is to handle trial being aborted. It does nothing here but requires
		// the code below this switch statement to handle releasing resources in
		// the scheduler. This should be refactored into the terminating logic.

	case actor.PostStop:
		if !t.idSet {
			return nil
		}
		if t.Restarts > t.experiment.Config.MaxRestarts {
			if err := t.db.UpdateTrial(t.id, model.ErrorState); err != nil {
				ctx.Log().Error(err)
			}
			return errors.Errorf("trial %d failed and reached maximum number of restarts", t.id)
		}
		ctx.Log().Info("trial stopped successfully")
		endState := model.CompletedState
		if t.experimentState == model.StoppingCanceledState || t.Killed {
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

	if t.task == nil {
		if t.trialClosing() {
			ctx.Self().Stop()
		} else if !t.sequencer.UpToDate() && t.experimentState == model.ActiveState {
			slotsNeeded := t.experiment.Config.Resources.SlotsPerTrial
			label := t.experiment.Config.Resources.AgentLabel
			resourcePool := t.experiment.Config.Resources.ResourcePool
			var name string
			if t.idSet {
				name = fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experiment.ID)
			} else {
				name = fmt.Sprintf("Trial (Experiment %d)", t.experiment.ID)
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
	} else if t.experimentState != model.ActiveState {
		_ = t.releaseResource(ctx)
	}

	return nil
}

func (t *trial) processOperations(ops []searcher.Operation) {
	for _, op := range ops {
		switch op := op.(type) {
		case searcher.Runnable:
			t.sequencer.OperationRequested(op)
		case searcher.Close:
			t.close = &op
		}
	}
}

func (t *trial) runningReceive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.ResourcesAllocated, sproto.ReleaseResources:
		return t.processSchedulerMsg(ctx)

	case containerConnected, sproto.TaskContainerStateChanged:
		return t.processContainerMsg(ctx)

	case *websocket.Conn, *apiv1.KillTrialRequest:
		return t.processAPIMsg(ctx)

	case workload.CompletedMessage:
		if err := t.processCompletedWorkload(ctx, msg); err != nil {
			return err
		}

	case sendNextWorkload:
		if msg.runID != t.RunID {
			ctx.Log().Warnf("ignoring sendNextWorkload with stale RunID %d", msg.runID)
			return nil
		}
		if err := t.sendNextWorkload(ctx); err != nil {
			return err
		}

	case actor.ChildFailed:
		ctx.Log().Info("found child actor failed, terminating forcibly")
		t.terminate(ctx, true)

	case killTrial:
		ctx.Log().Info("received killing request")
		t.Killed = true
		t.terminate(ctx, true)

	case allReadyTimeout:
		if msg.runID == t.RunID &&
			time.Now().After(t.lastContainerConnectedTime.Add(allReadyTimeoutPeriod)) {
			ctx.Tell(t.logger, model.TrialLog{
				TrialID: t.id, Message: "some containers are taking a long time to " +
					"connect to master; when running on kubernetes this may happen " +
					"because only some of the pods have been scheduled; it is possible " +
					"that some pods will never be scheduled without adding compute " +
					"resources or pausing / killing other experiments in the cluster",
			})
		}

	case terminateTimeout:
		if msg.runID == t.RunID {
			ctx.Log().Info("forcibly terminating unresponsive trial after timeout expired")
			t.terminate(ctx, true)
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
	if !t.allReady(ctx) {
		t.CancelUnready = true
		t.terminate(ctx, true)
	} else {
		t.terminate(ctx, false)
	}
	return nil
}

func (t *trial) processContainerMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case containerConnected:
		if err := t.processContainerConnected(ctx, msg); err != nil {
			return err
		}

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

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processAPIMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *websocket.Conn:
		a := api.WrapSocket(msg, workload.CompletedMessage{}, false)
		if ref, created := ctx.ActorOf("socket", a); created {
			ctx.Respond(ref)
		}

	case *apiv1.KillTrialRequest:
		ctx.Log().Info("received API request to kill trial")
		t.Killed = true
		t.terminate(ctx, true)
		ctx.Respond(&apiv1.KillTrialResponse{})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processID(id int) {
	t.id = id
	t.idSet = true
	t.sequencer.SetTrialID(id)
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
			t.create.RequestID,
			t.experiment.ID,
			model.JSONObj(t.create.Hparams),
			t.warmStartCheckpointID,
			int64(t.create.TrialSeed))
		if err := t.db.AddTrial(modelTrial); err != nil {
			ctx.Log().WithError(err).Error("failed to save trial to database")
			t.terminate(ctx, true)
			return err
		}
		t.processID(modelTrial.ID)
		ctx.AddLabel("trial-id", t.id)
		if t.experiment.Config.PerformInitialValidation {
			if err := t.db.AddNoOpStep(model.NewNoOpStep(t.id, 0)); err != nil {
				ctx.Log().WithError(err).Error("failed to save zeroth step for initial validation")
				t.terminate(ctx, true)
				return err
			}
		}
		ctx.Tell(t.rm, sproto.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experiment.ID),
			TaskHandler: ctx.Self(),
		})
		// We snapshot here to shorten the window where a trial can exist without the
		// searcher snapshotting the trialCreated message.
		if err := t.tellWithSnapshot(ctx, ctx.Self().Parent(), func(s trialSnapshot) interface{} {
			return trialCreated{create: t.create, trialSnapshot: s}
		}); err != nil {
			return errors.Wrap(err, "failed to snapshot trial after creation")
		}
	}

	w, err := t.sequencer.Workload()
	if err != nil {
		return errors.Wrap(err, "failed to get workload from sequencer after allocation")
	}

	if err = saveWorkload(t.db, w); err != nil {
		ctx.Log().WithError(err).Error("failed to save workload to the database after allocation")
	}

	ctx.Log().Infof("starting trial container: %v", w)

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
	for rank, a := range msg.Allocations {
		t.containerRanks[a.Summary().ID] = rank
		taskSpec := *t.taskSpec
		taskSpec.AgentUserGroup = t.agentUserGroup
		taskSpec.TaskToken = taskToken
		taskSpec.SetInner(&tasks.StartTrial{
			ExperimentConfig:    t.experiment.Config,
			ModelDefinition:     t.modelDefinition,
			HParams:             t.create.Hparams,
			TrialSeed:           t.create.TrialSeed,
			LatestCheckpoint:    t.sequencer.LatestCheckpoint,
			InitialWorkload:     w,
			WorkloadManagerType: t.sequencer.WorkloadManagerType(),
			AdditionalFiles:     additionalFiles,
			IsMultiAgent:        len(t.allocations) > 1,
			Rank:                rank,
		})
		a.Start(ctx, taskSpec)
	}

	return nil
}

func (t *trial) processCompletedWorkload(ctx *actor.Context, msg workload.CompletedMessage) error {
	if msg.ExitedReason == nil || *msg.ExitedReason == workload.UserCanceled ||
		*msg.ExitedReason == workload.InvalidHP {
		if err := markWorkloadCompleted(t.db, msg); err != nil {
			ctx.Log().Error(err)
		}
	}

	ctx.Log().Infof("trial completed workload: %v", msg.Workload)
	out := trialCompletedWorkload{
		completedMessage: msg,
		unitsCompleted:   model.UnitsFromBatches(msg.Workload.NumBatches, t.sequencer.unitContext),
	}

	isBestValidation := ctx.Ask(ctx.Self().Parent(), trialValidation{
		validationMetrics: msg.ValidationMetrics,
	})
	op, metrics, err := t.sequencer.WorkloadCompleted(msg, isBestValidation)
	switch {
	case err != nil:
		return errors.Wrap(err, "failed to pass completed message to sequencer")
	case op != nil:
		out.completedOps = append(out.completedOps, trialCompletedOperation{op: op, metrics: metrics})
	}

	trialErrored := false
	if msg.ExitedReason != nil {
		ctx.Log().Infof("exiting trial early: %s", *msg.ExitedReason)
		t.EarlyExit = true
		out.earlyExit = &trialExitedEarly{trialID: t.id, exitedReason: *msg.ExitedReason}
		if *msg.ExitedReason == workload.Errored {
			trialErrored = true
		}
	}

	if err := t.tellWithSnapshot(ctx, ctx.Self().Parent(), func(s trialSnapshot) interface{} {
		out.trialSnapshot = s
		return out
	}); err != nil {
		return errors.Wrap(err, "failed to send workloadCompleted with snapshot")
	}

	switch {
	case trialErrored:
		return nil
	case out.completedOps != nil:
		// If we completed a searcher operation, synchronize with the searcher to allow it to relay any
		// new operations to the trial. Otherwise just continue the trial immediately.
		ctx.Tell(ctx.Self().Parent(), sendNextWorkload{runID: t.RunID})
		return nil
	default:
		return t.sendNextWorkload(ctx)
	}
}

func (t *trial) sendNextWorkload(ctx *actor.Context) error {
	terminateNow := false
	var w workload.Workload
	var err error
	switch {
	// We have another workload to run.
	case !t.PendingGracefulTermination && !t.sequencer.UpToDate():
		w, err = t.sequencer.Workload()
		if err != nil {
			return errors.Wrap(err, "failed to get next workload from sequencer")
		}
		ctx.Log().Infof("continuing trial: %v", w)

	// We have no workloads, but the current step needs to be checkpointed before we can terminate.
	case t.sequencer.PrecloseCheckpointWorkload() != nil:
		w = *t.sequencer.PrecloseCheckpointWorkload()
		if w.Kind != workload.CheckpointModel {
			return errors.New(
				"sequencer.PrecloseCheckpointWorkload() returned a non-checkpoint workload")
		}
		ctx.Log().Infof("checkpointing trial before terminating trial runner: %v", w)

	// We have nothing at all to do, so terminate now.
	default:
		ctx.Log().Info("terminating gracefully because there are no more workloads")
		terminateNow = true
		t.terminate(ctx, false)
	}

	// Command the trial runner to do the thing we decided on (if this is not a replay).
	var msg interface{}
	if terminateNow {
		w = *t.sequencer.TerminateWorkload()
		msg = &trialMessage{
			RunWorkload: &runWorkload{
				Workload: w,
			},
		}

		t.TerminationSent = true
		actors.NotifyAfter(ctx, terminateTimeoutPeriod, terminateTimeout{runID: t.RunID})
	} else {
		if err := saveWorkload(t.db, w); err != nil {
			ctx.Log().WithError(err).Error("failed to save workload to the database")
		}
		msg = &trialMessage{
			RunWorkload: &runWorkload{
				Workload: w,
			},
		}
	}
	for _, socket := range t.containerSockets {
		if err := api.WriteSocketJSON(ctx, socket, msg); err != nil {
			ctx.Log().WithError(err).Error("cannot write to websocket")
		}
	}
	return nil
}

func (t *trial) processContainerConnected(ctx *actor.Context, msg containerConnected) error {
	t.lastContainerConnectedTime = time.Now()
	if len(t.containers) < len(t.allocations) {
		actors.NotifyAfter(ctx, allReadyTimeoutPeriod, allReadyTimeout{runID: t.RunID})
	}

	// Check to make sure this is not a connection from a stale container.
	if _, ok := t.containerRanks[msg.ContainerID]; !ok {
		ctx.Respond(errors.Errorf("socket connection from stale container: %s", msg.ContainerID))
		return nil
	}

	a := api.WrapSocket(msg.socket, workload.CompletedMessage{}, false)
	ref, _ := ctx.ActorOf(fmt.Sprintf("socket-%s", msg.ContainerID), a)
	t.containerSockets[msg.ContainerID] = ref
	ctx.Respond(ref)

	if err := t.pushRendezvous(ctx); err != nil {
		return errors.Wrap(err, "failed to push rendezvous to trial containers")
	}

	return nil
}

func formatAddress(p cproto.Address) string {
	return fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
}

func (t *trial) killAndRemoveSocket(ctx *actor.Context, id cproto.ID) {
	if skt, ok := t.containerSockets[id]; ok {
		addr := skt.Address().Local()
		if ref := ctx.Child(addr); ref != nil {
			if ok := ctx.Kill(addr); !ok {
				ctx.Log().Warnf("failed to kill container socket: %s", id)
			}
		}
		delete(t.containerSockets, id)
	}
}

// allReady returns true if and only if all the containers are reported to be started with the
// ContainerStarted message and their sockets to be connected with the containerConnected
// message. The two messages are not guaranteed to come in-order. During each run of the
// trial, once all the containers are ready this function will return true afterward because this
// function is used in deciding if the trial should be forcibly killed when terminating.
func (t *trial) allReady(ctx *actor.Context) bool {
	// If a trial has passed allReady it can never return to a state of not ready until the
	// current containers are all terminated.
	if t.allReadySucceeded {
		return true
	}

	// Ensure all ContainerStarted messages have arrived.
	if len(t.containers) < len(t.allocations) {
		return false
	}

	// Finally, ensure all sockets have connected.
	t.allReadySucceeded = len(t.containerSockets) == len(t.allocations)
	return t.allReadySucceeded
}

// pushRendezvous gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (t *trial) pushRendezvous(ctx *actor.Context) error {
	ctx.Log().Info("pushing rendezvous information")
	if !t.allReady(ctx) {
		ctx.Log().Info("found not all containers are connected")
		return nil
	}
	ctx.Log().Info("found all containers are connected successfully")

	type CAddress struct {
		Container cproto.Container
		Addresses []cproto.Address
		Ordinal   int
	}

	var caddrs []CAddress
	for k, v := range t.containers {
		caddr := CAddress{
			Container: v,
			Addresses: t.containerAddresses[k],
			Ordinal:   t.containerRanks[k],
		}
		caddrs = append(caddrs, caddr)

		sort.Slice(caddr.Addresses, func(i, j int) bool {
			a := caddr.Addresses[i]
			b := caddr.Addresses[j]

			return a.ContainerPort < b.ContainerPort
		})
	}

	sort.Slice(caddrs, func(i, j int) bool {
		a := caddrs[i]
		b := caddrs[j]
		switch {
		case a.Ordinal == 0 && b.Ordinal != 0:
			return true
		case a.Ordinal != 0 && b.Ordinal == 0:
			return false
		default:
			return a.Container.ID < b.Container.ID
		}
	})

	var rcontainers []*rendezvousContainer
	var addrs1 []string
	var addrs2 []string
	for _, caddr := range caddrs {
		var addresses []*rendezvousAddress

		var addrs []cproto.Address
		for _, addr := range caddr.Addresses {
			if MinLocalRendezvousPort <= addr.ContainerPort && addr.ContainerPort <= MaxLocalRendezvousPort {
				addrs = append(addrs, addr)
			}

			addresses = append(addresses, &rendezvousAddress{
				ContainerPort: addr.ContainerPort,
				ContainerIP:   addr.ContainerIP,
				HostPort:      addr.HostPort,
				HostIP:        addr.HostIP,
			})
		}

		if numAddrs := len(addrs); numAddrs == 2 {
			addrs1 = append(addrs1, formatAddress(addrs[0]))
			addrs2 = append(addrs2, formatAddress(addrs[1]))
		} else {
			ctx.Log().Errorf(
				"found %d rendezvous addresses instead of 2 for container %s; dropping rendezvous addresses",
				numAddrs, caddr.Container.ID)
		}

		rcontainers = append(rcontainers, &rendezvousContainer{
			Addresses: addresses,
		})
	}

	for _, caddr := range caddrs {
		c := caddr.Container
		socket := t.containerSockets[c.ID]

		if err := api.WriteSocketJSON(ctx, socket, &trialMessage{
			RendezvousInfo: &rendezvousInfoMessage{
				Addrs:      addrs1,
				Addrs2:     addrs2,
				Rank:       caddr.Ordinal,
				Containers: rcontainers,
			},
		}); err != nil {
			ctx.Log().WithError(err).Error("cannot write to socket")
		}
	}

	return nil
}

func (t *trial) processContainerRunning(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) error {
	ctx.Log().Infof("found container running: %s (rank %d)",
		msg.Container.ID, t.containerRanks[msg.Container.ID])

	t.containers[msg.Container.ID] = msg.Container
	t.containerAddresses[msg.Container.ID] = msg.ContainerStarted.Addresses
	if err := t.pushRendezvous(ctx); err != nil {
		return errors.Wrap(err, "failed to push rendezvous to trial containers")
	}
	return nil
}

func (t *trial) processContainerTerminated(
	ctx *actor.Context, msg sproto.TaskContainerStateChanged,
) {
	ctx.Log().Infof("found container terminated: %s", msg.Container.ID)
	t.terminatedContainers[msg.Container.ID] = terminatedContainerWithState{
		exitStatus:                 *msg.ContainerStopped,
		isLeader:                   t.containerRanks[msg.Container.ID] == 0,
		pendingGracefulTermination: t.PendingGracefulTermination,
		needsCheckpoint:            t.sequencer.PrecloseCheckpointWorkload() != nil,
	}

	_, ok := t.containers[msg.Container.ID]
	delete(t.containers, msg.Container.ID)
	delete(t.containerAddresses, msg.Container.ID)

	t.killAndRemoveSocket(ctx, msg.Container.ID)

	exitMsg := msg.ContainerStopped.String()
	t.insertLog(ctx, msg.Container, exitMsg)

	// Terminate the task if the container never started (since this prevents the gang
	// from ever being able to start), if the leader of the gang has exited out, or if
	// one of the containers exited with a failure.
	if !ok || t.containerRanks[msg.Container.ID] == 0 || msg.ContainerStopped.Failure != nil {
		t.terminate(ctx, true)
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

func classifyStatus(state terminatedContainerWithState) aproto.ContainerStopped {
	switch status := state.exitStatus; {
	case status.Failure != nil && status.Failure.FailureType != aproto.TaskAborted:
		return status.ContainerStopped
	case !state.pendingGracefulTermination || state.needsCheckpoint:
		return aproto.ContainerError(aproto.AgentError, errors.New(
			"container exited when it wasn't supposed to"))
	default:
		return status.ContainerStopped
	}
}

func (t *trial) reset() error {
	totalBatches, err := t.sequencer.RollBackSequencer()
	if err != nil {
		return errors.Wrap(err, "failed to rollback trial sequencer in reset")
	}
	err = t.db.RollBackTrial(t.id, totalBatches)
	if err != nil {
		return errors.Wrap(err, "failed to rollback trial in reset")
	}
	return nil
}

func (t *trial) trialClosing() bool {
	return t.EarlyExit || t.Killed || t.Restarts > t.experiment.Config.MaxRestarts ||
		(t.close != nil && t.sequencer.UpToDate()) ||
		model.StoppingStates[t.experimentState]
}

func (t *trial) terminate(ctx *actor.Context, kill bool) {
	switch {
	case len(t.allocations) == 0:
		ctx.Log().Info("aborting trial before resources are allocated")
		t.terminated(ctx)
		ctx.Tell(ctx.Self(), trialAborted{})
	case kill:
		ctx.Log().Info("forcibly terminating trial")
		if t.task != nil && t.allocations != nil {
			for _, allocation := range t.allocations {
				allocation.Kill(ctx)
			}
		}
	case !t.PendingGracefulTermination:
		ctx.Log().Info("gracefully terminating trial")
		t.PendingGracefulTermination = true
	}
}

// terminated handles errors and restarting for trials when they are failed, paused, canceled,
// or killed.
func (t *trial) terminated(ctx *actor.Context) {
	// Collect container terminated states.
	getLeaderState := func() (terminatedContainerWithState, bool) {
		for _, c := range t.terminatedContainers {
			if c.isLeader {
				return c, true
			}
		}
		return terminatedContainerWithState{}, false
	}
	status := aproto.ContainerError(aproto.AgentError, errors.New("no error status provided"))
	if len(t.startedContainers) == 0 {
		// If there are no containers started executing, consider as aborted.
		// The trial state will be restart.
		status = aproto.ContainerError(aproto.TaskAborted, errors.New("task aborted"))
	} else if leaderState, ok := getLeaderState(); ok {
		status = classifyStatus(leaderState)
	}

	terminationSent := t.TerminationSent

	t.RunID++

	if t.task != nil {
		if err := t.db.DeleteTaskSessionByTaskID(string(t.task.ID)); err != nil {
			ctx.Log().WithError(err).Error("error delete task session for a trial")
		}
	}
	t.task = nil
	t.allocations = nil
	t.containerRanks = make(map[cproto.ID]int)
	ctx.Tell(t.rm, sproto.ResourcesReleased{TaskActor: ctx.Self()})

	t.allReadySucceeded = false
	t.PendingGracefulTermination = false
	t.TerminationSent = false
	t.terminatedContainers = make(map[cproto.ID]terminatedContainerWithState)
	t.startedContainers = make(map[cproto.ID]bool)

	switch {
	case status.Failure == nil:
		ctx.Log().Info("trial runner stopped successfully")
		return
	case terminationSent:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since termination was requested",
		)
		return
	case t.CancelUnready:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since it was canceled or paused " +
				"before all containers are connected",
		)
		return
	case t.Killed:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since it was killed",
		)
		return
	case status.Failure.FailureType == aproto.TaskAborted:
		ctx.Log().Info("trial runner is aborted successfully")
		return
	}

	ctx.Log().Errorf("unexpected failure of trial after restart %d/%d: %v",
		t.Restarts, t.experiment.Config.MaxRestarts, status)
	t.Restarts++
	if t.Restarts <= t.experiment.Config.MaxRestarts {
		ctx.Log().Infof("resetting trial %d", t.id)
		if err := t.reset(); err != nil {
			ctx.Log().Warn("failed to reset trial", err)
		}
		return
	}

	var w workload.Workload
	var err error
	switch {
	case !t.sequencer.UpToDate():
		w, err = t.sequencer.Workload()
		if err != nil {
			panic(err)
		}
	case t.sequencer.PrecloseCheckpointWorkload() != nil:
		w = *t.sequencer.PrecloseCheckpointWorkload()
	default:
		panic("trial terminated due to failure but had nothing to fail")
	}

	if err := markWorkloadErrored(t.db, w); err != nil {
		ctx.Log().
			WithError(err).
			Error("failed to mark workload errored when terminated")
	}

	e := workload.Errored
	erroredMessage := workload.CompletedMessage{
		Workload:     w,
		ExitedReason: &e,
	}
	if err := t.processCompletedWorkload(ctx, erroredMessage); err != nil {
		ctx.Log().
			WithError(err).
			Error("failed to process errored message")
	}
}

func (t *trial) Snapshot() (json.RawMessage, error) {
	sequencerSnapshot, err := t.sequencer.Snapshot()
	if err != nil {
		return nil, errors.Wrap(err, "failed to snapshot trial workload sequencer")
	}
	t.TrialWorkloadSequencerState = sequencerSnapshot
	trialSnapshot, err := json.Marshal(t.trialState)
	return trialSnapshot, errors.Wrap(err, "failed to snapshot trial")
}

func (t *trial) Restore(snapshot json.RawMessage) error {
	if err := json.Unmarshal(snapshot, &t.trialState); err != nil {
		return errors.Wrap(err, "failed to unmarshal trial snapshot")
	}
	if err := t.sequencer.Restore(t.TrialWorkloadSequencerState); err != nil {
		return errors.Wrap(err, "failed to restore trial workload sequencer state")
	}
	if err := t.reset(); err != nil {
		return errors.Wrap(err, "failed to reset trial")
	}
	return nil
}

func (t *trial) tellWithSnapshot(
	ctx *actor.Context, ref *actor.Ref, withSnapshot func(s trialSnapshot) interface{},
) error {
	ss, err := t.Snapshot()
	if err != nil {
		return errors.Wrap(err, "failed to snapshot trial for early exit")
	}
	ctx.Tell(ref, withSnapshot(trialSnapshot{
		requestID: t.create.RequestID,
		trialID:   t.id,
		snapshot:  ss,
	}))
	return nil
}
