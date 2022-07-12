package agentrm

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
)

type slots struct{}

func (s *slots) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case []device.Device:
		for _, d := range msg {
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
	case echo.PATCH:
		patch := patchSlot{}
		if err := api.BindPatch(&patch, apiCtx); err != nil {
			ctx.Respond(errors.Wrap(err, "error patching agent"))
			return
		}
		ctx.Tell(ctx.Self().Parent(), patchSlotState{enabled: &patch.Enabled, drain: &patch.Drain})
		ctx.Respond(apiCtx.NoContent(http.StatusNoContent))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}
