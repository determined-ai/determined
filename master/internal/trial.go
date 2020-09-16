package internal

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"github.com/determined-ai/determined/master/pkg/workload"
	"sort"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/container"
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
	restoreTrial struct{}

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
	// of the number of runID that the trial had at the time we issued the TERMINATE. If we
	// receive the terminateTimeout message and t.runID has not changed, we forcibly kill the
	// running containers.
	terminateTimeout struct{ runID int }

	containerConnected struct {
		ContainerID scheduler.ContainerID
		socket      *websocket.Conn
	}
)

// Trial-specific external messages.
type trialMessage struct {
	RendezvousInfo *rendezvousInfoMessage `union:"type,RENDEZVOUS_INFO" json:"-"`
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

// terminatedContainerWithState records the terminatedContainer message with some state about the
// trial at the time termination was received. That information is analyzed when determining if a
// trial should be considered to have errored or not.
type terminatedContainerWithState struct {
	exitStatus                 agent.ContainerStopped
	isLeader                   bool
	pendingGracefulTermination bool
	needsCheckpoint            bool
}

// trial is an actor which is responsible for handling:
//  - messages from the scheduler,
//  - messages from the experiment,
//  - messages from the trial container(s),
//  - replay logic, and
//  - keeping the trial table of the database up-to-date.
//
// It is not responsible for maintaining the current state of the task running in the trial
// container, or the desired state as described by searcher operations; that is offloaded onto the
// workloadSequencer.
type trial struct {
	id    int
	idSet bool

	rp              *actor.Ref
	logger          *actor.Ref
	db              *db.PgDB
	experimentState model.State
	experiment      *model.Experiment
	modelDefinition archive.Archive

	warmStartCheckpointID *int

	create searcher.Create
	close  *searcher.Close

	sequencer *trialWorkloadSequencer

	// restarts is essentially a failure count, it increments when the trial fails and we retry it.
	restarts int

	// runID is a count of how many times the task container(s) have stopped and restarted, which
	// could be due to a failure or due to normal pausing and continuing. When runID increments,
	// it effectively invalidates any outstanding terminateTimeout messages so that we don't
	// accidentally kill a fresh container due to the terminateTimeout message from an older
	// container.
	runID int

	replaying bool
	earlyExit bool
	killed    bool

	task                       *scheduler.Task
	pendingGracefulTermination bool
	terminationSent            bool
	cancelUnready              bool

	privateKey []byte
	publicKey  []byte

	// numContainers is the number of containers that the scheduler has most recently assigned to run
	// this trial.
	numContainers              int
	startedContainers          map[container.ID]bool
	containers                 map[scheduler.ContainerID]scheduler.Container
	terminatedContainers       []terminatedContainerWithState
	lastContainerConnectedTime time.Time

	// sockets maps each running container for this trial to the corresponding websocket actor.
	sockets map[scheduler.ContainerID]*actor.Ref

	agentUserGroup        *model.AgentUserGroup
	taskContainerDefaults *model.TaskContainerDefaultsConfig
}

// newTrial creates a trial which will try to schedule itself after it receives its first workload.
func newTrial(
	exp *experiment,
	create searcher.Create,
	firstCheckpoint *model.Checkpoint,
) actor.Actor {
	var warmStartCheckpointID *int
	if firstCheckpoint != nil {
		checkpointID := firstCheckpoint.ID
		warmStartCheckpointID = &checkpointID
	}
	return &trial{
		rp:                    exp.rp,
		logger:                exp.trialLogger,
		db:                    exp.db,
		experimentState:       exp.State,
		experiment:            exp.Experiment,
		modelDefinition:       exp.modelDefinition,
		warmStartCheckpointID: warmStartCheckpointID,

		sequencer: newTrialWorkloadSequencer(exp.Experiment, create, firstCheckpoint),

		create:    create,
		replaying: exp.replaying,

		startedContainers: make(map[container.ID]bool),
		containers:        make(map[scheduler.ContainerID]scheduler.Container),
		sockets:           make(map[scheduler.ContainerID]*actor.Ref),

		agentUserGroup:        exp.agentUserGroup,
		taskContainerDefaults: exp.taskContainerDefaults,
	}
}

func (t *trial) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.AddLabel("experiment-id", t.experiment.ID)

	case model.State:
		t.experimentState = msg
	case []searcher.Operation:
		for _, operation := range msg {
			switch op := operation.(type) {
			case searcher.Runnable:
				if err := t.sequencer.OperationRequested(op); err != nil {
					return errors.Wrap(err, "error passing runnable to sequencer")
				}
			case searcher.Close:
				t.close = &op
			}
		}

	// Restoration-related messages.
	case trialCreated:
		t.processID(ctx, msg.trialID)
		ctx.Tell(ctx.Self().Parent(), msg)
	case restoreTrial:
		t.restore(ctx)
		t.replaying = false

	case sproto.ContainerLog:
		t.processLog(ctx, msg)

	case actor.PostStop:
		if !t.idSet {
			return nil
		}
		if t.restarts > t.experiment.Config.MaxRestarts {
			if !t.replaying {
				if err := t.db.UpdateTrial(t.id, model.ErrorState); err != nil {
					ctx.Log().Error(err)
				}
			}
			return errors.Errorf("trial %d failed and reached maximum number of restarts", t.id)
		}
		ctx.Log().Info("trial stopped successfully")
		endState := model.CompletedState
		if t.experimentState == model.StoppingCanceledState || t.killed {
			endState = model.CanceledState
		}
		if !t.replaying {
			if err := t.db.UpdateTrial(t.id, endState); err != nil {
				ctx.Log().Error(err)
			}
		}
		return nil
	default:
		if t.task != nil || t.replaying {
			if err := t.runningReceive(ctx); err != nil {
				return err
			}
		}
	}

	if t.task == nil {
		if t.trialClosing() {
			ctx.Self().Stop()
		} else if !t.sequencer.UpToDate() && t.experimentState == model.ActiveState &&
			!t.replaying {
			slotsNeeded := t.experiment.Config.Resources.SlotsPerTrial
			label := t.experiment.Config.Resources.AgentLabel
			name := "Pending Trial"
			if t.idSet {
				name = fmt.Sprintf("Trial %d", t.id)
			}

			t.task = ctx.Ask(t.rp, scheduler.AddTask{
				Name:         fmt.Sprintf("%s (Experiment %d)", name, t.experiment.ID),
				Group:        ctx.Self().Parent(),
				SlotsNeeded:  slotsNeeded,
				CanTerminate: true,
				Label:        label,
				FittingRequirements: scheduler.FittingRequirements{
					SingleAgent: false,
				},
				TaskHandler: ctx.Self(),
			}).Get().(*scheduler.Task)
		}
	} else if t.experimentState != model.ActiveState {
		ctx.Tell(t.rp, scheduler.TerminateTask{TaskID: t.task.ID})
	}

	return nil
}

func (t *trial) runningReceive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case
		scheduler.TaskAssigned,
		scheduler.TerminateRequest,
		scheduler.TaskAborted,
		scheduler.TaskTerminated,
		scheduler.ContainerStarted:
		return t.processSchedulerMsg(ctx)

	case containerConnected, sproto.ContainerStateChanged:
		return t.processContainerMsg(ctx)

	case *websocket.Conn, *apiv1.KillTrialRequest:
		return t.processAPIMsg(ctx)

	case workload.CompletedMessage:
		if err := t.processCompletedWorkload(ctx, msg); err != nil {
			return err
		}

	case sendNextWorkload:
		if msg.runID != t.runID {
			ctx.Log().Warnf("ignoring sendNextWorkload with stale runID %d", msg.runID)
			return nil
		}
		if err := t.sendNextWorkload(ctx); err != nil {
			return err
		}

	case actor.ChildFailed:
		t.terminate(ctx, true)

	case killTrial:
		t.killed = true
		t.terminate(ctx, true)

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

	case terminateTimeout:
		if msg.runID == t.runID {
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
	case scheduler.TaskAssigned:
		if err := t.processAssigned(ctx, msg); err != nil {
			return err
		}

	case scheduler.TerminateRequest:
		if !t.allReady(ctx) {
			t.cancelUnready = true
			t.terminate(ctx, true)
		} else {
			t.terminate(ctx, false)
		}

	case scheduler.TaskAborted:

	case scheduler.TaskTerminated:
		t.processTaskTerminated(ctx, msg)

	case scheduler.ContainerStarted:
		t.containers[msg.Container.ID()] = msg.Container

		if err := t.pushRendezvous(ctx); err != nil {
			return errors.Wrap(err, "failed to push rendezvous to trial containers")
		}

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processContainerMsg(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case containerConnected:
		if err := t.processContainerConnected(ctx, msg); err != nil {
			return err
		}

	case sproto.ContainerStateChanged:
		if msg.Container.State != container.Assigned {
			t.startedContainers[msg.Container.ID] = true
		}
		switch msg.Container.State {
		case container.Terminated:
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
		t.killed = true
		t.terminate(ctx, true)
		ctx.Respond(&apiv1.KillTrialResponse{})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (t *trial) processID(ctx *actor.Context, id int) {
	t.id = id
	t.idSet = true
	t.sequencer.SetTrialID(id)
	ctx.AddLabel("trial-id", id)
}

func (t *trial) processAssigned(ctx *actor.Context, msg scheduler.TaskAssigned) error {
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
			t.experiment.ID,
			model.JSONObj(t.create.Hparams),
			t.warmStartCheckpointID,
			int64(t.create.TrialSeed))
		if err := t.db.AddTrial(modelTrial); err != nil {
			ctx.Log().WithError(err).Error("failed to save trial to database")
			t.terminate(ctx, true)
			return nil
		}
		t.processID(ctx, modelTrial.ID)
		if t.experiment.Config.PerformInitialValidation {
			if err := t.db.AddNoOpStep(model.NewNoOpStep(t.id, 0)); err != nil {
				ctx.Log().WithError(err).Error("failed to save zeroth step for initial validation")
				t.terminate(ctx, true)
				return nil
			}
		}
		ctx.Tell(t.rp, scheduler.SetTaskName{
			Name:        fmt.Sprintf("Trial %d (Experiment %d)", t.id, t.experiment.ID),
			TaskHandler: ctx.Self(),
		})
		ctx.Tell(ctx.Self().Parent(), trialCreated{create: t.create, trialID: t.id})
	}

	// We need to complete cached checkpoints here in the event that between when we last shutdown
	// and now the searcher asked for a checkpoint we already created (this happens in PBT).
	switch op, metrics, err := t.sequencer.CompleteCachedCheckpoints(); {
	case err != nil:
		return errors.Wrap(err, "Error completing cached checkpoints")
	case op != nil:
		ctx.Tell(ctx.Self().Parent(), trialCompletedOperation{t.id, op, metrics})
	}

	w, err := t.sequencer.Workload()
	if err != nil {
		return errors.Wrap(err, "error getting workload from sequencer")
	}

	t.numContainers = len(msg.Assignments)

	if err = saveWorkload(t.db, w); err != nil {
		ctx.Log().WithError(err).Error("failed to save workload to the database after assigned")
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

	for _, a := range msg.Assignments {
		a.StartTask(tasks.TaskSpec{
			StartContainer: &tasks.StartContainer{
				ExperimentConfig:    t.experiment.Config,
				ModelDefinition:     t.modelDefinition,
				HParams:             t.create.Hparams,
				TrialSeed:           t.create.TrialSeed,
				LatestCheckpoint:    t.sequencer.LatestCheckpoint(),
				InitialWorkload:     w,
				WorkloadManagerType: t.sequencer.WorkloadManagerType(),
				AdditionalFiles:     additionalFiles,
				AgentUserGroup:      t.agentUserGroup,
				IsMultiAgent:        t.numContainers > 1,
			},
		})
	}

	return nil
}

func (t *trial) processCompletedWorkload(ctx *actor.Context, msg workload.CompletedMessage) error {
	if !t.replaying && (msg.ExitedReason == nil || *msg.ExitedReason == workload.UserCanceled) {
		if err := markWorkloadCompleted(t.db, msg); err != nil {
			ctx.Log().Error(err)
		}
	}

	ctx.Log().Infof("trial completed workload: %v", msg.Workload)

	completedSearcherOp := false
	units := model.UnitsFromBatches(msg.Workload.NumBatches, t.sequencer.unitContext)
	isBestValidation := ctx.Ask(ctx.Self().Parent(), trialCompletedWorkload{t.id, msg, units})
	op, metrics, err := t.sequencer.WorkloadCompleted(msg, isBestValidation)
	switch {
	case err != nil:
		return errors.Wrap(err, "Error passing completed message to sequencer")
	case op != nil:
		ctx.Tell(ctx.Self().Parent(), trialCompletedOperation{t.id, op, metrics})
		completedSearcherOp = true
	}

	switch op, metrics, err = t.sequencer.CompleteCachedCheckpoints(); {
	case err != nil:
		return errors.Wrap(err, "Error completing cached checkpoints")
	case op != nil:
		ctx.Tell(ctx.Self().Parent(), trialCompletedOperation{t.id, op, metrics})
		completedSearcherOp = true
	}

	if msg.ExitedReason != nil {
		ctx.Log().Info("exiting trial early")
		ctx.Tell(ctx.Self().Parent(), trialExitedEarly{t.id, msg.ExitedReason})
		t.earlyExit = true
		if *msg.ExitedReason == workload.Errored {
			return nil
		}
	}

	// If we completed a searcher operation, synchronize with the searcher to allow it to relay any
	// new operations to the trial. Otherwise just continue the trial immediately.
	if completedSearcherOp {
		ctx.Tell(ctx.Self().Parent(), sendNextWorkload{runID: t.runID})
		return nil
	}
	return t.sendNextWorkload(ctx)
}

func (t *trial) sendNextWorkload(ctx *actor.Context) error {
	terminateNow := false
	var w workload.Workload
	var err error
	switch {
	// We have another workload to run.
	case !t.pendingGracefulTermination && !t.sequencer.UpToDate():
		w, err = t.sequencer.Workload()
		if err != nil {
			return errors.Wrap(err, "error getting workload from sequencer")
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
		terminateNow = true
		t.terminate(ctx, false)
	}

	// Command the trial runner to do the thing we decided on (if this is not a replay).
	if !t.replaying {
		var msg interface{}
		if terminateNow {
			w = *t.sequencer.TerminateWorkload()
			msg = &tasks.TaskSpec{
				RunWorkload: &tasks.RunWorkload{
					Workload: w,
				},
			}

			t.terminationSent = true
			actors.NotifyAfter(ctx, terminateTimeoutPeriod, terminateTimeout{runID: t.runID})
		} else {
			if err := saveWorkload(t.db, w); err != nil {
				ctx.Log().WithError(err).Error("failed to save workload to the database")
			}
			msg = &tasks.TaskSpec{
				RunWorkload: &tasks.RunWorkload{
					Workload: w,
				},
			}
		}
		for _, socket := range t.sockets {
			if err := api.WriteSocketJSON(ctx, socket, msg); err != nil {
				ctx.Log().WithError(err).Error("cannot write to websocket")
			}
		}
	}
	return nil
}

func (t *trial) processContainerConnected(ctx *actor.Context, msg containerConnected) error {
	t.lastContainerConnectedTime = time.Now()
	if len(t.containers) < t.numContainers {
		actors.NotifyAfter(ctx, allReadyTimeoutPeriod, allReadyTimeout{runID: t.runID})
	}

	// If we have all of the container IDs from ContainerStarted messages, we can guard against
	// stale containers trying to connect.
	if len(t.containers) == t.numContainers {
		if _, ok := t.containers[msg.ContainerID]; !ok {
			ctx.Respond(errors.Errorf(
				"socket connection from stale container: %s", msg.ContainerID))
			return nil
		}
	}
	a := api.WrapSocket(msg.socket, workload.CompletedMessage{}, false)
	ref, _ := ctx.ActorOf(fmt.Sprintf("socket-%s", msg.ContainerID), a)
	t.sockets[msg.ContainerID] = ref
	ctx.Respond(ref)

	if err := t.pushRendezvous(ctx); err != nil {
		return errors.Wrap(err, "failed to push rendezvous to trial containers")
	}

	return nil
}

func formatAddress(p scheduler.Address) string {
	return fmt.Sprintf("%s:%d", p.HostIP, p.HostPort)
}

func (t *trial) killAndRemoveSocket(ctx *actor.Context, id scheduler.ContainerID) {
	if skt, ok := t.sockets[id]; ok {
		addr := skt.Address().Local()
		if ref := ctx.Child(addr); ref != nil {
			if ok := ctx.Kill(addr); !ok {
				ctx.Log().Warnf("failed to kill container socket: %s", id)
			}
		}
		delete(t.sockets, id)
	}
}

// allReady returns true if and only if an appropriate ContainerStarted message and a corresponding
// containerConnected message have been received from each container in the trial. The two messages
// are not guaranteed to come in-order.
func (t *trial) allReady(ctx *actor.Context) bool {
	// Ensure all ContainerStarted messages have arrived.
	if len(t.containers) < t.numContainers {
		return false
	}

	// Detect websockets which are not from the most current set of container IDs.
	//
	// Stale containers from this trial may still exist either after a trial crash or after a
	// master restart. Since ContainerStarted and containerConnected messages can come in any
	// order, it is not possible to detect which connections are from stale containers until after
	// all of the ContainerStarted messages have arrived.
	for id := range t.sockets {
		if _, ok := t.containers[id]; !ok {
			ctx.Log().Warnf("detected stray socket for unknown container: %s", id)
			t.killAndRemoveSocket(ctx, id)
		}
	}

	// Finally, ensure all sockets have connected.
	return len(t.sockets) == t.numContainers
}

// pushRendezvous gathers up the external addresses for the exposed ports and sends them to all the
// containers in the trial.
func (t *trial) pushRendezvous(ctx *actor.Context) error {
	if !t.allReady(ctx) {
		return nil
	}

	type CAddress struct {
		Container scheduler.Container
		Addresses []scheduler.Address
	}

	var caddrs []CAddress
	for _, c := range t.containers {
		caddr := CAddress{
			Container: c,
			Addresses: c.Addresses(),
		}
		caddrs = append(caddrs, caddr)

		sort.Slice(caddr.Addresses, func(i, j int) bool {
			a := caddr.Addresses[i]
			b := caddr.Addresses[j]

			return a.ContainerPort < b.ContainerPort
		})
	}

	sort.Slice(caddrs, func(i, j int) bool {
		a := caddrs[i].Container
		b := caddrs[j].Container
		switch {
		case a.IsLeader() && !b.IsLeader():
			return true
		case !a.IsLeader() && b.IsLeader():
			return false
		default:
			return a.ID() < b.ID()
		}
	})

	var rcontainers []*rendezvousContainer
	var addrs1 []string
	var addrs2 []string
	for _, caddr := range caddrs {
		var addresses []*rendezvousAddress

		var addrs []scheduler.Address
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
				numAddrs, caddr.Container.ID())
		}

		rcontainers = append(rcontainers, &rendezvousContainer{
			Addresses: addresses,
		})
	}

	for rank, caddr := range caddrs {
		c := caddr.Container
		socket := t.sockets[c.ID()]

		if err := api.WriteSocketJSON(ctx, socket, &trialMessage{
			RendezvousInfo: &rendezvousInfoMessage{
				Addrs:      addrs1,
				Addrs2:     addrs2,
				Rank:       rank,
				Containers: rcontainers,
			},
		}); err != nil {
			ctx.Log().WithError(err).Error("cannot write to socket")
		}
	}

	return nil
}

func (t *trial) processContainerTerminated(
	ctx *actor.Context,
	msg sproto.ContainerStateChanged,
) {
	c := t.containers[scheduler.ContainerID(msg.Container.ID)]
	delete(t.containers, scheduler.ContainerID(msg.Container.ID))

	t.killAndRemoveSocket(ctx, scheduler.ContainerID(msg.Container.ID))

	exitMsg := msg.ContainerStopped.String()
	t.processLog(ctx, sproto.ContainerLog{
		Container:  msg.Container,
		Timestamp:  time.Now(),
		AuxMessage: &exitMsg,
	})

	if c != nil {
		t.terminatedContainers = append(t.terminatedContainers, terminatedContainerWithState{
			exitStatus:                 *msg.ContainerStopped,
			isLeader:                   c.IsLeader(),
			pendingGracefulTermination: t.pendingGracefulTermination,
			needsCheckpoint:            t.sequencer.PrecloseCheckpointWorkload() != nil,
		})
	}

	// Terminate the task if the container never started (since this prevents the gang
	// from ever being able to start), if the leader of the gang has exited out, or if
	// one of the containers exited with a failure.
	if c == nil || c.IsLeader() || msg.ContainerStopped.Failure != nil {
		t.terminate(ctx, true)
	}
}

func (t *trial) processLog(ctx *actor.Context, msg sproto.ContainerLog) {
	// Log messages should never come in before the trial ID is set, since no trial runners are
	// launched until after the trial ID is set. But for futureproofing, we will log an error while
	// we protect the database.
	if !t.idSet {
		ctx.Log().Warnf("not saving log message from container without a trial ID: %s", msg.String())
		return
	}

	if t.logger == nil {
		// A trial created for a unit test does not have a logger.
		return
	}

	ctx.Tell(t.logger, model.TrialLog{TrialID: t.id, Message: msg.String() + "\n"})
}

func (t *trial) processTaskTerminated(ctx *actor.Context, msg scheduler.TaskTerminated) {
	getLeaderState := func() (terminatedContainerWithState, bool) {
		for _, c := range t.terminatedContainers {
			if c.isLeader {
				return c, true
			}
		}
		return terminatedContainerWithState{}, false
	}

	status := agent.ContainerError(agent.AgentError, errors.New("no error status provided"))
	if len(t.startedContainers) == 0 {
		// If we have no containers, we haven't started executing anything, so
		// let resetTrial reset our state rather than treating this as an error.
		status = agent.ContainerError(agent.TaskAborted, errors.New("task aborted"))
	} else if leaderState, ok := getLeaderState(); ok {
		status = classifyStatus(leaderState)
	}

	t.resetTrial(ctx, status)
}

func classifyStatus(state terminatedContainerWithState) agent.ContainerStopped {
	switch status := state.exitStatus; {
	case status.Failure != nil && status.Failure.FailureType != agent.TaskAborted:
		return status
	case !state.pendingGracefulTermination || state.needsCheckpoint:
		return agent.ContainerError(agent.AgentError, errors.New(
			"container exited when it wasn't supposed to"))
	default:
		return status
	}
}

func (t *trial) resetTrial(
	ctx *actor.Context,
	status agent.ContainerStopped,
) {
	terminationSent := t.terminationSent

	t.runID++
	t.task = nil
	t.pendingGracefulTermination = false
	t.terminationSent = false
	t.terminatedContainers = nil
	t.startedContainers = make(map[container.ID]bool)

	switch {
	case status.Failure == nil:
		ctx.Log().Info("trial runner stopped successfully")
		return
	case terminationSent:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since termination was requested",
		)
		return
	case t.cancelUnready:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since it was canceled or paused " +
				"before all containers are connected",
		)
		return
	case t.killed:
		ctx.Log().WithField("failure", status.Failure).Info(
			"ignoring trial runner failure since it was killed",
		)
		return
	case status.Failure.FailureType == agent.TaskAborted:
		return
	}

	ctx.Log().Errorf("unexpected failure of trial after restart %d/%d: %v",
		t.restarts, t.experiment.Config.MaxRestarts, status)
	t.restarts++
	if t.restarts <= t.experiment.Config.MaxRestarts {
		t.restore(ctx)
		return
	}

	if w, err := t.sequencer.Workload(); err != nil {
		if !t.replaying {
			if err := markWorkloadErrored(t.db, w); err != nil {
				ctx.Log().Error(err)
			}
		}
	}
	e := workload.Errored
	w, err := t.sequencer.Workload()
	if err != nil {
		ctx.Log().Error(err)
		panic(err)
	}
	erroredMessage := workload.CompletedMessage{
		Workload:     w,
		ExitedReason: &e,
	}
	if err := t.processCompletedWorkload(ctx, erroredMessage); err != nil {
		ctx.Log().Error(err)
	}
}

func (t *trial) restore(ctx *actor.Context) {
	// If the trial has not been created in the database yet (which can happen during master restart),
	// it can't have any state to restore.
	if !t.idSet {
		return
	}

	trial, err := t.db.TrialByID(t.id)
	if err != nil {
		ctx.Log().Error(err)
	} else if _, ok := model.TerminalStates[trial.State]; ok {
		ctx.Self().Stop()
		return
	}

	step := t.sequencer.RollBackSequencer()

	ctx.Log().Infof("restoring trial %d to end of step %d", t.id, step)

	// Delete things in the database that are now invalid. (Even if the last completed
	// step had its checkpoint done, the following step may have been started and thus
	// added to the database already.)
	if err := t.db.RollBackTrial(t.id, step); err != nil {
		ctx.Log().Error(err)
	}
}

func (t *trial) trialClosing() bool {
	return t.earlyExit || t.killed || t.restarts > t.experiment.Config.MaxRestarts ||
		(t.close != nil && t.sequencer.UpToDate()) ||
		model.StoppingStates[t.experimentState]
}

func (t *trial) terminate(ctx *actor.Context, kill bool) {
	if kill {
		ctx.Log().Info("forcibly terminating trial")
		if t.task != nil {
			ctx.Tell(t.rp, scheduler.TerminateTask{TaskID: t.task.ID, Forcible: true})
		}
	} else {
		ctx.Log().Info("gracefully terminating trial")
		t.pendingGracefulTermination = true
	}
}
