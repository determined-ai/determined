package agentrm

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
)

type slots struct{}

func (s *slots) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case aproto.AgentStarted:
		for _, d := range msg.Devices {
			_, ok := ctx.ActorOf(d.ID, &slotProxy{device: d})
			check.Panic(check.True(ok, "error registering slot, slot %s already created", d.ID))
		}
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
		result := ctx.Ask(ctx.Self().Parent(), model.SlotsSummary{}).Get().(model.SlotsSummary)
		ctx.Respond(apiCtx.JSON(http.StatusOK, result))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}
