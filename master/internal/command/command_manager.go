package command

import (
	"fmt"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
)

type commandManager struct {
	db        *db.PgDB
	commandID int
}

func (c *commandManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop, actor.ChildFailed, actor.ChildStopped:

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

	case tasks.GenericCommandSpec:
		c.commandID++
		taskID := model.TaskID(fmt.Sprintf("%s-%d", model.TaskTypeCommand, c.commandID))
		return createGenericCommandActor(ctx, c.db, taskID, msg, nil)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
