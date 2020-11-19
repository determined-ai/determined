package agent

import (
	"net/http"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Initialize creates a new global agent actor.
func Initialize(system *actor.System, e *echo.Echo, c *actor.Ref) {
	_, ok := system.ActorOf(actor.Addr("agents"), &agents{cluster: c})
	check.Panic(check.True(ok, "agents address already taken"))
	// Route /agents and /agents/<agent id>/slots to the agents actor and slots actors.
	e.Any("/agents*", api.Route(system, nil))
}

type agents struct {
	cluster *actor.Ref
}

type agentsSummary map[string]AgentSummary

func (a *agents) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case api.WebSocketConnected:
		id, resourcePool := msg.Ctx.QueryParam("id"), msg.Ctx.QueryParam("resource_pool")
		if ref, err := a.createAgentActor(ctx, id, resourcePool); err != nil {
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

func (a *agents) createAgentActor(ctx *actor.Context, id, resourcePool string) (*actor.Ref, error) {
	if id == "" {
		return nil, errors.Errorf("invalid agent id specified: %s", id)
	}
	ctx.Log().Infof("Creating new Agent actor with resource pool: %s", resourcePool)
	if resourcePool == "" {
		ctx.Log().Info("Resource pool was a blank string")
		resourcePool = "default"
	}
	if a.cluster.Child(resourcePool) == nil {
		return nil, errors.Errorf("cannot find specified resource pool %s for agent %s", resourcePool, id)
	}
	ctx.Log().Infof("Adding a new agent with resource pool: %s", resourcePool)
	ref, ok := ctx.ActorOf(id, &agent{resourcePool: a.cluster.Child(resourcePool), resourcePoolName: resourcePool})
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
