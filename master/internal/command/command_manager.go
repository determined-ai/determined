// Package command provides utilities for commands.
//
//nolint:dupl
package command

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
)

// CreateGeneric is a request to managers to create a generic command.
type CreateGeneric struct {
	ModelDef []byte
	Spec     *tasks.GenericCommandSpec
}

type commandManager struct {
	db *db.PgDB
	rm rm.ResourceManager
}

func (c *commandManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		tryRestoreCommandsByType(ctx, c.db, c.rm, model.TaskTypeCommand)

	case actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetCommandsRequest:
		resp := &apiv1.GetCommandsResponse{}
		users := make(map[string]bool, len(msg.Users))
		for _, user := range msg.Users {
			users[user] = true
		}
		userIds := make(map[int32]bool, len(msg.UserIds))
		for _, user := range msg.UserIds {
			userIds[user] = true
		}
		for _, command := range ctx.AskAll(&commandv1.Command{}, ctx.Children()...).GetAll() {
			typed := command.(*commandv1.Command)
			if (len(users) == 0 && len(userIds) == 0) || users[typed.Username] || userIds[typed.UserId] {
				resp.Commands = append(resp.Commands, typed)
			}
		}
		ctx.Respond(resp)

	case *apiv1.DeleteWorkspaceRequest:
		ctx.TellAll(msg, ctx.Children()...)

	case CreateGeneric:
		taskID := model.NewTaskID()
		jobID := model.NewJobID()
		msg.Spec.CommandID = string(taskID)
		if err := createGenericCommandActor(
			ctx, c.db, c.rm, taskID, model.TaskTypeCommand, jobID,
			model.JobTypeCommand, msg.Spec, msg.ModelDef,
		); err != nil {
			ctx.Log().WithError(err).Error("failed to launch command")
			ctx.Respond(err)
		} else {
			ctx.Respond(taskID)
		}

	case echo.Context:
		ctx.Respond(echo.NewHTTPError(http.StatusNotFound, api.ErrAPIRemoved))

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
