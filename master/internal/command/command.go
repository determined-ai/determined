package command

import (
	"context"
	"fmt"
	"sync"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
	"golang.org/x/exp/slices"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmerrors"
	"github.com/determined-ai/determined/master/internal/rm/tasklist"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

// terminatedDuration defines the amount of time the command stays in a
// terminated state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

// queueStates are allocation states which the API and UI will show as "Queued".
var queueStates = []model.AllocationState{
	model.AllocationStatePending,
	model.AllocationStateAssigned,
}

// Command is executed in a containerized environment on a Determined cluster.
// Locking in: Start, OnExit, DeleteIfInWorkspace, ToV1Command/Shell/Notebook/Tensorboard.
type Command struct {
	mu sync.Mutex

	db *db.PgDB
	rm rm.ResourceManager

	tasks.GenericCommandSpec

	registeredTime time.Time
	taskID         model.TaskID
	taskType       model.TaskType
	jobType        model.JobType
	jobID          model.JobID
	allocationID   model.AllocationID
	lastState      task.AllocationState
	exitStatus     *task.AllocationExited
	restored       bool

	contextDirectory []byte // Don't rely on this being set outsides of PreStart non restore case.

	logCtx logger.Context
	syslog *logrus.Entry
}

// CreateGeneric is a request to the CommandService to create a generic command.
type CreateGeneric struct {
	ContextDirectory []byte
	Spec             *tasks.GenericCommandSpec
}

func commandFromSnapshot(
	db *db.PgDB,
	rm rm.ResourceManager,
	snapshot *CommandSnapshot,
) (*Command, error) {
	taskID := snapshot.TaskID
	taskType := snapshot.Task.TaskType
	jobID := snapshot.Task.Job.JobID

	logCtx := logger.Context{
		"job-id":    jobID,
		"task-id":   taskID,
		"task-type": taskType,
	}

	cmd := &Command{
		db:                 db,
		rm:                 rm,
		registeredTime:     snapshot.RegisteredTime,
		GenericCommandSpec: snapshot.GenericCommandSpec,
		taskID:             taskID,
		taskType:           taskType,
		jobType:            snapshot.Task.Job.JobType,
		jobID:              jobID,
		restored:           true,
		logCtx:             logCtx,
		syslog:             logrus.WithFields(logrus.Fields{"component": "command"}).WithFields(logCtx.Fields()),
	}
	return cmd, cmd.Start(context.TODO())
}

// Start starts the command & its respective allocation. Once started, it persists to the db.
func (c *Command) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	priorityChange := func(priority int) error {
		return c.setNTSCPriority(priority, false)
	}
	if err := tasklist.GroupPriorityChangeRegistry.Add(c.jobID, priorityChange); err != nil {
		return err
	}
	c.allocationID = model.AllocationID(fmt.Sprintf("%s.%d", c.taskID, 1))

	if !c.restored {
		if err := db.Bun().RunInTx(ctx, nil, c.registerJobAndTask); err != nil {
			return err
		}
		if err := c.persistAndEvictContextDirectoryFromMemory(); err != nil {
			return err
		}
	}

	priority := c.Config.Resources.Priority
	if priority != nil {
		if err := c.setNTSCPriority(*priority, true); err != nil {
			return errors.Wrapf(err, "setting priority of task %v", c.taskID)
		}
	}

	var idleWatcherConfig *sproto.IdleTimeoutConfig
	if c.Config.IdleTimeout != nil && (c.WatchProxyIdleTimeout || c.WatchRunnerIdleTimeout) {
		idleWatcherConfig = &sproto.IdleTimeoutConfig{
			ServiceID:       string(c.taskID),
			UseProxyState:   c.WatchProxyIdleTimeout,
			UseRunnerState:  c.WatchRunnerIdleTimeout,
			TimeoutDuration: time.Duration(*c.Config.IdleTimeout),
			Debug:           c.Config.Debug,
		}
	}

	err := task.DefaultService.StartAllocation(c.logCtx,
		sproto.AllocateRequest{
			AllocationID:        c.allocationID,
			TaskID:              c.taskID,
			JobID:               c.jobID,
			JobSubmissionTime:   c.registeredTime,
			IsUserVisible:       true,
			Name:                c.Config.Description,
			SlotsNeeded:         c.Config.Resources.Slots,
			ResourcePool:        c.Config.Resources.ResourcePool,
			FittingRequirements: sproto.FittingRequirements{SingleAgent: true},
			ProxyPorts:          sproto.NewProxyPortConfig(c.GenericCommandSpec.ProxyPorts(), c.taskID),
			IdleTimeout:         idleWatcherConfig,
			Restore:             c.restored,
			ProxyTLS:            c.TaskType == model.TaskTypeNotebook,
		}, c.db, c.rm, c.GenericCommandSpec, c.OnExit)
	if err != nil {
		return err
	}

	// Once the command is persisted to the dbs & allocation starts, register it with the local job service.
	jobservice.DefaultService.RegisterJob(c.jobID, c)

	if err := c.persist(); err != nil {
		c.syslog.WithError(err).Warnf("command persist failure")
	}
	return nil
}

// registerJobAndTask registers the command with the job service & adds the command to the job & task dbs.
func (c *Command) registerJobAndTask(ctx context.Context, tx bun.Tx) error {
	c.registeredTime = time.Now().Truncate(time.Millisecond)
	if err := db.AddJobTx(ctx, tx, &model.Job{
		JobID:   c.jobID,
		JobType: c.jobType,
		OwnerID: &c.Base.Owner.ID,
	}); err != nil {
		return fmt.Errorf("persisting job %v: %w", c.taskID, err)
	}

	if err := db.AddTaskTx(ctx, tx, &model.Task{
		TaskID:     c.taskID,
		TaskType:   c.taskType,
		StartTime:  c.registeredTime,
		JobID:      &c.jobID,
		LogVersion: model.CurrentTaskLogVersion,
	}); err != nil {
		return fmt.Errorf("persisting task %v: %w", c.taskID, err)
	}
	return nil
}

func (c *Command) persistAndEvictContextDirectoryFromMemory() error {
	if c.contextDirectory == nil {
		c.contextDirectory = make([]byte, 0)
	}

	if _, err := db.Bun().NewInsert().Model(&model.TaskContextDirectory{
		TaskID:           c.taskID,
		ContextDirectory: c.contextDirectory,
	}).Exec(context.TODO()); err != nil {
		return fmt.Errorf("persisting context directory files: %w", err)
	}

	c.contextDirectory = nil
	return nil
}

func (c *Command) persist() error {
	snapshot := &CommandSnapshot{
		TaskID:             c.taskID,
		RegisteredTime:     c.registeredTime,
		AllocationID:       c.allocationID,
		GenericCommandSpec: c.GenericCommandSpec,
	}
	_, err := db.Bun().NewInsert().Model(snapshot).
		On("CONFLICT (task_id) DO UPDATE").
		Exec(context.TODO())
	return err
}

// OnExit runs when an command's allocation exits. It marks the command task as complete, and unregisters where needed.
// OnExit locks ahead of gc -> unregisterCommand.
func (c *Command) OnExit(ae *task.AllocationExited) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.exitStatus = ae

	if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
		c.syslog.WithError(err).Error("marking task complete")
	}
	if err := user.DeleteSessionByToken(context.TODO(), c.GenericCommandSpec.Base.UserSessionToken); err != nil {
		c.syslog.WithError(err).Errorf(
			"failure to delete user session for task: %v", c.taskID)
	}

	go func() {
		time.Sleep(terminatedDuration)
		c.garbageCollect()
	}()
}

// gc garbage collects the exited command.
func (c *Command) garbageCollect() {
	if err := tasklist.GroupPriorityChangeRegistry.Delete(c.jobID); err != nil {
		c.syslog.WithError(err).Error("deleting command from GroupPriorityChangeRegistry")
	}

	if c.exitStatus == nil {
		if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
			c.syslog.WithError(err).Error("marking task complete")
		}
	}

	go jobservice.DefaultService.UnregisterJob(c.jobID)
	go DefaultCmdService.unregisterCommand(c.taskID)
}

func (c *Command) setNTSCPriority(priority int, forward bool) error {
	if forward {
		switch err := c.rm.SetGroupPriority(sproto.SetGroupPriority{
			Priority:     priority,
			ResourcePool: c.Config.Resources.ResourcePool,
			JobID:        c.jobID,
		}).(type) {
		case nil:
		case rmerrors.UnsupportedError:
			c.syslog.WithError(err).Debug("ignoring unsupported call to set group priority")
		default:
			return fmt.Errorf("setting group priority for command: %w", err)
		}
	}

	c.Config.Resources.Priority = &priority

	return nil
}

// DeleteIfInWorkspace deletes a command's allocation matching a workspaceID.
func (c *Command) DeleteIfInWorkspace(req *apiv1.DeleteWorkspaceRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Metadata.WorkspaceID == model.AccessScopeID(req.Id) {
		err := task.DefaultService.Signal(
			c.allocationID,
			task.KillAllocation,
			"user requested workspace delete",
		)
		if err != nil {
			c.syslog.WithError(err).Warn("failed to kill allocation while deleting workspace")
		}
	}
}

// ToV1Command takes a *Command from the command service registry & returns a *commandv1.Command.
func (c *Command) ToV1Command() *commandv1.Command {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &commandv1.Command{
		Id:           c.stringID(),
		State:        enrichState(allo.State),
		Description:  c.Config.Description,
		Container:    allo.SingleContainer().ToProto(),
		StartTime:    protoutils.ToTimestamp(c.registeredTime),
		Username:     c.Base.Owner.Username,
		UserId:       int32(c.Base.Owner.ID),
		DisplayName:  c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool: c.Config.Resources.ResourcePool,
		ExitStatus:   c.exitStatus.String(),
		JobId:        c.jobID.String(),
		WorkspaceId:  int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

// ToV1Notebook takes a *Command from the command service registry & returns a *notebookv1.Notebook.
func (c *Command) ToV1Notebook() *notebookv1.Notebook {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &notebookv1.Notebook{
		Id:             c.stringID(),
		State:          enrichState(allo.State),
		Description:    c.Config.Description,
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

// ToV1Shell takes a *Command from the command service registry & returns a *shellv1.Shell.
func (c *Command) ToV1Shell() *shellv1.Shell {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	return &shellv1.Shell{
		Id:             c.stringID(),
		State:          enrichState(allo.State),
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		PrivateKey:     *c.Metadata.PrivateKey,
		PublicKey:      *c.Metadata.PublicKey,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		Addresses:      toProto(allo.SingleContainerAddresses()),
		AgentUserGroup: protoutils.ToStruct(c.Base.AgentUserGroup),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

// ToV1Tensorboard takes a *Command from the command service registry & returns a *tensorboardv1.Tensorboard.
func (c *Command) ToV1Tensorboard() *tensorboardv1.Tensorboard {
	c.mu.Lock()
	defer c.mu.Unlock()

	allo := c.refreshAllocationState()
	state := enrichState(allo.State)
	return &tensorboardv1.Tensorboard{
		Id:             c.stringID(),
		State:          state,
		Description:    c.Config.Description,
		StartTime:      protoutils.ToTimestamp(c.registeredTime),
		Container:      allo.SingleContainer().ToProto(),
		ServiceAddress: c.serviceAddress(),
		ExperimentIds:  c.Metadata.ExperimentIDs,
		TrialIds:       c.Metadata.TrialIDs,
		Username:       c.Base.Owner.Username,
		UserId:         int32(c.Base.Owner.ID),
		DisplayName:    c.Base.Owner.DisplayName.ValueOrZero(),
		ResourcePool:   c.Config.Resources.ResourcePool,
		ExitStatus:     c.exitStatus.String(),
		JobId:          c.jobID.String(),
		WorkspaceId:    int32(c.GenericCommandSpec.Metadata.WorkspaceID),
	}
}

// ToV1Command(), ToV1Notebook(), ToV1Shell(), ToV1Tensorboard() helper functions:
// refreshAllocationState, enrichState, toProto, serviceAddress, stringID

// Refresh our view of the allocation state. If the allocation has sent us an exit status,
// we don't ask for a refresh because it won't respond. Otherwise, ask with a timeout
// since there is another ask in the opposite direction, and even though it's probably
// 1 in a million runs, we don't want to deadlock.
func (c *Command) refreshAllocationState() task.AllocationState {
	if c.exitStatus != nil {
		return c.exitStatus.FinalState
	}

	state, err := task.DefaultService.State(c.allocationID)
	if err != nil {
		c.syslog.WithError(err).Warn("refreshing allocation state")
	} else {
		c.lastState = state
	}
	return c.lastState
}

func enrichState(state model.AllocationState) taskv1.State {
	if slices.Contains(queueStates, state) {
		return taskv1.State_STATE_QUEUED
	}
	return state.Proto()
}

func toProto(as []cproto.Address) []*structpb.Struct {
	res := make([]*structpb.Struct, 0, len(as))
	for _, a := range as {
		res = append(res, protoutils.ToStruct(a))
	}
	return res
}

func (c *Command) serviceAddress() string {
	return fmt.Sprintf("/proxy/%s/", c.taskID)
}

func (c *Command) stringID() string {
	return c.taskID.String()
}
