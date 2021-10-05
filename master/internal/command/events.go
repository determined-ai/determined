package command

import (
	"container/ring"
	"net/http"
	"strconv"
	"time"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	webAPI "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
)

const defaultEventBufferSize = 200
const ctxMissingSender = "message is missing sender information"

func countNonNullRingValues(ring *ring.Ring) int {
	// TODO(DET-4206) we could work on a constant time solution.
	count := 0
	ring.Do(func(val interface{}) {
		if val != nil {
			count++
		}
	})
	return count
}

type logSubscribers = map[*actor.Ref]webAPI.BatchRequest

// GetEventCount is an actor message used to get the number of events in buffer.
type GetEventCount struct{}

type eventManager struct {
	bufferSize   int
	buffer       *ring.Ring
	closed       bool
	seq          int
	isTerminated bool
	logStreams   logSubscribers

	description string
}

func newEventManager(description string) *eventManager {
	return &eventManager{
		bufferSize:   defaultEventBufferSize,
		buffer:       ring.New(defaultEventBufferSize),
		logStreams:   make(logSubscribers),
		isTerminated: false,

		description: description,
	}
}

func (e *eventManager) removeSusbscribers(ctx *actor.Context) {
	for actor := range e.logStreams {
		ctx.Tell(actor, webAPI.CloseStream{})
	}
	e.logStreams = nil
}

func (e *eventManager) processNewLogEvent(ctx *actor.Context, msg sproto.Event) {
	for streamActor, logRequest := range e.logStreams {
		if eventSatisfiesLogRequest(logRequest, &msg) {
			ctx.Tell(streamActor, logger.EntriesBatch([]*logger.Entry{msg.ToLogEntry()}))
		}
	}

	if msg.TerminateRequestEvent != nil || msg.ExitedEvent != nil {
		e.isTerminated = true
		e.removeSusbscribers(ctx)
	}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case sproto.Event:
		msg.ParentID = ctx.Self().Address().Parent().Local()
		msg.ID = uuid.New().String()
		msg.Seq = e.seq
		msg.Time = time.Now().UTC()
		msg.Description = e.description
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
		e.processNewLogEvent(ctx, msg)

	case webAPI.BatchRequest:
		if ctx.Sender() == nil {
			panic(ctxMissingSender)
		}
		ctx.Respond(true)

		total := countNonNullRingValues(e.buffer)
		msg.Offset = webAPI.EffectiveOffset(msg.Offset, total)

		matchingEvents := e.getMatchingEvents(msg)
		var logEntries []*logger.Entry
		for _, ev := range matchingEvents {
			if logEntry := ev.ToLogEntry(); logEntry != nil {
				logEntries = append(logEntries, logEntry)
			}
		}
		ctx.Tell(ctx.Sender(), logger.EntriesBatch(logEntries))

		limitMet := msg.Limit > 0 && len(matchingEvents) >= msg.Limit

		if msg.Follow && !e.isTerminated && !limitMet {
			e.logStreams[ctx.Sender()] = msg
		} else {
			ctx.Tell(ctx.Sender(), webAPI.CloseStream{})
		}

	case webAPI.CloseStream:
		if ctx.Sender() == nil {
			panic(ctxMissingSender)
		}
		delete(e.logStreams, ctx.Sender())

	case actor.PostStop:
		e.removeSusbscribers(ctx)

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

func eventSatisfiesLogRequest(req webAPI.BatchRequest, event *sproto.Event) bool {
	return event.Seq >= req.Offset
}

func (e *eventManager) getMatchingEvents(req webAPI.BatchRequest) []*sproto.Event {
	events := e.buffer
	var logs []*sproto.Event

	for i := 0; i < e.bufferSize; i++ {
		if events.Value != nil {
			event := events.Value.(sproto.Event)
			if eventSatisfiesLogRequest(req, &event) && (req.Limit < 1 || len(logs) < req.Limit) {
				logs = append(logs, &event)
			}
		}
		events = events.Next()
	}
	return logs
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
