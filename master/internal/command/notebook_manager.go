package command

import (
	"regexp"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
)

var jupyterReadyPattern = regexp.MustCompile("Jupyter Server .*is running at")

type notebookManager struct {
	db *db.PgDB
}

func (n *notebookManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *apiv1.GetNotebooksRequest:
		resp := &apiv1.GetNotebooksResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, notebook := range ctx.AskAll(&notebookv1.Notebook{}, ctx.Children()...).GetAll() {
			if typed := notebook.(*notebookv1.Notebook); len(users) == 0 || users[typed.Username] {
				resp.Notebooks = append(resp.Notebooks, typed)
			}
		}
		ctx.Respond(resp)

	case GenericCommandReq:
		return createGenericCommandActor(ctx, n.db, msg, map[string]readinessCheck{
			"notebook": func(log sproto.ContainerLog) bool {
				return jupyterReadyPattern.MatchString(log.String())
			},
		})
	}
	return nil
}
