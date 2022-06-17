package command

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/determined-ai/determined/master/pkg/actor"
)

// Remove all of event stream...
type eventManager struct{}

func newEventManager(description string) *eventManager {
	return &eventManager{}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch ctx.Message().(type) {
	case actor.PreStart, actor.PostStop:
	case echo.Context:
		fmt.Println("DO WE GET HERE?")
		ctx.Respond(echo.NewHTTPError(http.StatusNotFound, ErrAPIRemoved))
	default:
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}
