package agent

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	proto "github.com/determined-ai/determined/master/pkg/proto/apiv1"
)

// Initialize creates a new global agent actor.
func Initialize(s *actor.System, e *echo.Echo, c *actor.Ref) {
	_, ok := s.ActorOf(actor.Addr("agents"), &agents{cluster: c})
	check.Panic(check.True(ok, "agents address already taken"))
	e.Any("/agents*", api.Route(s))
}

type agents struct {
	cluster *actor.Ref
}

type agentsSummary map[string]agentSummary

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
	case *proto.GetAgentsRequest:
		response := &proto.GetAgentsResponse{}
		for _, a := range a.summarize(ctx) {
			if msg.Label == "" || msg.Label == a.Label {
				response.Agents = append(response.Agents, toProtoAgent(a))
			}
		}
		sort.Slice(response.Agents, func(i, j int) bool {
			a1, a2 := response.Agents[i], response.Agents[j]
			if msg.OrderBy == proto.GetAgentsRequest_ORDER_BY_DESC {
				a1, a2 = a2, a1
			}
			switch msg.SortBy {
			case proto.GetAgentsRequest_SORT_BY_TIME:
				return a1.RegisteredTime.Seconds < a2.RegisteredTime.Seconds
			case proto.GetAgentsRequest_SORT_BY_UNSPECIFIED, proto.GetAgentsRequest_SORT_BY_ID:
				return a1.Id < a2.Id
			default:
				panic(fmt.Sprintf("unknown sort type specified: %s", msg.SortBy.String()))
			}
		})
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
	results := ctx.AskAll(agentSummary{}, ctx.Children()...).GetAll()
	summary := make(map[string]agentSummary, len(results))
	for ref, result := range results {
		summary[ref.Address().String()] = result.(agentSummary)
	}
	return summary
}
