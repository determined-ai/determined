package agent

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/user"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Initialize creates a new global agents actor.
func Initialize(
	system *actor.System, e *echo.Echo, opts *aproto.MasterSetAgentOptions,
) {
	_, ok := system.ActorOf(sproto.AgentsAddr, &agents{opts: opts})
	check.Panic(check.True(ok, "agents address already taken"))
	// Route /agents and /agents/<agent id>/slots to the agents actor and slots actors.
	e.Any("/agents*", api.Route(system, nil))
	e.PATCH("/agents*", api.Route(system, nil),
		echo.MiddlewareFunc(user.GetService().ProcessAdminAuthentication))
}

type agents struct {
	opts *aproto.MasterSetAgentOptions
}

func (a *agents) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case api.WebSocketConnected:
		id := msg.Ctx.QueryParam("id")
		resourcePool := msg.Ctx.QueryParam("resource_pool")
		reconnect, err := msg.IsReconnect()
		if err != nil {
			ctx.Respond(errors.Wrapf(err, "parsing reconnect query param"))
			return nil
		}

		// Handle agent reconnecting after a network failure.
		if reconnect {
			if ctx.Child(id) != nil {
				// If the agent actor is still alive on our side when an
				// agent tries to reconnect, accept it.
				ctx.Respond(ctx.Ask(ctx.Child(id), msg).Get())
			} else {
				// In the event it has closed and the agent is trying to reconnect,
				// continue to deny it. This case is nearly impossible (master waits
				// longer than agent tries, to avoid it).
				ctx.Respond(aproto.ErrAgentMustReconnect)
			}
			return nil
		}

		// Handle agent reconnecting within the timeout after a crash/restart.
		// Based on ResourcePoolConfig AgentReattachEnabled, it will clear existing
		// containers or will try to reattach them.
		// That logic is located in agent.receive(ws.WebSocketConnected).
		if existingRef := ctx.Child(id); existingRef != nil {
			ctx.Log().Infof("restoring agent id: %s", id)
			ctx.Respond(ctx.Ask(existingRef, msg).Get())
			return nil
		}

		version := msg.Ctx.QueryParam("version")
		if ref, err := a.createAgentActor(ctx, id, version, resourcePool, a.opts); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(ctx.Ask(ref, msg).Get())
		}
	case *apiv1.GetAgentsRequest:
		response := &apiv1.GetAgentsResponse{}
		for _, a := range a.summarize(ctx) {
			response.Agents = append(response.Agents, a.ToProto())
		}
		ctx.Respond(response)
	case echo.Context:
		a.handleAPIRequest(ctx, msg)
	case actor.PreStart, actor.PostStop:
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (a *agents) createAgentActor(
	ctx *actor.Context, id, version, resourcePool string, opts *aproto.MasterSetAgentOptions,
) (*actor.Ref, error) {
	if id == "" {
		return nil, errors.Errorf("invalid agent id specified: %s", id)
	}
	if resourcePool == "" {
		ctx.Log().Info("resource pool is empty; using default resource pool: default")
		resourcePool = "default"
	}
	if err := sproto.ValidateResourcePool(ctx.Self().System(), resourcePool); err != nil {
		return nil, errors.Wrapf(err, "cannot find specified resource pool for agent %s", id)
	}

	resourcePoolRef := sproto.GetRP(ctx.Self().System(), resourcePool)
	rpConfig := ctx.Ask(resourcePoolRef, aproto.GetRPConfig{}).Get().(aproto.GetRPResponse)
	ref, ok := ctx.ActorOf(id, &agent{
		resourcePool:          resourcePoolRef,
		resourcePoolName:      resourcePool,
		version:               version,
		maxZeroSlotContainers: rpConfig.MaxZeroSlotContainers,
		agentReconnectWait:    time.Duration(rpConfig.AgentReconnectWait),
		agentReattachEnabled:  rpConfig.AgentReattachEnabled,
		opts:                  opts,
	})
	if !ok {
		return nil, errors.Errorf("agent already connected: %s", id)
	}
	return ref, nil
}

func (a *agents) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, a.summarize(ctx)))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (a *agents) summarize(ctx *actor.Context) model.AgentsSummary {
	results := ctx.AskAll(model.AgentSummary{}, ctx.Children()...).GetAll()
	summary := make(map[string]model.AgentSummary, len(results))
	for ref, result := range results {
		summary[ref.Address().String()] = result.(model.AgentSummary)
	}
	return summary
}
