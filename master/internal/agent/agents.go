package agent

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	aproto "github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Initialize creates a new global agent actor.
func Initialize(
	system *actor.System, e *echo.Echo, opts *aproto.MasterSetAgentOptions,
) {
	_, ok := system.ActorOf(sproto.AgentsAddr, &agents{opts: opts})
	check.Panic(check.True(ok, "agents address already taken"))
	// Route /agents and /agents/<agent id>/slots to the agents actor and slots actors.
	e.Any("/agents*", api.Route(system, nil))
}

type agents struct {
	opts *aproto.MasterSetAgentOptions
}

type agentsSummary map[string]AgentSummary

func (a *agents) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case api.WebSocketConnected:
		id, resourcePool := msg.Ctx.QueryParam("id"), msg.Ctx.QueryParam("resource_pool")
		if ref, err := a.createAgentActor(ctx, id, resourcePool, a.opts); err != nil {
			ctx.Respond(err)
		} else {
			ctx.Respond(ctx.Ask(ref, msg).Get())
		}
	case *apiv1.GetAgentsRequest:
		response := &apiv1.GetAgentsResponse{}
		for _, a := range a.summarize(ctx) {
			response.Agents = append(response.Agents, ToProtoAgent(a))
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
	ctx *actor.Context, id, resourcePool string, opts *aproto.MasterSetAgentOptions,
) (*actor.Ref, error) {
	if id == "" {
		return nil, errors.Errorf("invalid agent id specified: %s", id)
	}
	if resourcePool == "" {
		ctx.Log().Info("resource pool is empty; using default resource pool: default")
		resourcePool = "default"
	}
	if err := sproto.ValidateRP(ctx.Self().System(), resourcePool); err != nil {
		return nil, errors.Wrapf(err, "cannot find specified resource pool for agent %s", id)
	}
	ref, ok := ctx.ActorOf(id, &agent{
		resourcePool:     sproto.GetRP(ctx.Self().System(), resourcePool),
		resourcePoolName: resourcePool,
		opts:             opts,
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

func (a *agents) summarize(ctx *actor.Context) agentsSummary {
	results := ctx.AskAll(AgentSummary{}, ctx.Children()...).GetAll()
	summary := make(map[string]AgentSummary, len(results))
	for ref, result := range results {
		summary[ref.Address().String()] = result.(AgentSummary)
	}
	return summary
}
