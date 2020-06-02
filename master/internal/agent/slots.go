package agent

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
)

type slots struct {
	cluster *actor.Ref
}

type slotsSummary map[string]slotSummary

func (s *slots) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case slotsSummary:
		ctx.Respond(s.summarize(ctx))
	case aproto.AgentStarted:
		for _, d := range msg.Devices {
			enabled := slotEnabled{
				agentEnabled: true,
				userEnabled:  true,
			}
			c := containerForDevice(d, msg.RecoveredContainers)
			s := &slot{cluster: s.cluster, enabled: enabled, device: d, container: c}
			_, ok := ctx.ActorOf(d.ID, s)
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

func (s *slots) summarize(ctx *actor.Context) slotsSummary {
	results := ctx.AskAll(slotSummary{}, ctx.Children()...).GetAll()
	summary := make(map[string]slotSummary, len(results))
	for ref, result := range results {
		summary[ref.Address().String()] = result.(slotSummary)
	}
	return summary
}

func (s *slots) sendToSlots(ctx *actor.Context, c container.Container, msg actor.Message) {
	for _, d := range c.Devices {
		ctx.Tell(ctx.Child(d.ID), msg)
	}
}

func containerForDevice(
	target device.Device, rcs []aproto.ContainerRecovered) *container.Container {
	for _, rc := range rcs {
		for _, d := range rc.Container.Devices {
			if d == target {
				return &rc.Container
			}
		}
	}
	return nil
}
