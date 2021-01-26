package command

import (
	"fmt"
	"net/http"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	requestContext "github.com/determined-ai/determined/master/internal/context"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/resourcemanagers"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
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
	taskSpec              *tasks.TaskSpec
}

// CommandLaunchRequest describes a request to launch a new command.
type CommandLaunchRequest struct {
	CommandParams *CommandParams
	User          *model.User
}

func (c *commandManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetCommandsRequest:
		resp := &apiv1.GetCommandsResponse{}
		for _, command := range ctx.AskAll(&commandv1.Command{}, ctx.Children()...).GetAll() {
			resp.Commands = append(resp.Commands, command.(*commandv1.Command))
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

	case echo.Context:
		c.handleAPIRequest(ctx, msg)
	}
	return nil
}

func (c *commandManager) processLaunchRequest(
	ctx *actor.Context,
	req CommandLaunchRequest,
) (*summary, int, error) {
	commandReq, err := parseCommandRequest(
		ctx.Self().System(), c.db, *req.User, req.CommandParams, &c.taskSpec.TaskContainerDefaults,
	)
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	if commandReq.AgentUserGroup == nil {
		commandReq.AgentUserGroup = &c.defaultAgentUserGroup
	}

	ctx.Log().Info("creating command")

	command := c.newCommand(commandReq)
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

func (c *commandManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		userFilter := apiCtx.QueryParam("user")
		ctx.Respond(apiCtx.JSON(
			http.StatusOK,
			ctx.AskAll(getSummary{userFilter: userFilter}, ctx.Children()...)))

	case echo.POST:
		var params CommandParams
		if err := apiCtx.Bind(&params); err != nil {
			respondBadRequest(ctx, err)
			return
		}
		user := apiCtx.(*requestContext.DetContext).MustGetUser()
		req := CommandLaunchRequest{
			User:          &user,
			CommandParams: &params,
		}
		summary, statusCode, err := c.processLaunchRequest(ctx, req)
		if err != nil || statusCode > 200 {
			ctx.Respond(echo.NewHTTPError(statusCode, err.Error()))
			return
		}
		ctx.Respond(apiCtx.JSON(http.StatusOK, summary))

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
			petname.Generate(expconf.TaskNameGeneratorWords, expconf.TaskNameGeneratorSep),
		)
	}
	if len(config.Entrypoint) == 1 {
		config.Entrypoint = append(shellFormEntrypoint, config.Entrypoint...)
	}
	setPodSpec(&config, c.taskSpec.TaskContainerDefaults)

	return &command{
		taskID:    resourcemanagers.NewTaskID(),
		config:    config,
		userFiles: req.UserFiles,

		owner:          req.Owner,
		agentUserGroup: req.AgentUserGroup,
		taskSpec:       c.taskSpec,
	}
}
