package command

import (
	"fmt"
	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
)

// If an entrypoint is specified as a singleton string, Determined will follow the "shell form"
// convention of Docker that executes the command with "/bin/sh -c" prepended.
//
// https://docs.docker.com/engine/reference/builder/#shell-form-entrypoint-example
var shellFormEntrypoint = []string{"/bin/sh", "-c"}

type commandManager struct {
	db *db.PgDB

	defaultAgentUserGroup model.AgentUserGroup
	makeTaskSpec          tasks.MakeTaskSpecFn
}

// CommandLaunchRequest describes a request to launch a new command.
type CommandLaunchRequest struct {
	CommandParams *CommandParams
}

func (c *commandManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetCommandsRequest:
		resp := &apiv1.GetCommandsResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, command := range ctx.AskAll(&commandv1.Command{}, ctx.Children()...).GetAll() {
			if typed := command.(*commandv1.Command); len(users) == 0 || users[typed.Username] {
				resp.Commands = append(resp.Commands, typed)
			}
		}
		ctx.Respond(resp)

	case CommandLaunchRequest:
		summary, statusCode, err := c.processLaunchRequest(ctx, msg)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(
				statusCode,
				errors.Wrap(err, "failed to launch command").Error(),
			))
			return nil
		}
		ctx.Respond(summary.ID)
	}
	return nil
}

func (c *commandManager) processLaunchRequest(
	ctx *actor.Context,
	req CommandLaunchRequest,
) (*summary, int, error) {
	ctx.Log().Info("creating command")

	command := c.newCommand(req.CommandParams)
	if err := check.Validate(command.config); err != nil {
		return nil, http.StatusBadRequest, err
	}

	a, _ := ctx.ActorOf(command.taskID, command)
	summaryFut := ctx.Ask(a, getSummary{})
	if err := summaryFut.Error(); err != nil {
		return nil, http.StatusInternalServerError, err
	}
	summary := summaryFut.Get().(summary)
	ctx.Log().Infof("created command %s", a.Address().Local())
	return &summary, http.StatusOK, nil
}

func (c *commandManager) newCommand(params *CommandParams) *command {
	config := params.FullConfig

	// Postprocess the config.
	if config.Description == "" {
		config.Description = fmt.Sprintf(
			"Command (%s)",
			petname.Generate(model.TaskNameGeneratorWords, model.TaskNameGeneratorSep),
		)
	}
	if len(config.Entrypoint) == 1 {
		config.Entrypoint = append(shellFormEntrypoint, config.Entrypoint...)
	}
	setPodSpec(config, params.TaskSpec.TaskContainerDefaults)

	return &command{
		taskID:    sproto.NewTaskID(),
		config:    *params.FullConfig,
		userFiles: params.UserFiles,
		owner: commandOwner{
			ID:       params.User.ID,
			Username: params.User.Username,
		},
		agentUserGroup: params.AgentUserGroup,
		taskSpec:       params.TaskSpec,

		db: c.db,
	}
}
