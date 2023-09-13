// Package command provides utilities for commands.
package command

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

type tensorboardManager struct {
	db *db.PgDB
	rm rm.ResourceManager
}

func (t *tensorboardManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		tryRestoreCommandsByType(ctx, t.db, t.rm, model.TaskTypeTensorboard)

	case actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetTensorboardsRequest:
		resp := &apiv1.GetTensorboardsResponse{}
		users := make(map[string]bool, len(msg.Users))
		for _, user := range msg.Users {
			users[user] = true
		}
		userIds := make(map[int32]bool, len(msg.UserIds))
		for _, user := range msg.UserIds {
			userIds[user] = true
		}
		for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
			typed := tensorboard.(*tensorboardv1.Tensorboard)
			if msg.WorkspaceId != typed.WorkspaceId && msg.WorkspaceId != 0 {
				continue
			}
			if (len(users) == 0 && len(userIds) == 0) || users[typed.Username] || userIds[typed.UserId] {
				resp.Tensorboards = append(resp.Tensorboards, typed)
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
			ctx, t.db, t.rm, taskID, model.TaskTypeTensorboard, jobID,
			model.JobTypeTensorboard, msg.Spec, msg.ModelDef,
		); err != nil {
			ctx.Log().WithError(err).Error("failed to launch tensorboard")
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
