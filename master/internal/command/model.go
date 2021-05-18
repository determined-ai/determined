package command

import (
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
)

// CommandParams describes parameters for launching a command.
type CommandParams struct {
	UserFiles      archive.Archive
	Data           map[string]interface{}
	FullConfig     *model.CommandConfig
	TaskSpec       *tasks.TaskSpec
	User           *model.User
	AgentUserGroup *model.AgentUserGroup
}
