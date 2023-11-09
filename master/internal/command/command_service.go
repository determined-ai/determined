package command

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
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
	commands map[model.TaskID]*command
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
		commands: make(map[model.TaskID]*command),
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

// createGenericCommand creates NTSC commands and persists them to the database.
func (cs *CommandService) createGenericCommand(
	taskType model.TaskType,
	jobType model.JobType,
	req *CreateGeneric,
) (*command, error) {
	taskID := model.NewTaskID()
	jobID := model.NewJobID()
	req.Spec.CommandID = string(taskID)
	req.Spec.TaskType = taskType

	cmd := &command{
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
	// Add it to the registry.
	cs.commands[cmd.taskID] = cmd

	return cmd, cmd.startCmd(context.TODO())
}

func (cs *CommandService) unregisterCommand(id model.TaskID) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	delete(cs.commands, id)
}

// getNTSC gets & checks type of a command given its ID.
func (cs *CommandService) getNTSC(cmdID model.TaskID, cmdType model.TaskType) (*command, error) {
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
) (cmds []*command, users map[string]bool, userIds map[int32]bool) {
	users = make(map[string]bool, len(reqUsers))
	for _, user := range reqUsers {
		users[user] = true
	}
	userIds = make(map[int32]bool, len(reqUserIDs))
	for _, user := range reqUserIDs {
		userIds[user] = true
	}

	cmds = []*command{}
	for _, c := range cs.commands {
		if c.taskType == cmdType {
			cmds = append(cmds, c)
		}
	}
	return cmds, users, userIds
}

// DeleteWorkspaceNTSC deletes all NTSC associated with a workspace ID.
func (cs *CommandService) DeleteWorkspaceNTSC(req *apiv1.DeleteWorkspaceRequest) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for _, c := range cs.commands {
		c.deleteIfInWorkspace(req)
	}
}
