package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/proxy"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/actors"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

const tickInterval = 5 * time.Second

type tensorboardManager struct {
	db            *db.PgDB
	tensorboardID int

	timeout  time.Duration
	proxyRef *actor.Ref
}

type tensorboardTick struct{}

func (t *tensorboardManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		actors.NotifyAfter(ctx, tickInterval, tensorboardTick{})
	case actor.PostStop, actor.ChildFailed, actor.ChildStopped:

	case *apiv1.GetTensorboardsRequest:
		resp := &apiv1.GetTensorboardsResponse{}
		users := make(map[string]bool)
		for _, user := range msg.Users {
			users[user] = true
		}
		for _, tensorboard := range ctx.AskAll(&tensorboardv1.Tensorboard{}, ctx.Children()...).GetAll() {
			if typed := tensorboard.(*tensorboardv1.Tensorboard); len(users) == 0 || users[typed.Username] {
				resp.Tensorboards = append(resp.Tensorboards, typed)
			}
		}
		ctx.Respond(resp)

	case tensorboardTick:
		services := ctx.Ask(t.proxyRef, proxy.GetSummary{}).Get().(map[string]proxy.Service)
		for _, boardRef := range ctx.Children() {
			boardSummary := ctx.Ask(boardRef, getSummary{}).Get().(summary)
			if boardSummary.State != container.Running.String() {
				continue
			}

			service, ok := services[string(boardSummary.ID)]
			if !ok {
				continue
			}

			if time.Now().After(service.LastRequested.Add(t.timeout)) {
				ctx.Log().Infof("killing %s due to inactivity", boardSummary.Config.Description)
				ctx.Ask(boardRef, &apiv1.KillTensorboardRequest{})
			}
		}

		actors.NotifyAfter(ctx, tickInterval, tensorboardTick{})

	case tasks.GenericCommandSpec:
		t.tensorboardID++
		taskID := model.TaskID(fmt.Sprintf("%s-%d", model.TaskTypeShell, t.tensorboardID))
		return createGenericCommandActor(ctx, t.db, taskID, msg, map[string]readinessCheck{
			"tensorboard": func(log sproto.ContainerLog) bool {
				return strings.Contains(log.String(), "TensorBoard contains metrics")
			},
		})

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}
