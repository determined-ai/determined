package api

import (
	"strings"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
)

var (
	upgrader = websocket.Upgrader{}
)

// Route aims at routing HTTP and websocket requests to an actor. It returns an
// echo handler function to register with endpoints. Requests will be routed to
// the specified actor reference if presented. Otherwise, it will use the same
// path as the request path to locate the actor path.
func Route(system *actor.System, recipient *actor.Ref) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		// Locate the recipient actor using the request path if no recipient is presented.
		var addr actor.Address
		if recipient != nil {
			addr = recipient.Address()
		} else {
			addr = parseAddr(ctx.Request().URL.Path)
		}

		// Route the requests to the actor.
		if ctx.IsWebSocket() {
			return handleWSRequest(system, addr, ctx)
		}
		return handleRequest(system, addr, ctx)
	}
}

func handleWSRequest(system *actor.System, recipient actor.Address, ctx echo.Context) error {
	switch resp := system.AskAt(recipient, WebSocketConnected{Ctx: ctx}); {
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

func handleRequest(system *actor.System, recipient actor.Address, ctx echo.Context) error {
	switch resp := system.AskAt(recipient, ctx); {
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
