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
	"github.com/determined-ai/determined/proto/pkg/taskv1"
)

// terminatedDuration defines the amount of time the command stays in a
// terminated state in the master before garbage collecting.
const terminatedDuration = 24 * time.Hour

// queueStates are allocation states which the API and UI will show as "Queued".
var queueStates = []model.AllocationState{
	model.AllocationStatePending,
	model.AllocationStateAssigned,
}

// command is executed in a containerized environment on a Determined cluster.
// Locking in: toNTSC, startCmd, onExit, deleteIfInWorkspace.
type command struct {
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
) (*command, error) {
	taskID := snapshot.TaskID
	taskType := snapshot.Task.TaskType
	jobID := snapshot.Task.Job.JobID
	cmd := &command{
		db:                 db,
		rm:                 rm,
		registeredTime:     snapshot.RegisteredTime,
		GenericCommandSpec: snapshot.GenericCommandSpec,
		taskID:             taskID,
		taskType:           taskType,
		jobType:            snapshot.Task.Job.JobType,
		jobID:              jobID,
		restored:           true,
		logCtx: logger.Context{
			"job-id":    jobID,
			"task-id":   taskID,
			"task-type": taskType,
		},
		syslog: logrus.WithFields(logrus.Fields{
			"component": "command",
			"taskID":    taskID,
		}),
	}
	return cmd, cmd.startCmd(context.TODO())
}

// start starts the command & its respective allocation. Once started, it persists to the db.
func (c *command) startCmd(ctx context.Context) error {
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
		}, c.db, c.rm, c.GenericCommandSpec, c.onExit)
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
func (c *command) registerJobAndTask(ctx context.Context, tx bun.Tx) error {
	c.registeredTime = time.Now()
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

func (c *command) persistAndEvictContextDirectoryFromMemory() error {
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

func (c *command) persist() error {
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

// onExit runs when an command's allocation exits. It marks the command task as complete, and unregisters where needed.
// onExit locks ahead of gc -> unregisterCommand.
func (c *command) onExit(ae *task.AllocationExited) {
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
func (c *command) garbageCollect() {
	if err := tasklist.GroupPriorityChangeRegistry.Delete(c.jobID); err != nil {
		c.syslog.WithError(err).Error("deleting command from GroupPriorityChangeRegistry")
	}

	if c.exitStatus == nil {
		if err := c.db.CompleteTask(c.taskID, time.Now().UTC()); err != nil {
			c.syslog.WithError(err).Error("marking task complete")
		}
	}

	jobservice.DefaultService.UnregisterJob(c.jobID)
	DefaultCmdService.unregisterCommand(c.taskID)

	if err := user.DeleteSessionByToken(
		context.TODO(),
		c.GenericCommandSpec.Base.UserSessionToken,
	); err != nil {
		c.syslog.WithError(err).Errorf(
			"failure to delete user session for task: %v", c.taskID)
	}
}

// command NTSC methods: setNTSCPriority, deleteIfInWorkspace.
// These functions are not locked, rather only where they're called.
func (c *command) setNTSCPriority(priority int, forward bool) error {
	if forward {
		switch err := c.rm.SetGroupPriority(sproto.SetGroupPriority{
			Priority: priority,
			JobID:    c.jobID,
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

func (c *command) deleteIfInWorkspace(req *apiv1.DeleteWorkspaceRequest) {
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

// toCommand(), toNotebook(), toShell(), toTensorboard() helper functions:
// refreshAllocationState, enrichState, toProto, serviceAddress, stringID

// Refresh our view of the allocation state. If the allocation has sent us an exit status,
// we don't ask for a refresh because it won't respond. Otherwise, ask with a timeout
// since there is another ask in the opposite direction, and even though it's probably
// 1 in a million runs, we don't want to deadlock.
func (c *command) refreshAllocationState() task.AllocationState {
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

func (c *command) serviceAddress() string {
	return fmt.Sprintf("/proxy/%s/", c.taskID)
}

func (c *command) stringID() string {
	return c.taskID.String()
}
