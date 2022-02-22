package command

import (
	"container/ring"
	"net/http"
	"strconv"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	webAPI "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
)

const defaultEventBufferSize = 200

// GetEventCount is an actor message used to get the number of events in buffer.
type GetEventCount struct{}

type eventManager struct {
	bufferSize   int
	buffer       *ring.Ring
	closed       bool
	seq          int
	isTerminated bool

	description string
}

func newEventManager(description string) *eventManager {
	return &eventManager{
		bufferSize:   defaultEventBufferSize,
		buffer:       ring.New(defaultEventBufferSize),
		isTerminated: false,

		description: description,
	}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart, actor.PostStop:
	case sproto.Event:
		msg.ID = uuid.New().String()
		msg.Seq = e.seq
		e.seq++

		// Add the event to the event buffer.
		if e.bufferSize > 0 {
			e.buffer.Value = msg
			e.buffer = e.buffer.Next()
		}

		// Send the event to all active web sockets.
		for _, child := range ctx.Children() {
			if err := api.WriteSocketJSON(ctx, child, msg); err != nil {
				ctx.Log().WithError(err).Error("cannot write to socket")
			}
		}

		// The last event should be the exit status of the parent.
		if msg.ExitedEvent != nil {
			e.closed = true
			// Disconnect all web sockets listening for logs.
			for _, child := range ctx.Children() {
				child.Stop()
			}
		}

		if msg.TerminateRequestEvent != nil || msg.ExitedEvent != nil {
			e.isTerminated = true
		}

	case api.WebSocketConnected:
		follow, err := strconv.ParseBool(msg.Ctx.QueryParam("follow"))
		if msg.Ctx.QueryParam("follow") == "" {
			follow = true
		} else if err != nil {
			ctx.Respond(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
			break
		}

		tail, err := strconv.Atoi(msg.Ctx.QueryParam("tail"))
		if msg.Ctx.QueryParam("tail") == "" {
			tail = e.bufferSize
		} else if err != nil {
			ctx.Respond(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
			break
		} else if err := check.GreaterThanOrEqualTo(
			tail, 0, "tail option must be greater than 0"); err != nil {
			ctx.Respond(echo.NewHTTPError(http.StatusBadRequest, err.Error()))
			break
		}

		ws, ok := msg.Accept(ctx, nil, false)
		if !ok {
			break
		}

		events := e.buffer
		for i := 0; i < e.buffer.Len(); i++ {
			if events.Value != nil && i >= e.bufferSize-tail {
				if err := api.WriteSocketJSON(ctx, ws, events.Value); err != nil {
					ctx.Log().WithError(err).Error("cannot write to socket")
				}
			}
			events = events.Next()
		}
		if !follow || e.closed {
			ws.Stop()
		}
	case echo.Context:
		e.handleAPIRequest(ctx, msg)

	default:
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

// validEvent returns true if the given event's Seq value is within the bounds specified
// by greaterThanSeq and lessThanSeq. Note that these values can be nil, in which case the
// particular bound is not regarded.
func validEvent(e sproto.Event, greaterThanSeq, lessThanSeq *int) bool {
	if greaterThanSeq != nil && e.Seq <= *greaterThanSeq {
		return false
	}

	if lessThanSeq != nil && e.Seq >= *lessThanSeq {
		return false
	}
	return true
}

// handleAPIRequest handles HTTP API requests inbound to this actor.
func (e *eventManager) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		args := struct {
			GreaterThanID *int `query:"greater_than_id"`
			LessThanID    *int `query:"less_than_id"`
			Limit         *int `query:"tail"`
		}{}
		if err := webAPI.BindArgs(&args, apiCtx); err != nil {
			ctx.Respond(echo.NewHTTPError(http.StatusBadRequest))
			return
		}
		events := e.buffer
		clientEvents := make([]sproto.Event, 0)

		var limit int
		if args.Limit != nil {
			limit = *args.Limit
		} else {
			limit = e.bufferSize
		}

		for i := 0; i < e.bufferSize; i++ {
			if events.Value != nil {
				event := events.Value.(sproto.Event)
				if validEvent(event, args.GreaterThanID, args.LessThanID) && len(clientEvents) < limit {
					clientEvents = append(clientEvents, event)
				}
			}
			events = events.Next()
		}
		ctx.Respond(apiCtx.JSON(http.StatusOK, clientEvents))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}
