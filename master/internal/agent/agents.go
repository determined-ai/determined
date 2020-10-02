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
	ref, ok := system.ActorOf(actor.Addr("agents"), &agents{cluster: c})
	check.Panic(check.True(ok, "agents address already taken"))
	e.Any("/agents*", api.Route(system, ref))
}

type agents struct {
	cluster *actor.Ref
}

type agentsSummary map[string]AgentSummary

func (a *agents) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case api.WebSocketConnected:
		id := msg.Ctx.QueryParam("id")
		if id == "" {
			ctx.Respond(errors.Errorf("invalid id specified: %s", id))
			return nil
		}
		if ref, ok := ctx.ActorOf(id, &agent{cluster: a.cluster}); ok {
			ctx.Respond(ctx.Ask(ref, msg).Get())
		} else {
			ctx.Respond(errors.Errorf("agent already connected: %s", id))
			return nil
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
