// Package command provides utilities for commands.
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
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

type notebookManager struct {
	db *db.PgDB
	rm rm.ResourceManager
}

func (n *notebookManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		tryRestoreCommandsByType(ctx, n.db, n.rm, model.TaskTypeNotebook)

	case actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetNotebooksRequest:
		resp := &apiv1.GetNotebooksResponse{}
		users := make(map[string]bool, len(msg.Users))
		for _, user := range msg.Users {
			users[user] = true
		}
		userIds := make(map[int32]bool, len(msg.UserIds))
		for _, user := range msg.UserIds {
			userIds[user] = true
		}
		for _, notebook := range ctx.AskAll(&notebookv1.Notebook{}, ctx.Children()...).GetAll() {
			typed := notebook.(*notebookv1.Notebook)
			if !((len(users) == 0 && len(userIds) == 0) || users[typed.Username] || userIds[typed.UserId]) {
				continue
			}
			// skip if it doesn't match the requested workspaceID if any.
			if msg.WorkspaceId != 0 && msg.WorkspaceId != typed.WorkspaceId {
				continue
			}
			resp.Notebooks = append(resp.Notebooks, typed)
		}
		ctx.Respond(resp)

	case *apiv1.DeleteWorkspaceRequest:
		ctx.TellAll(msg, ctx.Children()...)

	case tasks.GenericCommandSpec:
		taskID := model.NewTaskID()
		jobID := model.NewJobID()
		msg.CommandID = string(taskID)
		if err := createGenericCommandActor(
			ctx, n.db, n.rm, taskID, model.TaskTypeNotebook, jobID,
			model.JobTypeNotebook, msg,
		); err != nil {
			ctx.Log().WithError(err).Error("failed to launch notebook")
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
