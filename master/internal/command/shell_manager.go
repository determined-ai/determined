package command

import (
	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

type shellManager struct {
	db     *db.PgDB
	logger *actor.Ref
}

func (s *shellManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetShellsRequest:
		resp := &apiv1.GetShellsResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, shell := range ctx.AskAll(&shellv1.Shell{}, ctx.Children()...).GetAll() {
			if typed := shell.(*shellv1.Shell); len(users) == 0 || users[typed.Username] {
				resp.Shells = append(resp.Shells, typed)
			}
		}
		ctx.Respond(resp)

	case tasks.GenericCommandSpec:
		taskID := model.NewTaskID()
		return createGenericCommandActor(ctx, s.db, s.logger, taskID, model.TaskTypeShell, model.JobTypeShell, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
