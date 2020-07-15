package api

import (
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

var (
	upgrader = websocket.Upgrader{}
)

// Route returns an echo handler for routing requests to actors in the actor system. Requests are
// routed to the actor with the same path as the request path.
func Route(system *actor.System, recipient *actor.Ref) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if ctx.IsWebSocket() {
			return handleWSRequest(system, recipient, ctx)
		}
		return handleRequest(system, recipient, ctx)
	}
}

func handleWSRequest(system *actor.System, recipient *actor.Ref, ctx echo.Context) error {
	var addr actor.Address
	switch {
	case recipient != nil:
		addr = recipient.Address()
	default:
		addr = parseAddr(ctx.Request().URL.Path)
	}

	switch resp := system.AskAt(addr, WebSocketConnected{Ctx: ctx}); {
	case resp.Source() == nil, resp.Empty():
		// The actor could not be found or the actor did not respond.
		return echo.ErrNotFound
	case resp.Get() == nil:
		return nil
	case resp.Error() != nil:
		// The actor responded with an error.
		return resp.Error()
	default:
		switch msg := resp.Get().(type) {
		case *actor.Ref:
			return msg.AwaitTermination()
		default:
			return errors.Errorf("%s: unexpected message (%T): %v",
				ctx.Request().URL.Path, resp.Get(), resp.Get())
		}
	}
}

func handleRequest(system *actor.System, recipient *actor.Ref, ctx echo.Context) error {
	var addr actor.Address
	switch {
	case recipient != nil:
		addr = recipient.Address()
	default:
		addr = parseAddr(ctx.Request().URL.Path)
	}

	switch resp := system.AskAt(addr, ctx); {
	case resp.Source() == nil, resp.Empty():
		// The actor could not be found or the actor did not respond.
		return echo.ErrNotFound
	case resp.Get() == nil:
		return nil
	case resp.Error() != nil:
		// The actor responded with either a response or error.
		return resp.Error()
	default:
		return errors.Errorf("%s: unexpected message (%T): %v",
			ctx.Request().URL.Path, resp.Get(), resp.Get())
	}
}

func parseAddr(rawPath string) actor.Address {
	rawPath = strings.TrimPrefix(rawPath, "/")
	var parsed []interface{}
	for _, part := range strings.Split(rawPath, "/") {
		parsed = append(parsed, part)
	}
	return actor.Addr(parsed...)
}
