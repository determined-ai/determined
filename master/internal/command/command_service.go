package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// DefaultCmdService is the global command service singleton.
var DefaultCmdService *CommandService

// CommandService tracks the different NTSC commands in the system.
type CommandService struct {
	db       *db.PgDB
	rm       rm.ResourceManager
	mu       sync.Mutex
	commands map[model.TaskID]*Command
	syslog   *logrus.Entry
}

// SetDefaultCmdService initializes & returns a new CommandService.
func SetDefaultCmdService(db *db.PgDB, rm rm.ResourceManager) {
	if DefaultCmdService != nil {
		logrus.Warn(
			"detected re-initialization of Command Service that should never occur outside of tests",
		)
	}

	DefaultCmdService = &CommandService{
		db:       db,
		rm:       rm,
		commands: make(map[model.TaskID]*Command),
		syslog:   logrus.WithField("component", "command-service"),
	}
}

// RestoreAllCommands restores all terminated commands whose end time isn't set.
func (cs *CommandService) RestoreAllCommands(
	ctx context.Context,
) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	snapshots := []CommandSnapshot{}
	err := db.Bun().NewSelect().Model(&snapshots).
		Relation("Allocation").
		Relation("Task").
		Relation("Task.Job").
		Where("allocation.end_time IS NULL").
		Where("allocation.state != ?", model.AllocationStateTerminated).
		Where("task.task_id = command_snapshot.task_id").
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to remake commands: %w", err)
	}

	for i := range snapshots {
		cmd, err := commandFromSnapshot(cs.db, cs.rm, &snapshots[i])
		if err != nil {
			return err
		}
		// Restore to the command service registry.
		cs.commands[cmd.taskID] = cmd
		cs.syslog.Debugf("restored & started generic command %s", cmd.taskID)
	}
	return nil
}

// LaunchGenericCommand creates NTSC commands and persists them to the database.
func (cs *CommandService) LaunchGenericCommand(
	taskType model.TaskType,
	jobType model.JobType,
	req *CreateGeneric,
) (*Command, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	taskID := model.NewTaskID()
	jobID := model.NewJobID()
	req.Spec.CommandID = string(taskID)
	req.Spec.TaskType = taskType

	cmd := &Command{
		db: cs.db,
		rm: cs.rm,

		GenericCommandSpec: *req.Spec,

		taskID:           taskID,
		taskType:         taskType,
		jobType:          jobType,
		jobID:            jobID,
		contextDirectory: req.ContextDirectory,
		logCtx: logger.Context{
			"job-id":    jobID,
			"task-id":   taskID,
			"task-type": taskType,
		},
		syslog: logrus.WithFields(logrus.Fields{
			"component": "command",
			"task-id":   taskID,
		}),
	}

	if err := cmd.startCmd(context.TODO()); err != nil {
		return nil, err
	}

	// Add it to the registry.
	cs.commands[cmd.taskID] = cmd

	return cmd, nil
}

func (cs *CommandService) unregisterCommand(id model.TaskID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.commands, id)
}

// getNTSC gets & checks type of a command given its ID.
func (cs *CommandService) getNTSC(cmdID model.TaskID, cmdType model.TaskType) (*Command, error) {
	c, ok := cs.commands[cmdID]
	if !ok {
		return nil, fmt.Errorf("get NTSC %s not found", cmdID)
	}

	if c.taskType != cmdType {
		return nil, fmt.Errorf("getNTSC: type mismatch: %s/%s", cmdType, c.taskType)
	}

	return c, nil
}

// listByType returns a list of NTSCs of one type.
func (cs *CommandService) listByType(
	reqUsers []string,
	reqUserIDs []int32,
	cmdType model.TaskType,
	workspaceID int32,
) []*Command {
	users := make(map[string]bool, len(reqUsers))
	for _, user := range reqUsers {
		users[user] = true
	}
	userIds := make(map[int32]bool, len(reqUserIDs))
	for _, user := range reqUserIDs {
		userIds[user] = true
	}

	cmds := []*Command{}
	for _, c := range cs.commands {
		wID := int32(c.GenericCommandSpec.Metadata.WorkspaceID)
		username := c.Base.Owner.Username
		userID := int32(c.Base.Owner.ID)
		// skip if it doesn't match the requested workspaceID if any.
		if workspaceID != 0 && workspaceID != wID {
			continue
		}
		if c.taskType == cmdType && ((len(users) == 0 && len(userIds) == 0) ||
			users[username] || userIds[userID]) {
			cmds = append(cmds, c)
		}
	}
	return cmds
}

// DeleteWorkspaceNTSC deletes all NTSC associated with a workspace ID.
func (cs *CommandService) DeleteWorkspaceNTSC(req *apiv1.DeleteWorkspaceRequest) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for _, c := range cs.commands {
		c.deleteIfInWorkspace(req)
	}
}

// SetNTSCPriority sets the NTSC's resource manager group priority.
func (cs *CommandService) SetNTSCPriority(
	id string, priority int,
) (*Command, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	c, err := cs.getNTSC(model.TaskID(id), model.TaskTypeCommand)
	if err != nil {
		return nil, err
	}

	err = c.setNTSCPriority(priority, true)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// KillNTSC sends a kill signal to the command's allocation.
func (cs *CommandService) KillNTSC(id string) (*Command, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	tID := model.TaskID(id)

	c, err := cs.getNTSC(tID, model.TaskTypeNotebook)
	if err != nil {
		return nil, err
	}

	completed, err := db.TaskCompleted(context.TODO(), tID)
	if err != nil {
		return nil, err
	}

	if !completed {
		err = task.DefaultService.Signal(c.allocationID, task.KillAllocation, "user requested kill")
		if err != nil {
			return nil, fmt.Errorf("failed to kill allocation: %w", err)
		}
	}

	return c, nil
}
