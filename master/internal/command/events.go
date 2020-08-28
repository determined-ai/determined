package command

import (
	"container/ring"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo"

	webAPI "github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/scheduler"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/logger"
)

const defaultEventBufferSize = 200

// event is the union of all event types during the parent lifecycle.
type event struct {
	Snapshot summary   `json:"snapshot"`
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

type requestParams struct {
	Offset int
}

type Subscriber struct {
	Send   webAPI.ServerSend
	Params requestParams
	// Handler ctx
}

type eventManager struct {
	bufferSize  int
	buffer      *ring.Ring
	closed      bool
	seq         int
	subscribers []Subscriber
}

func newEventManager() *eventManager {
	return &eventManager{
		bufferSize: defaultEventBufferSize,
		buffer:     ring.New(defaultEventBufferSize),
	}
}

func (e *eventManager) addSubscriber(s Subscriber) {
	// sync?
	msg := fmt.Sprintf("added subscriber")
	// fmt.Println(msg)
	s.Send(logger.Entry{ID: 0, Message: msg})
	e.subscribers = append(e.subscribers, s)
}

func (e *eventManager) removeSubscriber(s Subscriber) {
	// TODO
}

// type LogRequest interface {
// 	Props() LogStreamRequest
// }

func (e *eventManager) publish(ev *event) {
	// fmt.Println("publishing message")
	if ev.LogEvent == nil {
		fmt.Println("no log content")
		return
	}
	for _, s := range e.subscribers {
		// fmt.Printf("sending to subscriber %v\n", idx)
		s.Send(logger.Entry{ID: e.seq, Message: *ev.LogEvent})
	}
}

func (e *eventManager) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case Subscriber:
		e.addSubscriber(msg)
	case event:
		// str, _ := json.Marshal(msg)
		// fmt.Println(string(str))
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
		e.publish(&msg)

	case webAPI.LogStreamRequest:
		events := e.getClientEvents(msg)
		var logs []logger.Entry
		for _, event := range events {
			// FIXME use ok instead?
			evMsg, err := eventToAPILogResponse(event)
			if err != nil {
				continue
			}
			logs = append(logs, *evMsg)
		}
		ctx.Respond(logs)

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

func eventToAPILogResponse(ev event) (*logger.Entry, error) {
	// evJSON, err := json.Marshal(ev)
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "failed to marshal event")
	// }
	return &logger.Entry{
		ID: ev.Seq,
		// Message: string(evJSON),
	}, nil
}

func (e *eventManager) getClientEvents(req webAPI.LogStreamRequest) []event {
	events := e.buffer
	clientEvents := make([]event, 0)

	offset, limit := webAPI.EffectiveOffsetNLimit(req.Offset, req.Limit, e.bufferSize)

	for i := 0; i < e.bufferSize; i++ {
		if events.Value != nil {
			event := events.Value.(event)
			fmt.Printf("looking at event seq%d\n, off is %d, limit is %d \n", event.Seq, offset, limit)
			if event.Seq >= offset && (limit < 1 || len(clientEvents) < limit) {
				clientEvents = append(clientEvents, event)
			}
		}
		events = events.Next()
	}
	return clientEvents
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
