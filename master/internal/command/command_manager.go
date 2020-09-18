package command

import (
	"fmt"
	"net/http"

	"github.com/determined-ai/determined/master/pkg/tasks"

	"github.com/google/uuid"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
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
	defaultTaskSpec       *tasks.TaskSpec
}

func (c *commandManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetCommandsRequest:
		resp := &apiv1.GetCommandsResponse{}
		for _, command := range ctx.AskAll(&commandv1.Command{}, ctx.Children()...).GetAll() {
			resp.Commands = append(resp.Commands, command.(*commandv1.Command))
		}
		ctx.Respond(resp)

	case echo.Context:
		c.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (c *commandManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		var params commandParams
		if err := apiCtx.Bind(&params); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		req, err := parseCommandRequest(apiCtx, c.db, &params, &c.taskContainerDefaults)
		if err != nil {
			respondBadRequest(ctx, err)
			return
		}

		if req.AgentUserGroup == nil {
			req.AgentUserGroup = &c.defaultAgentUserGroup
		}

		ctx.Log().Info("creating command")

		command := c.newCommand(req)
		if err := check.Validate(command.config); err != nil {
			respondBadRequest(ctx, err)
			return
		}

		a, _ := ctx.ActorOf(command.taskID, command)
		ctx.Respond(apiCtx.JSON(http.StatusOK, ctx.Ask(a, getSummary{})))
		ctx.Log().Infof("created command %s", a.Address().Local())

	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (c *commandManager) newCommand(req *commandRequest) *command {
	config := req.Config

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
	setPodSpec(&config, c.defaultTaskSpec.TaskContainerDefaults)

	return &command{
		taskID:    scheduler.TaskID(uuid.New().String()),
		config:    config,
		userFiles: req.UserFiles,

		owner:           req.Owner,
		agentUserGroup:  req.AgentUserGroup,
		defaultTaskSpec: c.defaultTaskSpec,
	}
}
