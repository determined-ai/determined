package agent

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
)

type slots struct {
	resourcePool *actor.Ref
}

func (s *slots) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case model.SlotsSummary:
		ctx.Respond(s.summarize(ctx))
	case aproto.AgentStarted:
		for _, d := range msg.Devices {
			enabled := slotEnabled{
				agentEnabled: true,
				userEnabled:  true,
			}
			_, ok := ctx.ActorOf(d.ID, &slot{resourcePool: s.resourcePool, enabled: enabled, device: d})
			check.Panic(check.True(ok, "error registering slot, slot %s already created", d.ID))
		}
	case aproto.StartContainer:
		s.sendToSlots(ctx, msg.Container, msg)
	case patchSlot:
		for _, child := range ctx.Children() {
			ctx.Tell(child, msg)
		}
	case aproto.ContainerStateChanged:
		s.sendToSlots(ctx, msg.Container, msg)
	case echo.Context:
		s.handleAPIRequest(ctx, msg)
	case actor.PreStart, actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (s *slots) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, s.summarize(ctx)))
	case echo.PATCH:
		patch := patchSlot{}
		if err := api.BindPatch(&patch, apiCtx); err != nil {
			ctx.Respond(errors.Wrap(err, "error patching agent"))
			return
		}
		for _, child := range ctx.Children() {
			ctx.Tell(child, patch)
		}
		ctx.Respond(apiCtx.NoContent(http.StatusNoContent))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (s *slots) summarize(ctx *actor.Context) model.SlotsSummary {
	results := ctx.AskAll(model.SlotSummary{}, ctx.Children()...).GetAll()
	summary := make(map[string]model.SlotSummary, len(results))
	for ref, result := range results {
		summary[ref.Address().String()] = result.(model.SlotSummary)
	}
	return summary
}

func (s *slots) sendToSlots(ctx *actor.Context, c container.Container, msg actor.Message) {
	for _, d := range c.Devices {
		ctx.Tell(ctx.Child(d.ID), msg)
	}
}
