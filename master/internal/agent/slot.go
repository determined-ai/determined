package agent

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/container"
	"github.com/determined-ai/determined/master/pkg/device"
	proto "github.com/determined-ai/determined/proto/pkg/apiv1"
)

type slot struct {
	resourcePool *actor.Ref
	device       device.Device
	enabled      slotEnabled
	container    *container.Container
}

type slotEnabled struct {
	deviceAdded  bool
	agentEnabled bool
	userEnabled  bool
	draining     bool
}

func (s slotEnabled) Enabled() bool {
	return s.agentEnabled && s.userEnabled
}

type patchSlot struct {
	Enabled bool `json:"enabled"`
	Drain   bool `json:"drain"`
}

func (s *slot) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		s.patch(ctx)
	case model.SlotSummary:
		ctx.Respond(s.summarize(ctx))
	case patchSlot:
		s.enabled.userEnabled = msg.Enabled
		s.enabled.draining = msg.Drain
		s.patch(ctx)
	case aproto.StartContainer:
		check.Panic(check.True(s.enabled.Enabled(), "container allocated but slot is not enabled"))
		check.Panic(check.True(s.container == nil, "container already allocated to slot"))
		s.container = &msg.Container
	case aproto.ContainerStateChanged:
		check.Panic(check.Equal(s.container.ID, msg.Container.ID, "Invalid container id sent to slot"))
		s.container = &msg.Container
		if msg.Container.State == container.Terminated {
			s.container = nil
		}
	case *proto.GetSlotRequest:
		ctx.Respond(&proto.GetSlotResponse{Slot: s.summarize(ctx).ToProto()})
	case *proto.EnableSlotRequest:
		s.enabled.userEnabled = true
		s.patch(ctx)
		ctx.Respond(&proto.EnableSlotResponse{Slot: s.summarize(ctx).ToProto()})
	case *proto.DisableSlotRequest:
		s.enabled.userEnabled = false
		s.patch(ctx)
		ctx.Respond(&proto.DisableSlotResponse{Slot: s.summarize(ctx).ToProto()})
	case echo.Context:
		s.handleAPIRequest(ctx, msg)
	case actor.PostStop:
		s.enabled.agentEnabled = false
		// Disable the draining, to make sure any running containers are killed in `patch`.
		s.enabled.draining = false
		s.patch(ctx)
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (s *slot) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, s.summarize(ctx)))
	case echo.PATCH:
		patch := patchSlot{}
		if err := api.BindPatch(&patch, apiCtx); err != nil {
			ctx.Respond(errors.Wrap(err, "error patching slot"))
			return
		}
		s.enabled.userEnabled = patch.Enabled
		s.patch(ctx)
		ctx.Respond(apiCtx.NoContent(http.StatusNoContent))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (s *slot) patch(ctx *actor.Context) {
	if s.enabled.Enabled() && !s.enabled.deviceAdded {
		s.enabled.deviceAdded = true
		add := sproto.AddDevice{DeviceID: s.deviceID(ctx)}
		if s.container != nil {
			add.ContainerID = &s.container.ID
		}
		ctx.Tell(s.resourcePool, add)
	} else if !s.enabled.Enabled() {
		agentRef := ctx.Self().Parent().Parent()

		if !s.enabled.draining && s.enabled.deviceAdded {
			s.enabled.deviceAdded = false
			remove := sproto.RemoveDevice{DeviceID: s.deviceID(ctx)}
			ctx.Tell(s.resourcePool, remove)
		}

		// On `PostStop`, draining will be already set to false, and we'll kill the container
		// whether we have the device or not.
		if !s.enabled.draining && s.container != nil {
			ctx.Tell(agentRef, sproto.KillTaskContainer{ContainerID: s.container.ID})
		}
	}
}

func (s *slot) deviceID(ctx *actor.Context) sproto.DeviceID {
	return sproto.DeviceID{Agent: ctx.Self().Parent().Parent(), Device: s.device}
}

func (s *slot) summarize(ctx *actor.Context) model.SlotSummary {
	return model.SlotSummary{
		ID:        ctx.Self().Address().Local(),
		Device:    s.device,
		Enabled:   s.enabled.Enabled(),
		Container: s.container,
		Draining:  s.enabled.draining,
	}
}
