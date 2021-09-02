package command

import (
	"container/ring"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	webAPI "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/sproto"
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

// event is the union of all event types during the parent lifecycle.
type event struct {
	Snapshot summary   `json:"snapshot"`
	ParentID string    `json:"parent_id"`
	ID       string    `json:"id"`
	Seq      int       `json:"seq"`
	Time     time.Time `json:"time"`

	ScheduledEvent *model.AllocationID `json:"scheduled_event"`
	// AssignedEvent is triggered when the parent was assigned to an agent.
	AssignedEvent *sproto.ResourcesAllocated `json:"assigned_event"`
	// ContainerStartedEvent is triggered when the container started on an agent.
	ContainerStartedEvent *sproto.TaskContainerStarted `json:"container_started_event"`
	// ServiceReadyEvent is triggered when the service running in the container is ready to serve.
	// TODO: Move to ServiceReadyEvent type to a specialized event with readiness checks.
	ServiceReadyEvent *sproto.ContainerLog `json:"service_ready_event"`
	// TerminateRequestEvent is triggered when the scheduler has requested the container to
	// terminate.
	TerminateRequestEvent *sproto.ReleaseResources `json:"terminate_request_event"`
	// ExitedEvent is triggered when the command has terminated.
	ExitedEvent *string `json:"exited_event"`
	// LogEvent is triggered when a new log message is available.
	LogEvent *string `json:"log_event"`
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
}

func newEventManager() *eventManager {
	return &eventManager{
		bufferSize:   defaultEventBufferSize,
		buffer:       ring.New(defaultEventBufferSize),
		logStreams:   make(logSubscribers),
		isTerminated: false,
	}
}

func (e *eventManager) removeSusbscribers(ctx *actor.Context) {
	for actor := range e.logStreams {
		ctx.Tell(actor, webAPI.CloseStream{})
	}
	e.logStreams = nil
}

func (e *eventManager) processNewLogEvent(ctx *actor.Context, msg event) {
	for streamActor, logRequest := range e.logStreams {
		if eventSatisfiesLogRequest(logRequest, &msg) {
			entry := eventToLogEntry(&msg)
			ctx.Tell(streamActor, logger.EntriesBatch([]*logger.Entry{entry}))
		}
	}

	if msg.TerminateRequestEvent != nil || msg.ExitedEvent != nil {
		e.isTerminated = true
		e.removeSusbscribers(ctx)
	}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case event:
		msg.ParentID = ctx.Self().Address().Parent().Local()
		msg.ID = uuid.New().String()
		msg.Seq = e.seq
		msg.Time = time.Now().UTC()
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
			logEntry := eventToLogEntry(ev)
			if logEntry != nil {
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
func validEvent(e event, greaterThanSeq, lessThanSeq *int) bool {
	if greaterThanSeq != nil && e.Seq <= *greaterThanSeq {
		return false
	}

	if lessThanSeq != nil && e.Seq >= *lessThanSeq {
		return false
	}
	return true
}

func eventToLogEntry(ev *event) *logger.Entry {
	description := ev.Snapshot.Config.Description
	var message string
	switch {
	case ev.ScheduledEvent != nil:
		message = fmt.Sprintf("Scheduling %s (id: %s)", ev.ParentID, description)
	case ev.ContainerStartedEvent != nil:
		message = fmt.Sprintf("Container of %s has started", description)
	case ev.TerminateRequestEvent != nil:
		message = fmt.Sprintf("%s was requested to terminate", description)
	case ev.ExitedEvent != nil:
		message = fmt.Sprintf("%s was terminated: %s", description, *ev.ExitedEvent)
	case ev.LogEvent != nil:
		message = fmt.Sprintf(*ev.LogEvent)
	default:
		// The client could rely on logEntry IDs and since some of these events aren't actually log
		// events we'd need to notify of them about these non existing logs either by adding a new
		// attribute to our response or a sentient log entry or we could keep it simple and normalize
		// command events as log struct by setting a special message.
		message = ""
	}
	return &logger.Entry{
		ID:      ev.Seq,
		Message: message,
		Time:    ev.Time,
	}
}

func eventSatisfiesLogRequest(req webAPI.BatchRequest, event *event) bool {
	return event.Seq >= req.Offset
}

func (e *eventManager) getMatchingEvents(req webAPI.BatchRequest) []*event {
	events := e.buffer
	var logs []*event

	for i := 0; i < e.bufferSize; i++ {
		if events.Value != nil {
			event := events.Value.(event)
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
		clientEvents := make([]event, 0)

		var limit int
		if args.Limit != nil {
			limit = *args.Limit
		} else {
			limit = e.bufferSize
		}

		for i := 0; i < e.bufferSize; i++ {
			if events.Value != nil {
				event := events.Value.(event)
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
