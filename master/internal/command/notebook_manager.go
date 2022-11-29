// Package command provides utilities for commands.
//nolint:dupl
package command

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/task"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

type notebookManager struct {
	db         *db.PgDB
	rm         rm.ResourceManager
	taskLogger *task.Logger
}

func (n *notebookManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		tryRestoreCommandsByType(ctx, n.db, n.rm, n.taskLogger, model.TaskTypeNotebook)

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
			if len(users) == 0 || users[typed.Username] || userIds[typed.UserId] {
				resp.Notebooks = append(resp.Notebooks, typed)
			}
		}
		ctx.Respond(resp)

	case tasks.GenericCommandSpec:
		taskID := model.NewTaskID()
		jobID := model.NewJobID()
		if err := createGenericCommandActor(
			ctx, n.db, n.rm, n.taskLogger, taskID, model.TaskTypeNotebook, jobID,
			model.JobTypeNotebook, msg,
		); err != nil {
			ctx.Log().WithError(err).Error("failed to launch notebook")
			ctx.Respond(err)
		} else {
			ctx.Respond(taskID)
		}

	case echo.Context:
		ctx.Respond(echo.NewHTTPError(http.StatusNotFound, ErrAPIRemoved))

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
