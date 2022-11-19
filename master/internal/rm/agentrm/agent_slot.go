package agentrm

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

type slotProxy struct {
	device device.Device
}

type patchSlot struct {
	Enabled bool `json:"enabled"`
	Drain   bool `json:"drain"`
}

func (s *slotProxy) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case *proto.GetSlotRequest:
		result := s.handlePatchSlotState(ctx, PatchSlotState{ID: s.device.ID})
		if result != nil {
			ctx.Respond(&proto.GetSlotResponse{Slot: result.ToProto()})
		}
	case *proto.EnableSlotRequest:
		enabled := true
		result := s.handlePatchSlotState(ctx, PatchSlotState{ID: s.device.ID, Enabled: &enabled})
		if result != nil {
			ctx.Respond(&proto.EnableSlotResponse{Slot: result.ToProto()})
		}
	case *proto.DisableSlotRequest:
		enabled := false
		result := s.handlePatchSlotState(ctx, PatchSlotState{ID: s.device.ID, Enabled: &enabled})
		if result != nil {
			ctx.Respond(&proto.EnableSlotResponse{Slot: result.ToProto()})
		}
	case echo.Context:
		s.handleAPIRequest(ctx, msg)
	case actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (s *slotProxy) handlePatchSlotState(
	ctx *actor.Context, msg PatchSlotState,
) *model.SlotSummary {
	agentRef := ctx.Self().Parent().Parent()
	resp := ctx.Ask(agentRef, PatchSlotState{ID: s.device.ID})
	if err := resp.Error(); err != nil {
		ctx.Respond(err)
		return nil
	}

	result := resp.Get().(model.SlotSummary)
	return &result
}

func (s *slotProxy) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		result := s.handlePatchSlotState(ctx, PatchSlotState{ID: s.device.ID})
		if result != nil {
			ctx.Respond(apiCtx.JSON(http.StatusOK, result))
		}
	case echo.PATCH:
		patch := patchSlot{}
		if err := api.BindPatch(&patch, apiCtx); err != nil {
			ctx.Respond(errors.Wrap(err, "error patching slot"))
			return
		}
		agentRef := ctx.Self().Parent().Parent()
		resp := ctx.Ask(agentRef, PatchSlotState{ID: s.device.ID, Enabled: &patch.Enabled})
		if err := resp.Error(); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(apiCtx.NoContent(http.StatusNoContent))
		}
	default:
		if ctx.ExpectingResponse() {
			ctx.Respond(echo.ErrMethodNotAllowed)
		}
	}
}
