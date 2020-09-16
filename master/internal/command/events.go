package command

import (
	"container/ring"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"
	"github.com/pkg/errors"

	webAPI "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
)

const defaultEventBufferSize = 200
const ctxMissingSender = "message is missing sender infromation"

// event is the union of all event types during the parent lifecycle.
type event struct {
	Snapshot Summary   `json:"snapshot"`
	ParentID string    `json:"parent_id"`
	ID       string    `json:"id"`
	Seq      int       `json:"seq"`
	Time     time.Time `json:"time"`

	ScheduledEvent *scheduler.TaskID `json:"scheduled_event"`
	// AssignedEvent is triggered when the parent was assigned to an agent.
	AssignedEvent *scheduler.ResourcesAllocated `json:"assigned_event"`
	// ContainerStartedEvent is triggered when the container started on an agent.
	ContainerStartedEvent *sproto.TaskContainerStarted `json:"container_started_event"`
	// ServiceReadyEvent is triggered when the service running in the container is ready to serve.
	// TODO: Move to ServiceReadyEvent type to a specialized event with readiness checks.
	ServiceReadyEvent *sproto.ContainerLog `json:"service_ready_event"`
	// TerminateRequestEvent is triggered when the scheduler has requested the container to
	// terminate.
	TerminateRequestEvent *scheduler.ReleaseResources `json:"terminate_request_event"`
	// ExitedEvent is triggered when the command has terminated.
	ExitedEvent *string `json:"exited_event"`
	// LogEvent is triggered when a new log message is available.
	LogEvent *string `json:"log_event"`
}

// GetEventCount is an actor message used to get the number of events in buffer.
type GetEventCount struct{}

type eventManager struct {
	bufferSize   int
	buffer       *ring.Ring
	closed       bool
	seq          int
	isTerminated bool
	logStreams   map[actor.Address]webAPI.LogsRequest // TODO actor.Ref
}

func newEventManager() *eventManager {
	return &eventManager{
		bufferSize:   defaultEventBufferSize,
		buffer:       ring.New(defaultEventBufferSize),
		logStreams:   make(map[actor.Address]webAPI.LogsRequest),
		isTerminated: false,
	}
}

// QUESTION where do we house these utilities?
func countNonNullRingValues(ring *ring.Ring) int {
	// OPT we could use log_buffer here instead of a plain ring buffer.
	count := 0
	ring.Do(func(val interface{}) {
		if val != nil {
			count++
		}
	})
	return count
}

func (e *eventManager) RemoveSusbscribers(ctx *actor.Context) {
	for actorAddr := range e.logStreams {
		// OPT this will trigger a bunch of CloseStream message that'll come back to eventManager.
		ctx.Self().System().TellAt(actorAddr, webAPI.CloseStream{})
	}
	e.logStreams = make(map[actor.Address]webAPI.LogsRequest)
}

func (e *eventManager) ProcessNewLogEvent(ctx *actor.Context, msg event) {
	// Publish.
	for streamActor, logRequest := range e.logStreams {
		// OPT we could probably use actor hirearchy to message multiple logStreamActors at once.
		if eventSatisfiesLogRequest(logRequest, &msg) {
			entry := eventToLogEntry(&msg)
			ctx.Self().System().TellAt(streamActor, *entry)
		}
	}

	// Remove terminated subscribers.
	// TODO let the logstreamactor handle this.
	if msg.TerminateRequestEvent != nil || msg.ExitedEvent != nil {
		e.isTerminated = true
		e.RemoveSusbscribers(ctx)
	}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case event:
		msg.ParentID = ctx.Self().Address().Parent().Local()
		msg.ID = uuid.New().String()
		msg.Seq = e.seq
		msg.Time = time.Now()
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
		e.ProcessNewLogEvent(ctx, msg)

	case webAPI.CloseStream:
		if ctx.Sender() == nil {
			return errors.New(ctxMissingSender)
		}
		delete(e.logStreams, ctx.Sender().Address())

	case webAPI.LogsRequest:
		total := countNonNullRingValues(e.buffer)
		offset, limit := webAPI.EffectiveOffsetNLimit(msg.Offset, msg.Limit, total)
		msg.Limit = limit
		msg.Offset = offset

		// stream existing matching entries
		logEntries := e.getLogEntries(msg)

		// CHECK is it safe to store and msg.Handler actors from actor ref pointer vs address.
		if ctx.Sender() == nil {
			return errors.New(ctxMissingSender)
		}
		for _, entry := range logEntries {
			if entry != nil {
				ctx.Tell(ctx.Sender(), *entry)
			}
		}

		if msg.Follow && !e.isTerminated {
			e.logStreams[ctx.Sender().Address()] = msg
		} else {
			ctx.Tell(ctx.Sender(), webAPI.CloseStream{})
		}

	case actor.PostStop:
		// Whenever the event manager goes does we should stop all logStreamActors.
		// QUESTION should we make logStreamActors a child of eventmanager? this would
		// automatically be handled.
		e.RemoveSusbscribers(ctx)

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
		// We rely on log entry IDs to provide pagination and since some of these events aren't actually
		// log events we'd need to notify of them about these non existing logs either by adding a new
		// attribute to our response or a sentient log entry or we could keep it simple and normalize
		// command events as log struct by setting a special message.
		// return nil, errors.New(fmt.Sprintf("event %v has no supported log message", ev))
		message = ""
	}
	return &logger.Entry{
		ID:      ev.Seq,
		Message: message,
		Time:    ev.Time,
	}
}

func eventSatisfiesLogRequest(req webAPI.LogsRequest, event *event) bool {
	return event.Seq >= req.Offset
}

func (e *eventManager) getLogEntries(req webAPI.LogsRequest) []*logger.Entry {
	events := e.buffer
	// var logs []*logger.Entry
	logs := make([]*logger.Entry, 0)

	for i := 0; i < e.bufferSize; i++ {
		if events.Value != nil {
			event := events.Value.(event)
			if eventSatisfiesLogRequest(req, &event) && (req.Limit < 1 || len(logs) < req.Limit) {
				logEntry := eventToLogEntry(&event)
				logs = append(logs, logEntry)
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
		// TODO maybe rewrite using clientEvents?
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
